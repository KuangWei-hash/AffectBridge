# ALMA 輸出翻譯研究：第 02 輪

> 日期：2026-08-17（Asia/Taipei）
> 階段：wire/domain contract 深挖；仍不選定最終 wording 方案。
> ALMA 證據版本：commit `7c6eef75efae4dbfa7798c43900f22d7a4f001be`。

## 本輪問題

1. `GET /affect/{name}` 每個欄位在目前 ALMA fork 中精確代表什麼？
2. AffectBridge 能驗證哪些 invariant，哪些不能從單一 response 推出？
3. `AffectSnapshot` 應保留什麼，Renderer contract 又應排除什麼？
4. 第 01 輪八個候選方案在正確 contract 下有何變化？

## 本輪讀取的專案證據

- 第 01 輪已確立最多八個候選方案、九個評估維度與十二類 fixture。
- `internal/affect/alma/client.go` 尚未實作 `GET /affect/{name}` wire DTO，而是呼叫 `/apply` 並把 `model.Character` 當 ALMA payload；因此目前沒有能直接沿用的 domain contract。
- `internal/model.Character` 的 `Mood` 與 `EmotionSet` 不足以無損表達 dominant、baseline、active、mood word/tendency/default、paused 與觀測狀態。
- `assets/persona/Lisa.json` 的 global emotion baseline 是 `0.1`，William 是 `0.0`。這證明使用 raw intensity 比較兩角色會混入人格 baseline 差異；relative intensity 才是同一套 active/dominant 語義。
- `config.json` 的既有未提交 Groq 設定與本研究無關，本輪未修改。

## 外部來源與日期

本輪直接 clone 並讀取 `KuangWei-hash/ALMA`，固定在 commit `7c6eef75efae4dbfa7798c43900f22d7a4f001be`，讀取日期 2026-08-17：

- [`AlmaRestServer.java`](https://github.com/KuangWei-hash/ALMA/blob/7c6eef75efae4dbfa7798c43900f22d7a4f001be/src/de/affect/rest/AlmaRestServer.java)：REST path、locking、response JSON、tendency fallback。
- [`EmotionVector.java`](https://github.com/KuangWei-hash/ALMA/blob/7c6eef75efae4dbfa7798c43900f22d7a4f001be/src/de/affect/emotion/EmotionVector.java)：relative intensity 排序與 dominant。
- [`EmotionType.java`](https://github.com/KuangWei-hash/ALMA/blob/7c6eef75efae4dbfa7798c43900f22d7a4f001be/src/de/affect/emotion/EmotionType.java)：完整 enum 與 OCC category。
- [`Emotion.java`](https://github.com/KuangWei-hash/ALMA/blob/7c6eef75efae4dbfa7798c43900f22d7a4f001be/src/de/affect/emotion/Emotion.java)：intensity/baseline 範圍、elicitation time 與 appraisal。
- [`Mood.java`](https://github.com/KuangWei-hash/ALMA/blob/7c6eef75efae4dbfa7798c43900f22d7a4f001be/src/de/affect/mood/Mood.java)：PAD → mood word 與 mood intensity 的精確算法。
- [`MoodEngine.java`](https://github.com/KuangWei-hash/ALMA/blob/7c6eef75efae4dbfa7798c43900f22d7a4f001be/src/de/affect/compute/MoodEngine.java)：active emotions 如何形成 mood tendency、無 active emotion 時如何回 default。
- [`AffectManager.java`](https://github.com/KuangWei-hash/ALMA/blob/7c6eef75efae4dbfa7798c43900f22d7a4f001be/src/de/affect/manage/AffectManager.java)：runtime 可用 emotion 與 PAD relation 從 AffectML 載入。
- [`deploy/conf/AffectComputation.aml`](https://github.com/KuangWei-hash/ALMA/blob/7c6eef75efae4dbfa7798c43900f22d7a4f001be/deploy/conf/AffectComputation.aml)：目前 deploy 範例的啟用 emotion 與 PAD mapping；它是配置實例，不是不可變 protocol constant。

## Wire 語義校正

### Endpoint 與一致性

- 單角色讀取是 `GET /affect/{name}`；`GET /affect` 回全部角色；非 GET 回 405。
- server 在 `CharacterManager → EmotionEngine` 的固定 lock 順序內，同時產生 dominant 與完整 emotion list。因此同一 response 內的資料應視為同一個一致 snapshot。
- response 沒有 server-side snapshot timestamp 或 schema version。

### Emotion 集合不是 active list

`EmotionVector` 對 runtime `AvailableEmotions` 中每個 type 保存一個 emotion；manager 另固定加入 `Undefined` 與 `Physical`。REST `emotions[]` 會輸出整個 vector，包括 baseline-only/inactive 項目。因此：

```text
wire emotions[]                 全部 runtime enabled emotions
domain active_emotions[]        僅 active == true 且 invariant 合法的項目
```

mapper 不能把 `len(emotions)` 當同時存在的情緒數量。

### Active 與 relative intensity

ALMA invariant：

```text
0 <= baseline <= intensity <= 1
relative_intensity = intensity - baseline
active = relative_intensity > 0
```

`EmotionVector` 依 relative intensity 升冪排序；dominant 是最後一項。所有 relative intensity 都為 0 時，dominant 改成 `Undefined`。這推翻「raw intensity 最大就是 dominant」的做法。

相等 relative intensity 沒有可依賴的 tie-break：排序 comparator 回 0，而來源是 `HashMap.values()`。因此 wire array 在 tie 時的次序不是穩定 contract。AffectBridge 若需要穩定 secondary ordering，必須用：

1. relative intensity 降冪；
2. canonical enum ordinal 或明確 lexicographic key 作 tie-break；
3. 不使用 elicitation time 作 tie-break，除非產品確定 recency 應改變顯著性。

### Elicitor、time 與 appraisal

- baseline-only 且沒有 appraisal 時，REST 把 `elicitor`、`elicited_at` 與 `appraisal` 輸出為 null，即使 Java emotion 物件內部有建立時間與 personality elicitor。
- active 或帶 appraisal 時才輸出 elicitor/time。
- 這些欄位適合內部 trace，不應直接進 renderer。
- `elicited_at` 是 emotion 建立時間，不是 snapshot 時間，也不能證明 intensity 正在上升或下降。

### Emotion PAD 是 runtime configuration

一般 OCC emotion 的 PAD 從 `EmotionsPADRelation` 取得；該 mapping 由啟動時讀入的 `AffectComputation.aml` 設定。`Physical` 才使用 emotion 自身 PAD。

因此：

- AffectBridge 若需要該 PAD，應讀 response 中的 `emotion.pad`，不可複製 deploy 範例成硬編碼常數。
- 同一 emotion name 在另一份 runtime AffectML 可以有不同 PAD。
- missing relation 可能令 `pad` 為 null；mapper 必須接受或明確拒絕，不可補猜。

### Mood word 的精確 octant

除原點為 `Neutral` 外，word 只由三軸正負號決定；零被歸在非負側：

| P | A | D | word |
|---:|---:|---:|---|
| + | + | + | Exuberant |
| + | + | − | Dependent |
| + | − | + | Relaxed |
| + | − | − | Docile |
| − | + | + | Hostile |
| − | + | − | Anxious |
| − | − | + | Disdainful |
| − | − | − | Bored |
| 0 | 0 | 0 | Neutral |

這些是 ALMA 的技術 label，不應任意以一般中文詞典直譯。例如 `Dependent` 是 PAD octant 名稱，不足以斷言角色在關係上「依賴對方」。

### Mood intensity 已有固定算法

令 `norm = sqrt(P² + A² + D²)`：

| 條件 | wire intensity |
|---|---|
| `norm == 0` | neutral |
| `0 < norm <= 0.5` | slightly |
| `0.5 < norm <= 1.0` | moderate |
| `norm > 1.0` | fully |

主文件原本寫「ALMA 已提供離散強度時優先保留」是對的；AffectBridge 不應再用另一組 mood 門檻重分。

### Mood tendency 的 provenance 缺口

REST adapter 以 reflection 取得 private `fMoodEngine`：

- engine 正常時，`getCurrentMoodTendency()` 未曾設定也會回 default mood；
- reflection、security 或 engine 失敗時，adapter 也靜默回 default mood；
- wire 沒有 `tendency_source` 或 `fallback` 標誌。

所以 `mood_tendency == default_mood` 至少有三種解釋，mapper 無法區分。v1 不應把它翻成「心情正朝 X 發展」。最多保留 raw optional 值與 `provenance=unverified`，直到 server contract 增加來源旗標。

### Paused 的正確解讀

paused 會停止角色的 emotion decay 與 mood computation，也禁止部分新輸入；但 `GET /affect` 仍回最後一個一致狀態。因此：

- paused 不等於 parse failure，也不必強制 neutral；
- renderer 可以使用最後狀態，但內部 metadata 應標記 `computation_status=paused`；
- paused 持續多久無法從 response 得知，不能聲稱 fresh；是否允許玩家流程使用 paused state 是 orchestrator policy，不是 mapper 猜測。

## 建議的 Wire DTO（本輪 contract 草案）

Wire DTO 應忠實允許 nullable 欄位，不直接使用 domain enum 造成整包 decode 失敗：

```text
alma.AffectResponse
  name: string
  affect_computation_paused: bool
  personality: PersonalityResponse
  dominant_emotion: *EmotionResponse
  mood: MoodResponse
  mood_tendency: *MoodResponse
  default_mood: MoodResponse
  emotions: []EmotionResponse

alma.EmotionResponse
  name: string
  intensity: number
  baseline: number
  active: bool
  elicitor: *string
  elicited_at: *int64
  pad: *PADResponse
  appraisal: *AppraisalResponse
```

unknown `name` 先保留原字串；domain mapper 再轉成 `Known/Unknown` tagged value並產生 warning。這讓新 enum 可以降級，而不是讓整個角色停止回應。

## 建議的 AffectSnapshot（本輪 contract 草案）

```text
AffectSnapshot
  character_id
  source_character_name
  observed_at                 AffectBridge 成功收完整 body 的時間
  source_schema               "alma-rest/affect-v1"（本地 adapter version）
  computation_status          running / paused

  current_mood
    kind                      known enum + raw label
    intensity                 ALMA enum
    pad

  dominant_emotion            optional；Undefined 正規化為 absent
    kind                      known enum + raw label
    intensity
    baseline
    relative_intensity
    active

  active_emotions[]           deterministic order
    kind
    intensity
    baseline
    relative_intensity
    pad                       optional；只作結構資料

  default_mood                domain/diagnostic；預設不進 renderer
  mood_tendency               optional, provenance=unverified；v1 預設不進 renderer
  personality                 domain/diagnostic；預設不進每輪 renderer

  diagnostics
    warnings[]
    dropped_emotion_count
```

`observed_at` 只能表示 AffectBridge 何時觀測，不代表 ALMA 何時計算。若未來需要 freshness，HTTP client 必須另記 request start/end、timeout 與 snapshot age policy。

## Mapper 驗證與降級矩陣

| 條件 | 處理 | Renderer 可用性 |
|---|---|---|
| 缺 `name/personality/mood/default_mood/emotions` | contract error | 不可用；不得轉 raw JSON |
| dominant null/Undefined 且無 active | 合法 neutral-emotion state | 只用 current mood |
| intensity/baseline 超界或 intensity < baseline | 丟棄該 emotion + error warning；dominant 若受影響則 snapshot degraded | 可用剩餘 mood，需 metrics |
| `active != (intensity > baseline)` | 以數值 invariant 為 canonical 並警告，或 strict mode 拒絕 | policy 待第 05 輪測試 |
| unknown emotion | 保留 raw kind、排除 semantic guidance、警告 | 仍可用已知狀態 |
| unknown mood word 但 PAD 合法 | 保留 raw + PAD，不自行命名 | 只用安全 dimensional context 或 neutral wording |
| mood word/intensity 與 PAD 算法不一致 | 警告；不靜默改字 | current mood 是否可用待 policy |
| emotion PAD null | emotion label/strength 仍可用 | 禁止由 PAD 產 guidance |
| tendency 等於 default | 保留但不宣稱方向 | v1 不進 renderer |
| paused | 標記 paused，不清空 | 由 orchestrator 決定是否使用 |
| response decode failure | 整體失敗 | 使用上一個有 age bound 的 snapshot 或 minimal no-affect context，不能用 raw |

## Canonical fixture 形狀

後續測試應以 wire JSON fixture 為輸入，不只測手工建立的 domain object：

```json
{
  "name": "Lisa",
  "affect_computation_paused": false,
  "personality": {
    "openness": 0.2,
    "conscientiousness": 0.1,
    "extraversion": 0.0,
    "agreeableness": 0.3,
    "neurotism": 0.4,
    "derived": false,
    "emotion_influence": 0.2
  },
  "dominant_emotion": {
    "name": "Fear",
    "intensity": 0.6,
    "baseline": 0.1,
    "active": true,
    "elicitor": "chat-fixture/event-1",
    "elicited_at": 1786896000000,
    "pad": {"pleasure": -0.64, "arousal": 0.6, "dominance": 0.43},
    "appraisal": null
  },
  "mood": {
    "word": "Anxious",
    "intensity": "moderate",
    "pleasure": -0.4,
    "arousal": 0.5,
    "dominance": -0.2
  },
  "mood_tendency": {
    "word": "Anxious",
    "intensity": "moderate",
    "pleasure": -0.5,
    "arousal": 0.5,
    "dominance": -0.3
  },
  "default_mood": {
    "word": "Docile",
    "intensity": "slightly",
    "pleasure": 0.1,
    "arousal": -0.1,
    "dominance": -0.1
  },
  "emotions": []
}
```

注意：正式 fixture 必須讓 dominant 同時出現在 `emotions[]`；上例只展示欄位形狀，不能作 invariant-passing golden fixture。這個刻意不完整案例可作 validator negative test。

## 與第 01 輪的差異

1. dominant 的定義由「待確認」提升為已由 source 證實的 relative-intensity 規則。
2. mood intensity 不再是待校準項；直接保留 ALMA 的固定 norm bucket。
3. emotion PAD 從「可能固定 lexicon 資訊」改為 runtime-configured data。
4. tendency 從一般 renderer 欄位降為 provenance 不足的 optional diagnostic。
5. paused 從「可能 stale/error」拆成一致但停止演進的狀態；freshness 另訂 policy。
6. 加入 tie ordering、wire-enabled vs active、observed time 與 schema version 問題。

## 八方案的本輪比較

本輪只評 contract 相容性，不做總排名：

| 方案 | contract 適配結果 | 新風險/改良 |
|---|---|---|
| S1 最小結構直傳 | 可行 | 必須先 normalise unknown、Undefined 與 deterministic order |
| S2 中文字典 | 可行 | mood octant label 不能做日常人格/關係直譯 |
| S3 表達旋鈕 | 證據仍不足 | PAD mapping 動態，不能用 emotion name 的硬編常數 |
| S4 組合語法 | 可行 | 只組 active；不得用 tendency 聲稱變化方向 |
| S5 模板變體 | contract 無直接障礙 | deterministic seed 不能依 wire tie order |
| S6 結構 + renderer 規則 | 很適合保留 provenance | 需限制 payload，不能把 diagnostics/personality 全塞入 prompt |
| S7 LLM wording | 無法取代 mapper | LLM 前仍需完成所有 invariant 與降級處理 |
| S8 混合分級 | 可行但 routing 更複雜 | unknown enum 不應自動交 LLM 猜，只能安全保留/降級 |

## 風險與失敗模式

- **REST 無 schema version**：upstream 加欄位可相容，刪/改欄位會在 runtime 才發現。應以 adapter contract test 釘住 commit/fixture。
- **tie 非確定性**：直接保留 wire order 會讓相同狀態 prompt 漂移。
- **tendency false precision**：把 fallback default 翻成未來方向會製造不存在的動態訊息。
- **配置漂移**：硬編 emotion PAD 或假設所有 enum 永遠啟用，會與 runtime AffectML 不一致。
- **半有效 response**：只要一項 emotion 異常就整體 fail closed 可能太脆弱；全部忽略又會掩蓋 adapter bug。需要 strict/degraded policy 與 metrics。
- **過期快取**：response 沒有 snapshot timestamp，fallback 使用上次 snapshot 必須由 AffectBridge 自己限制 age。
- **負 fixture 誤用**：本輪示例 `emotions=[]` 與 dominant 不一致，只能作 schema 形狀或 negative test，不可被後續誤標為 golden success。

## 本輪改良

- 把 wire 與 domain nullable/unknown 邊界寫成可實作 contract。
- 明確區分 enabled、active、dominant 三個集合概念。
- 建立 deterministic ordering 與 invariant validation 規則。
- 把 observed time、schema version、diagnostics 納入 snapshot。
- 為每一類異常定義候選降級，不再只寫「失敗時 neutral 或停止」。
- 依 contract 重新檢查八方案，未增加第九種方案。

## 尚未解答

1. `active` mismatch 應 strict reject 整個 snapshot，還是 canonicalize 數值並警告？
2. 上次 snapshot fallback 的最大 age 應是多少；paused 是否使用不同 age policy？
3. dominant 與 list 在相同最大 relative intensity tie 時，domain 是否保留 server 選擇或自行 canonicalize？
4. `Physical` emotion 是否屬於 AffectBridge v1 的合法 renderer input，還是只做 diagnostics？
5. emotion appraisal 是否完全不進 `AffectSnapshot`，或為 trace 保存 redacted/internal reference？
6. renderer 最少需要 current mood + dominant，還是 dominant 可缺且只用 mood？
7. `source_schema` 應用手動 adapter version、ALMA commit、還是 server 新增正式版本欄位？

## 下一輪計畫

第 03 輪聚焦「語意 lexicon 與安全表達邊界」。逐一研究 24 個常用 OCC emotion 與 9 個 ALMA mood octant，區分可直接翻譯的 affect label、需要角色/事件 context 才能成立的關係詞，以及不能產生的動機/行為推論。產生候選 lexicon schema 與 collision/混合情緒規則；仍不選最終方案。
