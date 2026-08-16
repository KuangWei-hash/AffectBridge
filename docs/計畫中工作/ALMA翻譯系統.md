# ALMA 翻譯系統

> 狀態：初步設計。
>
> 本系統負責把玩家對話或受信任的遊戲事件，翻譯成 ALMA REST API 可接受的情緒運算輸入。它不直接指定最終 emotion 或 mood；最終情緒狀態仍由 ALMA 核心運算。

## 目的

玩家說出的是自然語言，ALMA 接受的則是結構化 affect input。ALMA 翻譯系統處理中間的語意差距：

```text
Player Dialogue / Trusted Game Event
                    │
                    ▼
             ALMA 翻譯系統
        ┌──事件辨識與切分
        ├──事件語意評價
        ├──選擇 ALMA 輸入層級
        ├──強度／數值產生
        └──結構與安全驗證
                    │
          ┌─────────┤
          ▼          ▼
     /appraisal        /eec
          └──────┬──────┘
                 ▼
                ALMA
          emotion / mood
```

## 核心邊界

- 玩家不會看到或直接提供 ALMA tag、EEC、elicitor 或 REST payload。
- 翻譯系統只產生「這個事件對角色意味著什麼」的 appraisal input。
- ALMA 負責根據 appraisal、personality、現有 emotion 與 mood 運算最終 affective state。
- 翻譯系統不直接寫入「Anger = 0.8」或「Mood = Hostile」。
- ALMA client 只負責傳輸與 wire DTO；不在 HTTP client 內進行語意判斷。

## 每輪處理原則

每則玩家輸入都會進入翻譯流程，但不代表每句話都必須改變情緒：

```text
玩家輸入
  → 抽取 0..N 個待評價事件
  → 每個事件產生 0..1 個 ALMA input
  → 中性／不確定事件可不送 ALMA 更新 API
```

一句話可能包含多個事件，也可能只是無需影響情緒的一般對話。系統不應為了強制產生情緒而憑空補造事件意義。

## ALMA 的兩條合法輸入路徑

`POST /appraisal` 與 `POST /eec` 都是 ALMA 正式支援的輸入。它們會進入同一個 ALMA 情緒運算核心，差別在於 appraisal 是由誰完成。

### 路徑 A：`/appraisal`

適用於翻譯系統只決定「發生哪一類事」與「本次有多強」，再讓 ALMA 讀取角色 persona 內預先定義的 appraisal 規則。

輸入：

```json
{
  "character": "Lisa",
  "tag": "BadActOther",
  "intensity": 0.8,
  "elicitor": "chat-123/event-1"
}
```

翻譯系統負責：

- 從 ALMA 固定 18 種 Basic appraisal tag 中選擇精確一種。
- 產生 `0.0～1.0` 的本次 `intensity`。
- 建立穩定且可重試的 `elicitor`。

ALMA 負責：

- 使用 tag 查找該角色 persona 中的 appraisal rule。
- 將 rule 內的 appraisal 數值依本次 `intensity` 縮放。
- 產生 EEC 並計算 emotion / mood。

18 種 tag 分為：

- Event：`GoodEvent`、`BadEvent`、與他人相關的 4 種 event、未來事件 4 種、`EventConfirmed`、`EventDisconfirmed`。
- Action：`GoodActSelf`、`BadActSelf`、`GoodActOther`、`BadActOther`。
- Object：`NiceThing`、`NastyThing`。

tag 是 ALMA 固定 enum 與角色 appraisal rule 的 key，不是自由命名的主題標籤。

### 路徑 B：`/eec`

適用於翻譯系統已完整計算本次 appraisal，希望直接把 ALMA BasicEEC 數值交給 OCC emotion 運算。

輸入：

```json
{
  "character": "Lisa",
  "desirability": 0.0,
  "praiseworthiness": -0.8,
  "appealingness": 0.0,
  "likelihood": 0.0,
  "realization": 0.0,
  "liking": 0.0,
  "agency": "other",
  "elicitor": "chat-123/event-1"
}
```

翻譯系統負責：

- 產生 `desirability`、`praiseworthiness`、`appealingness`、`likelihood`、`realization`、`liking`。
- 產生 `self` 或 `other` agency。
- 選擇 ALMA 支援的 EEC 非零欄位組合。
- 產生穩定 elicitor。

ALMA 負責：

- 直接使用 BasicEEC 運算 emotion / mood。
- 不再透過 persona 的 18-tag Basic appraisal rule 反查數值。

## 選擇路徑的原則

目前先保留兩條路徑，不在本文過早宣告唯一方案。後續需要透過實例與測試決定 v1 的預設路徑。

| 條件 | 傾向 `/appraisal` | 傾向 `/eec` |
|---|---|---|
| 希望角色 persona 定義各事件類型的固定評價 | 是 | 否 |
| 只需分類事件 + 本次強度 | 是 | 否 |
| 需要每個事件動態產生細緻 appraisal 數值 | 否 | 是 |
| 希望保留 ALMA 原生 tag-rule 層 | 是 | 否 |
| 外部系統已完成 appraisal | 否 | 是 |

同一個領域事件只能選擇其中一條路徑。不得將同一事件同時送到 `/appraisal` 與 `/eec`，否則會重複影響 ALMA state。

## 初步輸入契約

```text
TranslationRequest
  request_id             本輪穩定 ID
  character_id           AffectBridge 角色 ID
  alma_character_name    內部 ALMA 角色名稱
  interlocutor_id        當前說話者
  player_input           玩家原始訊息
  scene_context          可選；受信任的當前場景／遊戲事件
  fact_context           可選；記憶－事實模塊的輸出
```

記憶－事實模塊目前是 Deferred，因此 `fact_context` 的空值必須合法。ALMA 翻譯系統在沒有記憶時仍能以玩家當前輸入與受信任場景運作，只是不得假設未提供的過去事件、關係或動機。

## 初步輸出契約

翻譯結果使用 tagged union，避免一個結構同時帶 tag 與 EEC 而被重複送出：

```text
TranslationResult
  events[]
    event_id
    source_span           原輸入對應片段
    confidence
    route                 none / appraisal / eec

    appraisal             route=appraisal 時必填
      tag
      intensity
      elicitor

    eec                   route=eec 時必填
      desirability
      praiseworthiness
      appealingness
      likelihood
      realization
      liking
      agency
      elicitor
```

`route=none` 是正常結果，表示本事件不需更新 ALMA，不是系統錯誤。

## 系統內部分層

```text
Event Extractor
  玩家訊息 → 0..N 個可評價事件

Semantic Appraiser
  事件 + 可用 context → 事件類型、方向、強度與置信度

Route Selector
  決定 none / appraisal / eec

ALMA Input Compiler
  語意結果 → 型別安全的 AppraisalInput 或 EECInput

Validator
  欄位、數值、tag、EEC 組合、elicitor 與重複事件預檢

Dispatcher
  每個 event 只呼叫一個 ALMA endpoint
```

LLM 可用於 Event Extractor 與 Semantic Appraiser，但 LLM 不直接發 HTTP request。Compiler、Validator 與 Dispatcher 必須是可測試的確定性程式邏輯。

## 驗證規則

### `/appraisal`

- `tag` 必須是 ALMA 固定 18 種之一，且大小寫精確。
- `intensity` 必須是 JSON number `0.0～1.0`。
- `elicitor` 必填、不可超過 200 字元、不可使用 ALMA reserved value。

### `/eec`

- 六個 EEC 數值都必填，且必須在 `-1.0～1.0`。
- 未使用的數值填 `0.0`。
- `agency` 只能是 `self` 或 `other`。
- 非零欄位必須符合 ALMA 支援的 8 種 EEC 組合。
- 拒絕 ALMA 3.0 已知的不安全 Love/Hate compound。
- `elicitor` 使用與 `/appraisal` 相同的規則。

## Elicitor 與重複事件

- 每個抽取後的領域事件使用穩定 elicitor，例如 `chat-{request_id}/event-{index}`。
- 同一 request 重試時使用相同 elicitor，不產生新 ID。
- 同一事件不同 route 不得同時 dispatch。
- `EventConfirmed` / `EventDisconfirmed` 與對應的 prospective event 必須沿用同一 elicitor。
- ALMA 是 stateful backend，因此 AffectBridge 仍需要設計 dispatch ledger 或其他 idempotency 機制，不能假設單靠 elicitor 就會自動去重。

## 失敗與降級

- 無法安全理解玩家訊息時，使用 `route=none`，不憑空產生情緒評價。
- LLM 回傳無效 schema、非法 tag 或非法 EEC 組合時，不呼叫 ALMA 更新 API。
- 翻譯系統失敗不得把 raw prompt、ALMA payload 或內部錯誤暴露給玩家。
- 是否允許「本輪不更新 ALMA，但仍使用現有 `AffectSnapshot` 生成回應」作為降級策略，待上層 orchestrator 設計確認。

## 初步工作計畫

### Phase 1：契約與輸入路徑決策

- [ ] 確定 `TranslationRequest`、`TranslationResult`、`AppraisalInput`、`EECInput`。
- [ ] 決定 v1 預設使用 `/appraisal`、`/eec` 或有明確規則的 hybrid routing。
- [ ] 確定一句話允許拆分的事件數上限。
- [ ] 確定低置信度與中性訊息的 no-op 條件。

### Phase 2：語意評價

- [ ] 設計 Event Extractor 與 Semantic Appraiser prompt/schema。
- [ ] 準備 18-tag 分類與 EEC 數值評價 fixture。
- [ ] 測試讚美、侮辱、威脅、承諾、否認、不確定未來事件與中性對話。
- [ ] 測試多事件句子、否定、反諷、引用他人說法與虛構假設。

### Phase 3：Compiler、Validator 與 Dispatcher

- [ ] 實作 typed `SendAppraisal` 與 `SendEEC` client methods。
- [ ] 實作 18-tag、數值範圍、8 種 EEC 組合與 elicitor validator。
- [ ] 實作每事件單一 route dispatch。
- [ ] 實作重試與 idempotency 保護。

### Phase 4：與角色 pipeline 整合

- [ ] 在記憶搜尋與 ALMA 更新之間接入翻譯系統。
- [ ] ALMA 更新後讀取新 `AffectSnapshot`。
- [ ] 將事實 context + affect context 交給 Final Renderer。
- [ ] 記錄 route、tag/EEC kind、latency 與 no-op rate，但不在一般 log 洩漏完整玩家對話。

## 待確認問題

1. v1 應以 `/appraisal` 還是 `/eec` 為預設路徑？
2. 是否允許同一句話拆成多個 ALMA event？上限是多少？
3. 何種 context 可以影響 appraisal：當前場景、事實、關係、角色信念、當前 mood？
4. 是否要避免將當前 mood 餵回 Semantic Appraiser，以免情緒自我放大？
5. 翻譯失敗時，本輪是繼續回應還是停止？
6. `/appraisal` 的 persona rule 要如何校準，才能讓不同角色對同一 tag 有穩定但不同的反應？
7. 如果使用 hybrid routing，如何避免相似語意因 route 不同而產生不一致強度？

## 與其他系統的邊界

- `記憶-事實模塊.md`：未來可提供評價事件需要的事實、關係與過去記憶；目前可為空。
- `如何配合ALMA_REST_SERVER.md`：負責 ALMA client、wire DTO、狀態讀取與錯誤轉換。
- `ALMA輸出翻譯.md`：將 ALMA 運算後的 raw affect state 轉成 Renderer 可用的情緒 context。
- `如何利用LLM.md`：提供 provider-neutral LLM 介面；不直接發送 ALMA request。
- `最後一層renderer.txt`：使用運算後的 ALMA affect context 生成角色回應；不自己重新 appraisal。
