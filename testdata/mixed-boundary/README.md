# Mixed boundary offline fixtures v1

This is an offline experiment dataset, not a user trace or a product parser.
The acceptance gates are fixed in `design/mixed-input-experiment-acceptance.md`.
The 3,000 Chinese lexical prototypes use declared synthetic intentions and
readings transcribed from `data/lexicon/words.tsv`; attribution and source terms
remain those of `data/sources/ledger.tsv` and `data/sources/licenses/`.
Literal synthetic templates share families. Public device identifiers preserve
original ASCII token spelling, source line, vendor and product; whole vendor and
known manufacturer aliases remain together. Original USB GPLv3 license and PCI
attribution/redistribution notice are under `sources/`. No user data is included.

Sampling budgets, extraction rules, fixed source commits and pre-quality
structural corrections are in `protocol.json`. Broad coverage categories live
in `metadata.json`; they do not turn unrelated lexical prototypes into one
family. Variants of an actual fixed-tail synthetic template remain grouped.
No further source is added merely to reach a confidence bound.

Rebuild only into a new dataset version/directory; generation refuses to
overwrite frozen fixtures. From the repository root, with the pinned original
catalog files downloaded to disposable directories:

```sh
python3 testdata/mixed-boundary/generate.py
python3 testdata/mixed-boundary/add_usbids.py /path/to/pinned-usbids
python3 testdata/mixed-boundary/add_pciids.py /path/to/pinned-pciids /path/to/pinned-usbids
. /workspace/.0c-tools/activate.sh
umask 022
go run -tags=mixedeval ./cmd/0c-mixed-eval -phase assign
go run -tags=mixedeval ./cmd/0c-mixed-eval -phase validate
go run -tags=mixedeval ./cmd/0c-mixed-eval -phase correctness -out /tmp/new-correctness
go run -tags=mixedeval ./cmd/0c-mixed-eval -phase lock
go run -tags=mixedeval ./cmd/0c-mixed-eval -phase enumerate -split all -out /tmp/new-enumeration
```

Python 3 with `jsonschema` validates Draft 2020-12, duplicate JSON keys, aliases,
NFC/ASCII-casefold families and the exact deterministic transitive assignment.
Go validates canonical readings, strict physical spelling, ownership spans and
composition/window events. Existing assignments and run locks cannot be
silently overwritten. Enumeration verifies every frozen source/data hash.
Reports and raw candidate/event logs belong outside the public tree.
Phase1 has no learned proposed display or ranking success. An exploratory pilot
cannot satisfy full GO when independent test groups or required metrics are
missing, irrespective of its point estimates.
