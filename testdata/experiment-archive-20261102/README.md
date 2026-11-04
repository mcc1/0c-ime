# Historical experiment evidence archive

Captured for the user's handover to Claude on 2026-11-04. These archives preserve
original bytes and paths; they are **not** formal run locks or new model results.
Each gzip archive is split at 40 MiB when needed to stay below GitHub file limits.
`archive-index.json` contains ordered parts, whole-archive hashes and every
original file's path, size and SHA-256. All archived file bytes were read back and
verified against the index before commit.

Groups:

- `shift-return-records`: P0–P3, original failing evidence, pre-quality errata,
  acquisition v1, all task/annotation/rejection/alias/closure/qualification records,
  final decision, parent checks and acquisition plans.
- `mixed-boundary-records`: previous completed negative experiment, raw events and
  candidates, frozen fit/dev/test evidence, independent rechecks and source inputs.
- `review-records`: EGM reviews, source snapshots, focused re-audits and corrections.
- `public-runtime-data`: exact public Chinese/English data packs, LM, manifests and
  notices used by the environment. Preserve their own attribution/license terms.

Verify without extracting:

```sh
python3 testdata/experiment-archive-20261102/restore.py --verify-only
```

Restore to a fresh inspection directory:

```sh
python3 testdata/experiment-archive-20261102/restore.py --destination /tmp/ime-records
```

The original root is `/workspace`; therefore the restored directory contains
`scratch/...` and `0c-ime/assets/data/...`. Use `--destination /workspace` only when
restoring that layout intentionally. Existing differing files and symlinks are
rejected; nothing is silently overwritten. `--group shift-return-records` restores
only that group. Original absolute paths inside records are unchanged.

The index lists any omissions: local instruction/private filenames and regenerable
Python/compiled caches are excluded. Original generated evidence and historical
failure records are retained. No personal model/user.json, credentials, cloud
configuration secrets, installer binaries or repository .git directory is included.
The known missing pre-errata report_test.go source copy remains an explicit gap;
this archive does not invent it or claim complete historical source provenance.
