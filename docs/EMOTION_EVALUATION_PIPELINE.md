# Emotion Evaluation Pipeline

## Status

The complete application-layer path from recent dialogue through ALMA
`POST /appraisal` is implemented and tested. It is callable as
`EmotionEvaluationPipeline.Evaluate` but is not yet mounted on the production
HTTP chat route.

## End-to-end flow

```text
9 committed utterances + current player message
                    ↓
              128-token Story
                    ↓
       Topic/Event Propose (no mutation)
                    ↓
     18 appraisal workers at the same time
                    ↓
       complete 18-result validation
                    ↓
          Topic/Event versioned Commit
                    ↓
 matched results → numeric intensity mapping
                    ↓
 ALMA POST /appraisal in canonical tag order
```

One outer context with a 60-second timeout covers Story generation, Topic/Event
matching, all 18 workers, Event commit, and ALMA dispatch. A request waiting for
another evaluation of the same character consumes the same deadline. Different
characters may be evaluated concurrently.

Dialogue history is not committed by this service. The chat layer must store
the player/character exchange only after reply generation succeeds.

## LLM calls

A normal evaluation makes 20 LLM calls:

- one Story compression call;
- one constrained Topic/Event matching call;
- 18 appraisal worker calls released together.

Story, Topic Matcher, and each appraisal Worker request at most 128 output
tokens. Topic matching produces a non-mutating proposal. Worker failure,
cancellation, or timeout before Event commit leaves the Topic/Event Pool and
ALMA unchanged.

## Complete analysis requirement

ALMA dispatch requires exactly 18 validated worker results in canonical order.
Only `matched=true` results produce external signals. `matched=false` results
are omitted rather than sent with zero intensity because ALMA Basic appraisal
does not treat zero as a guaranteed no-op.

The semantic intensity mapping is deterministic:

| Worker intensity | ALMA request intensity |
|---|---:|
| 極輕微 | 0.10 |
| 輕微 | 0.25 |
| 中等 | 0.45 |
| 強烈 | 0.65 |
| 非常強烈 | 0.85 |
| 極端 | 1.00 |

`不適用` can never be dispatched.

## ALMA request

For a matched `GoodEvent` result:

```json
{
  "character": "Lisa",
  "tag": "GoodEvent",
  "intensity": 0.65,
  "elicitor": "E-000001"
}
```

The ALMA client sends this exact object to `POST /appraisal` with the shared
pipeline context. It accepts only a successful HTTP response whose
acknowledgement matches character, tag, and elicitor.

Signals are sent sequentially in canonical 1..18 tag order. This avoids
concurrent access to ALMA's single pending Event/Action/Object EEC slots.

## External atomicity limitation

The current ALMA REST adaptor accepts one Basic tag per request and exposes no
transaction or rollback endpoint. AffectBridge validates the entire local batch
before the first write, but if ALMA accepts one tag and a later HTTP call fails,
the earlier emotion cannot be rolled back.

`AppraisalDispatchError` therefore reports:

- the failed signal;
- how many earlier signals were accepted;
- all receipts collected before failure.

The pipeline does not retry automatically because replaying an already accepted
tag may duplicate affect. True all-or-nothing multi-tag application requires an
ALMA adaptor batch endpoint with pre-validation and a defined core failure
policy. Until that exists, callers must treat a partial dispatch error as an
explicit reconciliation condition.

The Topic/Event identity is committed immediately before dispatch because a new
Event ID is required as the ALMA elicitor. A failure during ALMA dispatch can
therefore leave a committed Event with partially applied affect; the returned
report makes that state visible.

## Code

- `internal/model/appraisal_signal.go`: ALMA Basic request and receipt types.
- `internal/affect/appraisal_sender.go`: provider-neutral sender interface.
- `internal/affect/alma/client.go`: context-aware `POST /appraisal` client.
- `internal/service/appraisal_dispatcher.go`: complete-batch validation,
  intensity mapping, matched-tag filtering, and ordered dispatch.
- `internal/service/emotion_evaluation_pipeline.go`: shared-deadline orchestrator.
- HTTP, mapping, timeout, no-mutation, and end-to-end tests live beside those
  files.

## Remaining integration work

- mount the orchestrator on the chosen production API/chat path;
- supply persistent character-specific views;
- create/recreate the corresponding ALMA runtime character before evaluation;
- decide how to reconcile a partial multi-tag ALMA failure;
- query `/affect/{name}` and feed the resulting state into reply generation;
- commit the completed dialogue exchange.
