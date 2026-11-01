# Shift-return：資訊組合與決策順序

正式定義與門檻以 [protocol v2](shift-return-protocol-v2.md)為準。B是初始訊號，L是已輸入的後續長度，W是後文字詞與統計解碼證據。資訊固定與ownership封存是兩個不同變因。

## 1. 四種資訊組合

| 模型 | 初始B | 長度L | 字詞W |
| --- | --- | --- | --- |
| E0 | 有 | 無 | 無 |
| EL | 有 | 有 | 無 |
| EW | 有 | 無 | 有 |
| EC（主要） | 有 | 有 | 有 |

E0固定B的初次資訊，不表示必然封存決策；它是否可改判由下面的流程決定。四模型同資料、訓練程序與預算，各自擬合係數，unused係數為0；不要把它們說成同一係數的主實驗。

在相同revise流程比較EL−E0、EW−E0、EC−EL與EC−EW。error尺度交互作用 I=errorEC−errorEL−errorEW+errorE0，報paired component bootstrap，不用負I單獨宣稱因果。EW可檢查長度是否與詞訊號重疊或干擾。

## 2. 兩種封存流程

| 流程 | 首次有可辨首音節候選時 | 後續資訊 |
| --- | --- | --- |
| fixed | 選擇後封存當時display的ownership／reading，pending視literal | 只能相容延伸；同音字面仍可更新 |
| revise | 暫選，不封存原鍵 | 重新比較同段所有未明確鎖定的拆法 |

封存時點不讀gold。兩流程使用同一份模型與margin，不重訓、不重選門檻。共同generator先產候選，fixed只在selection層篩除不相容者；「存在但被封存排除」不能算generator沒找到。

## 3. 實驗矩陣及解釋

四個模型×兩流程共8個test replays，模型只有4份。margin先在dev-revise選定並凍結，八個final conditions各跑一次；主要仍EC-revise，不在看test後改挑最高分。

同一模型比fixed/revise，回答提早定案的影響；同一流程比模型，回答資訊組合的影響。不能把EC-revise與E0-fixed之差全歸因於context。共享候選、固定向量累加順序下，先計算B或W沒有額外語意；本輪研究的是封存時機，而非全部soft-rule優先排列。

若EC只在revise改善，保留重判空間是該方法必要條件；若EW更好，記為下一版假說而非事後改本次主方案。中文改善但literal超標仍不通過；字面改善但切點沒改善，亦只證明選字改善。
