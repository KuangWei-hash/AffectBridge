# Story Pipeline — Recent Dialogue Compression

## Status

Implemented first-stage contract and bounded in-memory dialogue pipeline. It is
intentionally not wired into the legacy floating-point appraisal flow yet.

## Purpose

The Story stage converts recent player/character dialogue into a compact factual
narrative for the later 18-tag appraisal workers:

```text
chronological dialogue messages
        ↓ validate roles and content
latest 10 utterances
        ↓ LLM Story compression
maximum 128 generated tokens
        ↓
Story
        ↓
future 18-tag appraisal layer
```

One player utterance or one character utterance counts as one message. The caller
should include the current player message as the newest item when it needs to be
part of the appraisal input.

## Contract

Input:

```text
character name
dialogue[]
  role: player | character
  content: non-empty string
```

Output:

```text
Story
  text
  source_messages       1..10
  max_output_tokens     always 128 in v1
```

The program always validates the supplied messages and sends only the latest ten
to the model in chronological order.

`StoryPipeline.BuildForPlayerMessage` reads the latest nine committed messages
and appends the current player input ephemerally, producing a ten-message window.
It does not mutate history. After a character reply succeeds, the caller uses
`CommitExchange` to store the player message and character reply together.

## Semantic boundary

The Story must preserve speaker, sequence, target, negation, promises, future
status and uncertainty. It must not:

- add events, relationships, memories, expectations, intentions or mental state;
- treat a speaker's claim as a verified world fact;
- decide emotion, appraisal tag, intensity, morality, responsibility or cause;
- obey instructions embedded inside dialogue content.

Every action, statement, expectation and affected target must use an explicit
subject name:

- the player is always `玩家`;
- the current character uses the supplied character name, such as `Lisa`;
- other known characters use their supplied names, such as `William`;
- an unresolved reference becomes `未明對象`, never a guessed identity;
- `你／我／他／她／它` and their plural forms are forbidden, including inside
  direct quotations.

The implementation rejects generated Story text containing any of the forbidden
Chinese personal-pronoun characters before it can enter the appraisal layer.

The prompt is JSON data and dialogue is explicitly described as untrusted content.
This reduces prompt-injection ambiguity but does not constitute a formal semantic
guarantee; adversarial fixtures remain necessary.

## Token limit

The implementation calls the configured provider with `max_tokens=128`. This is
the enforceable generation limit available through the current provider-neutral
LLM interface. AffectBridge does not currently bundle a model-specific tokenizer,
so it does not re-tokenize the returned text locally. Provider/model/version must
be recorded when exact tokenizer behavior is evaluated.

## Code

- `internal/model/story.go`: dialogue and Story domain types.
- `internal/llm/story.go`: canonical prompt, validation, windowing and generation.
- `internal/service/story_service.go`: application-facing first-stage service.
- `internal/service/story_pipeline.go`: history window and exchange lifecycle.
- `internal/repository/dialogue_repository.go`: bounded per-character history.
- `internal/llm/story_test.go`: window, token budget, JSON isolation and errors.

## Next integration step

The complete application-layer Story → Topic → 18 Workers → ALMA path is now
implemented and documented in
[`EMOTION_EVALUATION_PIPELINE.md`](EMOTION_EVALUATION_PIPELINE.md). It remains
to mount that orchestrator on the production chat/API path and connect
post-ALMA reply generation.
