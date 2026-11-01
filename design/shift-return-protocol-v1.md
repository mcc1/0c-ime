# Shift 英文後自動回中文：實驗 protocol v1

狀態：事前實驗設計，尚無本實驗品質結果。依[目標文件](shift-english-auto-return-experiment.md)細化。這是離線局部切點研究，不是替換產品 parser 的規格；輸出持有／原生整合另有明確關卡。舊 mixed-eval-v1 只作開發參考，不更改舊資料或宣稱它已驗證本題。

開跑前補充：[組合與決策順序](shift-return-combination-order.md)新增EW（B+W），形成4模型×2封存流程；主要方案仍為EC-revise，其他門檻不改。下文的E0/EL/EC主要資訊比較皆指O-revise，O-fixed只作相同模型下的封存對照。

## 1. 要回答的問題與介入範圍

使用者 Shift 已標示英文起點；檢查英文結束的位置及中文讀音能否隨後文變準。允許在同一未提交組字段內重估前面的英文／中文 ownership，沒有第二聲調截止或一次改判上限。使用者不再按第二次 Shift，也不手動選字。

這輪只處理一次 Shift 英文片段後接一段中文，以及該中文起頭誤按後的恢復。更後面再次轉英文另切成新任務，不假裝已解決多次自動切換。實驗模組只選該片段的 literal／Chinese spans；同音字仍由原有 decoder 產生。一般中文、既有學習、UI、鄰鍵修正與自動進英文不變。

主要假說 H1：相對未使用後文重判的局部判斷，使用後文可將歧義隊列的最終切點／讀音錯誤降低至少30%，並同時達到整體返回成功率90%、最終 literal 誤轉 Wilson95 上限1%的要求。H0 是未達上述聯合條件，不是「後文永遠無用」。

## 2. 同一任務的對照

| ID | 行為 | 用途 |
| --- | --- | --- |
| M | 真實原版：Shift 進英文後不再切回 | 確認現有行為、省第二次 Shift 的成本；不當成偵測準確率基準 |
| R | 在 gold 終點人工回中文，其餘走原版 | 參考正常解碼上限；gold 僅此組可見，不參與選模 |
| C/A | preview2 的保守／積極混打，獨立 Session 重播相同 raw | 行為對照，揭露現有修正收益與帳號誤傷；不是實際 Shift 自動返回 |
| E0 | 同一局部選擇器，候選初次可辨讀時的資訊固定，不使用後文詞證據重評它 | 主要無後文對照 |
| EL | E0 資訊，加上當時後續音節／未解按鍵的數量，不加入後文內容分數 | 長度負對照 |
| EW | E0 資訊，加上後文字詞分數，不使用長度features | context與長度的組合對照 |
| EC | 同一局部選擇器，後文到達後更新詞與語言證據並重選 | 待驗證的 context 組 |

四個feature模型共用原鍵、候選生成、可見前文、詞庫、decoder、訓練資料、估計器、訓練預算及 margin 清單；feature mask 是資訊比較的唯一研究變因。各自因果重播，不讀 gold past state。模型因可用 features 不同分別按同一程序擬合，不能把係數差異當作某個 feature 的獨立因果效應。

另做不訓練、不選模的配對 probe：每個 raw prefix 的同一組候選及凍結 EC 係數，分別輸入正常 context features 與初次觀測 features。只解釋該 EC 對後文是否敏感，不把這種訓練分布外遮蔽當主要成效。若只 EL 改善、EC 沒有額外優勢，不能稱字詞內容已幫助區分。

版本：M/R/C/A 的來源固定 preview2 `e754019571573051d8c29e06526802c31d5172e2`。上游 `v0.2.7` 可另報歷史行為，不與主要 context 對照混算。詞庫及系統統計模型使用當前公開 `assets/data`，實際 manifests、英文詞庫和工具鏈 hashes 在執行前寫入 lock。禁用 neighbor/fuzzy/autocomplete/neural；既有 neural API 沒有跨 literal／讀音校準能力，本輪不假設它具備。

## 3. 實驗的觀測與候選契約

- 策略只收到 Shift 事件、已發生的 raw key prefix、已確認且允許使用的左文、layout、設定與明確鎖定。不收到 intended span、gold checkpoint、任務類別、paired ID 或未來按鍵。
- 起點是實際 Shift，不是 fixture 人工指定的正確中文起點。標準 layout，ASCII 實體字元，明確3/4/6/7與第一聲空白均包含；每列使用真正鍵值，不對錯字自動補鍵。
- 全 literal 解釋永遠保留。對每個可行 offset 枚舉 `literal prefix + exact Chinese suffix`；先前錯誤按鍵可包含在 literal prefix。中文以 standard exact spelling 解析，首音節須有完整讀音及詞庫exact或合法prefix支持；prefix-only可保留待續，不能當完成中文字。完整中文輸出須有exact word coverage，不能因某音節只能出現在詞中就提前丟棄。
- 中文候選保留完整有序讀音、每個 byte 的半開區間 ownership、完成／未完成狀態及原鍵。後續非完整音節先作可還原 pending tail；不能因下一字未完成就丟掉首字的另一解釋。中文之後再次含無法解釋的 raw tail 須明列，最終完整讀音計分不能忽略它。
- 同音字不擴成不同 boundary 候選；交給既有 decoder。在相同 boundary 下出現的字面變更與跨 boundary 變更分別記錄。literal branch 不使用中文字數或 LM 分數冒充可比較機率。
- 研究容量固定每片段512 raw bytes、最多256個 boundary hypotheses；不以音節次數關閉。先依 offset 升序、canonical reading UTF8 排序截取，literal 另保留；不按 gold／結果剪枝。若來源排列存在更多解析，排序規則同上，須記 uncapped/capped 差異。overflow 時記覆蓋失敗，保留全部原鍵作 literal fallback，不從分母移除。
- 首字 offset 的所有未鎖候選可在後續每一完成音節／聲調與每一新可見字元重估；未完成尾鍵可更新暫時顯示，但不使用未完成詞當完成詞。Enter 不作最後一次額外排名，只提交最後可見結果。手選／明確 literal lock 封存該區，不能跨越改判。
- 禁用日常 user store；每列 disposable store，假設路徑只有 immutable inputs／read-only scoring，不透過 Session 反覆試打。從 Shift 進英文的實驗 eligibility 與 generic English mode 分開，不能為了跑候選取消 private／paused／blocked 保護。

舊 generator 有64-byte／兩音節與不支援空白的限制，舊 Machine 有 early freeze，不能直接套用。可重用 SpellExact、read-only decoder 與 guard parity 工具，需新增獨立版本及測試。不能修改舊 lock 以冒充同一實驗。

## 4. 最小局部選擇器：只研究資訊，不重建全域模型

採同一個 conditional linear ranker 比較該片段的 boundary hypotheses。它不接管片段外的中文，不替換 normal decoder。這是本輪固定的探針；若失敗，只能否定這個探針在此資料／條件下的表現，不否定所有 context 方法。

固定 feature groups：

| 組 | features（每項獨立） | 定義 |
| --- | --- | --- |
| B，初次資訊 | literal flag、英文 prefix exact membership、prefix 含數字、prefix 含非字母數字、prefix numberLike、原保守建議等價 flag、原積極建議等價 flag、首完整音節 exact statistical log score、該分數缺值 flag | 在該 first-boundary／first-reading 初次完整出現的 raw prefix 上取得後固定；literal 首分數為0、缺值flag=1。prefix 比較保留大小寫原文，英文詞籍依原 helper 規則 |
| L，長度 | 後續完成音節數、未解 tail byte 數 | 每個 checkpoint 更新；literal 候選兩值皆0 |
| W，詞內容 | 起於第一中文音節的最長 exact word 音節數、全完成中文路徑中可由≥2音節 exact words覆蓋的最大音節比例、完整解碼 mean log score、完整解碼缺值flag | 最長詞含單字=1；沒有完成路徑則0。coverage 以不重疊動態規劃求最大值，單字不增加分子。score 用現有 decoder 的完整讀音 Best、prefix=false，除已完成音節數；不將 prefix-only score 混入 |

E0 用 B，EL 用 B+L，EW 用 B+W，EC 用 B+L+W；unused coefficients 強制0。W 只提供可學習的證據，不先設定「有詞一定中文」。原作者 heuristics 的逐項命中另作 audit 輸出，不事後新增特徵。hard modes／lock／privacy 不交給模型投票。

B的snapshot key固定為`(first Chinese byte offset, first exact reading)`，literal另用固定key。時間是首音節按鍵首次完整的prefix，即使當時只有詞庫prefix支持也先記missing／guard資訊；不能等後面成詞後才建立B，否則E0會偷看到後文。後續讀音延伸共用該B；四組共用每個checkpoint的label-free候選，不能只替EC保存正解。同一O-revise下不提前封存E0原鍵；O-fixed的額外封存依補充設計獨立比較。

Decoder 初始 left context 只用當時可確認的中文左文；literal prefix 重設中文延續到 BOS，suffix 內完整解碼使用原統計模型，故後續字詞可改變先前字面與整條 reading 的分數。系統詞庫、LM 與 decoder options 要寫入每個 run。詞組／統計 context 與句子語意理解不同；本輪不宣稱測到了神經語意模型的能力。

實作前須將 B 的保守／積極建議與初次 score adapter 對應至具體 helper、固定來源與輸入欄位，做 parity；若任一項無法可靠投影，停止而非換成近似值。此 source map、derived-feature worked vectors、可讀報表 schema 是 code freeze 的必要產物。

## 5. 擬合與 dev 選擇，結果前固定

四個模型都使用相同 train primary 的完整觀測 checkpoints：每個完成音節及最後一個 key，重複 checkpoint 去重。gold 只作訓練 label，先投影到當時 prefix，不要求未輸入音節出現；未輸入到 intended boundary 前的正解是literal pending。錯誤候選依canonical identity全列，不抽容易negatives。

目標為 pairwise logistic loss：gold與每個negative的feature差。各 intent、group、row、checkpoint、negative 逐級等權；同一equivalence group的互斥意圖都保留，不當成資料錯誤刪除。gold 不在候選者不能製造 gold candidate：訓練該 checkpoint 不產生pair並記錄，評測仍算失敗。所有無競爭候選列仍計評測分母。

訓練設定：float64、zero init、1000 full-batch gradient steps、learning rate0.05、L2=0.001、無early stopping、無額外模型搜尋。各連續feature以train primary candidate observations的mean/std標準化；std=0時輸出0，missing score先置0並另旗標。標準化統計只存train，四模型共用對應欄位的統計。固定canonical排序，單執行緒數值庫與實際版本入lock；不使用test調整預算。報loss與gradient，沒有收斂不能泛稱線性模型失敗。

literal score作比較基準；對最高分Chinese candidate，只有 `scoreChinese-scoreLiteral > margin` 才顯示中文。其餘literal。同分先literal，再first offset升序、canonical identity升序；沒有額外黏住前次答案的margin。這允許context到達後重新選擇，不加第二個穩定門檻。

四模型各自只從固定margin清單 `[0,0.1,0.25,0.5,1,2,4,8]` 選擇。dev先取literal-final點估≤1%且return成功≥90%的候選，再依return成功率高、literal錯誤低、margin小排序；若無可行候選，按literal錯誤低、return成功率高、margin小選一個作負結果測試，清楚標記dev未達標。不得因test選另一個。正式literal區間門檻仍以test判，不以dev點估代替。

code／資料／features凍結後才fit；數值模型、標準化與dev選擇紀錄提交後，四模型×兩流程的final replay各跑一次。O-fixed沿用O-revise選定的相同margin，不另選模。重算既有raw metrics不是新選模；修bug使受影響結果失效，見過的test不再作新方法未見資料。

## 6. 資料取得與資格

固定取得預算為最多3000個 return 原型與3000個 literal 原型，另最多500個正常中文／模式／提交回歸原型。取得階段只可做結構／詞籍／數量驗證，不讀策略輸出。來源限公開且可引用內容或人工指定任務；每個中文字面與literal意圖先有來源／標註者記錄，不以decoder答案當gold。禁止實際密碼與私人trace。

Return涵蓋已知／未知／大寫英文、英數識別字後中文、數字聲母、誤按後恢復。literal涵蓋一般英文、帳號、刻意注音式帳號、型號版本、日期時間小數、URL路徑、識別字與隨機英數。每組含單字、2–4音節短詞、5–16音節段落、17–64音節長段的預先分層任務；字串超過研究容量仍保留。第一聲空白至少占return primary的20%。

對同一歧義首段建立後文配對：成詞中文、較長中文、正常literal、注音式literal、立即結束。配對是同一group，不能充獨立樣本。gold「後文有區別」須由欲輸出spans與前綴碰撞結構在看結果前標記，不按EC是否修好選列；詞籍驗證只記metadata，不決定意圖。

Families至少包含原英文prefix、中文詞／句原型、literal原型、模板實例／來源群、已知別名與相同可觀測條件的collision。共享任一family transitive union。廣泛coverage類別不是template family；同一真實固定模板不得拆。固定hash split：`SHA256("shift-return-v1:" + JSON(sorted_component_ids))`視為大端整數，JSON使用UTF8、無空白、ensure_ascii=false；mod10為0–5 train、6–7 dev、8–9 test。各group每種意圖各選一個primary，先按source priority（人工已驗證、公開已驗證、synthetic）、再id字典序固定；不因只選一列而丟掉相反意圖。同意圖的其他列作variants。

正式test至少400 return groups、400 literal-only groups、400 intended-English primary spans（可來自return）；每個主coverage類至少20 groups，context-contrast隊列至少100 groups。train/dev各至少100 return與100literal groups。數量不足不改seed／拆群；未讀品質前可按事前來源預算完成取得，但不在看結果後補到CI過關。最終資料、aliases、group、primary、split與標註exceptions全凍結。

v1及已在對話／既有測試出現的案例作development／mandatory，不算未見test。`microsoft593`及其相反意圖必須保留作碰撞示範。完全相同觀測而不同意圖的列都評分：先每意圖等權，再group平均；這種group的max可達分依標籤計算並明列，不能把兩答案皆算正確。

## 7. 計分與事前判定

所有primary品質先列各意圖逐group成績及paired counts，不報一個混合總分。對一個group的同意圖primary，完整span任何一處失敗即該項失敗；額外variants另報worst-case，不混入主分母。相反意圖的collision列各自進相應分母，保持同split且CI依component，不當兩個獨立group。

| Gate | 主指標與通過要求 |
| --- | --- |
| G0 工具／資料資格 | source parity、完整prelude至Enter、raw ownership、gold validity、split、資料數量全部通過；gold／ownership缺失不能填零 |
| G1 搜尋可行性 | test gold完整partition/readings在capped set ≥99%；mandatory正解全部存活；uncapped、單字／長段與overflow另報。未通過先停止fit |
| G2 返回 | EC final key後、Enter前完整首boundary及完成讀音成功≥90%；沒有手動選字。錯同音字另報，不掩蓋完整字面任務成功率 |
| G3 後文內容有效 | 預定context-contrast隊列EC錯誤相對E0下降≥30%，並且EC較EL的淨修好group數>0；三者同literal風險限制下比較。E0零錯時要求EC零錯，標無可量測改善。報paired修好／改壞與group bootstrap95區間，不把點估改善當顯著性證明 |
| G4 保留literal | EC literal-only與return英文span的最終誤轉，各自點估與two-sided Wilson95上限≤1%；EC比E0、EL不增加最終literal錯誤；必要作者保留案例全過 |
| G5 不破壞 | zero丟鍵／重複／跨手選鎖定／改寫已提交文字；Enter逐字等於最後顯示；假設學習／持久化0；private／paused／blocked實驗呼叫0；正常中文回歸零新增失敗 |

Literal錯誤包括字母大小寫／標點改寫及英文尾碼被借入中文；不能只看是否出現漢字。return成功要求全部應完成讀音，不只首offset；pending tail未完成與本來任務要完成但未完成分開。字面gold在所有正式primary必備，報同音字錯誤與整段字面成功率；現階段G2主門檻只評意圖／讀音，字面並非可以省略的證據。

Bootstrap使用component作單位、10000次、固定seed `271828`，paired重抽保持四模型及兩流程同group；不得拆collision。在純獨立primary incidence分母使用Wilson（z=1.959963984540054）；若同group有多個英文primary span則先合成任一錯誤，不當獨立span做Wilson。零分母是null。

過程診斷：第一完成音節、第二音節、每個後續完成音節、gold詞末、final key各記cut/readings/surface；詞末只供評分。報首次正確、最後穩定、context修好／改壞、暫時literal誤轉的持續鍵數、boundary反覆數。**中途錯但自動修好不算final失敗；中途對最後改壞算失敗。** 一旦錯誤被送入應用程式則屬G5，不是可逆暫時顯示。

## 8. 能下什麼結論

- **GO_OUTPUT_EXPERIMENT：** G0–G5全過，支持局部context方法，才做原鍵持有／完整composition的輸出驗證；不是產品或安裝包GO。
- **NO_GO_METHOD：** 有效比較未達G1–G5任一必要門檻。依搜尋缺失、評分選錯、context只改同音字、長度效應或不可辨識取捨分析；不由單一估計器宣稱context無用。
- **INCONCLUSIVE：** 必要工具、資料、paired metric／gold不可用或洩漏，停止品質宣稱。已可用負證據照報，不改稱通過。

若EC與E0相同，先檢查feature actually consumed及候選存活；若有字面改善但切點不改善，結論是context幫到選字但尚未解決中英意圖。若成詞中文改善而注音式帳號誤傷，結論是先驗取捨未解，不是成功辨認使用者意圖。

輸出階段再量英文即時顯示／送出延遲、整段反覆與機器延遲，於該階段開跑前另固定UX門檻。辨識期採虛擬未提交composition，不聲稱實際英文直通已可回收；512-byte容量也不是推薦產品等待長度。原生Windows TSF、跨欄位、手動反悔與效能未驗證之前不出新安裝包。

## 9. 預定產物與開始順序

新增獨立 `shiftreturneval` 實驗版本，不能原地修改舊mixed-eval-v1；具體package／CLI需在code freeze manifest列齊。產物為schema與範例、source map、候選／worked feature vectors、資料取得manifest／family assignment、run lock、train設定、四個模型及dev選擇、八個conditions的逐鍵rows／candidates／events、gate report與錯誤分類。

先實作並驗證工具→凍結資料與code→做候選召回→按本規格fit/dev→凍結數值模型→一次final test→主agent獨立重算。任何新模型、feature、窗口或門檻變更先版本化；已看過test轉為回歸，不再當未見資料。

本次僅完成protocol與[手算計分例](shift-return-worked-scoring.md)，沒有收新測試資料、實作選擇器、訓練或評測。實作前的source-map parity與schema是必做產物，不可跳過後宣稱本文件等於已跑實驗。
