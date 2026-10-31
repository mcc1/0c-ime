# 中英混打離線 evaluator 契約 v1

狀態：Phase 0 的實作規格；尚未執行候選生成、排名訓練或輸入實驗。此文件與[實驗計畫](mixed-input-experiment-plan.md)共同補足提案 C1–C3，仍須獨立審查。它只授權未來離線工具的設計，不授權修改產品行為、發布安裝包或啟用 live ranker。

規格 ID `mixed-eval-v1`，generator `exact-suffix-v1`，features `mixed-features-v1`，fixture `mixed-fixture-v1`，split `mixed-family-v1`。任何語意變更均升版、重建 split 與報告，不能沿用舊 held-out 結論。

## 來源與不可變依賴（C2）

本次核對來源 HEAD：`a731b8a2069a97c3c76ba873d7b00a0f2e9686c2`。原提案的 preview-2 稽核基準是 `e754019571573051d8c29e06526802c31d5172e2`；兩者 provenance 分開記錄；core tree OID 同為 `6e79202abd70dad3b1252b8809bb6be8f57bb8b9`，internal tree 同為 `66d917ab40bc02413f592e005546f522c9048163`，cmd tree 同為 `baa28e98c8bce40783574a93ccb0d4b85add9290`，差異僅提案／稽核文件。主 baseline 固定後者，因 runtime source tree 等價，可在本 HEAD 加純觀察 adapter，報告需記錄此對應；未來 runtime source 有變動則不能直接沿用。

| Source body | 已核對的行為與設計決定 |
| --- | --- |
| [mixed.go](../core/session/mixed.go) `mixedComposer`, `keysInSyllableOrder`, `readingReachable`, `readingContinues` | 不能只搬 dictionary lookup。strict spelling 要求 Initial→Medial→Final 的 class 嚴格遞增且不覆寫；reachability 還會看 literal boundary、前置 syllables、retained phonetic prefix。v1 用以下純 projection。 |
| [mixed_policy.go](../core/session/mixed_policy.go) | `mixedExactReading` 不做 neutral fallback；aggressive 的群組順序與 digit-initial override 是 baseline，不是新 generator 的 eligibility。 |
| [mixed.go](../core/session/mixed.go) `finishMixedTone`, `mixedKey`, `flushMixed` | 現行 `mixed.raw` 64-byte flush 與 tone 後清空不等於 retained-run cap。Enter 保持顯示；各 baseline 必須實際 replay。 |
| [symbols.go](../core/session/symbols.go) `decodeWithLiterals`, `insertLiteral`；[pending_literal.go](../core/session/pending_literal.go) `prefixLiteralBefore`, `joinPrefixLiteral`, `completeDecodable` | literal 分割 decoding；只作 prefix 的原樣注音有獨立語意，不能混為一般 literal。hypothesis 不能呼叫這些會寫 session 的函式。 |
| [decode.go](../core/decode/decode.go) `Decoder`, `Best`, `InitialLeft`；[input.go](../core/input/input.go) | `Best(...,false)` 無完整路徑時為 `-Inf`；`true` 可採 prefix。`input.Exact` 不承載不同 literal 長度；每個 suffix 各自解碼。 |
| [personal.go](../core/personal/personal.go) `NewModel`, `NewStore`, `Prob`；[lexicon.go](../core/lexicon/lexicon.go) `Source` | dictionary 不是 scoring model。空 personal store 加系統 LM 才是本實驗統計來源。 |
| [dictionary.go](../core/english/dictionary.go) `Has`；[english.go](../core/personal/english.go) `EnglishKnown`；[vocab.go](../core/personal/vocab.go) `VocabEnglish` | dictionary exact 為 ASCII case-insensitive；custom exact 為 case-sensitive；personal known 與 privacy eligibility 另計。英文權重不是 corpus probability。 |
| [neighbor_recovery.go](../core/session/neighbor_recovery.go) | 只修 bounded suffix、排除 valid original、automatic repair 有不學習 provenance。v1 關閉，不能把修正混入 exact recall。 |
| [rerank.go](../core/session/rerank.go) `rerankPath`, `rerankAllowed`, `rerankSegment` | 同 reading 的中文 spans 才可 neural rerank。v1 scorer nil，不呼叫 neural、不跨 reading 比值。 |
| [session.go](../core/session/session.go) `Handle`, `commit`, `reset`, `DefaultOptions` | `Handle` 可能 sync time／學習；private 會改 English eligibility。hypothesis 不可呼叫；public baseline 不用 Private 假裝唯讀。 |
| [datadir.go](../internal/datadir/datadir.go) `Open`, `openPack`；[provenance.go](../internal/eval/provenance.go)；[eval.go](../internal/eval/eval.go) | 驗證資料包後另記 English manifest。重用 Provenance/WriteReport/Percentile；不用 content-based two-way `Split`，不用 live `Typist` 枚舉 hypothesis。 |

每次 run 先以 `datadir.Open(assets/data)` 驗證公開 sealed package；要求 Chinese `manifest.json` 與非空 `english/manifest.json`，缺失、prototype fallback、hash mismatch 都 fail。報告記錄兩份 manifest 原始檔 SHA256、manifest 內 payload hashes、實際檔案 hashes、source/evaluator commit 與 dirty diff SHA256、Go／OS／arch、規格與 fixture／assignment hashes。此規格不固定目前資料包的數值雜湊；執行前產生 `run-lock.json`，後續 train/dev/test 必須用完全相同 lock，否則新 experiment ID。

每個 fixture／policy 建立新的 `personal.NewModel(lex,sys)`、`personal.NewStore()`、DefaultParams、`Now=nil`，不 load/save user.json、不讀 HOME、不開 personal 目錄、不呼叫 Learn/SyncTime/VocabAdd。主資料集 Vocab=nil、Store.English 為空；custom-token 子實驗只用 fixture 的公開 `custom_tokens`，以 `vocab.New()` 後 `Set.Add([]vocab.Entry{{Text:token,Readings:[]string{}}}, vocab.SourceManual, 0)` 初始化（整批依 token UTF-8排序；任一 error fail），初始化完成後唯讀、不得寫入磁碟。英文 exact/prefix 查詢一律唯讀。baseline replay 可能觸發既有 commit 學習，但只在自己 disposable store 中；每列重建，不能把學習傳到下一列。

### FrozenContext 與 pure projections

公開 API 型別（欄位名稱固定）：

```go
// All slices/maps are deep copies; no Session or Model pointer is accepted.
type Context struct {
    Generation string
    Units []Unit // in-composition left prefix, ending exactly at cursor
    RetainedPrefix *Retained // only prefix=true pendingLiteralBefore equivalent
    LastWord string // frozen last complete Chinese displayed word; empty means BOS
    Private, Paused, LocalBlocked, English, Toneless bool
    Layout string
    CustomTokens []string // literal, case-sensitive, sorted unique ASCII tokens
}
type Unit struct { Text string; Reading string; Literal bool; OriginalKeys string }
type Retained struct { Reading string; OriginalKeys string }
```

v1 只在組字尾端，`Units` 每項是一個 Chinese syllable 或一個 literal rune；`Reading` 是 canonical `string(zhuyin.Syllable)`，literal 時空字串。`RetainedPrefix` 只有緊鄰游標、prefix=true、tone 尚保留的 pending literal 才存在；其佔用的 Units 已是 literal，不另附重複 Chinese Unit。來源是 fixture 的 prelude 由 isolated aggressive A baseline replay 後匯出的 snapshot（C保有自己的prelude session作compatibility，E統一用A frozen context），或 unit-test 人工 typed Context，禁止讀 application 任意前文。prelude在最後一事件後以純觀察snapshot凍結，不自動Commit、不新增physical delimiter；prelude不進入本window。像`ru,3WQ5j04fu06`的context case明確編成prelude=`ru,3`、events=`WQ5j04fu06`，另以A/C重播完整stream記compatibility。這是實驗者指定的context/run切口，不證明live engine已能自動找到切口；live run creation仍須Phase3規格。主pilot的boundary品質只針對宣告的run起點，不能外推為全stream segmentation。主 pilot 含 empty、Chinese tail、literal number、retained-prefix 四類；不支持游標中插入、跨已提交 document context。`LastWord` 由最後一個非 partial、End==cursor 且不含 literal 的 frozen path segment 取 Word，否則空；保留 prefix 後 LastWord 空。主資料集與 baseline 均 DocumentContextWeight=0，沒有跨提交 context。

候選在 run 開頭插入非空 literal prefix 時，將 Context 的 RetainedPrefix 清空、LastWord 清空；把 prefix 每個 rune 作 Literal Unit 附加。隨後每個 Chinese span 從左至右附加 Chinese Unit。前置 literal 的確會切斷 contextual continuation。任何 guard 只讀這個 projected value，禁止建立／clone Session 再試打候選。

`SpellExact(body,tone)`：Standard.Symbol 原大小寫查詢（不把大寫轉小寫）；body 1–3 ASCII bytes，每鍵非 Tone、class 嚴格遞增、無同 slot。Composer.Add tone 成功且結果 Valid、非 Untoned；不做 neutralFallback。`Reachable(ctx,syl)`：單音節 Exact 或 HasPrefix；或 retained prefix+syl 的 Exact/HasPrefix；或最近 continuous nonliteral Units 的每個 suffix+syl 的 Exact/HasPrefix，最多 `decode.MaxWordLen-1` 個既有 syllables。掃描停在 literal，不跨 literal。這是 `readingReachable/readingContinues` 的純對應；所有 lookup 使用同一唯讀 lexicon，不把前文 syllables 算入本次 key coverage。

獨立 `LegacyReading(ctx,body,tone)` 供 guard parity，沿 `mixedReading` 的 Composer、不強制 class order，並依 `neutralFallback` 在 Tone5 不可達但 Tone1 可達時退回 Tone1。不能用它生成 strict exact 路徑。`AfterChinese`、`LiteralNumberBefore`、`PrefixLiteralBefore` 按上述 Units/Retained 模擬來源 helper。`LiteralNumberBefore` 只收緊鄰的 literal 單 ASCII rune `0123456789.,:/-`，並非在所有文字裡搜尋數字。

統計 feature decoder 固定：`Lex=lex`, `Score=empty-store model`, `Left=effective LastWord`（遇候選 literal prefix 重置 BOS），`Beam=0`, `TopM=64`, `TopB=4`, `Continue=.5`, `SeparateContinuation=true`, `SuggestPrior=5`；Evidence／CorrectionPenalty／CorrectionUsage=nil，MaxCost／MaxCandidates=0，OnlyCosted=false。對 `input.Exact{}.Expand(readings)` 呼叫 `Best(lat,nil,false)`，只解本次完整 suffix，不能改寫 frozen prefix 或讓詞跨越它。記錄完整原始 log score，feature 取 score/n。prefix support 使用另外的 dictionary cover，不偷偷改成 `Best(...,true)` 補分。顯示 prefix-only 時直接顯示 canonical 注音，完整路徑顯示 Best 的 Word 串接；同 reading 同分沿 decoder 穩定順序。空 suffix 不呼叫 Best。

這是刻意窄化：可利用 frozen Chinese last word 的 LM context 與 contextual reachability，但不重新分詞 frozen prefix，也不解三音節以上歧義。跨舊詞分詞需要下一版實驗；不能宣稱已測過全 live session。

## Exact generator（C1/C2）

輸入 `Generate(Context,raw,lastToneEnd,legacyObservation)`，raw 是本 window 的 ASCII physical bytes（含 tone bytes），不含 prelude；Context frozen。只在 `3/4/6/7` 的事件執行新搜尋。space 是 finalizer，沒有新增 space-tone 搜尋；一聲只在 baseline compatibility 測，不在 exact pilot 的推論範圍。

1. 保留完整 literal lane `[0,len(raw))`。對每個 start `a=0..len(raw)-2` 枚舉 suffix；prefix `[0,a)` 一律保持原 bytes，允許失敗 tone 留在 prefix。
2. 從 a 起嚴格解析 1 或 2 個連續 `body tone`；body 1–3、tone 只能 3/4/6/7，不能跳 byte。解析必須剛好到當前 tone 末尾。body 中任何 tone 都不能當 body；不枚舉 interior literal gaps。一個 suffix 中可有兩個 exact words，但不得夾 literal；invalid first syllable 後 resumption 由較後 start 完成。
3. 對每個 syl 用上一節 projection 的 Reachable。另做 suffix dictionary segmentation DP：允許每段為 Exact word，最後一段可為 HasPrefix。`whole_word`=整 suffix 有 Exact；`complete_cover`=全由一個或兩個 Exact 覆蓋；`prefix_only`=無 complete_cover 但至少一個末段 HasPrefix cover。若三者皆否，且只靠 context Reachable，標 `context_only` 並保留；score missing 是合法結果，不可把它當完整詞。每段覆蓋 witness 記在 candidate，不能拿前置 literal 合成詞。
4. candidate 是 `segments` array：可選一段 literal prefix，其後一／二段 Chinese，半開 byte spans、Chinese 段含 tone；literal text 就是 slice；Chinese reading 是 canonical Syllable 字串。candidate identity 是 compact JSON array `[[kind,begin,end,reading],...]` 的 UTF-8（literal reading 空字串），ID=`c:`+SHA256(identity) 小寫 hex。dedup 只看 partition+reading，合併 witness／來源 flags，不能按可見文字 dedup。
5. 依 `(complete_cover desc, Chinese byte count desc, first Chinese begin asc, readings UTF-8 lex asc, identity asc)` 保留最多 8 個 strict exact candidates。記錄 uncapped 與 capped 集合、cap pruning 原因。加 protected literal 及 protected legacy observation，總最多 10 lanes。若 legacy 與 exact 相同則標記同 lane；否則 legacy 為 opaque baseline-only lane，不能參與 exact recall、feature scoring／訓練或 longest-suffix 選擇。strict generator 加 literal 是 proposed ranker 的全集，不能為救 baseline 硬塞 nonexact reading。

舊 aggressive 可能 neutral fallback、反序 whole-reading 或其他特殊處理；因此 opaque legacy 不承諾 strict validity。報告分開 `legacy_present`, `legacy_exact_equivalent`，以及「intended path only in legacy」；後者不算 exact recall 成功。longest-valid-suffix comparator 在 capped strict candidates 中選 Chinese bytes 最多、complete_cover 優先、identity asc；若無 strict path 選 literal。另報 uncapped longest diagnostic，不能拿它替代同 cap 比較。

## Offline state machine 與顯示（C1）

三條互不共享 session 的 replay：A preview-2 aggressive、C preview-2 conservative、E experiment。A/C 各真實 Handle 一次事件，E 只純 values。E 狀態 `Idle`, `Collecting`（零 tone）、`Open1`（第一次 tone）、`Closed`；附 `raw`, frozen Context, firstToneEnd, shown partition/text, pending suffix, revisionCount, closureReason。Closed 的 ownership 永遠 immutable；它不等於 application commit。每個 window 最多 64 raw bytes、最多兩個 tone opportunities，含 literal intent／failed tone。這裡 opportunity 指非空 body 後的 3/4/6/7（第一次無 legal reading 也記一次）；run 起點立即遇到 tone 當一個 literal byte，仍記 opportunity。如此不能靠一直失敗 tone 讓 window 無限 open。

E 有三個 policy 模式，必須在報告區分：

- `enumerate`（Phase 1）：記錄 candidates、vectors 與 baseline observation；`shown` 借用 A 的 display 作「參考 history」，不是 proposed UI。`proposed_display=null`、proposed stability metrics=null。candidate feature 的 stability 用前一 tone 的 A partition 投影，標 `history=legacy-lagged-v1`；不可把標籤填成 previous display。
- `scripted`：unit event fixtures 以明確 candidate ref 選 shown；只驗證 state machine，不納入 accuracy、穩定性成效或訓練。
- `ranked`（Phase 2 才實作）：首 tone 用凍結 linear ranker 選 shown；次 tone 用同 ranker 與 stability margin。history 僅用此 policy 上一個實際 shown partition，與 train labels 無關。訓練前的 feature cache 中 stability 欄不能直接沿用 legacy history 當 rollout 結果。fitting 規格必須採 causal rollout 重算 stability 或將 stability coefficient 固定 0；在該後續規格批准前不開始 fit。本 Phase 1 不選 coefficients／optimizer。

| 事件 | E 的確定轉移與禁止事項 |
| --- | --- |
| 合資格首個 printable ASCII non-space key | Idle→Collecting，capture Context；原 key 追加 raw，一次事件只 append 一次。大寫保留，只有 lowercase layout keys 可作 body。 |
| 尚未完成 tone 的後續 key | 追加 pending；shown 已完成區維持不變，pending 以 raw bytes 顯示。v1 pending display 刻意用 literal，並非對產品 phonetic pending UI 的 parity 主張；不可 rerank 前 boundary。 |
| 第一 tone | 追加 tone、enumerate／選首 provisional，Collecting→Open1；raw 不清空。prefix-only 顯示 canonical phonetic reading（含非一聲調）；pending 清空。沒有 strict candidate 也留 literal lane。 |
| 第二 tone | 追加 tone、重新搜尋整 raw，依 causal policy 最多一次 revision，Open1→Closed。revision 只比較截至 firstToneEnd 的舊 bytes／completed Chinese spans。普通 `WQ賺`→`WQ賺錢` 不算；`WQ5萬`→`WQ賺錢` 算。 |
| Closed 後一般 key | 建立下一 window，以 closed prefix 的 value context 為起點；舊 ownership 不可重新搜尋，舊 homophone 也在 v1 offline frozen。 |
| Enter/Commit | 先 freeze，輸出恰等於本 policy 最後 visible text+pending；不 enumerate/rank；清 raw／alternatives→Idle。不是讓 baseline commit E 顯示。 |
| Down/open_candidates | freeze→Closed 後才開候選；此處只允許同 partition/reading 的 word 與 original keys。記 menu token，不顯示 alternative-boundary menu。 |
| select | 必須匹配最近未失效 menu 的 candidate ref；選詞只鎖該 span；original 選取將該 span materialize 為 raw literal。隨即 freeze 全 window，日後不能取回其 raw。未先 open、ref 非同 reading／original、span 越界則 fixture hard error。 |
| space / punctuation | freeze 當前 display、discard alternatives，將原 event 交 legacy finalization adapter 恰一次。space 不在 E 新搜尋，若 existing pending 能一聲完成由 adapter 原路完成，已 frozen ownership 不可改。逗號句點等標準 body keys不是 punctuation event；fixture 必須用 `finalize` 明確表達 delimiter，不可把 mapped body key 當任意 delimiter。 |
| cursor/navigation/backspace/delete | freeze、輸出 materialized display+raw ownership ledger 予 legacy editing adapter，delegate 一次，直到 composition reset 不再開 E window（v1 editing bypass）。不重建被刪 alternatives。 |
| cancel/reset/focus_end | 清 composition、raw、menu、locks、history→Idle；focus_end 的 `commit_visible` 決定記錄目前 visible 作 commit 或 discard，不能自行猜 shell 政策。 |
| mode/private/paused/blocked/layout/toneless 改變 | 先記最後 visible、清 alternatives；之後 E bypass，直到 reset。privacy event 處理前即檢查 flags，零 experimental generate/score；以 value copy 保留 display，不為 freeze 補跑 inference。 |
| model/settings generation 改變 | 舊 snapshot freeze→Closed；清 alternatives，新 window 才用新 generation。不重算已顯示內容。 |

`finalize` 的 offline delegate 不可直接把 E bytes replay through Session.Handle 來重建 hypothesized state；這會重新解析／學習。v1 E editing trace 的 adapter 是 value editing：backspace 移除最後一個 materialized rune（Chinese span 還原與其他 legacy phonetic undo 標 `delegated_unverified`）；cursor 只記錄 frozen value／target，terminal bypass；space／punctuation 的 baseline 結果另記錄，不拿來宣稱 E session parity。這刻意將 live materialization、cursor UTF-16 與 undo 留給 Phase 3。Phase 1 可以驗證 freeze/delegate exactly once，不能聲稱驗過 live editing output。fixture 可以對 A/C 要求實際顯示；E 僅對 freeze 與事件計數要求 assertion。

### 63／64／65 bytes（tone 也計數）

每次 append **之前**：若 len(raw)==64，freeze 舊 visible+pending，closureReason=`overflow`，清 alternatives、將整舊 raw 移入 frozen ledger，再用當前第65個 key 開新 window offset=64；不得丟 byte、不得以第65鍵重評舊 window。若第64鍵為第二 tone，正常 second-tone close 優先，原因 `second_tone`。第64鍵為首 tone則 Open1 仍可存在，下一鍵先 overflow。第65鍵為 tone時不能完成舊 pending body：它是新 window 的 literal tone opportunity。legacy pending-token overflow 仍照其本來算法，另報差異。

必備 scripted trace（`A^n` 指 n 個獨立 `key:A` 事件；不是一個多 byte event）：

| keys | event index → expected E state |
| --- | --- |
| `A^63` | 63→Collecting, raw=63, revisions=0 |
| `A^64` | 64→Collecting, raw=64，尚未 overflow |
| `A^65` | 65→Collecting，新 raw=`A`，frozen raw bytes=64，overflow count=1 |
| `A^60su3A` | 63→Open1（`su3` span [60,63)）；64→Open1,pending=`A`；65 再輸入 `A`→新 Collecting,raw=`A`，舊 [0,64) 固定 |
| `A^61su3` | 64→Open1，首 tone 佔第64 byte；第65鍵 `f`→新 raw=`f`，不能續評首音節 |
| `A^60su3f` + `6` | 64→Open1,pending=`f`；65→新 Open1，raw=`6` literal，舊 f 不得與6成音節 |
| `A^58su3cl3` | 64→Closed,reason=second_tone，tone spans 含3，不 overflow |

常用兩音 trace：`WQ5j04` 首 tone後 raw 長6，scripted legacy-equivalent `[L0:3,C3:6=ㄨㄢˋ]`；`f`、`u`、`0` 依序只改 pending；最後 `6` 可 scripted 選 `[L0:2,C2:6=ㄓㄨㄢˋ,C6:10=ㄑㄧㄢˊ]`，useful revision=1，Closed。這是 state fixture，並非模型預測。Enter 後 commit=`WQ賺錢` 不再 rank。

### Legacy replay 與 ownership observer

A/C 使用 pinned preview-2 Session.New、DefaultOptions（Beam8、PrefixAware=true、Continue=.5、SeparateContinuation=true，但 DocumentContextWeight=0），Standard、MixedInput=true、Preview02=true、MixedInputAggressive 分別 true/false，Neighbor=false、Toneless=false、Fuzzy zero、Rerank.Scorer=nil、Punct=nil、Simplified=nil、EnglishAutocomplete=false。每列獨立 session/store；公開列 Private=false、Paused=false、blocked=false；專門 privacy rows 才改 flags。settings 的其餘值均 New 預設，不從使用者 config 載入。

adapter 位於 `core/session/mixed_eval_adapter.go`，`//go:build mixedeval`；只暴露 baseline runner／deep-copy observation，不暴露任意 restore。第一版刻意只保證 append/end-only 的 ownership extraction；原版 Session 本身正常 replay 所有事件並報 text/commit。adapter 在事件前記目前 raw byte ledger，事件後以 typed units 左至右消耗 ledger：literal Unit 只能消耗完全相等的原 bytes；Chinese Unit 以 `mixedKeys[pos]` 的原始 keys 長度消耗（比較 slice，不搜尋子字串）；pending `mixed.raw` 消耗尾端；來源已保存的 `literalKeys` 用於 original 選取。不可由可見中文反查 physical keys。tone 仍在 keys span 裡；不同 syllables 相同 keys 也不會產生搜尋歧義。

為 prefix-only pending literal 唯一新增的觀察 hook 是 `insertMixedTyped(syl,keys)`：在進入時捕捉 cursor 與對應當前 raw token IDs，defer 於完成後觀察是否成為 pendingLiteral；將該 object 的 start／canonical reading／original byte IDs 記入 adapter-owned ledger。`joinPrefixLiteral` 成功時從前後觀察差異及已記 IDs 合併，無須新 mutation hook。hook interface/no-op 定義在 `mixed_eval_observer.go`，一般 Session observer nil，不能影響 return或排序；runner只在mixedeval build可用。不得為了枚舉 hypothesis呼叫此hook或Session。

若存在 raw-rewriting English assistance、`reviseLetterI`／`reviseDigitWord` 等無法從既有 mixedKeys/literalKeys與已記token IDs唯一重建的路徑，或 editing後來源丟失，觀察標 `ownership_status=unsupported`，保留 text/commit結果與原raw，不猜位置、不追加廣泛mutation hooks。privacy／editing fixtures可驗text/freeze/inference，但不能假稱這些路徑的E materialization已驗證。每事件要求所有可映射byte恰有owner、原raw重組相等；失配是`invalid`並hard fail，只有事先列明的特殊路徑可為unsupported。

報告必含每policy ownership `mapped/unsupported/invalid` 的rows/events分子分母及case IDs。generator intended recall仍對全部有label的Chinese rows計算（不需要baseline ownership）；baseline text output error也不受影響。first-boundary、legacy-lagged stability與ownership revisions若任何所需row unsupported，整個paired metric=null並報coverage，不能用剩下rows報passing rate；可另列mapped-only diagnostic但不得當主成效或改變分母。opaque legacy lane可僅含text、partition=null，不能供feature12 reference；該row的stability feature=null並排除所有feature fitting（Phase1仍算recall）。mandatory或release primary row有required metric unsupported就fail readiness，必須補最小observer並重新審規格，不能剔除失敗案例。若unmapped是新且未列明source path，直接fail本run。

所有可映射事件的ledger與`mixedKeys`、`literalKeys`、pendingLiterals、literal bitmap、cursor、snapshot核對。A/C與未加observer原版逐事件snapshot/commit/store digest parity必須成立，均在disposable stores執行。這個窄observer不承諾全Session serialization，也不在live Session反覆試打hypotheticals。

## Feature vector（C2）

已知值全部 float64，schema 順序固定；observer unsupported 時第12欄可null，整列不可用於fit，並依上述規則報coverage。Boolean 0/1；不做 z-score、imputation 或 clipping。除 `mean_log_score` 可為任意 finite 負實數，其餘皆 [0,1]。raw score 非 finite 時 JSON 寫 null 並記 `score_status=unreachable|nan|positive_inf|negative_inf`，vector 用0與missing=1，不傳 NaN/Infinity。查詢 failure 不是 missing：abort run。

| index/name | 精確定義 |
| --- | --- |
| 0 literal_lane | fully literal=1，strict Chinese=0 |
| 1 coverage | Chinese spans 的 bytes（含 tone）/len(raw) |
| 2 syllables | 本 suffix syllable count/2 |
| 3 whole_word | full suffix Exact 非空 |
| 4 complete_cover | suffix 可被 Exact entries 完整分割 |
| 5 prefix_only | 無 complete_cover，末段 HasPrefix cover |
| 6 context_only | 只有 contextual Reachable，無上三種 suffix cover |
| 7 mean_log_score | 上述 Best false 的 finite score/n；不可用 single-syllable exp score 代替 |
| 8 score_missing | 無完整 finite score=1，否則0 |
| 9 english_dictionary_exact | candidate literal prefix P 非空且 English.Has(P)；case-insensitive ASCII |
| 10 custom_english_exact | Preview02=true 且 custom_tokens exact P（case-sensitive），與9獨立 |
| 11 personal_english_exact | EnglishKnown(P) 且 eligibility；主資料集固定0，synthetic-store parity test 才非0 |
| 12 boundary_changed | candidate 與前 shown 投影到前 completed tone end，owner/spans 不同=1；無前 tone=0；opaque reference 無法 map 則此欄 null，依 ownership coverage 規則禁止 passing paired metric |
| 13–25 guards | 以下13個 boolean，精確順序如下 |

對 strict candidate 第一 Chinese span `[a,b)` 設 P=raw[:a]、B=raw[a:b-1]、K=raw[b-1:b]、T=Standard.Symbol(K)。所有 guards 都在「frozen Context 加 P」的同一 projection 計算，不能有的看 raw 全 run、有的看 E shown。guard 須按來源原大小寫與 `unicode` 判斷，輸入已限 ASCII；不可對 B/P lower-case。來源 guards 作 evidence，不是 hard veto。literal lane **完整 vector 固定** `[1,0,0,0,0,0,0,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]`，numeric evidence 可另記 diagnostics，不能悄悄改 vector。

| index/name | pure projection of source helper、實參 |
| --- | --- |
| 13 number_like_run | `numberLike(raw)`，整 window，含 tone |
| 14 number_like_prefix | `numberLike(P)` |
| 15 english_then_number | `englishThenNumber(B,K)` |
| 16 digit_tone_number | `digitToneNumber(B,T)`；含 preceding literal number、AfterChinese、neutral reachability |
| 17 number_after_literal | `numberAfterLiteral(B,K)`；LiteralNumberBefore 從 projection 重建 |
| 18 rare_digit_reading | `rareDigitReading(B,T)`；其舊 `Best(...,true)` isolated probability 及 system floor 僅此 guard 內沿用，不能與 feature7混用 |
| 19 literal_digit_before | `literalDigitBefore()` |
| 20 english_then_punct | `englishThenPunct(B)` |
| 21 mixed_literal_prefix | `mixedLiteralPrefix(P,B,K)`，P 空時按原 helper 返回，不自行特判 |
| 22 digit_word | `digitWord(B,T)`（explicit tone 下固定 false，保留為空白 compatibility parity 欄） |
| 23 punct_key_word | `punctKeyWord(B)` |
| 24 digits_read_as_chinese | `digitsReadAsChinese(B,T)` |
| 25 mixed_space_number | `mixedSpaceNumber(B,T)`（診斷原 helper boolean，並非真的有 space event） |

所有 guard 的 transitive dependencies 固定上述來源 bodies，包括 `mixedReading` 的 neutral fallback、EnglishKnown eligibility、keysInSyllableOrder、mixedAfterChinese、literalNumberBefore；禁止用改寫的「數字 regex」替換。`digitWordKeys`、`reviseDigitWord` 是 existing-state mutation/provenance 邏輯，不是 guard vector；baseline保留實際text結果，這些路徑的ownership可依前節標unsupported；strict hypothesis不執行。未知帳號 evidence 本版只用 dictionary/custom flag 缺失、上述各 guard 與 literal lane，沒有新造 identifier regex。URL／underscore 是保留原 literal 的 coverage 類別，不假稱已有專門 URL probability。

worked vectors 用 synthetic immutable lexicon/scorer，避免捏造公開 LM 成績：English 僅 `hello`，custom=`WQ`，無 personal evidence；Exact `賺=[ㄓㄨㄢˋ]`、`錢=[ㄑㄧㄢˊ]`、`賺錢=[ㄓㄨㄢˋ,ㄑㄧㄢˊ]`；假 scorer Prob 每詞 `exp(-2)`、Prior 同，PRW=1，不代表 fitted model。代表輸入都在 empty context、無 previous tone，guard ordering 如上。以 name map 列 vector 非零值，未列欄為0：

- literal 任意 raw：literal_lane=1, score_missing=1。
- `WQ5j04`，P=`WQ`、suffix `[2,6)=ㄓㄨㄢˋ`：coverage=4/6, syllables=.5, whole_word=1, complete_cover=1, mean_log_score=-2, custom_english_exact=1。digits_read_as_chinese=1，其餘 guard=0（B=5j0 含 digit、有 reachable reading、不是 numberLike；P 大寫不在 English，custom 不等於 mixedEnglishWord）。
- `WQ5j04fu06`，同一個 exact two-syllable word：coverage=.8, syllables=1, whole_word=1, complete_cover=1, mean_log_score=-1（單一詞 log=-2 /2）, custom_english_exact=1；digits_read_as_chinese=1，其餘 guard=0。
- 移除 `賺錢` entry 後相同 raw 為 two exact words：whole_word=0, complete_cover=1, prefix_only=0, mean_log_score=-2，其他同上。
- 只留 `賺錢` entry，`WQ5j04` 為 prefix-only：coverage=4/6, syllables=.5, prefix_only=1, mean_log_score=0, score_missing=1, custom_english_exact=1；digits_read_as_chinese=1；不把 partial word 當完整 output。
- 上例 P 改 `hello`：dictionary=1、custom=0、mixed_literal_prefix=1，其餘按 raw 長度重算；大小寫 `HELLO` dictionary 同1，body 大寫不正規化。

worked-vector tests 必須以 tiny lexicon 直接跑 decoder驗證；若 decoder 的 repeat/mass 等實際分數不等此假 scorer預期，停止修訂契約／測試並升版，不在 production adapter 另寫 scoring 逼近表格。

## Fixture / event / partition schema（C3）

[fixture.schema.json](mixed-input-experiment/fixture.schema.json) 是 Draft 2020-12 結構檢查；[examples.jsonl](mixed-input-experiment/examples.jsonl) 是公開 synthetic 格式範例，不是實驗資料集或 baseline gold results。讀 JSONL 拒絕 duplicate keys、未知欄、NaN/Infinity、重複 id；UTF-8 LF。每列必填 id/intent/source/families/prelude/custom_tokens/events/assertions，literal/chinese 必須有 final accepted partition；ambiguous 必須至少一 literal 與一 Chinese partition。event 單鍵字元，沒有多 byte string 假裝一事件。`at` 為零起算 events index，prelude 不計入；`window` 為零起算 window index，span 對該 window raw 半開 byte offsets。

`partition` 是 segment array，literal reading/text 欄以 schema 區分。逐事件 validator 重建 raw/window ledger，要求 spans 從0連續、無空段、無重疊、尾端恰覆蓋當時全部 consumed raw；Chinese 每 span 一個 canonical reading、body+tone strict roundtrip `SpellExact` 一致；禁止 `Untoned`、neutral fallback 等寫進 strict accepted partition。`pending` 可以存在 assertion 中最後一個 literal segment，且標 stage=`pending`；這類 assertion 不計 final recall。已 complete partition Chinese 不可含未消耗 tone。literal 必須 text==raw[begin:end]。語意 validator 用 Go zhuyin.Syllable.Valid + WithTone/String canonical roundtrip，不只正則。

`assertions[].policy` 為 `generator|scripted|aggressive|conservative`；generator 的 `accepted` 表示 required candidate witnesses，`absent` 表示禁出，與 ranking gold 分開。`scripted` 的 accepted[0] 作該 tone 要顯示的 candidate，第二 tone 才可換；ambiguous 的多個 accepted 都必須 surviving。assert `state`, `closed_reason`, `commit`, `delegate_count` optional；若省略是明確「不 assert 此值」，不能推定通過。release unambiguous 每個 terminal window 必須有 final assertion（stage=final），不得靠省略隱藏失敗。final標籤可在最後tone事件assert生成集合，之後仍必須有明確enter/commit/cancel/focus_end；EOF不自動commit。

selection event ref 為 `{menu_event,kind,span,reading,text}`；menu_event 指之前 open_candidates 的 event index；kind=`word` 的 span 必須 matching frozen Chinese span sequence、reading 為 ordered canonical array，text 必須同 reading dictionary candidate，kind=`original` text 必須 span 原 physical keys。不能用易變的 UI row number／candidate rank；legacy adapter 在 menu 中找相同 canonical tuple，零或多重匹配則 fail fixture。selection refs 與 partition candidate hash 分開，因同 partition 有多 homophones。

每事件 observation 保存 `before`, `after`, `closed_windows`，assertion 的 `window` 查該事件的window snapshot（含剛closed／committed者），不是只查after.current.raw。selected→Enter後仍可assert已commit之舊partition，commit欄從該event輸出取值；不對清空後raw長0驗舊partition。

finalizer實參與baseline proto mapping固定如下：`enter`→`proto.Request{Type:proto.Key,Key:proto.KeyEnter}`，`commit`→`proto.Commit`，`space`→`proto.Request{Type:proto.Key,Key:" "}`，`punctuation`必填單個printable ASCII `text`→`proto.Request{Type:proto.Key,Key:text}`，其餘text禁止；`cursor`必填非負`target`，單位為composition中的syllable/literal-rune slot→`proto.Request{Type:proto.Cursor,To:target}`；`backspace`→`proto.Backspace`，`delete`→`proto.Request{Type:proto.Key,Key:proto.KeyDelete}`；`open_candidates`→`proto.Request{Type:proto.Key,Key:proto.KeyDown}`；`cancel`與`reset`→`proto.Cancel`；`focus_end`必填`commit_visible`，true→`proto.Request{Type:proto.Commit}`、false→`proto.Request{Type:proto.Cancel}`（這是fixture宣告的shell政策）。除cursor外禁止target；除focus_end外禁止commit_visible。baseline select以canonical ref定位目前menu的實際candidate ID，再送`proto.Request{Type:proto.Select,Index:id}`恰一次；找不到或多義即fixture error。裸proto.Commit/Backspace/Cancel皆指`proto.Request{Type:該常數}`，其他欄零值；每個request的Private取目前fixture private旗標。environment為harness控制事件，先對E做freeze/bypass，A/C按固定順序只對變更欄位送Pause(On)、Mode(On)、`proto.Request{Type:proto.Display,Private:flag,Columns:0}`（synthetic harness control request，Columns不變）；private=false不能解除本composition的sticky private，直到Cancel才重建flags。blocked、toneless、layout、generation沒有等價單一proto request：該類mid-composition事件僅測E pure freeze，A/C configuration-transition觀察標unsupported，不納入主品質資料；reset後再按新snapshot建立新session。layout只允許standard/eten/hsu；主品質資料只允許standard。一個environment fixture事件可能包含多個internal control requests；另記control_request_count，不當成physical key重送，也不混入finalizer的delegate_count。禁止發明desktop request。fixture欄位不是JSON desktop協定。


`prelude` 使用相同 events，但不得含 selection/settings；以真 baseline frozen snapshot 建 Context。examples prelude 為空。所有 events 有語意限制：`key` printable ASCII `[!-~]`、不含空白或 backtick；`finalize.reason` 精確 enum；`focus_end` 必須指定 commit_visible；`environment` 固定帶完整 flags/layout/generation；editing bypass 後再收 key 只能 baseline 評測。這些是 harness contract，不是桌面協定變更。

### Family grouping、split 與防洩漏

每列 `families` 至少一個顯式語意 family ID。ID=`kind:value`，kind 為 `prefix|word|literal|template|source|context`，value NFC、ASCII casefold（只 A–Z→a–z）、trim ASCII whitespace；不得刪數字、標點、聲調或用 arbitrary UUID 規避 grouping。Chinese word value 是標籤 accepted text 的完整中文詞（保留同義／不同讀音手工 alias 至同 canonical family）；literal account 以 literal family，無中文仍有 prefix/template。ambiguous 取兩種 intended alternatives 的所有 families。`aliases.json` 固定結構為 `{"version":"mixed-family-v1","aliases":{"word:別名":"word:正名"}}`；空表合法。key/value先按同一normalization，kind必須相同，禁止self-alias、chain、cycle及重複normalized key，canonical value本身不能是key；只作exact replacement、不支援regex。資料作者提供 alias table，review 確認不同大小寫、數字變體、同一 bug 的前後文變體共享 template；同 source template 生成的近重複共享 source family。fixtures 不能把整個 `synthetic` 類別當一個 family；source family 指獨立生成原型，非所有 synthetic rows。

validator 自動補 overlap edges：相同 normalized explicit family；相同 full raw ASCII-casefold 字串；相同 `(normalized literal prefix,ordered reading sequence)` 的任一 accepted Chinese interpretation；相同完整 literal-intent token casefold；相同 normalized custom token；相同 prelude context signature（非空才加）。跨 intent 同 raw 一定合併；相同raw但不同明確prelude/custom/environment條件可有不同意圖，仍同group；完整條件相同卻互斥unambiguous標籤則要求改為一列ambiguous或hard error。用 union-find 求 transitive connected components：A共享prefix與B、B共享word與C，ABC同一 group，禁止只選其中一 family分開。可形成巨型 component，不能為了湊樣本任意斷 edge。

每 component 的 group_id=`g:`+SHA256(UTF8(sorted normalized family IDs + auto-edge keys，每項以JSON string encoding，以LF串接)))；IDs 去重、UTF8 byte排序，最後不加LF。split hash=`SHA256(UTF8("mixed-family-v1\nseed=20261101\n"+group_id))`，取 digest 前8 bytes unsigned big-endian，mod10：0–5 train、6–7 dev、8–9 test。seed 是固定字串，不用 PRNG／語言內建 hash。將 assignment.json 寫成 `{"version":"mixed-family-v1","seed":"20261101","fixtures_sha256":"<原JSONL bytes SHA256>","aliases_sha256":"<原aliases.json bytes SHA256>","rows":{"row-id":{"group_id":"g:<hex>","split":"train|dev|test"}}}`，所有object keys依UTF-8排序、UTF-8 LF、JSON indent2與末尾LF；人工只 review，不 re-roll seed或搬 group。任何新增/alias/label 造成 component 改變都產生新版 dataset與split，舊 held-out 結論失效。既有 assignment 與重算不符、同 edge 跨 split、duplicate raw contradictory gold且未標 ambiguous，都 fail closed。

pilot 至少100 Chinese、100 literal、30 ambiguous rows；split 後 train/dev/test 都必須有 Chinese、literal、ambiguous，且每 split 至少10個 independent unambiguous Chinese groups、10 literal groups、3 ambiguous groups。未達就補獨立來源資料並新 dataset，不把相關 variants 當 independent；目前格式 examples 刻意不達門檻，驗證 examples 只跑 schema/semantic，不跑 adequacy gate。release 額外要求 frozen test 至少300 Chinese-intent groups及300 literal-intent groups；按 group 等權主指標，每組一個由 `SHA256("primary-v1\n"+row_id)` 最小、row_id tie asc 的 unambiguous row 作 primary；其餘 variants 報 worst-case group incidence與 row-level diagnostic。Chinese/literal family overlap可在同 group 同時各取一 primary，但全 split仍同組，報 overlap數。

這裡「獨立」只保證已知 grouping 的不交叉，不能由 hash 證明人類語料統計獨立；release 需要作者／reviewer確認來源原型獨立，對未知 dependence 報限制。不得讀 held-out 後為了改善結果改 alias／group。
