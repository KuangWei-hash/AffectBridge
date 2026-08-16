# UI 計畫

> **用途：內部開發與快速驗證。**
>
> 本 UI 不是正式玩家介面。它用來快速測試 AffectBridge 的完整對話流程，並在同一頁觀察角色回覆與 ALMA 情緒狀態是否按照預期變化。

## 第一項工作：開發驗證用對話頁面

建立一個雙欄對話頁面，讓開發者選擇角色、送出玩家訊息，並即時查看角色目前狀態與經 AffectBridge 轉換後的 ALMA 情緒資料。

```text
┌─────────────────────────────┬──────────────────────────────────────┐
│ 即時狀態面板                │ 對話視窗                             │
│                             │                                      │
│ 目前角色                    │ 玩家訊息                             │
│ 角色基本資料                │ 角色回覆                             │
│ ALMA 情緒／心情摘要         │                                      │
│ AffectSnapshot 更新時間     │                                      │
│ request / error 狀態        │                                      │
│                             │ ┌──────────────────────────────┐     │
│                             │ │ 輸入玩家訊息                 │ 送出│
│                             │ └──────────────────────────────┘     │
└─────────────────────────────┴──────────────────────────────────────┘
```

桌面版以左側狀態面板、右側對話視窗為預設；窄螢幕可改成上下排列，不要求第一版支援完整行動裝置體驗。

## 使用流程

1. 開發者開啟頁面並選擇一名現有角色。
2. 頁面載入角色基本資料與目前 affect snapshot。
3. 開發者在對話區輸入一段玩家訊息。
4. UI 呼叫 AffectBridge 的 `/characters/{id}/chat`。
5. 對話區加入玩家訊息與角色回覆。
6. 狀態面板以該輪回傳的狀態更新；若 chat response 沒有完整 snapshot，再由 AffectBridge 的 trusted debug endpoint 重新取得。
7. 開發者可以直接比較「玩家說了什麼、角色如何回答、情緒如何變化」。

## 左側：即時角色與 ALMA 狀態面板

第一版至少顯示：

- 當前角色 ID、名稱與基本角色資訊。
- 當前 dominant emotion、active emotions 及其強度。
- 當前 mood、mood tendency 與必要的強度摘要。
- `AffectSnapshot` 的取得或更新時間。
- 本輪請求狀態：等待、處理中、成功、失敗或資料已過期。
- AffectBridge 回傳的可安全顯示錯誤，不顯示憑證或後端內部資訊。

可提供「開發者詳細資料」摺疊區，以結構化 JSON 顯示 AffectBridge 自己的 domain/debug DTO，方便檢查欄位；但不得讓瀏覽器直接取得 ALMA wire response。

## 右側：玩家與角色對話視窗

第一版至少提供：

- 可捲動的訊息歷史。
- 清楚區分玩家訊息、角色訊息與系統錯誤。
- 多行文字輸入框與送出按鈕。
- Enter 送出、Shift+Enter 換行。
- 請求進行中禁止重複送出，並顯示角色正在回應。
- 送出失敗時保留玩家輸入，允許重新嘗試。
- 切換角色時清楚重設或隔離畫面上的對話紀錄，避免把上一個角色的訊息誤認為目前角色的記憶。

## API 與安全邊界

UI 只能呼叫 AffectBridge，不得直接呼叫 ALMA：

```text
Development UI
   ├── POST /characters/{id}/chat
   ├── GET  /characters/{id}
   └── GET  /characters/{id}/affect   （trusted debug/admin only）
                         │
                         ▼
                    AffectBridge
                         │ private outbound calls
                         ▼
                       ALMA
```

- ALMA 仍然是本城市內部的情緒運算核心，不是玩家或瀏覽器直接使用的 API。
- UI 顯示的是 AffectBridge 的穩定 DTO，例如 `AffectSnapshot`，不是原始 ALMA payload。
- `/characters/{id}/affect` 若保留，必須受 trusted debug/admin 邊界保護，正式玩家環境預設不可使用。
- UI 不得保存或要求輸入 ALMA 位址、ALMA token 或其他後端憑證。
- 不在畫面或 browser log 顯示不必要的 elicitor、內部 ID、prompt、資料庫資訊或 stack trace。

## 狀態同步原則

- 每次 chat response 應攜帶該輪對應的角色回覆與 affect snapshot，讓兩欄顯示同一輪結果。
- 若必須另外讀取 affect，應使用 response version、request ID 或時間戳避免舊請求覆蓋新狀態。
- 快速連續操作時，只接受目前角色與最新有效 request 的結果。
- 無法取得情緒狀態時，對話區仍應顯示已成功取得的角色回覆；狀態面板則標示 unavailable，不應假裝為中性情緒。
- ALMA 狀態更新失敗與「沒有 active emotion」必須使用不同畫面狀態。

## MVP 工作項目

### 頁面與互動

- [ ] 建立雙欄頁面骨架與窄螢幕上下排列。
- [ ] 建立角色選擇與角色基本資料顯示。
- [ ] 建立訊息列表、輸入框、送出與重試操作。
- [ ] 加入 loading、empty、error、unavailable 與 stale 狀態。
- [ ] 在角色切換時隔離或清除本機對話狀態。

### Affect 顯示

- [ ] 定義 UI 使用的 `AffectSnapshot` view model。
- [ ] 顯示 dominant emotion、active emotions、mood 與更新時間。
- [ ] 加入可選的安全 debug JSON 檢視。
- [ ] 讓每輪訊息可對照該輪情緒 snapshot，而不只顯示最新值。

### API 整合

- [ ] 串接角色讀取 API。
- [ ] 串接 chat API 並確認 response 能關聯角色回覆與 affect snapshot。
- [ ] 視需要串接受保護的 affect debug API。
- [ ] 定義 timeout、取消、重試與亂序 response 的行為。
- [ ] 確認開發模式與正式環境的 debug API 存取控制。

### 驗證

- [ ] 測試正常對話後，訊息與情緒面板同步更新。
- [ ] 測試沒有 active emotion 時的顯示。
- [ ] 測試 ALMA／affect 不可用，但角色流程可降級時的顯示。
- [ ] 測試 chat 失敗、逾時、重試與快速連續送出。
- [ ] 測試切換角色時不會混用訊息或 affect state。
- [ ] 驗證瀏覽器沒有直接向 ALMA 發出 request，也沒有取得 ALMA 憑證或原始 wire payload。

## 第一版非目標

- 正式遊戲美術與玩家體驗設計。
- 玩家帳號、社交、配對或多人聊天室。
- 記憶－事實搜尋模塊的管理後台。
- 直接修改 ALMA 參數或任意注入 `/appraisal`、`/eec`。
- 在瀏覽器中實作 ALMA 翻譯或 ALMA 輸出翻譯邏輯。
- 對話內容與情緒歷史的永久保存；除非後續另行定義開發紀錄需求。

## 待確認事項

1. 第一版 UI 使用哪一種前端技術與放置目錄？
2. chat response 是否直接包含該輪 `AffectSnapshot`，或需要 UI 再呼叫一次 debug endpoint？
3. 頁面只在本機開發環境啟用，還是需要部署到受保護的測試環境？
4. 是否需要逐輪保留 affect snapshot，讓開發者回看情緒變化？
5. 是否需要顯示 ALMA 輸入翻譯結果；若需要，應使用另一個受保護的 debug DTO，而不是原始 ALMA request。
