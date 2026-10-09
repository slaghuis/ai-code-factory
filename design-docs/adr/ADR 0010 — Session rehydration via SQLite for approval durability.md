---
id: 0010
title: Session rehydration via SQLite for approval durability
status: Accepted
date: 2024-11-28
deciders: [slaghuis]
tags: [reliability, human-in-the-loop]
---

# ADR 0010: Session rehydration via SQLite for approval durability

## Context

Approval sessions originally existed only in memory. A server restart (crash,
redeploy, laptop sleep) orphaned any in-flight approvals: the Telegram
message still existed with its buttons, but tapping produced "session
expired" because the in-memory `replyCh` was gone.

Overnight autonomous runs rely on approvals that may sit for hours. We need
these to survive restarts.

## Decision

Every session (`notify`, `ask`, `approval`) is persisted to a SQLite
database at creation. On startup, the Hub calls `Rehydrate`:

1. Any pending row whose `expires_at` has passed is marked `timeout`.
2. Remaining pending rows are loaded as **orphaned** `Session` structs with
   `replyCh == nil`.
3. The Telegram message IDs persist with the rows, so button taps still
   resolve the correct session.

When a user taps a button on an orphaned session, the DB row is updated and
the Telegram message is edited to indicate the resolution happened
post-restart (♻️ marker). The original waiter (the agent) has already
disconnected and will have retried; the retry can poll the resolved state
via `GET /pipeline/approve/{id}`.

A background janitor sweeps expired in-memory sessions every 30 seconds for
long-uptime cases.

## Consequences

### Positive

- Overnight approvals work: the laptop can sleep and wake, the server can be
  restarted, and the tap on the phone still records a decision.
- No duplicate Telegram messages after retry, thanks to idempotent
  `session_id` on the approval endpoint.
- Visual distinction (♻️ vs ✅) tells the user when context changed.

### Negative

- Added complexity in the Hub's state model (`Orphaned` flag, `replyCh` can
  be nil).
- SQLite becomes a hard dependency; not just a convenience.
- Pipeline retries must supply a stable `session_id` to benefit from
  idempotency.

## Alternatives considered

- **Keep in-memory only**: simpler but overnight runs become unreliable.
- **Resolve orphans silently**: possible but then the human has no idea their
  tap was late; the ♻️ marker is better UX.