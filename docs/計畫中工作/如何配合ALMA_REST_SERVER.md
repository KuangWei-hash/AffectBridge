# 如何配合 ALMA REST Server

> 規劃中文件。寫於讀完 `KuangWei-hash/ALMA` 的 `AlmaRestServer.java` 與 `使用教學.md` 之後。
> 程式碼還沒實作，本文件是「要做什麼、怎麼對接」的技術紀錄。

## 目的

AffectBridge 把 LLM（透過 pgEdge）當聲帶、玩家訊息當輸入，但**角色的心理狀態本身**由外部的 ALMA REST server 持有與運算。

```
                    Player
                       │  text
                       ▼
              ┌────────────────┐
              │  AffectBridge  │
              │  (this repo)   │
              │                │
              │  LLM layer     │  ← internal/llm (pgEdge)
              │  HTTP layer    │
              │  pipeline      │
              └────────┬───────┘
                       │  HTTP / REST
                       ▼
              ┌────────────────┐
              │  ALMA REST     │  ← external: KuangWei-hash/ALMA
              │  Server        │     runs on localhost:8081
              │                │     Java + AffectManager
              └────────────────┘
                       │
                       ▼
                ALMA core
              (mood / emotion / decay / appraisal)
```

關鍵邊界：
- **ALMA 是 single source of truth**。所有 mood、emotion、decay 都在它內部。
- **AffectBridge 不再保存 affective state**。in-memory repository 只留薄薄一層 character name 對照，不再保存 mood/emotion 欄位。
- **LLM 不再憑空生出 state**。LLM 收到 ALMA 給的 state 然後表達。

## 我們呼叫的 endpoint

完整 endpoint 列表見 `AlmaRestServer.java` 的 `registerHandlers()`。我們這層用到的：

| Method | Path | 用途 |
|---|---|---|
| `GET` | `/health` | liveness check |
| `POST` | `/characters` | 建立角色（吃完整 persona JSON）|
| `GET` | `/characters` | 列已建立角色名稱 |
| `GET` | `/affect/{name}` | 讀單一角色完整 affective state |
| `GET` | `/affect` | 列所有角色 state |
| `POST` | `/eec` | 餵 appraisal（外部系統已完成 appraisal）|
| `POST` | `/pad` | 餵原始 PAD（biosensor-style）|
| `POST` | `/pause` / `/resume` / `/step` | timer 控制（debug 用）|

不用：`/appraisal`（tag-based 沒彈性）、`/act`、`/emotion-display`、`/mood-display`、`/groups`（v1 不用多角色互動）。

## 角色建立

`POST /characters` 吃的是完整 persona payload。我們現有的 `assets/persona/Lisa.json` 跟 `William.json` 已經是這個格式，**直接轉送**即可。

來源（從 `handleCreateCharacter` 抓的真實驗證規則）：

- **top level required**：`name`, `personality`, `mood`, `emotion`, `appraisal`
- **top level optional**：`complex_appraisal`, `internal_affect_appraisal`
- **name**：`[A-Za-z0-9_. -]{1,80}`，**不可含 `" - "`**（群組 summary delimiter）
- **personality keys**（5 trait 必填 + `emotion_influence` 必填 + `derived` optional）
- **mood keys exact 3 個**：`decay_time` / `decay_period` / `neurotism_stability`
- **emotion keys exact 4 個**：`decay_time` / `decay_period` / `decay_function` / `baseline`
- 任何多餘或缺少的 key 直接 400，不會被靜默忽略

response 201：

```json
{
  "created": true,
  "name": "Lisa",
  "derived": false,
  "internal_affect_appraisal": false,
  "persistent": false
}
```

`persistent: false` 很重要 — 重啟 ALMA server 角色就消失，client 端要保留建立用 JSON 供重建。

## 讀 state

`GET /affect/{name}` 回傳：

```json
{
  "name": "Lisa",
  "affect_computation_paused": false,
  "personality": {
    "openness": ..., "conscientiousness": ..., "extraversion": ...,
    "agreeableness": ..., "neurotism": ...,
    "derived": false, "emotion_influence": 0.2
  },
  "dominant_emotion": {
    "name": "Joy",
    "intensity": 0.42, "baseline": 0.05, "active": true,
    "elicitor": "chat-...", "elicited_at": 1786720000000,
    "pad": { "pleasure": 0.4, "arousal": 0.2, "dominance": 0.1 },
    "appraisal": { ... }   // or null if pure baseline
  },
  "mood":         { "word": "Exuberant", "intensity": "moderate",
                    "pleasure": 0.46, "arousal": 0.37, "dominance": 0.40 },
  "mood_tendency": { ... },
  "default_mood":  { ... },
  "emotions": [ ... ]
}
```

9 個 mood word：`Exuberant, Bored, Dependent, Disdainful, Relaxed, Anxious, Docile, Hostile, Neutral`。

## 餵 appraisal：LLM → EEC

LLM 透過 `internal/llm.Appraise` 產出（`internal/model/appraisal.go`）：

| 欄位 | 值域 | 語意 |
|---|---|---|
| `agency` | `"self"` / `"other"` | 誰造成 |
| `desirability` | `[-1, 1]` | 對角色好或壞 |
| `unexpectedness` | `[0, 1]` | 多意外 |
| `blameworthiness` | `[0, 1]` | 應責備程度 |
| `praiseworthiness` | `[0, 1]` | 應讚許程度 |

EEC `POST /eec` 強制 9 個欄位（character + 6 數值 + agency + elicitor），但**只有 8 種合法非零組合**（從 `validateEecCombination` 抓）：

| 合法組合 | 對應 ALMA type |
|---|---|
| `realization` 唯一 | Prospective event 確認/否認 |
| `desirability` | Event |
| `desirability + likelihood` | Prospective Event |
| `desirability + liking` | Event concerning other |
| `praiseworthiness` | Action（agency 決定 self/other）|
| `appealingness` | Object |
| `desirability + praiseworthiness` | Event+Action compound |
| `praiseworthiness + appealingness` | Action+Object pair（**agency=other 同號會 422，Love/Hate bug**）|

### LLM → EEC 映射決策（v1）

選 **A：選一個主維度 → 對應的 1 個 EEC 組合**。`unexpectedness` 暫時丟掉。

```
if praiseworthiness > 0:
    use praiseworthiness (Action)
else if blameworthiness > 0:
    use praiseworthiness = -blameworthiness (Action, negative)
else if desirability != 0:
    use desirability (Event)
else:
    no-op (don't call /eec at all)
```

`agency` 直接轉。`liking / appealingness / likelihood / realization` 沒用到，給 0。

缺點：lossy。優點：一次 EEC call 就夠，pipeline 簡單。

未來改進方向：把 LLM prompt 改成只生 4 個對應欄位（去掉 `unexpectedness`），或拆成多次 EEC call。

## Elicitor 策略

Elicitor 是 ALMA 用來**關聯同一事件的多個信號**的穩定 ID。

規則（從 source 抓）：
- 必填，非空
- **≤ 200 字元**
- **不可為 `"alma internal emotion appraisal"` 或 `"alma internal mood appraisal"`**（reserved，會 422）

我們的策略：
- 每次 chat message 產生一個：`chat-{uuid}` 或 `chat-{timestamp_ms}-{counter}`
- uuid 36 字元 + prefix 約 50 字元，遠低於 200 上限
- 同一個 message 的 EEC 與將來的 `EventConfirmed` / `EventDisconfirmed` 必須沿用同一 elicitor

## 錯誤處理

ALMA 會回：

| Code | 意義 | 我們的處理 |
|---|---|---|
| 400 | JSON / schema / 組合不對 | 透傳錯誤給 client，log |
| 404 | character / path 不存在 | 透傳 |
| 409 | 名稱衝突 / paused 狀態衝突 | 透傳 |
| 413 | body > 1 MiB | 透傳（理論上 client 不會送這麼大）|
| 422 | 落入原始核心危險路徑（Love/Hate）| 透傳 + log warn（這代表我們 client-side 預檢漏了）|
| 500 | ALMA core 失敗 | log error + 透傳 |

**重要**：不要在 client 端吞掉 ALMA 錯誤或重新包裝，**透傳**讓 client 知道是什麼問題。

Client-side 我們有兩個預檢：
1. EEC 8 種合法組合的預檢（在 `internal/affect/alma/client.go` 送之前就 fail，省一次 round trip）
2. Elicitor 長度 + reserved string 預檢（同上）

預檢失敗直接在我們這層回 400，不要送 ALMA 讓它回 400。

## Configuration

從 `.env` / `.env.example` 來：

```bash
ALMA_ADDR=http://localhost:8081   # REST server base URL
ALMA_HOME=/path/to/alma          # optional，informational only
```

`routes.go` 的 wiring 邏輯：
- `ALMA_ADDR` 設了 → 用 `alma.Client`
- 沒設 → `affect.NewNoopEngine()`（server 仍能起，但 affect 完全不做事）

## 已知落差 / TODO

- [ ] **EEC 8 種合法組合的 client-side validator** 還沒寫。要在 `internal/affect/alma/` 加一個 `eec.go` 或直接放 `client.go` 裡。
- [ ] **LLM Appraisal → EEC 映射函式** 還沒寫。要在 service 層加一個 `mapToEEC(appraisal) EECInput`。
- [ ] **model 完全重整** 還沒做。現在的 `Character / Mood / Emotion` 跟 ALMA 的 `Affect / Mood / Emotion` 對不上。Mood 缺 `Word / Intensity`，Emotion 從 `map[string]float64` 變 `[]Emotion{...}`，Character 缺 `DominantEmotion / MoodTendency / DefaultMood / AffectComputationPaused`。
- [ ] **in-memory repository 角色定位** 還沒決定。要嘛移除（ALMA 唯一來源），要嘛只保留 name → payload 對照。
- [ ] **chat pipeline 中表達階段**目前給 LLM 的 prompt 是用我自己組的 `state`，要改成讀 ALMA 回的 `Affect` 結構（含 `Mood.Word`、dominant emotion 等）。
- [ ] **streaming** 還沒接。
- [ ] **integration test** 還沒寫。要 mock 一個 `pgedgellm.Client` 介面、mock 一個 `alma.Client` 介面，驗 pipeline。

## 參考來源

- Server source：`https://github.com/KuangWei-hash/ALMA/blob/master/src/de/affect/rest/AlmaRestServer.java`
- 使用教學：`https://github.com/KuangWei-hash/ALMA/blob/master/使用教學.md`
- Go 整合指南：`https://github.com/KuangWei-hash/ALMA/blob/master/Go整合指南.md`
- 範例 payload：`https://github.com/KuangWei-hash/ALMA/blob/master/scripts/CreateCharacterExample.json`

## 相關檔案（待改）

- `internal/model/{personality,mood,emotion,character,appraisal}.go` — 完全重整
- `internal/affect/alma/client.go` — 整個重寫
- `internal/affect/alma/engine.go` — 對齊新 client
- `internal/affect/engine.go` — 介面可能微調
- `internal/service/*` — 配合新 model
- `internal/controller/*` — 配合新 model
- `internal/repository/character_repository.go` — 簡化或移除
- `api/routes.go` — wiring
- `.env.example` — 已有 `ALMA_ADDR` 設定
