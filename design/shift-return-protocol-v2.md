# Shift 後自動回中文：實驗契約 v2

狀態：EGM 修正稿；尚未收新資料、實作 evaluator 或跑品質評測。此文件與同目錄的 [source map](shift-return-source-map-v2.md)、[schemas](shift-return-v2/schema.json)、[例子](shift-return-v2/examples.jsonl)、[數值例子](shift-return-v2/numerical.json)共同構成契約，取代舊 v1。舊審查版本保留於 Git `e91bbd7`。JSON schema 固定交換形狀，本文件固定跨欄位語意；兩者衝突即工具錯誤，不能任選其一。

## 1. 問題、方法與替代方案

Shift 指定英文起點；在一個未提交片段內，判斷何處恢復中文，允許後文重切先前原鍵。只研究一次英→中，以及首段誤按後的恢復；不改一般中文 decoder、不自動進英文、不修正誤按本字、不處理後續再次進英文。第一聲空白及3/4/6/7均在範圍內。

| 方法 | 可回答什麼 | 成本／限制 | 本輪決定 |
| --- | --- | --- | --- |
| 原版手動切換 | 現有操作成本、正常 decoder 上限 | 不推斷意圖，需要第二次 Shift | 保留 M/R 對照 |
| 固定詞證據規則，例如首段能接成二字詞就轉中文 | 詞證據是否可作觸發器 | 易解釋、不需訓練；無法自行校準帳號、詞長、詞頻與 literal 的取捨，依規則優先序可能只是轉移誤傷 | 不選為主要探針；不是宣稱它無效。若學習探針缺乏收益，優先考慮此替代，而非直接換更大模型 |
| 局部線性比較 | 在同一組 raw hypotheses 下量測 B/L/W 訊號的聯合效果與 literal 取捨 | 要資料與校準；固定訓練預算可能不足；失敗不代表 context 本身無效 | 選作可審計探針，僅13 features、四個預定 masks；不取代全域 parser |
| 持續保留原鍵、讓使用者手動決定 | 避免自動誤轉 | 沒有達成省第二次 Shift 的主要目標 | 作 literal fallback，不冒充自動成功 |

選擇學習探針是因為中文 LM 分數本來不與 literal 分數校準，且本題要同時測訊號互動與稀有誤傷。四個 masks 比較共享估計程序，沒有再搜尋其他模型種類。方法選擇不增加實驗 arms；本輪不宣稱已證明比固定規則好。

主要條件固定 EC-revise。E0=B、EL=B+L、EW=B+W、EC=B+L+W，各自同程序擬合，**係數可以不同**。固定係數的遮蔽 probe 另報，只測 EC 敏感度，不當主要效益比較。

## 2. 版本、檔案、介面與階段

產品基準固定 preview2 `e754019571573051d8c29e06526802c31d5172e2`。舊 `core/mixedparse`、`internal/mixedeval`、`cmd/0c-mixed-eval` 不改。重用 API 逐一列於 source map；執行 lock 固定來源 hashes，禁止用「舊 helper 同名」代替契約。

以下全部為**未來 Create**；目前只建立設計檔及範例驗證器，不建立下列 evaluator：

| 路徑 | 職責／公開介面 |
| --- | --- |
| `internal/shiftreturn/{types,validate,assign}.go` | schema 對應 values；`Validate(Fixture) error`、`Assign([]Fixture) Assignment` |
| `internal/shiftreturn/{candidate,features,render}.go` | `Generate(Query) (Result,error)`、`Features(Query,Hypothesis) Vector`、`Render(Query,Hypothesis) Display`；不接 Session |
| `internal/shiftreturn/{state,baseline}.go` | `Step(State,Event,Policy) (State,Observation,error)`；真實 M/R/C/A Session 與 application ledger；不試打 hypothetical 候選 |
| `internal/shiftreturn/{metrics,report,lock}.go` | `Summarize([]Row,Assignment) Report`、`VerifyLock(path) error` |
| `internal/shiftreturn/*_test.go` | 下列各階段對應 assertions |
| `cmd/0c-shift-eval/main.go` | `validate/assign/verify/enumerate/replay/summarize` 子命令 |
| `internal/shiftreturn/fit.py` | `fit/select`，只讀已凍結候選／feature observations |
| `internal/shiftreturn/reduce.py` | 獨立Python report reducer，只讀凍結rows／assignment；不重新選模或重播 |
| `testdata/shift-return-v2/{fixtures.jsonl,assignment.json,sources.json,run-lock.json,fit-lock.json}` | 凍結未見資料、分配、來源與來源程式／工具 hashes；不存日常輸入 |

所有 Go 檔使用 `//go:build shiftreturneval`，不改 production package。Query=`{raw,eligible,context_units,custom_tokens}`，Hypothesis／Vector／Observation／Model／Row／Report 依 schema；State 還包含 `fragment_id,raw,app,prelude,eligible,closed,selection,freeze,observations`，片段內 raw offset 從0起。app 是已提交字串，prelude 是當時已有的 composition，不包含研究片段。字串比較以 Unicode codepoints，raw offsets 以 ASCII bytes。hash採UTF8、SHA256小寫hex。

CLI全接受 `--fixtures --assignment --lock --out`，資料目錄另用 `--data`（預設 `assets/data`）；`--out` 必須不存在，不覆寫。`assign` 只讀 fixtures；`validate` 不評分；`verify` 跑 tiny／parity 不讀正式品質；`enumerate` 產候選召回；`replay` 加 `--model` 和 `--flow fixed|revise --split train|dev|test`；`summarize` 加 `--rows`。錯誤 exit2，資料資格不足 exit3，品質未過 exit4，通過 exit0；不以成功寫檔等於品質通過。unsupported／耗盡寫出明確狀態後 exit3，保留先前證據。

commands的完整呼叫前綴為 `go run -tags=shiftreturneval ./cmd/0c-shift-eval <command>`；具體資料路徑固定 `--fixtures testdata/shift-return-v2/fixtures.jsonl --assignment testdata/shift-return-v2/assignment.json --lock testdata/shift-return-v2/run-lock.json --data assets/data`，各phase的out另用尚不存在的 `/workspace/scratch/shift-return-v2/<phase>`。assign忽略尚不存在的assignment／lock，只產`assignment.json`；validate可先不驗source-lock但必驗schema／labels／assignment。verify及後續必驗lock，source／data或版本不符即exit2，不修改lock。fit呼叫 `python3 internal/shiftreturn/fit.py fit --observations <enumerate-out>/train.jsonl --assignment testdata/shift-return-v2/assignment.json --lock testdata/shift-return-v2/fit-lock.json --out <new-train-out>`，select改子命令為select並傳 `--dev <dev-replays-dir> --models <train-out>`，不接test。verify-numerical只接一個例子檔。

Generate結果為`{candidates,uncapped_identities,calls,overflow,status}`；Vector是13個finite float64；Display為`{text,segments}`。Step每次由同一State+Event+固定Policy得相同Observation；不得讀全fixture／gold。serialized shapes依schema；內部State不可序列化成額外未定義的模型輸入。out固定含`report.json`（Report）、`rows.jsonl`（Row）、`events.jsonl.gz`（Snapshot）、`candidates.jsonl.gz`（Candidate）；enumerate另產`train.jsonl/dev.jsonl/test.jsonl` observations，各列是candidate、fixture_id、component_id、event_index、raw prefix、gold hypothesis id集合，fit只可讀train檔。canonical JSON皆UTF8、keys字典序、無空白、禁止NaN/Inf；hash不含尾端LF，JSONL每列加LF。

| 階段 | 必須留下的可建置產物與命令 | 完成條件 |
| --- | --- | --- |
| P0 交換契約 | types/validator/assign/CLI骨架；`go test -tags=shiftreturneval ./internal/shiftreturn -run 'TestSchema|TestAssignment'` | malformed events/spans/labels、重複id、跨family split皆拒收；固定分配可重現 |
| P1 純探針 | candidate/features/render/state；`... -run 'TestSpell|TestCandidate|TestFeature|TestRender|TestFlow'` | 本文worked transitions；B不讀未來；字典prefix-only保留；tie／overflow／封存投影可重現 |
| P2 真實對照 | baseline與ledger；`... -run 'TestBaseline|TestPrivacy|TestLedger|TestFinalize'` | M/R/C/A observer on/off snapshot/store parity；空白／Toggle／unhandled／prelude正確；private零實驗呼叫 |
| P3 數值與報表 | fit/metrics/locks；`python3 internal/shiftreturn/fit.py verify-numerical design/shift-return-v2/numerical.json`；`... -run 'TestWeights|TestMetrics|TestBootstrap|TestLimits'` | 手算及獨立reducer一致、檔案hash與工具版本齊；toy不構成品質結果 |
| P4 凍結及實驗 | 凍結資料／來源程式→`enumerate`→fit/dev→凍結數值→8次test replay→獨立summarize | 依G0–G5；不能跳過前述工具驗證 |

每階段再跑 `go test -tags=shiftreturneval ./internal/shiftreturn ./cmd/0c-shift-eval` 與相同路徑 `go vet`。完成P3時跑正常 `go test ./...`／`go vet ./...`、`python3 scripts/check_public_tree.py`。沒有 production 變更時不把原生安裝當工具測試。

## 3. 事件、模式及真實基準

正式品質 fixture固定：初始中文，prelude keys先按原版組字，接一個 `shift`、至少一個raw key、`enter`。主任務raw限定ASCII可見字元及空白；gold欲輸出內容必須先標記，不傳給策略。其他事件只在獨立 compatibility fixtures，不混入return百分比。

eligible只由中文狀態收到shift產生，且opt-in實驗開啟、standard、非toneless、非private/paused/blocked、無CapsLock。generic mode=English及啟動英文不產生eligible。hard gate先於任何generator／score；不藉修改Session英文旗標繞過。純Query的eligible代表研究片段，與真實Session英文模式分離。

| 事件 | 實驗State轉移（初次Shift前走原版） |
| --- | --- |
| shift，中文→英文 | 保留當時prelude可見字串與原鍵；研究raw清空；開eligible，開始虛擬未提交片段 |
| key | 加入raw；超過512時只保留原鍵／literal，標capacity_failure；否則重算候選及當前顯示 |
| 再次shift | 封存目前整段顯示到app，清研究狀態，回中文；後續正常Session。是人工介入，不算自動return成功 |
| enter／commit | app附加prelude+最後研究display，清片段；Enter不評分。未eligible時原版處理 |
| focus／cursor／paste／mode／caps | 封存最後顯示，關eligible；focus清片段及跨欄上下文；cursor僅允許移至封存字串末尾，其他位置原版接手；paste文字直接附加app，不參與推斷；mode/caps交原版，不能自動重新eligible |
| backspace／cancel | backspace只刪研究raw最後一byte後重算並撤銷位於刪除部分的freeze；cancel丟研究raw並恢復prelude，不重學習；本輪不研究任意游標內編輯 |
| select_literal | 封存已指定研究raw原樣，後續研究key走literal直到下一次明確模式切換；不允許模型越鎖 |
| select_candidate | 鎖定指定candidate已顯示span；後續僅相容延伸；候選不在列表則拒收。主品質不含此事件 |
| protect=true／pause=true | 封存最後顯示、清eligible／暫存特徵；其後實驗generate/score=0；交原版保護狀態。解除不自動恢復eligible |

每次保存完整view=`app+prelude+display`。已送app的字不可改。虛擬研究片段由evaluator持有，尚不是實際Windows英文直通的解法。

M收到原始 `proto.Toggle`、key、Enter；R同M，但在gold首中文body key前額外Toggle（只有R可見gold）；C/A只省略該任務的第一個Shift，強制其餘初始中文、MixedInput=true，各用保守／積極，其餘raw／prelude／Enter不改。C/A是行為參考，不是自動Shift回切方案。設定其他欄位依source map。

真實對照每次呼叫 `Session.Handle` 後：先追加 `Snapshot.Commit` 到app；若key且 `Handled=false`，再追加該字元（非可見特殊鍵不加）；view=`app+Snapshot.Text()`。Enter未處理時只記提交動作，不把字串"Enter"輸出。完整prelude始終在此ledger內。C/A表只報可觀察完整surface正確率、手動成本及Snapshots，不從字面猜raw ownership；它們不是G3的配對boundary基準，因此不要求舊observer的unsupported路徑變成假mapping。G3只用具明確raw ownership的E0/EL/EW/EC。

主品質baselines只接受key/shift/enter/commit及prelude keys。原版映射分別是proto.Key、Toggle、KeyEnter、Commit，完全按實際順序；R額外Toggle只在gold首Chinese span的begin前一次。lifecycle compatibility只測虛擬State的closed／handoff／zero-calls，不假裝原版也有相同新行為；原版Mode/Select等parity以直接proto.Request fixture另測wrapper on/off。closed後State輸出handoff事件且不再評分，下一Session由原版接管，不在同一虛擬episode模擬新的中文字；因而不會拿手寫normal-mode模擬器取代原版。

## 4. 候選、待定顯示與決策順序

每個raw prefix枚舉cut=0..len(raw)-1。raw[:cut]是literal；從cut起，遇到第一個tone key（空白或3467）便以 `mixedparse.SpellExact(body,tone)` 解析1–3個body keys；失敗立即停止，此後raw是pending，不跳過內部錯字。開始的第一個音節須有 `Lex.Exact` 或 `Lex.HasPrefix`；首段誤按透過較後cut保留在literal prefix。後續逐音節同樣exact拼法，不做中性音替代、neighbor或fuzzy。

每個cut至多一條ordered reading；完全沒有完成首音節者不建Chinese hypothesis，literal仍存在。最長可由exact詞連續覆蓋的leading音節數為k（0..n），n是已拼完的全部音節數；用DP從0按1..8音節查 `Lex.Exact`，到不了的位置不視為覆蓋。`Lex.HasPrefix`只支持待續，不增加k。剩餘未拼完／非法尾鍵不丟失。

render=`literal prefix + Decoder.Best(readings[:k],prefix=false)的完整詞面 + raw從第k音節結尾至當前末尾`。k=0時全段顯示raw。ownership是literal prefix、k個Chinese reading spans、其後pending raw；pending不是完成中文字，也不是顯示已改成literal的判決。literal hypothesis則全部literal。W與Render共用同一k／完整解碼快取。所有scores NaN／+Inf為工具錯誤，-Inf改missing。

identity為無空白UTF8 JSON `[[kind,begin,end,reading],...]`，kind只允許literal/chinese/pending；literal/pending的reading=""；hypothesis identity使用全部n個音節的預期reading加tail，display identity使用實際k個Chinese+pending，兩者欄位分開。相同cut/first_reading的B快照共用，tail改變不重建B。ID=`"h:"+SHA256(identity)`。排序先cut升序、再identity UTF8升序；literal另保留。枚舉最多512個cut，排序後最多保留256個Chinese hypotheses；uncapped只作recall診斷，不參與選擇。沒有gold剪枝。

O-revise每個key重比全部未手選鎖定候選。O-fixed在第一次存在首音節exact spelling且有詞庫exact/prefix支持的事件封存一次當時display的ownership與reading；其中pending強制定義為literal，模擬提早定案，而非已完成中文字。其後只允許display projection在封存區與其相同的候選，homophone詞面可變、reading不可變。無相容延伸時凍結舊display並原樣附加新raw，記fallback，不計召回成功。共同generator先產生所有候選，再作封存篩選。

| raw／tiny詞庫 | k／顯示／ownership | 必須驗證 |
| --- | --- | --- |
| `x5`，詞庫有窄 | 無完整音節；`x5`全literal | 沒有憑未來593猜窄 |
| `x593`，只有「窄小」無單字窄 | n=1,k=0；顯示`x593`，hyp有ㄓㄞˇ，display suffix pending | O-fixed將此pending封為literal；revise可待續 |
| `x593vul3`，同上 | n=2,k=2；顯示x窄小，兩Chinese spans | revise能回修，fixed排除此projection |
| `x593v`，另有單字窄 | n=1,k=1；顯示x窄v，尾v為pending | W只評已完成窄；不能偷用ul3 |
| `x593v3`，v3解析不合法 | n=1,k=1；顯示x窄v3，tail pending | G2不當成完整兩字；原鍵仍齊 |

以上是候選render例，不假設該候選一定勝出。最終G2要求expected literal spans、Chinese spans與readings完全符合、預定完成部分無pending。字面同音字另計，不把render能出字等於指定字面正確。

## 5. 特徵與來源

**刪除v1未定義的「原保守／積極建議等價flag」**，不把13個guards當完整原作者決策。C/A仍保留真實行為對照及作者保留回歸。這是看新資料前的feature版本變更，不重用v1模型。13維固定順序：

| index | 值 | 更新／literal branch |
| --- | --- | --- |
| B0 | literal flag | literal=1，其餘0 |
| B1 | English.Has(raw[:cut])，長度1只接受a/i | snapshot；literal=0 |
| B2/B3 | prefix含ASCII數字／含非ASCII字母數字的可見字元（空白亦算） | snapshot；literal=0 |
| B4 | mixedparse.NumberLike(prefix) | snapshot；literal=0 |
| B5/B6 | 第一完整reading的prefix-supported log score／missing flag | snapshot；literal=0/1；-Inf=0/1 |
| L7/L8 | n-1／未解tail byte數 | current；literal=0/0 |
| W9 | 起於第一reading的最長exact word長度（≤8，無則0） | current；literal=0 |
| W10 | 在n個readings上不重疊≥2音節exact words的最大覆蓋數/n | DP可skip音節，n>0；literal=0 |
| W11/W12 | 對leading k個完整readings的Best log score/k／missing | k=0或-Inf→0/1；literal=0/1 |

B snapshot時間是cut後第一完整音節tone key事件，早於後續字；即使只有prefix支持也先記值。B5用prefix=true，W11用prefix=false，兩者語意不同，不直接相減當概率。literal features是固定向量 `[1,0,0,0,0,0,1,0,0,0,0,0,1]`。

詞庫／scorer固定為空store `personal.NewModel(loaded.Model.Lex,loaded.Model.LM)`，Now=nil、無custom/personal learning；取其Source()及同一model作Score。Decoder除Lex/Score外所有欄位置Go零值，Left=""即BOS；英文prefix非空時不跨literal延續中文LM。此輪不使用已確認左句語意作boundary特徵，研究的是片段內後文及原統計詞／bigram；prelude只作完整輸出與正常baseline上下文。這個限制不等於一般0c不使用長句。

四個models用同一train標準化統計與程序、不同mask：E0=0..6，EL=0..8，EW=0..6,9..12，EC=0..12。遮蔽probe使用同一EC係數，只將W替換為首音節時的W快照，L維持當前值；單列敏感度，不混成真正訓練對照。

## 6. Gold投影、數值與選模

主要品質episode最後raw固定完整，不用故意未完成尾音湊成功；incomplete tail僅compatibility例。每個完整音節事件及最後key為fit checkpoints，依key offset去重。gold hypotheses先投影已出現raw；若尚未完成首Chinese音節，gold={literal}；一旦完成，保留所有可接受完整gold readings的當前前綴；預定literal之後才打中文的prefix仍為literal。多個可接受gold去重，各自等權，不把另一個gold當negative。gold absent記miss，不製造候選。不得把未來字面傳給features。

數值定義：對train全部primary observations（包括無pair者）計continuous indices={5,7,8,9,10,11}的population mean/std，每個checkpoint的每個candidate各一次，重複candidate在不同checkpoint仍各一次；missing indices5/11排除該欄mean/std，轉換後缺值固定0；std=0或無有效值→全0。flags不標準化。四模型共用統計。population variance=`sum((x-mean)^2)/count`，不用n-1。

有pair的intent集合I等權；每intent的component等權；每component／intent的一個primary row；有效checkpoint等權；checkpoint內各gold等權；各negative等權。無pair checkpoint不分配權重，仍入coverage及評測；無有效pair row/component不參與fit權重並逐一報數，不改quality分母。collision相反意圖保留，但同split。

每pair差向量d=z(gold)-z(negative)，權重q按上述階層乘積，sum(q)=1。`loss=Σ q log(1+exp(-w·d)) + 0.001/2 Σ_active w_j²`；gradient=`-Σ q*d/(1+exp(w·d))+0.001*w`。無intercept，literal flag可學；inactive coefficients永為0。使用穩定logaddexp／sigmoid，float64、zero-init、1000次full-batch、lr0.05、無early stopping／clipping／shuffle。每次固定canonical pair順序累加，不能讓BLAS多執行緒改reduction。Python/NumPy確切版本及單執行緒設定入fit-lock，兩次tiny verification結果應在absolute1e-12內；實際模型reproduction比較bytes。

score按index0..12累加。最高Chinese score−literal score嚴格大於margin才選Chinese；其他literal。Chinese ties按cut、identity；沒有hysteresis。即使pending-only hypothesis獲勝也依第4節render，不宣稱完成中文。masked score缺值依上述規則，不允許NaN。

每模型dev在O-revise選margin `[0,.1,.25,.5,1,2,4,8]`：先篩return success≥90%、literal-only及return-English兩個final錯誤率各≤1%；候選按return error、literal-only error、return-English error、margin的lex順序升序。無可行時，按max(兩literal率)、兩literal率之和、return error、margin升序選一個負結果模型，記dev未過。率分母0時資料資格失敗，不自動填0。O-fixed沿用相同margin，不再選模。

參見 [numerical.json](shift-return-v2/numerical.json)：兩個observations x={0,2}→mean1/std1，差向量d=2，w0=0，q=1，初loss=log2、gradient=-1，第一步w=.05。給定w=.1時loss=`log(1+exp(-.2))+.0005*.01`，gradient=`-2/(1+exp(.2))+.0001`。例中是單一有效continuous維的算術，不是訓練實驗結果。

## 7. 資料取得、分組與獨立單位

資料取得最多3000 return任務原型、3000 literal原型、500compat原型；原型內最多5個後文variants，不額外充獨立樣本。來源程序固定為人工任務規格：作者先寫欲輸出的英文token＋中文片語／句子或完整literal，再按standard layout轉鍵；以公開詞庫驗讀音但不按策略答案標意圖。第二次獨立標註pass只看task／raw／gold、不看任何模型輸出；兩pass不一致者修正或拒收並記原因，不能執行後挑除。人工任務來源標為synthetic-intent，不冒稱真實使用頻率；真實密碼禁用。可引用公開原句時sources需URL、版本／hash、授權與所改spans；沒有此記錄就不得收為public。

只允許取得phase建立來源bank；bank凍結後依`SHA256("acquire-v2:"+prototype_id)`排序取到各類budget，策略結果不可見。原型ID=`SHA256(canonical JSON{source_ref,intent,prompt,raw,gold})`前24hex；candidate id不得來自此gold。已在v1／對話／測試出現的英文prefix、目標content word/reading原型與完整raw建立seen list，只作dev/mandatory，不計新test；bank及seen來源hash入lock。若budget內不足資格則INCONCLUSIVE，不看品質後換seed／補到過關。

return主類按以下優先順序唯一指定：typo（首中文前有指定誤按span）、identifier（欲保留prefix含數字或標點）、digit_initial（首reading聲母鍵是數字）、uppercase（prefix含大寫）、unknown（English.Has(prefix)=false）、known。每類取得budget500，總3000。literal主類固定各375：ordinary_english、account、phonetic_account、model_version、numeric_date_time、url_path、identifier、random_ascii。主類由任務prompt規則決定；不能按classifier輸出分類。

return每主類的length配額固定為音節數1/2–4/5–16/17–64各125；length不要求每個dependency component都有四層。literal沒有Chinese syllable strata，按raw長1–8/9–32/33–128/129–512分層，375依94/94/94/93分配。raw>512及>64音節另作compat，不進此取得budget，capacity行為仍必測。return空白一聲至少20% primary；每主類test至少20個independent components。這些是資格不是品質，不能用容易子類稀釋稀有類。

名詞固定：task=row；prototype=任務原型；family=相關性標籤；component=family關聯傳遞閉包，**本版本group即component，不存在component下再算獨立groups**。family key包含確切英文prefix（ASCII lowercase）、目標content word/readings原型、完整literal token、同任務／固定模板實例的variants、來源同段落／同vendor產品族，以及相同可觀測輸入collision。共用整份字典檔或廣泛coverage類別不是family；不得拆確實同一固定模板。semantic近重複由annotation pass標alias，凍結後不改。

component_id=`SHA256(JSON(sorted row IDs))`；JSON無空白、UTF8、ensure_ascii=false。split=`big-endian SHA256("shift-return-v2:"+component_id) mod10`，0..5train、6..7dev、8..9test。每component每intent預定一個primary：verified-public先、verified-synthetic次，再id字典序；完全同輸入相反意圖都留、同split，每intent只計一次。additional variants報worst-case，不充n。

若component含任一seen原型，整component標為regression，不進formal train/dev/test。其餘按上述hash分配。seen list列出來源檔、版本和匹配依據；不得只刪難列而讓同component剩下列留test。每個synthetic/public任務都需要兩個獨立標註者ID；第二人只見prompt及raw/gold、不見第一人的理由或模型輸出；不一致記adjudication後修正或拒收，來源bank和adjudication在quality前凍結。

context-contrast的預定predicate：在已完成首音節的raw prefix存在至少兩個不同gold boundary／literal intent的任務（先按source gold建pair），後續raw不同；return continuation至少再完成一個音節並含一個跨首音節的≥2音節exact content word；相反literal continuation由prompt明定為literal。此規則只用gold與固定字典，不按模型能否修好選列。它估計「指定成詞延續」的效果，不涵蓋所有句子context。相同完整raw／setting而相反意圖是identical_collision，不算有區別隊列，但留在各intent主表／challenge表，報可達上限。

正式test至少400個return components、400 literal-only components、400含return-English primary的components；context-contrast至少100個return components；train/dev各至少100return及100literal components。return primary的英文prefix全部作一次英文字串保留檢查，不能一字串拆多個獨立spans。每intent n=component數，兩intent可共用component，但不得相加假稱獨立樣本。

例如10個帳號variants共享同prefix→一component；只選一literal primary給Wilson n=1，其他9列另報variants worst case。若另一intent在同component，同intent n不增加，bootstrap一起抽。400個raw字串但只有150components就是n=150，不能通過400資格。

## 8. 報表、公式與門檻

三項成功必須分名：`boundary_reading_success`要求render完整span／reading對；`surface_success`另要求可接受文字完全相符；`materialization_success`要求app+composition/commit無丟字且首字實際輸出。90%門檻僅第一項，不能稱90%正確中文字面或真實使用體驗。完整字面gold全部primary必備；任何缺失皆G0不過，不填0。

每個intent／component primary任一必要span錯即該項錯。literal誤傷包含字母大小寫、標點、數字被改寫及尾碼被借入Chinese；比較render投影與expected prefix，不只搜尋漢字。最終主checkpoint是最後key，Enter只驗提交一致。

E0/EL/EW/EC皆在revise下報組合增益，fixed只作同模型流程比較。修好=`base_error=1,new_error=0`，改壞反之，net=修好−改壞。相對下降=(errBase−errNew)/errBase；base=0時要求new=0並記improvement=null。交互作用I=errorEC−errorEL−errorEW+errorE0。流程增益=errorFixed−errorRevise，context對回修的額外增益=EC流程增益−E0流程增益。不用EC-revise對E0-fixed歸因於context。

首次正確=第一個gold Chinese音節完成後，當時可見gold投影與render一致的最早key index；若從未正確=null。最後穩定=最早checkpoint，使其至final每個後續key投影均正確；final錯則null。暫時literal誤轉的每段長度為連續錯誤key事件數，結束時再轉對；shift/Enter不算key。反覆數為相鄰key共同已處理bytes的display ownership／reading投影改變次數；普通新增音節不算。字面同音變動另計。pending→Chinese是resolution，單獨計，不冒稱先前literal錯判。

Wilson使用各intent中獨立component的binary primary outcome，z=1.959963984540054：p=e/n，center=(p+z²/2n)/(1+z²/n)，half=z*sqrt(p(1−p)/n+z²/(4n²))/(1+z²/n)，上下截[0,1]。同component的其他variants不進n，另報whole-component any-variant風險。這是指定synthetic/public任務分布下的benchmark區間，不宣稱真實使用者誤傷頻率已驗證。

配對bootstrap在固定test component union上等概率有放回抽同樣數目，10000 replicates，Python `random.Random(271828)`；抽樣用 `randrange(N)`重複N次，不用NumPy RNG。每次對所有模型／流程同步抽。子群分母為0的replicate記invalid，不填0；有效replicates<9500該區間unavailable、G0不足。區間為排序有效樣本的2.5／97.5百分位，使用linear quantile index=(m−1)*p內插；資料／工具lock記Python版本。Wilson gate不以bootstrap代替。

bootstrap母體component IDs先字典序排序；同component兩intent的primary與所有conditions一起重抽。report.conditions固定8個`{E0,EL,EW,EC}-{fixed,revise}`，每條件metrics固定`return_boundary,return_surface,literal_only,return_english`；metric.errors/n是相應component的binary primary計數，value是errors/n，不用success命名誤導。逐類及variants診斷另存diagnostics_path，格式同metric並以category分層。safety固定彙總key_failures、commit_failures、app_rewrites、lock_violations、hypothetical_learning_updates、persistent_writes、protected_calls；unknown不可填0，任一必要row unsupported使G0 unavailable。derived表固定interaction_I、各model的flow_gain、context_extra_revision_gain及其component bootstrap區間。paired表包含四種feature對照（均revise）、四model的fixed→revise，以及主要EC相對E0在context-contrast的修好／改壞；沒有此隊列不能用all-return補。

| Gate | 固定要求 |
| --- | --- |
| G0 工具／資料 | P0–P3及來源／gold／split／樣本資格全過，必要paired metrics可用；不靠未來gold或反覆held-out調整 |
| G1 召回 | test primary完整gold在capped候選≥99%；mandatory全存活；overflow／unsupported仍算miss；未過停止fit |
| G2 恢復 | EC-revise final boundary_reading_success≥90%；surface另完整報告，不含糊稱選字全對 |
| G3 context | 預定contrast EC錯誤相對E0下降≥30%，EC對EL的net修好>0；EL長度效果及EW獨立報；base零時無可量測改善，不報通過30% |
| G4 literal | EC兩個風險分母各點估及Wilson95上限≤1%；相對E0及EL最終錯誤不增加；所有指定作者保留例全過 |
| G5 不破壞 | 假設候選評估期間learning updates／persistent writes=0；protected generate/score=0；zero掉鍵／重複／改寫app／越明確lock；Enter=最後完整view；正常中文regression零新增失敗 |

G3 base零且EC零只能寫「無可量測改善」，不是H1得到支持；整輪結論為context效果INCONCLUSIVE，不能拿G2成功率代替G3。所有對照報相同門檻表，但不要求E0/fixed達EC成效門檻。

dev模型／margin固定後八個test各只跑一次；主要方案不因test挑最高分換名。EC全過→GO_OUTPUT_EXPERIMENT，允許再研究輸出持有與UX，非產品GO。有效主品質失敗→NO_GO_METHOD；必要工具／資料缺失或無法量改善→INCONCLUSIVE，已有負證據仍保留。不得用這一個探針否定所有context方法。

## 9. 資源與不得跳過的驗證

每episode最多512研究bytes；每checkpoint枚舉cut≤512、保留Chinese≤256+literal。每hyp最多256個完整音節。單episode計數預算：250000次exact/prefix詞庫query、100000次Decoder.Best；超限保留原鍵，記resource_exhausted、停止該episode評分並使必要品質unavailable，不縮小困難案例分母。每model每step最多2million pairs；超過則整次fit exit3，不偷偷抽negative。全run raw evidence gzip最多8GiB，preflight free≥10GiB，超限整run安全停止／INCONCLUSIVE，保留已寫證據，不刪舊實驗。

估計6000原型、每列平均8音節、平均20候選時約1million完整解碼與數GB未壓縮artifacts；這只是容量估算，先用固定20個tiny/compat任務做時間／大小校準，不看正式品質、不改門檻。每階段鎖定實際預算和計數。uncapped diagnostics只保存identity／gold survival計數，不另解碼被剪候選。

必測case及assertion：standard所有1–3body×五tone的SpellExact roundtrip；tiny窄/窄小的prefix轉完整；第一聲空白不被丟；prelude+Shift+raw+Enter完整輸出；再Shift/焦點/貼上/mode/caps/private pause生命週期；literal/Chinese ties；無候選；63/64/65及511/512/513bytes；canonical cap與無gold依賴；future suffix／gold替換不改B；firstreading多gold均非negative；無pair階層權重重分配；數值worked gradient；component分組／seen／split可重現；Wilson n=400零錯上限約0.9512%；相依variants不加n；bootstrap空子群；全部八條件metrics用獨立Python reducer重算；正常Mode／manual literal／原鍵還原parity；old evaluator來源檔hash不變。

本文件不宣稱實作已存在。完成上述契約驗證才可凍結正式資料／工具；輸出持有、實際英文延遲、Windows TSF與安裝包仍另立原生階段，不在此偷偷實作。
