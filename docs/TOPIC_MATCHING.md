# Topic and Elicitor Matching — v0.1 Design

## Status

The in-memory repository, constrained LLM matcher, and two-phase application
service are implemented and tested. The application-layer emotion orchestrator
now consumes them, but that orchestrator is not yet wired into the HTTP chat
route.

## Decision

Each character owns an in-memory **hot event pool with at most 32 entries**.
This is not necessarily the 32 newest events by creation time. Unresolved
`pending` events are protected so a later confirmation or disconfirmation can
still reuse the original elicitor. Evictable completed events fill the remaining
slots according to recency.

The number 32 is a v0.1 operational bound, not part of ALMA semantics. It must be
measured against real conversation traces before becoming a persistent format.

## Why Topic and Event are separate

A Topic is a continuing narrative context. An Event is one occurrence inside
that context and is the identity used as ALMA's `elicitor`.

```text
Topic T-000007: 玩家與 Lisa 的信任關係
  Event E-000101: 玩家承諾返回
  Event E-000102: 玩家隱瞞情報
  Event E-000103: Lisa 原諒玩家
```

Reusing `T-000007` as the elicitor for all three occurrences would allow
unrelated appraisals to combine. ALMA therefore receives an Event ID such as
`E-000101`, never a Topic ID.

## Position in the pipeline

```text
latest 10 dialogue utterances
          ↓
128-token Story
          ↓
Topic/Event matcher: Propose (no Pool mutation)
  - read at most 32 event candidates
  - reuse an Event or create a new Event
          ↓
Story + validated event proposal
          ↓
18 appraisal workers released concurrently
          ↓
all 18 workers complete successfully
          ↓
Topic/Event matcher: Commit
  - reject a stale Pool version
  - allocate new IDs if required
          ↓
matched tags use the committed Event ID
          ↓
POST /appraisal
```

The eventual top-level pipeline deadline starts before Topic/Event matching and
remains one minute for the whole operation. The matcher does not receive an
additional minute before the 18-worker stage.

## Domain records

```text
Topic
  id                  opaque, monotonic, never reused
  character_id
  canonical_summary
  participants[]
  created_at
  last_seen_at

Event
  id                  opaque, monotonic, never reused; ALMA elicitor
  topic_id
  canonical_summary
  event_type
  subject
  target
  status
  created_at
  last_seen_at
  resolved_at
```

Initial Event statuses:

- `pending`: a future occurrence is still awaiting an outcome;
- `active`: an ongoing or currently discussed occurrence;
- `realized`: a pending occurrence was confirmed;
- `disconfirmed`: a pending occurrence was denied or contradicted;
- `closed`: no further correlation is expected.

Topic and Event IDs remain separate from editable summaries. Updating a summary
must never change identity.

## Candidate window

The matcher receives at most 32 Event candidates for the current character.
Candidate construction in v0.1 is deterministic:

1. Enforce the same character as a hard repository boundary.
2. List `pending` events first, ordered by `last_seen_at` descending.
3. Fill remaining slots from active and completed events by `last_seen_at`
   descending.
4. Preserve stable ordering for equal timestamps by Event ID.

Because Story is currently free text, v0.1 does not pretend it can enforce
participant compatibility before semantic analysis. All Event subject, target,
status and Topic metadata are shown to the LLM. A later structured Story/Event
extractor can add deterministic participant and event-type filters without
changing the repository contract.

If more than 32 protected pending events are relevant, v0.1 returns an explicit
capacity error. It must not silently hide a pending candidate or overwrite an
elicitor that ALMA may still need.

## Pool eviction

Before inserting a new Event into a full 32-entry pool, evict the first available
entry from these groups:

1. oldest `closed` event;
2. oldest `realized` or `disconfirmed` event;
3. oldest non-protected `active` event.

A live `pending` event is never evicted merely because it is old. If all 32
entries are protected, creation fails visibly. IDs from evicted entries are
never recycled.

A Topic with no remaining Events may be removed from the in-memory Topic pool.
Its ID is still never reused.

## Matcher contract

The LLM sees the new Story plus candidate records as JSON data and receives a
maximum output budget of 128 tokens. It may select an existing candidate or
request a new identity; it cannot invent an ID.

```json
{
  "decision": "reuse",
  "topic_id": "T-000007",
  "event_id": "E-000101",
  "reason": "故事描述玩家先前承諾的返回結果。"
}
```

For a new occurrence:

```json
{
  "decision": "new",
  "topic_id": "T-000007",
  "event_summary": "玩家向 Lisa 作出新的保密承諾",
  "event_type": "promise",
  "subject": "玩家",
  "target": "Lisa",
  "status": "pending",
  "reason": "既有候選沒有描述這項承諾。"
}
```

If the Topic is also new, `topic_id` is omitted and a short
`topic_summary` is required. The repository, not the LLM, assigns all new IDs.

The program rejects:

- an ID absent from the supplied candidate set;
- `reuse` without both an allowed Topic ID and Event ID;
- `new` without the required structured event fields;
- a confirmation matched to a non-pending Event;
- free-form output outside the JSON Schema.

Ambiguity resolves to `new`. A false split loses correlation, but a false merge
can combine unrelated emotions and corrupt prospect confirmation.

## Multiple events in one Story

The long-term contract allows a Story to reference more than one Event. Each
matched appraisal result must eventually identify the Event it concerns. The
current worker result has only one Boolean and one intensity per tag, so v0.1
must initially choose one primary Event for the Story or extend the worker
result to carry `event_id` before multi-event compilation is enabled.

The system must not silently assign every tag in a multi-event Story to one
elicitor.

## Two-phase mutation safety

`TopicMatcherService.Propose` records the character Pool version and returns a
validated decision without changing Topic or Event state. The structured
pipeline must run all 18 workers under the same outer deadline and call
`Commit` only after the complete batch succeeds.

`Commit` applies the proposal only if its Pool version is still current. A
concurrent committed Story makes older proposals fail with an explicit conflict
instead of creating duplicate or incorrectly resolved Events. Timeout,
cancellation, or worker failure before Commit discards the proposal and leaves
the Pool unchanged. ALMA dispatch begins after Commit and has the separate
external atomicity limitation documented in `EMOTION_EVALUATION_PIPELINE.md`.

`Resolve` is a convenience method that proposes and commits immediately for
standalone matching tests. The full appraisal pipeline must use the two-phase
API.

## Future vector retrieval seam

Vector similarity is an optional candidate ranker, not the identity authority.
The repository interface should permit a later implementation to rank Topics
and Events by embedding while retaining the same hard filters, 32-candidate
limit, LLM decision contract, and conservative `new` fallback.

The first implementation uses structured filters and recency only. This avoids
committing to an embedding model or vector database before real pool traces are
available.

## Required tests before runtime integration

- the same pending promise reuses the same Event ID across dialogue turns;
- confirmation and disconfirmation only reuse compatible pending Events;
- similar wording about a new occurrence creates a new Event ID;
- completed events are evicted before pending events;
- a full protected pool fails without overwriting data;
- IDs are never reused after eviction;
- candidate input never exceeds 32 Events;
- the LLM cannot select an ID that was not offered;
- timeout or cancellation before Commit leaves both the Pool and ALMA unchanged.

## Code

- `internal/model/topic.go`: Topic, Event, status, decision and proposal types.
- `internal/repository/topic_event_repository.go`: 32-entry per-character Pool,
  monotonic IDs, eviction, version checking and atomic commit.
- `internal/llm/topic_matcher.go`: 128-token constrained matcher and validation.
- `internal/service/topic_matcher_service.go`: two-phase propose/commit service
  and same-character standalone serialization.
- matching tests live beside each of those implementation layers.
