# ALMA 輸出翻譯

> 狀態：初步設計。
>
> 本系統負責把 ALMA REST API 回傳的 raw affective state，轉換成 AffectBridge 內部可使用的 `AffectSnapshot`，再整理成 Final Renderer 能正確理解的 `RendererAffectContext`。

## 目的

ALMA 輸出的是運算狀態，包含 emotion、mood、PAD、baseline、intensity、mood tendency 與人格等資料。Final Renderer 需要的則是：

> 角色此刻的主要情緒是什麼、強度如何、整體心情偏向哪裡，以及這些狀態應如何影響表達。

因此需要專門的輸出翻譯層：

```text
ALMA REST /affect/{character}
              │
              ▼
       ALMA wire response
              │
       ┌── schema validation
       ├── domain mapping
       ├── active-state selection
       ├── intensity interpretation
       ├── context compression
       └── renderer-safe wording
              │
              ▼
    RendererAffectContext
              │
              ▼
       Final Renderer / LLM
```

## 與 ALMA 輸入翻譯的關係

```text
玩家對話
   │
   ▼
ALMA 翻譯系統
   │  /appraisal 或 /eec
   ▼
 ALMA 運算核心
   │  /affect/{character}
   ▼
ALMA 輸出翻譯
   │  RendererAffectContext
   ▼
Final Renderer
```

- `ALMA翻譯系統.md` 處理 **事件 → ALMA input**。
- `ALMA輸出翻譯.md` 處理 **ALMA state → Renderer context**。
- 兩者都不直接生成角色最終台詞。

## 核心邊界

- ALMA wire JSON 只能存在 `internal/affect/alma` adapter 邊界內。
- `AffectSnapshot` 是 AffectBridge 穩定的內部 domain model，不等於 ALMA wire DTO。
- `RendererAffectContext` 是給 Final Renderer 的精簡語意資料，不是 public API 回應。
- 本系統只解釋已經存在的 affective state，不重新評價玩家事件。
- 本系統不改寫 ALMA state，不回送 `/appraisal`、`/eec` 或 `/pad`。
- 本系統不從情緒推導未提供的事實、意圖、原因或記憶。
- 情緒可以影響語氣、措辭、表達強度、距離感與透露意願，但不能改變事實。

## ALMA raw output

ALMA `GET /affect/{name}` 主要回傳：

```text
AffectResponse
  name
  affect_computation_paused
  personality
  dominant_emotion
    name
    intensity
    baseline
    active
    elicitor
    elicited_at
    pad
    appraisal
  mood
    word
    intensity
    pleasure
    arousal
    dominance
  mood_tendency
  default_mood
  emotions[]
```

不是所有欄位都需要進入 Renderer。例如 `affect_computation_paused`、raw elicitor、數值精度與內部 appraisal 主要用於 service logic、追蹤與除錯。

## 第一層輸出：AffectSnapshot

`AffectSnapshot` 保留 AffectBridge 運作需要的結構化狀態，但與 ALMA wire format 解耦：

```text
AffectSnapshot
  character_id
  observed_at
  computation_status

  personality
    openness
    conscientiousness
    extraversion
    agreeableness
    neurotism

  dominant_emotion
    kind
    intensity
    baseline
    active
    source_event_ref       可選；內部關聯

  active_emotions[]
    kind
    intensity
    baseline
    relative_intensity

  mood
    word
    intensity_label
    pad

  mood_tendency
    word
    intensity_label
    pad

  default_mood
    word
    pad
```

`active_emotions` 只包含 `intensity > baseline` 的情緒。純 personality baseline 不應被翻譯為「角色正在強烈感受某種情緒」。

## 第二層輸出：RendererAffectContext

Final Renderer 不需要完整 raw snapshot。應組裝一個精簡、有類型且可預測的 context：

```text
RendererAffectContext
  current_mood
    label
    strength
    direction

  dominant_emotion
    label
    strength
    active

  secondary_emotions[]
    label
    strength

  mood_tendency
    label
    direction

  expression_guidance
    emotional_pressure
    approach_or_withdraw
    openness
    response_energy

  uncertainty_warnings[]
```

`expression_guidance` 只能由已明確定義的映射產生，不能用 LLM 自由發揮心理動機。如果目前尚未能為某情緒建立可驗證的行為映射，應僅傳遞 mood / emotion 狀態，交由 Renderer 依 prompt 規則處理。

## 強度翻譯

ALMA 輸出數值不應直接以長小數或內部參數名稱塞進 Renderer prompt。建議使用集中且可測試的強度映射：

```text
relative_intensity = max(0, intensity - baseline)

0                    → inactive
(0, threshold_low]   → slight
(low, medium]        → moderate
(medium, high]       → strong
(high, 1]            → overwhelming
```

門檻必須由實驗與 Renderer 行為測試校準，不在本文預先寫死。同一套映射必須對所有 request 保持確定性，不讓 LLM 每次自由解釋相同數值。

ALMA mood 已提供 `slightly`、`moderate`、`fully` 等離散強度時，應優先保留核心輸出，而不用另一套門檻重新分類 mood。

## 情緒組合原則

角色可以同時存在多種 active emotion。輸出翻譯不應只保留一個標籤，也不應將所有情緒全部塞進 prompt。

初步規則：

1. 保留 ALMA 給出的 dominant emotion。
2. 再選擇少量有顯著 `relative_intensity` 的 secondary emotions。
3. 保留情緒間可以同時存在的可能，不強制合併成單一心理結論。
4. 若 dominant emotion 僅來自 baseline 而 `active=false`，Renderer 不應將它當成當前強烈反應。
5. mood 是較中期的狀態，emotion 是較短期的反應，輸出 context 必須保持這個層次。

## 事件原因與情緒的邊界

ALMA emotion 可能帶有 elicitor 與 appraisal。它們可以用於 AffectBridge 內部追蹤「哪個事件引發這個情緒」，但：

- 不把 raw elicitor ID 放入 Renderer prompt。
- 不從 emotion name 反推玩家做過的具體事情。
- 只有當 elicitor 能連回受信任的事件資料時，才能由 orchestrator 將事件事實放入 Renderer 的事實區塊。
- 情緒可以是不合理、矛盾或來自過去累積；輸出翻譯不得擅自建立一個「合理原因」。

## Renderer 文字組裝

`RendererAffectContext` 最後映射到 `最後一層renderer.txt` 的：

```text
[當前心理與情緒狀態]
{{alma_context}}
```

建議的語意格式類似：

```text
整體心情：目前呈現較明顯的焦慮偏向，整體情緒能量偏高。
主要情緒：此刻有中等程度的責備感。
次要情緒：仍有輕微的苦惱。
表達影響：說話較警戒、反應偏快，不容易直接放下防備。
```

這段文字是給 Renderer 的內部 context，不是要直接回給玩家。Renderer 仍必須只輸出角色會說的話，不可說出 ALMA、PAD、數值或系統欄位名稱。

## 是否使用 LLM

預設不需要另外呼叫 LLM。主要翻譯應使用確定性 mapper：

```text
ALMA wire DTO
  → AffectSnapshot
  → intensity buckets
  → emotion/mood lexicon
  → RendererAffectContext
```

原因：

- 相同 ALMA state 應得到相同翻譯。
- 避免多一次 latency 與 LLM cost。
- 避免 LLM 重新解釋或改寫 ALMA 運算結果。
- 便於為每種 emotion、mood 與強度寫固定測試。

若未來要使用 LLM 將 context 在不同語言間自然化，它只能作為已結構化結果的最後 wording layer，不能改變 emotion kind、強度、active 狀態或 mood。

## 失敗與降級

- ALMA response 無法解析或缺少必要欄位時，不得將 raw JSON 直接交給 Renderer。
- 未知 emotion/mood enum 必須保留為 `unknown` 並產生 internal warning，不得憑名稱猜測意義。
- 部分 secondary emotion 失敗時可降級，但 dominant emotion 與 current mood 的必要性待 orchestrator 設計確認。
- 輸出翻譯失敗時，是否允許使用「中性 affect context」繼續回應，或應停止本輪，尚待決定。
- 所有 raw ALMA error 與 payload 只記錄於內部受控 log，不透傳給玩家。

## 初步 interface

```go
type OutputTranslator interface {
    Translate(snapshot AffectSnapshot) (RendererAffectContext, error)
}

type AffectMapper interface {
    FromALMA(response alma.AffectResponse) (AffectSnapshot, error)
}
```

ALMA HTTP client 負責取得 `alma.AffectResponse`，`AffectMapper` 建立 domain snapshot，`OutputTranslator` 再產生 Renderer context。

## 建議模塊位置

```text
internal/affect/
  snapshot.go                 AffectSnapshot domain model
  output_translator.go        OutputTranslator interface
  renderer_context.go         RendererAffectContext

internal/affect/alma/
  client.go                   GET /affect/{name}
  dto.go                      ALMA wire response DTO
  mapper.go                   ALMA DTO → AffectSnapshot

internal/affect/translate/
  translator.go               AffectSnapshot → RendererAffectContext
  intensity.go                relative intensity + buckets
  lexicon.go                  emotion/mood semantic mapping
```

實際 package 分割等 interface 定稿後再決定，不因規劃文件提前建立空 package。

## 初步工作計畫

### Phase 1：domain contract

- [ ] 確定 `AffectSnapshot` 保留的 ALMA 資訊。
- [ ] 確定 `RendererAffectContext` 契約與 Final Renderer token budget。
- [ ] 確定 active emotion、dominant emotion、secondary emotion 與 baseline 規則。
- [ ] 確定 mood tendency 是否需要進入 Renderer。

### Phase 2：mapping 與語意字典

- [ ] 實作 ALMA wire DTO → `AffectSnapshot` mapper。
- [ ] 為 ALMA emotion 與 9 種 mood word 建立明確 lexicon。
- [ ] 定義 emotion relative intensity buckets。
- [ ] 定義哪些心理描述屬於安全直接映射，哪些會越界成動機猜測。

### Phase 3：Renderer 整合

- [ ] 實作 `AffectSnapshot` → `RendererAffectContext`。
- [ ] 將 context 組裝到 `{{alma_context}}`。
- [ ] 確保 Final Renderer 只將情緒當成表達條件，不當成新事實。
- [ ] 確保玩家回應不洩漏 ALMA、PAD、內部數值或欄位名稱。

### Phase 4：測試與校準

- [ ] 為每種 ALMA emotion 建立 translation fixture。
- [ ] 為 9 種 mood word 與強度建立 fixture。
- [ ] 測試 baseline-only、單一 active emotion、多 emotion、dominant 切換與 decay。
- [ ] 測試 mood 與短期 emotion 不一致或同時存在。
- [ ] 測試相同 snapshot 永遠得到相同 context。
- [ ] 測試翻譯文字不建立原 snapshot 中不存在的事實或動機。

## 待確認問題

1. Renderer 需要所有 active emotions，還是 dominant + 最多 N 個 secondary emotions？
2. emotion strength 門檻應使用固定分段，還是依 baseline、personality 或情緒類型校準？
3. PAD 應直接轉成 energy / approach / control 等表達信號，還是只保留 mood word？
4. personality 是否每輪都需放入 Renderer，或應由另一個基本角色設定 context 提供？
5. 是否需要告訴 Renderer 情緒正在增強或衰減？單一 snapshot 是否足以支持這個判斷？
6. 輸出翻譯失敗時，使用中性 context 還是停止本輪？
7. `expression_guidance` 應由本系統產生，還是只提供結構化 emotion/mood，讓 Final Renderer 依角色表現習慣決定？

## 與其他規劃文件的邊界

- `ALMA翻譯系統.md`：將玩家對話或遊戲事件轉成 `/appraisal` 或 `/eec` 輸入。
- `如何配合ALMA_REST_SERVER.md`：處理 ALMA REST client、wire DTO、錯誤轉換與 lifecycle。
- `記憶-事實模塊.md`：提供事實與記憶；情緒輸出不能取代事實搜尋。
- `最後一層renderer.txt`：使用 `{{alma_context}}` 影響角色語氣與表達，不暴露內部 ALMA 資訊。
