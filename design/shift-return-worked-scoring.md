# Shift-return v2：計分與獨立樣本演練

以下全部是假設輸出，不是實驗結果。機讀例子在 [examples.jsonl](shift-return-v2/examples.jsonl)，完整假設報表在 [worked-report.json](shift-return-v2/worked-report.json)，精確規則見 [protocol v2](shift-return-protocol-v2.md)。

## 1. 改善與誤傷一起計

| 任務 | 相同可見raw | 欲輸出 | E0-revise假設 | EC-revise假設 |
| --- | --- | --- | --- | --- |
| a 中文 | `microsoft593vul3` | microsoft窄小 | 原raw | microsoft窄小 |
| b 注音式帳號 | `microsoft593vul3` | 原raw | 原raw | microsoft窄小 |
| c 中文 | `x593gp ` | x窄身 | x窄身 | x窄身 |
| d 型號 | `gpt4` | 原raw | 原raw | 原raw |

a/b同component、不同intent；c/d各自一component。return分母為a/c的2components，E0錯1/2→EC錯0/2；literal分母為b/d的2components，E0錯0/2→EC錯1/2。不能只摘中文100%改善。這些數量不符正式資料資格，所以完整toy報告outcome為INCONCLUSIVE，並保留G4失敗。x窄身只示範空白一聲的形狀，不是自然語料。

到`microsoft593`仍顯示raw，後來`vul3`到達才自動組成窄小，可算a的最終repair；同一操作卻使b誤轉。系統看不到意圖欄，完全相同觀測不能同時猜對；同component等權兩intent可達上限為1/2。

## 2. 三種正確性不能混用

若顯示microsoft窄筱，切點／讀音可正確，但字面不是指定窄小，surface_success失敗。若影子判斷選窄小，但首字raw已送出而真正提交只剩小，materialization失敗。Enter時才另選答案違反提交契約。中途pending raw後來完成Chinese屬resolution，不直接當作先前literal誤轉。

## 3. 相關variants不灌分母

若10個帳號原型因相同prefix／alias連成一個component，只選一literal primary進主風險率，Wilson n=1；其餘9列報variants/worst-case。即使有400個字串，只合成150components，資格及Wilson分母仍為150。不能用對字串做bootstrap來補救相依性。

0錯／400獨立components的two-sided Wilson95上限約0.9512%，仍只是指定benchmark條件下的區間，不代表真實使用者的意圖分布。若有錯誤，上限可能不達1%，不追加案例直到過關。

## 4. 可重算的數值例

[numerical.json](shift-return-v2/numerical.json)固定mean/std、pair差向量、loss、gradient與第一步更新；是手算核對，不是正式fit。[check_examples.py](shift-return-v2/check_examples.py)驗證schema、原鍵spans、例子reading及數值，並拒絕缺key、ownership缺口與錯誤reading。正式完整SpellExact仍由未來Go P0/P1驗證，不把此小映射檢查當成語音合法性測試。

補充可重算例包括：canonical模型bytes/hash、same-hypothesis不同query的occurrences、seen名單使整component排除、非空prelude在每個關閉事件只輸出一次、primary自身而非variant的contrast資格，以及baseline-zero bootstrap replicates仍有效。稀疏errors=[1,0,0,0]修好唯一錯誤時，點估雖改善100%，絕對改善區間[0,.75]仍包含零，不能通過新的證據門檻。worked report包含全部10個paired entries，沒有contrast隊列時用n=0/null明列；未重播的安全計數全部null而非0。

```sh
python3 design/shift-return-v2/check_examples.py
```
