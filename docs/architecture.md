# AffectBridge Architecture

> Work in progress. This document will grow alongside the implementation.

## Pipeline

```
HTTP request
   │
   ▼
controller                (parse JSON, write JSON)
   │
   ▼
service                   (business logic)
   │
   ├── llm.Client          (semantic interpretation, expression)
   ├── repository          (identity, persona, lifecycle metadata)
   │
   └── affect.Engine       (internal affect abstraction)
          └── ALMA client  (private outbound REST calls)
                   │
                   ▼
           ALMA REST Server     (private dependency; never player-facing)
```

Players and game clients call AffectBridge endpoints only. ALMA endpoints and
wire payloads are implementation details inside `internal/affect/alma`; they
must not be exposed as a one-to-one public proxy.

## Layer Boundaries

| Layer        | Depends on                              | Does not depend on |
|--------------|-----------------------------------------|--------------------|
| controller   | service                                 | repository, engine |
| service      | repository, engine, llm, model          | controller, http   |
| repository   | model                                   | service, controller|
| affect       | model                                   | llm, controller    |
| llm          | model                                   | service, controller|
| model        | —                                       | anything internal  |

## Endpoints

| Method | Path                              | Purpose                                |
|--------|-----------------------------------|----------------------------------------|
| GET    | `/healthz`                        | liveness                               |
| POST   | `/characters`                     | create a character                     |
| GET    | `/characters/{id}`                | fetch full character state             |
| GET    | `/characters/{id}/affect`         | trusted debug/admin snapshot via AffectBridge DTO |
| POST   | `/characters/{id}/affect`         | trusted internal/test appraisal input  |
| POST   | `/characters/{id}/chat`           | send message, get reply + new state    |

The player-facing flow uses `/characters/{id}/chat` (and future game-domain
event endpoints). Direct appraisal input and affect inspection are not player
APIs. If retained, they require a trusted internal/admin boundary and must not
accept or return raw ALMA wire formats.

## Modules

- `cmd/server` — entry point and HTTP server lifecycle
- `api` — route wiring
- `internal/config` — environment-based configuration
- `internal/model` — stable AffectBridge domain models and public DTO inputs
- `internal/repository` — character identity, persona, and ALMA lifecycle metadata
- `internal/affect` — engine interface + noop backend
- `internal/affect/alma` — private ALMA-backed engine, wire DTOs, mapper, and HTTP client
- `internal/llm` — provider-agnostic LLM client + appraisal prompt
- `internal/service` — business logic
- `internal/controller` — HTTP handlers
