# LLM 整合實作筆記

> 對應 commit 前的內部 note。寫於 pgEdge adapter 落地之後。

## 為什麼要有這層

`internal/llm` 的存在目的只有一個：**在 server 與最終接的 LLM 之間放一道隔離層**。

server code 只認 `internal/llm.Client` interface，不直接 import 任何 provider SDK（openai-go、anthropic-sdk、vllm-client、...）。日後：

- 換 provider（OpenAI → Claude → vLLM → 本地 ollama）
- 換 LLM library（pgEdge 換成別的，或自己 hand-roll HTTP）
- 抽換整個 backend（離線測試、mock、failover）

server 程式碼完全不需要動。`internal/llm` 是唯一需要改的地方。

## 架構

```
                Your Server
                     │
                     │  llm.Client.Complete(ctx, prompt, opts...)
                     ▼
              ┌─────────────┐
              │  LLM Layer  │   ← internal/llm package
              └──────┬──────┘
                     │
         ┌───────────┼───────────┐
         ▼           ▼           ▼
      OpenAI       Claude       vLLM / Ollama / Gemini / Voyage
   (via pgEdge) (via pgEdge)   (via pgEdge, OpenAI-compat or native)
```

底下三個（其實五個）provider 在 LLM Layer 內部都透過 `pgedge-go-llm-lib` 統一介接，但對 server 而言是透明的。server 只看到上方的 `Client` interface，看不到 pgEdge 也看不到任何 provider。

## 狀態

`internal/llm` 已經透過 `pgedge-go-llm-lib` 接到 LLM。不是設計，是已實作。

## 檔案對照

```
internal/llm/                 ← 我們自己的 Client interface
  ├── client.go                 · Client interface
  │                             · NoopClient（無 key 時 fallback）
  │                             · WithSystem / WithTemperature / WithMaxTokens / WithJSONMode
  ├── appraisal.go              · Appraise() = LLM 結構化 appraisal 抽取
  └── pgedge.go                 · pgedgeClient = pgEdge 適配器
                                  任何 pgEdge 支援的 provider 都走這條
        │
        ▼
  github.com/pgEdge/pgedge-go-llm-lib
        │
        ▼
  anthropic / gemini / ollama / openai / voyage
```

`pgedge.Client` 是 pgEdge 的 interface；`pgedgeClient.inner` 直接持有一個，避免多餘的指標間接。

## 設定

| env | 用途 | 預設 |
|---|---|---|
| `LLM_PROVIDER` | `openai` / `anthropic` / `gemini` / `ollama` / `voyage` | `openai` |
| `LLM_API_KEY` | provider 的 API key | （無） |
| `LLM_MODEL` | model id | `gpt-4o-mini` |
| `LLM_BASE_URL` | OpenAI-compatible proxy / 本地 ollama | （空） |

`LLM_API_KEY` 一旦有設，`api/routes.go` 自動切到 pgEdge client，沒設就 NoopClient。`NewPgEdgeClient` 失敗也會 fallback 到 Noop，避免 server 開不起來。

## 已做

- 5 個 provider 全部透過同一個 client（pgEdge 已處理各家差異）
- Interpreter 走 JSON mode（pgEdge `ResponseFormat.Type = ResponseFormatJSON`）
- pgEdge 內建 retry / timeout / observability hooks，全部免費帶進來
- prompt + system prompt 用 `pgedgellm.UserText / SystemText` 兩種 block 組合
- chat 流程（expresser 階段）只取 `BlockText` 的內容回傳

## 沒做 / 限制

- **沒串 streaming**。`pgedgeClient.Complete` 走 `Chat` 沒走 `ChatStream`。長對話或即時互動會有 TTFB 問題。
- **沒用 tools / function calling**。Interpreter 目前是純文字 JSON prompt，未來可以改用 tool 強制 schema。
- **embedding / rerank** 完全沒用，pgEdge 有支援但目前不需。
- **provider extension** 沒用（例如 Anthropic 的 `WithToolCaching` / OpenAI 的 `EmbeddingDimensions`）。需要時再加。
- **HTTP proxy**（pgEdge 內建的 SSE 對外暴露）沒用，目前 LLM 呼叫都在 Go process 內。

## 順手修的 bug

`NoopClient` 原本回傳 `"[no llm configured]"` 字串。`Appraise()` 接到後 `json.Unmarshal` 直接炸，導致無 key 時 `POST /chat` 整個 500。

修法：noop 看到 `WithJSONMode` 就回合法的零值 Appraisal JSON，否則維持原本的 placeholder 字串。詳見 `internal/llm/client.go`。

## 如何加 provider

理論上不用加。`LLM_PROVIDER=xxx` 加對應的 key 就行。如果 pgEdge 沒註冊某個 provider 才需要：

1. 換成 `import _ "github.com/pgEdge/pgedge-go-llm-lib/llm/provider/xxx"` 把全包換成單包
2. 或在 `pgedge.go` 自己寫 client 實作 `pgedgellm.Client` interface

## 下一步（如果要繼續做）

- [ ] streaming（`ChatStream`）接出來，expresser 邊生成邊吐字
- [ ] 用 pgEdge 的 `OnRetry` hook 接到 metrics，觀察重試率
- [ ] 把 Interpreter 改成 tool calling，強制 JSON schema（比 prompt 可靠）
- [ ] 寫一個小測試 fixture：mock pgEdge client，驗 Interpreter 解析邏輯
- [ ] 評估 Anthropic 的 `WithToolCaching` 對長 session 的 token 成本影響

## 相關檔案

- `internal/llm/client.go` — interface + options + noop
- `internal/llm/pgedge.go` — adapter
- `internal/llm/appraisal.go` — Interpreter
- `internal/service/chat_service.go` — pipeline 用 client 的地方
- `api/routes.go` — 自動切換 client 的地方
- `internal/config/config.go` — env var 對應
