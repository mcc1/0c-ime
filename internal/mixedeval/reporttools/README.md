# Frozen mixed-input report reconstruction

These tools only aggregate existing raw evidence. They do not change fixtures,
train another estimator, select a margin, or replay the fitted test policy.
Run from the repository root, passing the repository and artifact root:

```sh
python3 internal/mixedeval/reporttools/correct_phase1_report.py "$PWD" /path/to/artifacts
python3 internal/mixedeval/reporttools/extract_examples.py "$PWD" /path/to/artifacts
python3 internal/mixedeval/reporttools/diagnose_training.py "$PWD" /path/to/artifacts
python3 internal/mixedeval/reporttools/finalize_report.py "$PWD" /path/to/artifacts
```

The artifact root contains `enumerate-v1/`, `trained-v1/`, `selected-v1/`,
`test-v1/`, `longest-causal-test-v1/`, `correctness-both-v1/` and the standalone
`full-state-exhaustive.json`. Original raw reports remain immutable.
`correct_phase1_report.py` refuses to overwrite its corrected output. For a
fresh reconstruction, use a new artifact root with symlinks to those immutable
input directories/files. Other scripts regenerate only derived summaries.

`diagnose_training.py` reuses the frozen estimator's data-construction AST and
stops before the optimizer; it performs zero coefficient updates. The diagnostic
is not an extra model or an optimization-convergence gate.

The frozen Phase1 automatic summary incorrectly used all rows for G3 and used
terminal candidate comparisons for literal conversion. The corrected report uses
test primary groups and marks unsupported paired metrics unavailable. Causal
longest replay is a separate unfitted comparator, preserving its own history.
The finalizer also preserves the whole-composition prelude presentation gap:
G1/G7 are unavailable even though the executed run-scoped subchecks passed.

The final experiment report records source/data/model hashes and freeze commits.
`artifact-index.json` in the artifact root hashes raw evidence and these sources.
