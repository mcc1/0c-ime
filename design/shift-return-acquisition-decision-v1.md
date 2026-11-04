# 2026-11-02 實驗狀態與主 agent 判讀

目前結論是 **INCONCLUSIVE：資料取得提前停止，G0 未通過；方法品質尚未測試**。不能據此判定積極組字、後文回修或四個模型有效／無效。

前兩次失去回應是主 agent 的交接疏漏：子 agent 完成指定階段後，工作停在主審，沒有及時接續與回報。本回合已接續完成工具核對、真實資料取得、第二 AI 盲標註、爭議裁定、相依群組合併，以及直接執行資料資格檢查。

## 已完成且有證據的工作

- 工具與契約修補提交為 `18d4f77`；保留原始 c512578 契約及失敗反例。主 agent 獨立重跑 tagged Go tests/vet、Python 8 個數值契約、design checker 與 public-tree checker，通過。59 檔工具／契約來源雜湊已逐檔核對，各階段 manifest 與檢查紀錄分開保留。
- 取得 1,500 個 synthetic-intent 任務：750 return、750 literal。兩個實際 AI 身份分別標註，不冒稱真人或自然輸入頻率。
- 第二次盲審原本接受 1,492 筆，8 筆因把非昆蟲置於昆蟲觀察語境而暫不接受。修正題目描述後，作者與盲審者重新標註 8 筆，最終 1,500 筆標籤一致；原題目、拒收、old/new ID 與修訂證據均保留。
- 同義近重複及固定 literal 內容模板真實連結，沒有把多個字串冒充多個獨立樣本。審核後為 372 個 components：train 209、dev 74、test 69、regression 20。兩個 intent 共用 component，不可相加算獨立 n。
- 最終 test 的 context-contrast components 為 69，低於預定 100；test return/literal 也各只有 69，低於各 400。其餘固定長度／類別取得配額未完成。
- 直接執行已建置 CLI 的 validate，exit **3**，訊息為 `fixed return acquisition quota incomplete/exceeded`。記錄在 `acquisition-v1/parent-accepted/through09-corrected/qualification-check.json`。沒有把 Go run 外層的 exit 1 誤認為工具的狀態碼。
- 另建立且獨立靜態核對 3 筆 mandatory 回歸：microsoft窄裙、prelude 解 + WQ賺錢、無 Shift 的 593→窄。只是按鍵／讀音／事件範圍驗證，尚未重播，未計新測試 n。

## 為何停止大量補題

原先 750 組後文對照的分配已填入，但重複路徑、版本號、長英數內容模板造成大量真實相依性。盲審補齊所有模板關係後，只留下 69 個 test context-contrast groups。繼續添加原計畫中沒有對照的一般任務，不能按原意補足這個特定隊列。

這是看到**資料結構與資格**後的提前停止，不是預先宣告的模型品質失敗，也不是聲稱 6,000 原型預算已用完。後續真實 family 合併會重算 hash；此處沒有證明所有可能合法資料都無法通過。沒有換 split seed、按分組結果補造題目、刪除困難列或降低門檻。

另有一份 pre-errata `internal/shiftreturn/report_test.go` 的 seen 來源歷史 bytes 尚缺 exact cache；原 seen hash 清單仍保留，不能宣稱來源稽核完整。長輸入校準也保留 64 音節情況耗盡固定 250,000 logical lex requests 的負例。這些都需在未來正式凍結前處理。

## 判讀與下一版取得方案方向

應先修正**來源取得與相依單位設計**，沒有證據要求更換模型或推翻後文資訊的核心想法。下一版應先建立多樣、可追溯的內容／來源 family sampling frame，核對來源供給與模板集中程度，再凍結取得程序與預算；不能先大量合成同一種路徑／版本／payload 模板，最後才發現有效 n 不足。

可評估有版本、授權與原文快取的公開識別字串，與明示合成任務分層；依來源／vendor／模板實際連 family。對照配置若需調整，必須另立 pre-quality revision，保留此次失敗取得紀錄，不能事後悄悄改原配置。既有品質門檻、四模型／八條件及主要 EC-revise 不因這次資料失敗而降低或更換。

正式 candidate 品質枚舉、訓練、dev 選模與 final test **皆未執行**；沒有取得模型品質結論。產品組字程式未變更。
