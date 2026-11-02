# Offline Shift-return v2 tools

All Go files here and in `cmd/0c-shift-eval` require `shiftreturneval`. Nothing is connected to the production Session. Baselines create fresh disposable models and observe the original Session; the scoring probe owns a different empty store.

The exchange contract remains `design/shift-return-protocol-v2.md`, its source map, and reviewed schema. The embedded schema is checked against the reviewed bytes. Formal commands reject demo records, mismatched locks, uncommitted implementation source, unsupported English fallback packs, and incomplete source/annotation/sample qualification. Output directories and files are create-only.

Verification commands, from the repository root after activating the pinned toolchain:

```sh
export GOCACHE=/workspace/.0c-tools/shift-return-go-cache
go test -count=1 -tags=shiftreturneval ./internal/shiftreturn ./cmd/0c-shift-eval
go vet -tags=shiftreturneval ./internal/shiftreturn ./cmd/0c-shift-eval
python3 internal/shiftreturn/fit.py verify-numerical design/shift-return-v2/numerical.json
python3 -m unittest discover -s internal/shiftreturn -p 'test_*.py'
```

`-count=1` matters after Python changes because Go's test cache does not track imported Python files. `SHIFTRETURN_DATA_INVENTORY=1` additionally enables the fixed public source inventory, baseline observer parity, and the prescribed 20-case operational calibration. These inspect public dictionaries and fixed compatibility inputs, not a new quality dataset.

`assign` and `validate` do not score quality. Missing bank qualification makes `validate` exit3, even if the exchange/assignment is structurally valid. `verify` runs the tagged and Python contracts, fixed public baseline parity, and 20-case calibration. Enumeration generates candidates/features and external labels, with zero linear score calls. Its exit0 only means its G1 phase gate passed; its report is not an overall quality GO. A replay containing one condition remains an incomplete eight-condition experiment and exits3; `support_complete` in `run.json` distinguishes that from unsupported/exhausted episodes. Final quality is decided only after the frozen complete evidence is summarized independently.

Training reads only primary train observations joined to the frozen assignment. It requires a frozen G1-passing enumeration report and completion metadata. Means, variances, gradients, and updates follow the specified order; Python3.12 compensated `sum` is deliberately avoided. Four models are fitted separately with the fixed masks. The fitter writes immutable copies for each of the eight prescribed dev margins; selection reads only matching dev-revise reports and checks their model hashes. No test option exists on selection.

Source/data locks stay immutable. The current implementation can use later execution/evidence locks that include the immutable initial `run-lock.json`, fit lock, selected models and evidence bytes. Models keep their initial source/fit hashes. A source commit may be an ancestor of a later metadata-only freeze commit; every frozen source byte and complete source-file coverage is still checked. The final lock must contain exactly one basename selected-model-manifest.json; only its four absolute model paths/hashes are read. This precise transport needs final parent review before any formal invocation; no initial lock is rewritten.

The internal Evidence bundle retains verified raw-file hashes for the source lock and models. Decoded JSON values cannot reconstruct original whitespace or final-LF file hashes. Source, bank, labels, selected-primary metadata, queries, capped/uncapped diagnostics, scores, strict-margin selection and row grades are checked again before reporting. The independent Python reducer reads the public binary dictionaries itself and independently rebuilds assignment, joins, grades, component metrics, bootstrap statistics and diagnostics.

Unavailable rows retain full component denominators. Their rates/intervals are null, paired/derived intervals are unavailable, and diagnostics state the conservative failure charge in the integer error count. Unrun safety counters are null, never fabricated zeros. Explicit preservation examples are required before G4 can pass. EC's fixed-coefficient first-W sensitivity is a diagnostic and never a replacement main arm.

The 10GiB free-space floor and aggregate 8GiB retained raw-evidence cap are unchanged. Old phase evidence is counted and never deleted. The fixed 64-syllable repeated-reading calibration currently exhausts the 250000 logical lex-query budget; this is retained operational evidence, not a quality score.

Parent adjudication fixes temporal indices to zero-based research-key ordinals and preserves locked prior display plus retained raw on failure. The protocol/schema have received explicit pre-quality errata; thresholds, masks and acquisition caps are unchanged. Formal data, source/numeric freezes, acquisition independence and quality remain separate gates. These tools do not authorize product integration or native installation.

The parent-approved2026-11-02 pre-quality erratum retains x593v3 as an API-semantic counterexample (n=2,k=1,L7=1,L8=0) and changes the invalid-body worked case to x593vv3 (duplicate class; n=1,k=1,L7=0,L8=3). No phonotactic guard or production change was introduced. The original failing P3 logs remain untouched. This is a contract revision before quality, not a claim that c512578 never changed.

The parent has accepted zero-based research-key ordinals, locked-display preservation on resource failure, and conservative unavailable error charges. Each metric now carries status, observed_errors and unavailable_components; errors is their sum. An unavailable charged count is not an observed error rate, and rates/intervals remain null. The exact selected-model-manifest transport is specified in protocol section10 and is awaiting final parent review before formal freeze. Formal quality remains prohibited.
