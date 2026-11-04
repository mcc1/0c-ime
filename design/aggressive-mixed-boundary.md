# 積極混打的英中切點：規則與待回修研究的想法

更新：2026-11-04。適用於 opt-in 的積極混打（`mixed-input-policy=aggressive`）；保守模式照上游 0.2.8。

## 目前規則（`core/session/mixed_policy.go`）

按下聲調鍵時，`mixedAggressiveBoundary` 依序找切點：

1. 前段是已知英文字、數字、標點或學過的英數詞（`mixedLiteralPrefix`）時，比較後段音節的字頻（`mixedMoveRatio`）。
2. 打錯一鍵的注音（`mixedAfterTypo`）。
3. 字典不認得的前段：先用上游 #241 的大寫邊界（`mixedAfterToken`／`mixedAfterLetter`），大寫字母不是注音鍵；沒有大寫邊界時才比較各切法的字頻（`mixedUnknownRatio`）。

每個切點最後經過 `mixedAbsorbNonLetters`：切點前一鍵是**數字或標點鍵**、併進後段仍是照順序的合法讀音時就併，直到組不成或碰到字母。數字與標點幾乎不會是英文字的一部分，所以中文優先。

代表結果（真實 v0.2.8 字典、統計選字）：`WQ5j04fu06`→WQ賺錢、`WD55j04fu06`→WD5賺錢、學過 `wq5` 的 `wq5j04xk7`→wq賺了、`microsoft593fmp6`→microsoft窄裙、`Vxk7`→V了、`Gq06`→G盤、`iPhoneji3`→iPhone我、`ps4`→ps4。

## 已試過、暫不採用的規則

在真實 v0.2.8 字典上以 `0c-cli` 比對（無神經選字）：

| 規則 | 修好 | 弄壞 |
|---|---|---|
| 不分鍵種「能組就盡量組」 | — | iPhon果、ma火、cod果、pytho所、linu裸、ye逆、appl果 |
| 同上，但前段是字典英文字就停 | — | jso所、vscod果、hellos以、AIr應 |
| 切在介音前且前一鍵是聲母就併（聲母＋介音＋韻母） | ok記、kevin記 | 同第一列 10 例 |
| 同上，且只在併走後剩下的前段是英文字時 | 無變化 | 無變化 |

原因：我、一、要、有、為等最常用字本身是零聲母、以介音開頭，所以「字母＋介音開頭」同時可能是「聲母＋介音＋韻母」或「英文字尾＋零聲母字」。字典不認得的前段（`kevinru4` 與 `jsonji3`）在按鍵上無法區分，只剩字頻。

## 與後文回修的關係

切點一旦決定，前段以字面文字插入（`insertMixedLiteral`），後段以固定讀音插入（`insertMixedTyped`）。之後的整句解碼只在同一讀音內換字（果→粿），不會把字母還給英文：

| 按鍵 | 目前 | 介音規則 |
|---|---|---|
| iPhoneji3ru04ji3cl3 | iPhone我見我好 | iPhon果見我好 |
| macji3vu.4 | mac我秀 | ma火宿 |

做回修（提交前保留多種切法、由後文整句分數決定）時，再重新評估「切在介音前就積極併入聲母」：有回修時猜錯可被後文改回，這條規則的代價會小很多。對應 `design/shift-return-v2` 的 fixed／revise 比較。
