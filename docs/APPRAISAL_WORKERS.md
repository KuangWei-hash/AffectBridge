# 18-Worker Appraisal Pipeline

## Status

Implemented as the second structured-appraisal stage. It is callable from Go
services and is now consumed by the application-layer orchestrator documented
in [`EMOTION_EVALUATION_PIPELINE.md`](EMOTION_EVALUATION_PIPELINE.md). The
orchestrator is not yet mounted on the production HTTP chat route.

## Purpose

The pipeline evaluates one immutable Story from the viewpoint of one character:

```text
Story + character name + character-specific view
                    ↓
          common synchronized start gate
                    ↓
     18 independent LLM calls at the same time
                    ↓
 validated Boolean + semantic intensity per tag
                    ↓
 results restored to canonical 1..18 tag order
```

Every worker receives the same Story and character view. A worker cannot modify
either value or observe another worker's output.

## Prompt strategy: variant A

The first implementation deliberately uses variant A:

- every worker sees all 18 natural-language appraisal questions;
- the complete list provides the ontology boundary;
- one target question is named as the only question the worker may answer;
- only the target tag receives detailed decision and intensity rules;
- character view and Story are encoded as JSON data and are explicitly treated
  as untrusted input rather than instructions.

This is the accuracy-oriented baseline. A shorter ontology prompt must not
replace it until an evaluation shows that the shorter form preserves the same
classification quality on boundary cases.

## Input

```text
characterName   required, non-empty
characterView   optional; an explicit no-special-view marker is substituted
Story.Text      required, non-empty
```

The canonical tag order is defined by `model.AllAppraisalTags`, from
`GoodEvent` through `NastyThing`.

## Worker output

Each call requests provider-side JSON Schema output:

```json
{
  "tag": "GoodEvent",
  "matched": true,
  "reason": "玩家履行承諾，直接促進 Lisa 重視的關係。",
  "intensity": "強烈"
}
```

Each worker request uses `max_tokens=128`. This budget covers the small JSON
object and one-sentence reason while preventing unnecessarily long output.

Allowed intensity values are:

- `極輕微`
- `輕微`
- `中等`
- `強烈`
- `非常強烈`
- `極端`
- `不適用`

The program rejects malformed JSON, a tag that differs from the requested tag,
an empty reason, an unknown intensity, `matched=true` with `不適用`, and
`matched=false` with any other intensity.

## Concurrency contract

`AppraisalWorkerService.Analyze` creates all 18 goroutines first. Each goroutine
signals that it is ready and waits at a shared start gate. The service opens the
gate only after all 18 are ready, allowing every call to enter
`Client.Complete` concurrently.

There is deliberately no semaphore, worker pool, batch size, retry loop, or
queue inside this pipeline.

The complete 18-worker batch has one shared 60-second timeout. The timeout
starts before fan-out and is not reset by individual worker progress. Under
normal operation the service returns only after all 18 workers finish. At the
deadline it cancels every unfinished request, abandons the entire batch, and
returns no partial appraisal results.

The supplied `llm.Client` must therefore:

- be safe for concurrent use;
- permit at least 18 in-flight requests;
- not be wrapped in an AffectBridge `Limiter` configured below 18.

The existing `Limiter` does not queue excess work: it returns `llm.ErrBusy`
immediately. Passing a client with `max_concurrent: 4` would start all workers
but cause up to 14 workers to fail instead of waiting. Production wiring must
use the unwrapped provider client for this stage or set the outer limit to at
least 18.

## Failure behavior

The service waits for every started worker. Successful results are returned in
canonical tag order even when other workers fail. All failures are reported in
one `AppraisalWorkerBatchError`, also in canonical tag order.

Timeout and caller cancellation are batch-level failures: all collected results
are discarded, including workers that completed before cancellation. This keeps
a timed-out batch from entering the later ALMA compiler as a partial appraisal.

The pipeline does not silently retry or convert a failed tag to `matched=false`.
A missing appraisal result is different from evidence that the appraisal did
not match. The later ALMA compiler must require a complete result set unless a
separate, explicit degradation policy is designed.

## Code

- `internal/model/appraisal_tag.go`: tag, intensity, result and validation types.
- `internal/llm/appraisal_worker.go`: variant A prompt, JSON Schema and one-worker call.
- `internal/service/appraisal_worker_service.go`: synchronized 18-way fan-out and aggregation.
- `internal/llm/appraisal_worker_test.go`: prompt and output contract tests.
- `internal/service/appraisal_worker_service_test.go`: simultaneous-start and partial-failure tests.

## Next integration step

The application-layer Story → Topic → Workers → ALMA path is implemented in
`EmotionEvaluationPipeline`. Its next integration step is the production chat
route, character-view storage, and post-ALMA reply generation.
