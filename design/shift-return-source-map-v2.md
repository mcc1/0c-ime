# Shift-return v2：來源、初始值及驗證對應

本文件屬於[契約 v2](shift-return-protocol-v2.md)。來源路徑指目前 checkout；run-lock須記實際blob hashes及基準commit，不因名稱相同就假定等價。以下是計畫使用的API，不是已完成parity的宣稱。

| 用途 | 具體API／檔案 | 固定輸入與語意 |
| --- | --- | --- |
| 公開資料 | [datadir.Open](../internal/datadir/datadir.go) | Open(`--data`)，必須有已驗hash的Chinese與English manifests；不接受fallback prototype English。記manifest及其引用檔hash |
| 空個人模型 | [personal.NewModel](../core/personal/personal.go) | 用loaded.Model.Lex／LM，每episode建新的Store；scoring專用另建一個、Now=nil；不調Learn／SyncTime，不設custom vocab；Source()保留原neutral-particle view |
| 完整拼法 | [mixedparse.SpellExact](../core/mixedparse/context.go) | body是該cut後直到下一tone的1–3個raw keys；tone=`layout.Standard.Symbol(key)`，不含tone在body，無neutral expansion |
| 數字格式 | [mixedparse.NumberLike](../core/mixedparse/guards.go) | 只對literal prefix傳入原文，不project到Session、無大小寫／空白修剪。NumberLike空字串=false |
| 英文詞籍 | [english.Dictionary.Has](../core/english/dictionary.go) | 傳raw prefix；Has依原碼ASCII lower lookup。單字元只許a/i且Has=true；其餘長度>1。空prefix=false |
| B5 | [decode.Decoder.Best](../core/decode/decode.go) + [input.Exact.Expand](../core/input/input.go) | `Decoder{Lex:model.Source(),Score:model}.Best(Exact.Expand([first_reading]),nil,true)`。零Left=BOS。直接取log，不exp；-Inf→B5=0,B6=1；finite→B6=0 |
| W9/W10 | [lexicon.Source.Exact/HasPrefix](../core/lexicon/lexicon.go) | 使用相同model.Source()。只用Exact算詞長／覆蓋；HasPrefix只證明可保留；Exact查最長8音節，DP skip只用於W10而非render k |
| W11 | Decoder.Best | 完整leading k音節、prefix=false，其餘欄位同B5。score/k，k=0不呼叫decoder、0/1 missing。不可用Best(true)的prefix mass當完整mean |
| 原完整決策 | [finishMixedTone／mixedBoundary](../core/session/mixed.go)、[mixedAggressiveBoundary](../core/session/mixed_policy.go) | 僅C/A真實Session對照使用；不抽取為B flags，不把Guards的13 predicates當原決策。B v2刻意無此兩個未定義features |
| 原單音節分數差異 | mixedSyllableScore in mixed.go | 原版是Best(true)後exp。B5只取相同decoder條件下的log；測exp(B5)與空store原分數相等（finite範圍），prefix-only支持時B5可finite。與W11的完整解碼不同 |
| mode/key/output | [Session.Handle／View](../core/session/session.go)、[proto.Toggle／Snapshot.Text](../core/proto/proto.go) | M是真實Toggle；C/A省略該起始Toggle，其他鍵保留。Baselinewrapper只觀察不改生產分支；application ledger另記unhandled keys。不要沿用舊observer的raw只收33..126規則 |

## Factory與隔離

baseline每列新建 `personal.NewModel(loaded.Model.Lex,loaded.Model.LM)`、Store新、Now=nil、`session.New(model)`；`MixedInput=true`、C的`MixedInputAggressive=false`、A=true、M/R=false；Preview02=true、English=loaded.English、EnglishAutocomplete=false、NeighborKeyRecovery=false；Opt.DocumentContextWeight=0。Fuzzy、rerank、punctuator、Simplified皆nil／零值，standard、toneless=false，所有其餘Session constructor預設不改。來源parity測試需比較同一constructor／設定與wrapper on/off，不聲稱上游v0.2.7等同此baseline。

正式品質custom_tokens必須空。compat自訂英文token以 [vocab.New／Add](../core/vocab/vocab.go)建立，Readings空、SourceManual、weight0，步驟與現有 [NewMixedEvalBaseline](../core/session/mixed_eval_adapter.go)相同，但不重用它不支援空白的ledger。顯式literal／candidate lock先於score，protected狀態每個key檢查，不能靠feature值代替禁呼叫。

## 固定worked feature vectors（tiny，不是public品質）

tiny詞庫用lexicon.NewMem，只Add「窄小」、reading=`ㄓㄞˇ ㄒㄧㄠˇ`、PRW=1。tiny scorer.Prob(left,word)在word="窄小"時回exp(-2)、其餘0，與left無關；English.Has("x")=false（單字元x不認英文）。這是tiny測試用scorer，不是public資料model的替代。已用既有decoder獨立檢查：首reading Best(true)=-2、Best(false)=-Inf，完整二reading Best(false)=-2。P1/P3仍須把這組向量納入未來實作測試，不能以本文件檢查代替feature adapter parity。

- `x593` cut=1、firstreadingㄓㄞˇ：B=`[0,0,0,0,0,-2,0]`，L=`[0,0]`，W=`[0,0,0,1]`；k=0，顯示raw、suffix pending。
- `x593vul3` 同cut：B不變，L=`[1,0]`，W=`[2,1,-1,0]`；k=2，顯示x窄小。
- `x593v`：B不變，L=`[0,1]`，W仍`[0,0,0,1]`；不得用未出現的ul3更新W。
- 只有單字「窄」且其log=-2的tiny：`x593v`的k=1，W=`[1,0,-2,0]`，顯示x窄v。
- literal固定13維向量依契約；B5/W11的missing flag分別為1。

## 必做parity與負例

1. SpellExact逐一驗body×空白/3467，包含不合法反序、重複class、tone-only；不得用「能產生某Composer」代替exact。
2. B5與空store原單音節helper在完整單字、只有詞首支持、無詞三種狀態對照；W11同時證明prefix=false時prefix-only為missing，完成窄小為-1mean。
3. English大小寫、a/i/x、unknown prefix、NumberLike(`593`)/(`x593`)/(`2.4`)來源對照；B snapshot後再加vul3不改B，mask只變可用資訊。
4. 真實M/R/C/A在首聲調空白、無composition英文pass-through、prelude、中性音及原作者kevin87/nina87/lisa0304/gpt4/h264/gpt4o/hello4/zoom3/hello2.4保留任務用新ledger比完整view／commit。C/A實際輸出不是gold，不為了parity把其誤傷當正確。
5. baseline Handle可在最終commit學進當列disposable store，獨立記錄；候選scoring store digest前後完全不變，不跨列reuse。private/protected所有實驗query=0；原版自身模式行為仍照原版，不宣稱它不執行普通中文組字。
