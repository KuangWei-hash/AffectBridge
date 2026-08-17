# ALMA 輸出翻譯

> 狀態：十輪研究完成；可進入實作，尚未部署或完成 Final Renderer 模型實測。
> 研究日期：2026-08-17（Asia/Taipei）。
> ALMA 證據版本：`KuangWei-hash/ALMA` commit `7c6eef75efae4dbfa7798c43900f22d7a4f001be`。

本系統把 ALMA REST raw affect state 轉成 AffectBridge 穩定 domain，再投影成 Final Renderer 可安全使用的情緒 context。它只解釋既有狀態，不重新 appraisal、不生成角色台詞，也不從情緒補造事件、對象、意圖、關係或行為。

## 決策摘要

v1 建議採用：

```text
ALMA wire response
  → bounded decode + source contract validation
  → canonical AffectSnapshot
  → typed/versioned SemanticAffectContext
  → deterministic layered-text formatter（S4）
  → {{alma_context}}
  → Final Renderer
```

具體選擇：

- **主要方案：S4 規則式分層文字**。固定呈現「整體 mood／短期 dominant／少量 secondary／target status」，保留時間尺度與混合情緒；不建立因果故事。
- **安全 fallback：S2 compact typed phrase**。同一 semantic core 的更短格式；formatter或budget問題時可回退，不需要另一套 lexicon。
- **保留實驗：S6 compact typed JSON**。在相同 semantic core 上與 S4做 E2 paired comparison；未證實本地 2B或 Groq qwen更遵循 JSON，因此不作 v1預設。
- **S1/S3/S5** 分別保留作 enum baseline、可選維度 probe、模板多樣性實驗；v1不單獨使用。
- **S7/S8 預設關閉**。額外 wording LLM或分級 router只有在 deterministic方案通過後，且實測證明 semantic non-inferiority與自然度收益時才考慮。

這項推薦分成兩種證據級別：

- **已由 source/code/離線規格支持**：wire/domain分層、relative intensity、target binding、安全 lexicon、deterministic semantic core、無額外 LLM、failure containment與可測性。
- **仍待 E2 模型實測**：S4相對 S2/S6在 Groq `qwen/qwen3.6-27b`與本地 `qwen3.5-2b`的 hallucination、自然度與小模型遵循差異。因此 S4是最符合現有文字 Renderer與證據的 provisional default，不是假裝已跑出的冠軍。

## 為何不使用額外 LLM 翻譯

ALMA output translation 的核心是資料契約與語意限制，不是創作。確定性方案可做到：

- 相同 snapshot得到相同 context；
- 本地/離線可工作，沒有第二次 network/TTFT；
- kind、strength、active、target與層次可作 golden/metamorphic test；
- provider outage不會同時讓「翻譯」與 Final Renderer失效；
- 避免 wording LLM把 unknown target寫成玩家，或加入原因/行為。

第 04 輪對 canonical snapshot 的靜態估算：S4約 70–105 affect input tokens；S7還需約 150–240 wording input、55–85 wording output，再把其輸出送 Final Renderer。這只是敏感度區間，正式 token/cost須取 provider usage；但已足以顯示 S7新增一次完整生成與失敗面。

## 系統邊界與 authority

```text
玩家／受信任事件
  → ALMA 輸入翻譯（/appraisal 或 /eec）
  → ALMA affect runtime
  → GET /affect/{character}
  → 本輸出翻譯
  → Final Renderer
```

- raw ALMA DTO只能存在 `internal/affect/alma`。
- `AffectSnapshot` 是 provider-neutral domain，不等於 `model.Character`或 public API。
- `SemanticAffectContext` 只含 Renderer有權使用的 typed semantics。
- formatter只改 representation，不能查 lexicon、改 strength、重排或增加語意。
- facts、persona、relationship、memory與situational behavior決定可說內容；affect只調整表達狀態。
- raw elicitor/appraisal/unknown enum/API error不進 Renderer。
- Final Renderer只輸出角色台詞，不說出 ALMA、PAD、enum、schema或 constraint。

## ALMA source contract

### Wire response

`GET /affect/{name}` 主要回：

```text
AffectResponse
  name
  affect_computation_paused
  personality
  dominant_emotion
  mood
  mood_tendency
  default_mood
  emotions[]
```

adapter必須用 nullable wire DTO忠實 decode；不要直接 decode到 domain enum，以免新 enum令整包失敗。

### 已確認語義

- `emotions[]` 是 runtime-enabled完整 vector，包含 inactive/baseline-only，不是 active list。
- invariant：`0 <= baseline <= intensity <= 1`。
- `relative_intensity = intensity - baseline`。
- canonical active：嚴格 `intensity > baseline`。
- dominant依 relative intensity；全部 relative為 0 時為 `Undefined`。
- equal-relative tie沒有可靠 wire順序；domain須 stable secondary sort。
- response同一 lock範圍產 dominant與list，可視為 coherent snapshot。
- response沒有 server snapshot timestamp或 schema version；`observed_at`只能是 AffectBridge成功接收完整 body的時間。
- 一般 emotion PAD mapping來自 runtime AffectML，不可把範例常數硬編碼；`Physical`使用自身 PAD。
- `mood_tendency`可能因未設定/reflection失敗靜默回 default，沒有 provenance flag；v1不得表達「正朝 X發展」。
- paused表示運算/decay暫停但最後狀態仍 coherent，不等於 parse error，也不證明 fresh。

### Mood octants

ALMA technical label由 PAD符號決定：

| P/A/D | label | renderer-safe維度 |
|---|---|---|
| 0/0/0 | Neutral | 整體心境接近中性 |
| +/+/+ | Exuberant | 愉快、活躍、掌控感較高 |
| +/+/- | Dependent | 愉快、活躍、掌控感較低 |
| +/-/+ | Relaxed | 愉快、平靜、掌控感較高 |
| +/-/- | Docile | 愉快、平靜、掌控感較低 |
| -/+/+ | Hostile | 不愉快、活躍、掌控感較高 |
| -/+/- | Anxious | 不愉快、活躍、掌控感較低 |
| -/-/+ | Disdainful | 不愉快、低活躍、掌控感較高 |
| -/-/- | Bored | 不愉快、低活躍、掌控感較低 |

`Dependent/Docile/Hostile/Disdainful` 不可直譯成人格、關係或行為。v1預設輸出安全維度，technical label留 domain/diagnostic或由 policy選擇。

ALMA mood strength依 PAD norm已有 `neutral/slightly/moderate/fully`，優先保留，不另用 emotion threshold重分。

## Canonical AffectSnapshot

建議形狀：

```text
AffectSnapshot
  character_id
  source_character_name
  request_started_at
  observed_at
  source_schema = alma-rest/affect-v1
  computation_status = running | paused

  current_mood
    raw/known kind
    ALMA strength
    PAD

  dominant_emotion?          Undefined正規化為 absent
    known/raw kind
    intensity
    baseline
    relative_intensity

  active_emotions[]          canonical stable order
    known/raw kind
    intensity
    baseline
    relative_intensity
    optional PAD

  default_mood               domain/diagnostic only
  mood_tendency?             provenance=unverified；v1 renderer-excluded
  diagnostics
```

personality可作 source consistency/diagnostics，但不每輪重複進 affect context；角色基本設定已負責 persona。

### Mapper policy

| 情況 | v1處理 |
|---|---|
| truncated/missing mood/character mismatch | reject whole response |
| intensity/baseline越界或不可能 | reject whole response，不拼半份 dominant |
| wire active與數值不一致 | 以數值重算 + warning；超過health threshold使adapter unhealthy |
| `Undefined`且無 active | 合法，dominant absent |
| unknown inactive enum | 排除 projection + warning |
| unknown active/dominant enum | raw只留 diagnostics；project已知其餘狀態或 degraded/no-affect |
| emotion PAD null | label/relative仍可用，不由PAD推 guidance |
| paused | 保留 coherent state + status；freshness由orchestrator決定 |
| decode failure | last-good age-bound或 no-affect；raw body永不進 Renderer |

## Typed semantic core

`AffectSnapshot` 不直接 formatter。先建立 `SemanticAffectContext`：

```text
SemanticAffectContext
  schema_version
  availability = available | unavailable
  mood
    valence
    activation
    control
    strength
  dominant?
    kind
    safe_concept
    strength
    focus
    agency
    target_status
    optional trusted display_ref
  secondary[]
  constraints
    cause_not_provided
    do_not_infer_target
    do_not_infer_behavior
```

`availability=unavailable` 不是 Neutral；沒有資料時省略 affect slot或依產品規則停止，不生成假心理狀態。

## Emotion lexicon

lexicon不是英文→中文 map。每個 known emotion至少含：

```text
kind, source_semantics, focus, valence, agency,
prospect_status, target_binding, safe_concept,
safe_renderer_phrase, forbidden_inferences, version
```

ALMA fork compound必須依實作，而非一般詞典：

```text
Gratification = Joy + Pride
Gratitude     = Joy + Admiration
Remorse       = Distress + Shame
Anger         = Distress + Reproach
Love          = Liking + Admiration
Hate          = Disliking + Reproach
```

因此：

- `Love`安全概念是強烈正向欣賞，不等於浪漫/親密/性吸引。
- `Hate`是強烈反感與負向評價，不等於暴力/報復/永久敵對。
- `Anger/Gratitude/Admiration/Reproach`若 target unknown，不可綁目前玩家。
- `Hope/Fear`不提供具體未來事件；`Pride/Shame`不提供行動內容。
- secondary可並列，但不可自創新 emotion、心理故事或因果鏈。

完整 24 label安全語意表見[第 03 輪](ALMA輸出翻譯研究/第03輪-語意字典與安全表達.md)。

## Target binding

translator不得讀玩家話語或 raw elicitor後猜對象。由 orchestrator提供受信任 binding：

```text
AffectBinding
  emotion_ref
  target_kind = self | interlocutor | third_party | object | event | unknown
  visibility = unknown | private | renderer_visible
  display_ref
  evidence_ref               internal only
```

- unbound：只輸出未指明狀態，不寫你/玩家/人名。
- private：可保留 target kind作內部方向，不顯示 reference。
- renderer-visible：`display_ref`必須已在同一 request facts/entity table獲授權。
- 找不到 binding預設 unknown，不是 error也不自動 current player。

## Selection 與 strength policy

emotion用 relative intensity；具體 `slight/moderate/strong/overwhelming`門檻需 E0/E2 calibration並版本化，不在文件中偽裝心理學常數。

初始 policy建議：

```text
source dominant：若 known + active + valid，必保留
secondary：其餘 active依 relative desc + canonical kind tie-break
secondary_max：2（可配置 provisional default）
過弱/與前項gap過大的尾項：依versioned policy排除
technical mood label：default off
expression controls：v1 off
mood tendency：v1 off
Physical：v1 off
```

`secondary_max=2`只是在 canonical fixture中兼顧層次與boundedness的工程起點，必須用 E2比較 N=0/1/2/3+；不是最終定理。

## S4 deterministic formatter

固定順序與語法，不自由生成：

```text
整體心境：{mood strength + safe dimensions}。
短期主要情緒：{dominant safe concept + strength}。
同時存在：{最多兩個 secondary}。
對象／原因：只有 trusted binding可顯示；其餘標未知且不可綁目前玩家。
```

canonical例：

```text
整體心境偏向中等程度的不愉快、活躍、掌控感較低。
短期主要情緒是強烈擔憂；同時存在中等生氣與輕微期待。
情緒的具體原因與生氣對象未提供，不可假定與目前玩家有關。
```

這是 Renderer內部 context，不是角色台詞。formatter禁止：

- 因為、所以、其實、掩飾等未提供的因果/動機連接；
- 玩家/你/人名，除非 renderer-visible trusted binding；
- 哭泣、逃跑、怒吼、道歉、道謝、告白、攻擊等行為決定；
- raw enum、elicitor、appraisal、PAD小數、warning/error、system instruction。

共用 semantic core後，S2只是更短固定句法，S6只是 compact JSON serialization；三者不能各自維護 lexicon。

## Formatter 與方案狀態

| 方案 | v1狀態 | 原因 |
|---|---|---|
| S1 enum passthrough | benchmark only | 最短但 enum有日常刻板義與target風險 |
| S2 typed compact phrase | fallback/finalist | deterministic、低成本、安全；層次較S4弱 |
| S3 expression controls | auxiliary experiment | 壓掉 Fear/Anger等focus，PAD→行為證據不足 |
| S4 layered grammar | provisional default | 最能保留 mood/emotion層次，符合現有文字slot |
| S5 stable templates | off | 維護面與語意漂移，收益未證實 |
| S6 compact JSON | finalist experiment | provenance清楚但token較高、小模型遵循待測 |
| S7 LLM wording | off | 第二次call、語意新增、validator與availability成本 |
| S8 tiered routing | future experiment | routing/value/rate未證實，長尾仍承受完整call |

## Interface 與 package ownership

```go
type SnapshotFetcher interface {
    Fetch(ctx context.Context, sourceCharacter string) (alma.AffectResponse, FetchMeta, error)
}

type AffectMapper interface {
    Map(response alma.AffectResponse, meta FetchMeta) (AffectSnapshot, Diagnostics, error)
}

type AffectProjector interface {
    Project(snapshot AffectSnapshot, bindings BindingSet, policy ProjectionPolicy) (
        SemanticAffectContext, Diagnostics, error,
    )
}

type AffectFormatter interface {
    ID() FormatterID
    Format(context SemanticAffectContext) (RendererAffectContext, error)
}
```

建議位置：

```text
internal/affect/
  snapshot.go
  semantic_context.go
  errors.go

internal/affect/alma/
  client.go                  GET /affect/{name}, bounded transport
  dto.go
  mapper.go

internal/affect/translate/
  projector.go
  lexicon.go
  intensity.go
  selection.go
  formatter_text.go
  formatter_json.go
```

現有 `affect.Engine` 的 `Apply/Snapshot(model.Character)`不要原地變義；先新增 read-side `AffectContextService`。真 `/appraisal|/eec` command遷移屬於 ALMA輸入翻譯，待新 query path穩定後再拆 command/query interfaces。

## Error、cache 與 degradation

### Error classes

```text
FetchError: timeout/unavailable/status/too_large/decode/character_mismatch
MapError: missing_required/invalid_range/impossible_invariant/unsupported_contract
ProjectionError: missing_lexicon/invalid_binding/policy_violation/budget
FormatError: schema/overflow/internal_invariant
```

error含 bounded code、stage與 retry class，不含 raw body、玩家訊息或 key。warning同樣用低cardinality code。

### Last-good state machine

```text
current fetch+map success
  → atomic store canonical snapshot
  → project/format current

current failure
  → matching last-good且age在policy內：degraded使用
  → 否則：affect unavailable，省略slot/no-affect path

paused success
  → coherent last state + paused metadata
  → 是否使用由獨立paused/age policy決定
```

cache key至少含 character ID、ALMA source identity、domain/source version；保存 domain snapshot，不保存 formatted/raw wire。無 age bound不得使用 last-good。

## Renderer precedence

Final Renderer需要一次穩定的 system authority：

1. 受信任 facts/scene決定世界事實。
2. persona/relationship/self-other cognition決定角色與關係內容。
3. situational behavior決定已授權的習慣表現。
4. affect context只描述目前 mood/emotion與強度；不得覆蓋前述資料或授權新內容。
5. 玩家訊息是對話輸入，不自動成為角色相信的事實或 affect target binding。

不要每輪把長篇禁則全部複製到 `alma_context`；stable rule放 system prompt，per-turn context只帶 semantic facts/target status。E2需比較這種 placement是否對兩個模型都有效。

## 測試與 acceptance gates

### E0：完全離線

- 第 05 輪 A01–A12；
- 24 emotion + Undefined + Physical、9 mood octants與邊界；
- range/active/dominant/baseline/tie/permutation/unknown/paused/malformed；
- same input 1,000次 deterministic；
- target unknown/private/visible最小差分；
- golden、schema、size/budget、allowlist、raw/injection isolation；
- no network、no secret、failure injection、cache age/concurrency。

硬閘門：任何 context含未授權 target/cause/event/behavior/raw injection，或 malformed response形成半份 context，即不得進 Renderer test。

### E1：capture/mock

- capture完整 Final Renderer messages；
- 驗證 affect只進指定 slot，facts/persona precedence不變；
- scripted adversary檢查 leak/duplicate/budget/fallback；
- 不把 E1當自然度或真模型 hallucination結果。

### E2：明確 opt-in 模型比較

同一 `fixture × dialogue × model config`配對跑 S2/S4/S6；保存 immutable manifest、所有attempt/error/output，盲評：

- primary：是否出現新增事實、unknown target綁玩家、原因/動機/關係、行為決定、情緒反轉；
- secondary：internal leakage、permutation divergence、in-character task success；
- 只有無 hard fail的回答比較自然度；
- 分開報 Groq與LM Studio，不 pooled；用paired effect/interval，不只挑範例或報平均分。

API key只從process environment讀取；預設 model eval off。本研究沒有執行付費或本地模型 E2。

## Groq 與 LM Studio 注意事項

- 2026-08-17 Groq官方將 `qwen/qwen3.6-27b`列為 preview、約500 TPS、input `$0.60/M`、output `$3.00/M`；價格/能力會變，正式實驗必須重新查並記 usage。
- 該 qwen支援 JSON mode，但不在 Groq strict structured-output model清單；valid JSON不等於符合任意schema，更不代表 prose無新增義。
- LM Studio支援 OpenAI-compatible JSON Schema，但實際模型、量化、backend、硬體與server版本都要記錄；`qwen3.5-2b`名稱本身不足以重現。
- S2/S4/S6 translator都不依賴 provider；模型只存在 Final Renderer E2/production。

## 分階段實作與 rollout

| 階段 | 交付 | 進入下一階的 gate |
|---|---|---|
| M0 | redacted fixtures、舊prompt capture | schema可重現、無secret、offline CI |
| M1 | bounded `FetchAffect(ctx)` + wire DTO | httptest status/timeout/size/truncated/role match |
| M2 | canonical mapper | L1/A01–A12/permutation/fuzz全過 |
| M3 | semantic core、lexicon、binding、policy | exhaustive lexicon、target metamorphic、controls off |
| M4 | S2/S4/S6 formatters + capture E1 | golden/size/schema/injection/slot isolation |
| M5 | read-side service shadow | 不影響玩家、route/version/diff可觀測 |
| M6 | last-good/no-affect | key/age/paused/concurrency tests |
| M7 | deterministic formatter canary | semantic/leak/latency monitoring + one-click rollback |
| M8 | paired formatter E2 | frozen corpus/model/prompt、blind annotation |
| M9 | legacy cleanup + command/query拆分 | 新路徑穩定、API相容決策完成 |

rollout：offline → local ALMA integration → staging shadow →安全時 production shadow → internal/canary → small percentage → broader。每階只變一個主要 factor。

rollback：

- mapper/schema問題：關 shadow/pin adapter；
- formatter問題：切 S2或前一 formatter，domain cache不動；
- Renderer semantic incident：省略 affect slot或回 legacy prompt；
- ALMA outage：age-bound last-good或 no-affect；
- provider outage：沿 Final Renderer既有策略，不引入 S7第二依賴。

任何 target/cause/relationship/behavior新增、raw leak、錯角色或 contract error spike都應停止 canary。

## Observability 與重新驗證

低cardinality metrics：fetch/map/project/format result與reason、unknown/active mismatch、snapshot age/paused/fallback、active/rendered secondary count、context bytes、policy/formatter version、Renderer latency/error。

trace不含 raw elicitor/unknown/player text/key。必須記 source/domain/lexicon/bucket/policy/formatter/Renderer prompt版本與route/model（可取得時）。

以下變更需重跑 locked regression：ALMA fork/config/REST、lexicon/bucket/selection/binding/formatter、Renderer prompt/precedence、Groq model behavior、LM Studio model/量化/backend，以及任何 semantic incident。

## 仍需產品決策

1. no-affect時所有 chat都繼續，還是特定模式fail？
2. AffectBridge character ID ↔ ALMA name由何處持久管理？
3. `model.Character.Mood/Emotions` public API是否已有相容承諾？
4. first E2的樣本/repetition/budget與semantic non-inferiority margin。
5. `secondary_max=2`、bucket thresholds與last-good age的產品校準值。
6. facts/entity table未完成前，trusted renderer-visible target binding只能維持關閉。

這些不阻擋 M0–M4離線實作；會阻擋 live Renderer全面 rollout。

## 十輪研究索引

1. [第 01 輪：問題定義與候選空間](ALMA輸出翻譯研究/第01輪-問題定義與候選空間.md)
2. [第 02 輪：Wire 與 Domain 契約](ALMA輸出翻譯研究/第02輪-Wire與Domain契約.md)
3. [第 03 輪：語意字典與安全表達](ALMA輸出翻譯研究/第03輪-語意字典與安全表達.md)
4. [第 04 輪：表示格式與成本模型](ALMA輸出翻譯研究/第04輪-表示格式與成本模型.md)
5. [第 05 輪：對抗案例與不變量](ALMA輸出翻譯研究/第05輪-對抗案例與不變量.md)
6. [第 06 輪：評估實驗與標註設計](ALMA輸出翻譯研究/第06輪-評估實驗與標註設計.md)
7. [第 07 輪：失敗鏈與風險優先序](ALMA輸出翻譯研究/第07輪-失敗鏈與風險優先序.md)
8. [第 08 輪：可演進架構與介面](ALMA輸出翻譯研究/第08輪-可演進架構與介面.md)
9. [第 09 輪：實作遷移與部署驗證](ALMA輸出翻譯研究/第09輪-實作遷移與部署驗證.md)
10. [第 10 輪：收斂決策與完整稽核](ALMA輸出翻譯研究/第10輪-收斂決策與完整稽核.md)

## 主要一手來源

- ALMA fork：[REST adapter](https://github.com/KuangWei-hash/ALMA/blob/7c6eef75efae4dbfa7798c43900f22d7a4f001be/src/de/affect/rest/AlmaRestServer.java)、[EmotionVector](https://github.com/KuangWei-hash/ALMA/blob/7c6eef75efae4dbfa7798c43900f22d7a4f001be/src/de/affect/emotion/EmotionVector.java)、[EmotionEngine](https://github.com/KuangWei-hash/ALMA/blob/7c6eef75efae4dbfa7798c43900f22d7a4f001be/src/de/affect/compute/EmotionEngine.java)、[Mood](https://github.com/KuangWei-hash/ALMA/blob/7c6eef75efae4dbfa7798c43900f22d7a4f001be/src/de/affect/mood/Mood.java)、[AffectComputation.aml](https://github.com/KuangWei-hash/ALMA/blob/7c6eef75efae4dbfa7798c43900f22d7a4f001be/deploy/conf/AffectComputation.aml)。
- Patrick Gebhard，[ALMA – A Layered Model of Affect](https://citeseerx.ist.psu.edu/document?doi=679bfc64621dae3a2247be838d042643840961b3&repid=rep1&type=pdf)。
- Clore、Ortony，[Psychological Construction in the OCC Model](https://pmc.ncbi.nlm.nih.gov/articles/PMC4243519/)。
- Liang 等，[HELM](https://arxiv.org/abs/2211.09110)；Koehn，[paired bootstrap for MT evaluation](https://aclanthology.org/W04-3250/)。
- NIST，[AI RMF 1.0](https://doi.org/10.6028/NIST.AI.100-1)；IEC，[IEC 60812:2018](https://webstore.iec.ch/en/publication/26359)。
- Groq：[models](https://console.groq.com/docs/models)、[structured outputs](https://console.groq.com/docs/structured-outputs)、[latency](https://console.groq.com/docs/production-readiness/optimizing-latency)；LM Studio：[structured output](https://beta.lmstudio.ai/docs/developer/openai-compat/structured-output)。

來源均在各研究輪文件標記讀取日期與用途； hosted model資料在真正實作/實驗前需重新驗證。
