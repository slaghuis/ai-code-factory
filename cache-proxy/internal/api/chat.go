package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/slaghuis/cache-proxy/internal/cache"
	"github.com/slaghuis/cache-proxy/internal/config"
	"github.com/slaghuis/cache-proxy/internal/costs"
	"github.com/slaghuis/cache-proxy/internal/embedder"
	"github.com/slaghuis/cache-proxy/internal/router"
	"github.com/slaghuis/cache-proxy/internal/upstream"
	"github.com/slaghuis/cache-proxy/internal/metrics"
)

type ChatHandler struct {
    Up         *upstream.Client
    Exact      *cache.Exact
    Semantic   *cache.Semantic
    Embedder   *embedder.Ollama
    Ledger     *costs.Ledger
    Escalator  *escalator.Escalator   // NEW
    Log        *slog.Logger
}

func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    cfg := config.Current()

    raw, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "read body", 400)
        return
    }
    var req ChatRequest
    if err := json.Unmarshal(raw, &req); err != nil {
        http.Error(w, "bad json", 400)
        return
    }

    tag := r.Header.Get("x-task-tag")
	targetModel := router.Route(cfg, &req, r.Header)
    policy := escalator.ResolvePolicy(cfg, r.Header)
    useEscalation := cfg.Escalation.Enabled &&
        h.Escalator != nil &&
        !req.Stream && // streaming through escalator is complex; skip for now
        len(req.Tools) == 0 && // tool-use requires raw passthrough
        policy.Mode != escalator.ModeLocal // local-only skips escalator

    if useEscalation && (policy.ShouldEscalate() || policy.ForceCloud()) {
        h.handleWithEscalation(w, r, &req, raw, policy, tag, start)
        return
    }

    // --- LEGACY PATH (streaming, tools, local-only) ---
    req.Model = targetModel
    fwdBody, _ := injectModel(raw, targetModel)
    if req.Stream {
        h.proxyStream(w, r, &req, fwdBody, targetModel, start, tag)
    } else {
        h.proxyUnary(w, r, &req, fwdBody, targetModel, start, tag)
    }
}

func (h *ChatHandler) proxyUnary(w http.ResponseWriter, r *http.Request,
	req *ChatRequest, body []byte, model string, start time.Time, tag string) {

	cfg := config.Current()
	respBody, status, err := h.Up.Call(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(respBody)

	if status != 200 {
		return
	}
	var resp ChatResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return
	}
	_ = h.Ledger.Record(cfg, model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens,
		"", time.Since(start), tag)

	metrics.Record(metrics.RecordParams{
		Model:       targetModel,
		Cache:       hit, // "exact" or "semantic"
		Tag:         tag,
		PromptTok:   resp.Usage.PromptTokens,
		OutputTok:   resp.Usage.CompletionTokens,
		Latency:     time.Since(start),
		SavedUSD:    computeCost(cfg, targetModel, resp.Usage), // what we would have paid
		SavedReason: "cache_" + hit,
	})

	// Cache store (fire and forget)
	if cfg.Cache.Enabled && cache.Cacheable(req) {
		go h.storeInCache(req, &resp, model)
	}
}

func (h *ChatHandler) proxyStream(w http.ResponseWriter, r *http.Request,
	req *ChatRequest, body []byte, model string, start time.Time, tag string) {

	cfg := config.Current()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, _ := w.(http.Flusher)

	assembled, upstreamModel, err := h.Up.Stream(r.Context(), body, func(chunk []byte) error {
		if _, err := w.Write(chunk); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	})
	if err != nil {
		h.Log.Warn("stream err", "err", err)
		return
	}

	// Record cost (approximate — streams don't give usage in older versions)
	// LiteLLM sends usage in a final chunk when enabled; otherwise estimate.
	in := estimateTokens(req)
	out := estimateTokens(&ChatRequest{
		Messages: []Message{{Role: "assistant", Content: json.RawMessage(fmt.Sprintf("%q", assembled))}},
	})
	_ = h.Ledger.Record(cfg, model, in, out, "", time.Since(start), tag)

	metrics.Record(metrics.RecordParams{
		Model:       result.ModelUsed,
		Cache:       "",
		Escalated:   result.Escalated,
		Tag:         tag,
		PromptTok:   resp.Usage.PromptTokens,
		OutputTok:   resp.Usage.CompletionTokens,
		CostUSD:     computeCost(cfg, result.ModelUsed, resp.Usage),
		SavedUSD:    computeSaved(cfg, result),
		SavedReason: savedReason(result),
		Latency:     time.Since(start),
		LocalScore:  result.Score,
		LocalDur:    result.LocalLatency,
		LocalModel:  policy.Profile.LocalModel,
	})

	if cfg.Cache.Enabled && cache.Cacheable(req) {
		resp := &ChatResponse{
			Model: upstreamModel,
			Choices: []Choice{{
				Message: ResponseMessage{Role: "assistant", Content: assembled},
				FinishReason: "stop",
			}},
			Usage: Usage{PromptTokens: in, CompletionTokens: out, TotalTokens: in + out},
		}
		go h.storeInCache(req, resp, model)
	}
}

func (h *ChatHandler) storeInCache(req *ChatRequest, resp *ChatResponse, model string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	exactKey, semText := cache.Normalize(req)
	_ = h.Exact.Put(exactKey, model, resp)

	cfg := config.Current()
	if len(semText) == 0 || len(semText) > cfg.Cache.MaxPromptChars {
		return
	}
	vec, err := h.Embedder.Embed(ctx, semText)
	if err != nil {
		h.Log.Warn("embed on store", "err", err)
		return
	}
	if err := h.Semantic.Store(ctx, vec, model, resp); err != nil {
		h.Log.Warn("semantic store", "err", err)
	}
}

func (h *ChatHandler) respond(w http.ResponseWriter, req *ChatRequest, resp *ChatResponse,
	hit, model string, start time.Time, tag string) {

	cfg := config.Current()
	w.Header().Set("Content-Type", "application/json")
	if req.Stream {
		// Convert cached response into a single SSE chunk for streaming clients.
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		chunk := map[string]any{
			"id": resp.ID, "object": "chat.completion.chunk",
			"model": resp.Model, "x_cache": hit,
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]string{
					"role":    "assistant",
					"content": resp.Choices[0].Message.Content,
				},
				"finish_reason": "stop",
			}},
		}
		b, _ := json.Marshal(chunk)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
		_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	} else {
		_ = json.NewEncoder(w).Encode(resp)
	}
	_ = h.Ledger.Record(cfg, model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens,
		hit, time.Since(start), tag)

	metrics.Record(metrics.RecordParams{
		Model:       result.ModelUsed,
		Cache:       "",
		Escalated:   result.Escalated,
		Tag:         tag,
		PromptTok:   resp.Usage.PromptTokens,
		OutputTok:   resp.Usage.CompletionTokens,
		CostUSD:     computeCost(cfg, result.ModelUsed, resp.Usage),
		SavedUSD:    computeSaved(cfg, result),
		SavedReason: savedReason(result),
		Latency:     time.Since(start),
		LocalScore:  result.Score,
		LocalDur:    result.LocalLatency,
		LocalModel:  policy.Profile.LocalModel,
	})
}

// injectModel rewrites the "model" field in the raw request body.
func injectModel(raw []byte, model string) ([]byte, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	m["model"], _ = json.Marshal(model)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// crude token estimate: 4 chars per token.
func estimateTokens(req *ChatRequest) int {
	n := 0
	for _, m := range req.Messages {
		n += len(m.Content) / 4
	}
	return n
}


func (h *ChatHandler) handleWithEscalation(w http.ResponseWriter, r *http.Request,
    req *ChatRequest, raw []byte, policy escalator.Policy, tag string, start time.Time) {

    cfg := config.Current()
    promptText := extractPromptText(req)

    localBody, err := escalator.InjectModel(raw, policy.Profile.LocalModel)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    cloudBody, err := escalator.InjectModel(raw, policy.Profile.CloudModel)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    result, err := h.Escalator.Run(r.Context(), policy, promptText, localBody, cloudBody)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadGateway)
        return
    }

    // Return the response
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("x-model-used", result.ModelUsed)
    w.Header().Set("x-escalated", fmt.Sprintf("%v", result.Escalated))
    w.Header().Set("x-score", fmt.Sprintf("%.2f", result.Score))
    if result.Escalated {
        w.Header().Set("x-escalation-reason", result.Reason)
    }
    _, _ = w.Write(result.ResponseBody)

    // Record cost & cache
    var resp ChatResponse
    _ = json.Unmarshal(result.ResponseBody, &resp)
    _ = h.Ledger.RecordEscalated(cfg, result.ModelUsed,
        resp.Usage.PromptTokens, resp.Usage.CompletionTokens,
        "", time.Since(start), tag, result.Escalated,
        result.Score, result.Reason)

	metrics.Record(metrics.RecordParams{
		Model:       result.ModelUsed,
		Cache:       "",
		Escalated:   result.Escalated,
		Tag:         tag,
		PromptTok:   resp.Usage.PromptTokens,
		OutputTok:   resp.Usage.CompletionTokens,
		CostUSD:     computeCost(cfg, result.ModelUsed, resp.Usage),
		SavedUSD:    computeSaved(cfg, result),
		SavedReason: savedReason(result),
		Latency:     time.Since(start),
		LocalScore:  result.Score,
		LocalDur:    result.LocalLatency,
		LocalModel:  policy.Profile.LocalModel,
	})

    // Only cache cloud (or local-pass) responses; never cache "would have escalated".
    if cache.Cacheable(req) {
        go h.storeInCache(req, &resp, result.ModelUsed)
    }
}

func extractPromptText(req *ChatRequest) string {
    var b strings.Builder
    for _, m := range req.Messages {
        if m.Role == "system" {
            continue
        }
        var s string
        if err := json.Unmarshal(m.Content, &s); err == nil {
            b.WriteString(s)
            b.WriteString("\n")
        }
    }
    return b.String()
}