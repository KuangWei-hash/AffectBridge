# 如何配合 ALMA REST Server

> 規劃中文件。寫於讀完 `KuangWei-hash/ALMA` 的 `AlmaRestServer.java` 與 `使用教學.md` 之後。
> 程式碼還沒實作，本文件是「要做什麼、怎麼對接」的技術紀錄。

## 目的

AffectBridge 把 LLM（透過 pgEdge）當語意理解與表達層、玩家訊息當輸入，但**角色的情緒狀態本身**由 ALMA REST server 持有與運算。

ALMA 是 **AffectBridge 的內部情緒運算依賴**，不是玩家或遊戲客戶端的 API。玩家只會呼叫 AffectBridge 公開的角色互動 API；何時呼叫 ALMA、送什麼 payload、如何處理 ALMA 回應，全部由 AffectBridge 內部決定。

```
              Player / Game Client
                       │  AffectBridge public API
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
              │  ALMA REST     │  ← private internal dependency
              │  Server        │     loopback / private network only
              │                │     Java + AffectManager
              └────────────────┘
                       │
                       ▼
                ALMA core
              (mood / emotion / decay / appraisal)
```

關鍵邊界：
- **ALMA 是 affective state 的 single source of truth**。所有 mood、emotion、decay 都在它內部。
- **AffectBridge 不保存第二份可變 affective state**。repository 可保存 AffectBridge character ID、ALMA character name、persona payload、session 與生命週期資訊，但不自己維護 mood/emotion 運算結果。
- **LLM 不憑空生出 affective state**。LLM 可以解讀事件並產生 appraisal，但最後的 mood/emotion 由 ALMA 運算。
- **ALMA REST API 永遠不直接暴露給玩家**。AffectBridge 不把 ALMA endpoint 做成一對一 proxy，也不把 ALMA 的 raw payload 或 raw error 當成公開 API contract。

## AffectBridge 內部呼叫的 ALMA endpoint

完整 endpoint 列表見 `AlmaRestServer.java` 的 `registerHandlers()`。以下只是 `internal/affect/alma` adapter 對 ALMA 發出的 outbound HTTP，**不是 AffectBridge 對玩家公開的 endpoint**：

| Method | Path | 用途 |
|---|---|---|
| `GET` | `/health` | liveness check |
| `POST` | `/characters` | 建立角色（吃完整 persona JSON）|
| `GET` | `/characters` | 列已建立角色名稱 |
| `GET` | `/affect/{name}` | 讀單一角色完整 affective state |
| `GET` | `/affect` | 列所有角色 state（管理／除錯用）|
| `POST` | `/appraisal` | 送入固定 18-tag 事件類型 + 本次強度，由 ALMA 查角色 rule |
| `POST` | `/eec` | 送入 AffectBridge 內部已完成的 appraisal |
| `POST` | `/pad` | 餵原始 PAD（未來特殊內部輸入，v1 不必要）|
| `POST` | `/pause` / `/resume` / `/step` | timer 控制（僅管理／debug 用）|

`/appraisal` 與 `/eec` 都是合法的情緒運算輸入；v1 要使用哪一條路徑，由 `ALMA翻譯系統.md` 的設計決定。同一事件不可重複送入兩條路徑。

目前不用：`/act`、`/emotion-display`、`/mood-display`、`/groups`（v1 不用多角色互動）。

### 玩家 API 與 ALMA API 的界線

- 玩家呼叫的是 AffectBridge 的 `/characters/{id}/chat` 或其他遊戲領域 API。
- AffectBridge service 根據請求內容判斷是否需要 appraisal、更新或讀取 affective state。
- 玩家不會提供 ALMA EEC、PAD、decay 或 persona schema。
- 即使 AffectBridge 未來對外提供 affect snapshot，回傳的也應是 AffectBridge 自己定義的 response DTO，不是 ALMA raw response。

## 角色建立

ALMA `POST /characters` 吃的是完整 persona payload。AffectBridge 從受信任的角色設定或 `assets/persona/Lisa.json`、`William.json` 讀取這些資料，再由內部 adapter 送給 ALMA。

**這不是把玩家 request body 直接轉送給 ALMA。** 玩家若能選擇角色，僅能選 AffectBridge 已認識的 character ID，trait、decay、appraisal rules 與 `internal_affect_appraisal` 等 ALMA 底層設定由 server 控制。

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

`persistent: false` 很重要 — 重啟 ALMA server 角色就消失，**AffectBridge** 要保留建立用 JSON 供重建，這是 server 內部的生命週期責任，不交給玩家。

## 讀 state

ALMA `GET /affect/{name}` 回傳的 raw JSON 僅存在於內部 adapter：

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

`internal/affect/alma` 應定義 ALMA wire DTO，然後映射成 AffectBridge 穩定的 `AffectSnapshot` domain model。Controller 不應直接將上面的 raw JSON 回傳給玩家。

## 餵 appraisal：LLM → EEC

> 本節只描述「直接 EEC」候選路徑，不再代表 v1 已決定只使用 `/eec`。`/appraisal` 與 `/eec` 的選擇、轉譯與 dispatch 規則統一放在 `ALMA翻譯系統.md`。

玩家只提供對話或遊戲行為。AffectBridge 內部的 semantic/appraisal 層先解讀該輸入，再由 ALMA adapter 組成 EEC。玩家不會直接呼叫 `/eec`，也不會看到 EEC wire format。

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
- 每個需要送入 ALMA 的對話或遊戲事件產生一個：`chat-{uuid}` 或 `event-{uuid}`
- uuid 36 字元 + prefix 約 50 字元，遠低於 200 上限
- 同一個領域事件的 EEC 與將來的 `EventConfirmed` / `EventDisconfirmed` 必須沿用同一 elicitor
- 若訊息無需改變 affective state，就不產生 EEC，也不因為「玩家送了一則訊息」而強制呼叫 ALMA 更新 API。

## 錯誤處理

ALMA 會回：

| Code | 意義 | AffectBridge 內部處理 |
|---|---|---|
| 400 | AffectBridge 組出的 JSON / schema / EEC 組合不對 | 記錄 ALMA raw error；視為 adapter bug，公開 API 不應回玩家 400 |
| 404 | ALMA 內沒有該 character | 嘗試依 server 保存的 persona 重建並重試一次；仍失敗則回內部服務錯誤 |
| 409 | 名稱衝突 / paused 狀態衝突 | 由生命週期或管理邏輯處理，不直接丟給玩家 |
| 413 | AffectBridge 產生的 body > 1 MiB | 視為 server 內部限制或 bug，log 並轉成內部錯誤 |
| 422 | 落入原始核心危險路徑（Love/Hate）| log warn/error；代表 adapter-side 預檢漏掉，不將 ALMA 細節暴露給玩家 |
| 500 / 連線失敗 | ALMA core 失敗或不可用 | log raw error，公開 API 轉成 AffectBridge 的 `502` / `503` 或穩定 domain error |

**重要**：ALMA 是內部依賴，raw status/body 只用於 log 與除錯。AffectBridge 對外只回自己穩定的 error contract，不吞錯誤，也不透傳 ALMA 實作細節。

Adapter-side 有兩個預檢：
1. EEC 8 種合法組合的預檢（在 `internal/affect/alma/client.go` 送之前就 fail，省一次 round trip）
2. Elicitor 長度 + reserved string 預檢（同上）

預檢失敗時不呼叫 ALMA。如果 EEC 是 AffectBridge 自己產生的，失敗代表內部 mapping bug；只有受信任的內部／管理 API 直接接收 appraisal 時，才可能將輸入問題回為 400。

## Configuration

ALMA 連線只使用專案根目錄的 `config.json`：

```json
{
  "server": {
    "port": 8080
  },
  "alma": {
    "host": "127.0.0.1",
    "port": 8081
  },
  "llm": {
    "provider": "openai",
    "host": "127.0.0.1",
    "port": 1234,
    "model": "deepseek/deepseek-r1-0528-qwen3-8b",
    "max_concurrent": 1
  }
}
```

`alma.host` 只填 hostname 或 IP，不含 `http://`；AffectBridge 會自行組成 REST base URL。這些值不得從玩家 request、header 或 query parameter 覆寫。ALMA 應只綁定 loopback 或放在受信任的 private network，不對公網暴露。

`routes.go` 的 wiring 邏輯：
- `config.json` 的 `alma.host` 與 `alma.port` 合法 → 使用 `alma.Client`
- 設定檔缺失或欄位不合法 → server 啟動失敗並回報清楚錯誤
- 不再用 `ALMA_HOME` 判斷是否啟用，也不靜默退回 noop 使玩家以為情緒運算正常

## 已知落差 / TODO

- [ ] **EEC 8 種合法組合的 adapter-side validator** 還沒寫。要在 `internal/affect/alma/` 加一個 `eec.go` 或直接放 `client.go` 裡。
- [ ] **LLM Appraisal → EEC 映射函式** 還沒寫。要在 service 層加一個 `mapToEEC(appraisal) EECInput`。
- [ ] **ALMA wire DTO 與 domain mapper** 還沒做。`internal/affect/alma` 應定義對齊 ALMA JSON 的 `AffectResponse / MoodResponse / EmotionResponse`，再映射為 AffectBridge 穩定的 `AffectSnapshot`；不要讓 ALMA wire schema 直接成為玩家 API model。
- [ ] **in-memory repository 角色定位**：保留 AffectBridge ID → ALMA name / persona payload / lifecycle metadata 對照，供 ALMA 重啟後重建角色；不保存第二份 mood/emotion state。
- [ ] **chat pipeline 中表達階段**目前給 LLM 的 prompt 是用我自己組的 `state`，要改成讀 ALMA 回的 `Affect` 結構（含 `Mood.Word`、dominant emotion 等）。
- [ ] **public API boundary** 還沒收緊。玩家只送對話／遊戲事件；直接提交 appraisal 或讀 raw affect 的 endpoint 僅能是受信任的內部、管理或測試界面。
- [x] **configuration 一致性**：`config.json` 是 server port 與 ALMA host/port 的單一來源；`internal/config` 驗證後組成 ALMA base URL。
- [ ] **streaming** 還沒接。
- [ ] **integration test** 還沒寫。要 mock 一個 `pgedgellm.Client` 介面、mock 一個 `alma.Client` 介面，驗 pipeline。

## 參考來源

- Server source：`https://github.com/KuangWei-hash/ALMA/blob/master/src/de/affect/rest/AlmaRestServer.java`
- 使用教學：`https://github.com/KuangWei-hash/ALMA/blob/master/使用教學.md`
- Go 整合指南：`https://github.com/KuangWei-hash/ALMA/blob/master/Go整合指南.md`
- 範例 payload：`https://github.com/KuangWei-hash/ALMA/blob/master/scripts/CreateCharacterExample.json`

## 相關檔案（待改）

- `internal/model/*` — 定義與 ALMA wire format 解耦的穩定 domain model / `AffectSnapshot`
- `internal/affect/alma/client.go` — 封裝內部 ALMA REST 呼叫、wire DTO、錯誤轉換
- `internal/affect/alma/mapper.go` — ALMA wire DTO ↔ AffectBridge domain model
- `ALMA輸出翻譯.md` — `AffectSnapshot` → Final Renderer 所需的情緒 context
- `internal/affect/alma/engine.go` — 對齊新 client
- `internal/affect/engine.go` — 介面可能微調
- `internal/service/*` — 配合新 model
- `internal/controller/*` — 僅暴露 AffectBridge public DTO，不暴露 ALMA raw schema/error
- `internal/repository/character_repository.go` — 只保存角色身分、persona 與 ALMA 生命週期對照，不保存 affective state 副本
- `api/routes.go` — wiring
- `config.json` — AffectBridge server、ALMA 與本機 LLM 的連線設定
- `.env.example` — 只保留遠端 provider 可能需要的 API key
