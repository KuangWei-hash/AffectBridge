# LM Studio 與 Groq 切換筆記

這份筆記用於將 AffectBridge 的 LLM 後端在本機 LM Studio 與 GroqCloud 之間切換。兩者都使用 OpenAI-compatible API，因此 `provider` 都設為 `openai`；主要差別是 Base URL、model ID 和 API key。

## 已完成的一次性改造

`config.json` 已改用完整的 `llm.base_url`，可接受 LM Studio 的 HTTP URL 與 Groq 的 HTTPS URL。`internal/config/config.go` 會驗證 scheme 與 hostname，再將 URL 傳給 pgEdge client。API key 仍只從環境變數 `LLM_API_KEY` 讀取。

預期的設定格式：

```json
{
  "llm": {
    "provider": "openai",
    "base_url": "http://127.0.0.1:1234/v1",
    "model": "deepseek/deepseek-r1-0528-qwen3-8b",
    "max_concurrent": 1
  }
}
```

## 切換到 LM Studio

1. 啟動 LM Studio local server，確認預設 port 為 `1234`。
2. 在 LM Studio 載入要使用的 model。
3. 將 `config.json` 的 `llm` 設為：

```json
"llm": {
  "provider": "openai",
  "base_url": "http://127.0.0.1:1234/v1",
  "model": "deepseek/deepseek-r1-0528-qwen3-8b",
  "max_concurrent": 1
}
```

4. 清除之前的 Groq key，再啟動 AffectBridge：

```bash
unset LLM_API_KEY
go run ./cmd/server
```

如果 model ID 不確定，查詢 LM Studio：

```bash
curl http://127.0.0.1:1234/v1/models
```

將回傳的 model `id` 完整複製到 `config.json`。

## 切換到 Groq

1. 在 Groq Console 建立 API key。
2. 將 `config.json` 的 `llm` 設為：

```json
"llm": {
  "provider": "openai",
  "base_url": "https://api.groq.com/openai/v1",
  "model": "qwen/qwen3.6-27b",
  "max_concurrent": 4
}
```

3. 將 Groq key 放在環境變數，不要寫進 `config.json`：

```bash
export LLM_API_KEY="<Groq API key>"
go run ./cmd/server
```

也可以先將 key 放入已被 Git 忽略的 `.env`：

```dotenv
LLM_API_KEY=<Groq API key>
```

啟動前載入：

```bash
export $(grep -v '^#' .env | xargs)
go run ./cmd/server
```

## 確認 Groq model ID

Groq 的可用 model 可能隨時調整，不要只依賴這份筆記中的範例 ID。可使用：

```bash
curl https://api.groq.com/openai/v1/models \
  -H "Authorization: Bearer $LLM_API_KEY"
```

將回傳的 model `id` 完整複製到 `config.json`。正式環境優先選擇 Groq 列為 production 的 model，避免 preview model 在短期內下架造成中斷。

## 快速檢查清單

切換後若無法呼叫 LLM，依序確認：

- `provider` 是否仍為 `openai`。
- `base_url` 是否包含正確的 scheme 與 API path。
- `model` 是否與 `/models` 回傳的 ID 完全一致。
- Groq 模式下 `LLM_API_KEY` 是否已設定。
- LM Studio 模式下 local server 是否已啟動並載入 model。
- 修改設定後是否已重新啟動 AffectBridge；目前不支援運行中熱切換。
