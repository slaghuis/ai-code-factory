---
id: 0009
title: Telegram as the human-in-the-loop channel
status: Accepted
date: 2024-11-12
deciders: [slaghuis]
tags: [human-in-the-loop, product]
---

# ADR 0009: Telegram as the human-in-the-loop channel

## Context

Autonomous agents must sometimes pause for human decisions: approving a
deploy, choosing between options, clarifying intent. We need a channel that:

- The human already has on their phone.
- Supports inline buttons for approvals.
- Supports free-text replies for open questions.
- Is reliable in the background and does not need the laptop open.
- Can be hit from any part of the stack (agent tools, pipeline, scripts).

## Decision

Use **Telegram** via the Bot API, consumed through long-polling (no public
inbound webhook needed).

A dedicated `telegram-mcp` service owns the bot, manages session state in
SQLite, and exposes two interfaces:

- **MCP tools** (`notify`, `ask_question`, `request_approval`, `list_pending`)
  for agents.
- **HTTP endpoints** (`/pipeline/notify`, `/pipeline/approve`, polling
  endpoint) for Dagger and scripts.

Both interfaces share the same bot, hub, and session store, so a decision
tapped on the phone resolves whichever request initiated it.

## Consequences

### Positive

- No new app to install; works on the phone the human already has.
- Inline buttons give unambiguous approval UX in two taps.
- Long-polling means no inbound firewall rules or public tunnels.
- Session persistence means approvals survive restarts (see ADR 0010).
- One bot, one chat, many agents.

### Negative

- Vendor dependency on Telegram.
- Markdown formatting is restrictive; code snippets must be escaped
  carefully.
- Rate-limited to ~30 msg/sec per bot; irrelevant for human-in-the-loop but
  noteworthy if an agent misbehaves.

## Alternatives considered

- **Slack**: equally capable but requires workspace setup; Telegram's bot
  flow is lower friction for a single-user factory.
- **Email**: too slow and lacks structured responses.
- **Custom mobile app**: unjustifiable complexity for one user.
- **ntfy.sh**: good for notifications but no reply-with-decision flow.