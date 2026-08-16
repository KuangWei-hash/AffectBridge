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
   │
   ├── affect.Engine       (mood + emotion dynamics)
   │
   └── repository          (character persistence)
```

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
| GET    | `/characters/{id}/affect`         | current affective snapshot             |
| POST   | `/characters/{id}/affect`         | apply an external appraisal            |
| POST   | `/characters/{id}/chat`           | send message, get reply + new state    |

## Modules

- `cmd/server` — entry point and HTTP server lifecycle
- `api` — route wiring
- `internal/config` — environment-based configuration
- `internal/model` — character, personality, mood, emotion, appraisal
- `internal/repository` — persistence (in-memory default)
- `internal/affect` — engine interface + noop backend
- `internal/affect/alma` — ALMA-backed engine and HTTP client
- `internal/llm` — provider-agnostic LLM client + appraisal prompt
- `internal/service` — business logic
- `internal/controller` — HTTP handlers
