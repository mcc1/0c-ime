# Review workflows available in this cloud environment

Read this index when starting a new task. Files are preinstalled environment dependencies, outside application source. Native skill-picker registration is not assumed: use these explicit file paths if the host does not discover them. A running chat need not refresh its initial skill catalog.

## EGM

Full upstream bundle at `/workspace/.0c-tools/review-skills/vendor/elephant-goldfish-model-403d8881d62022490637d740657b33f01e321ea8/`.
For design review, read `skills/egm-review/SKILL.md` and `docs/codex.md` in that bundle. Other EGM skills, reviewer personas, standards and license are preserved together; do not copy only SKILL.md and lose its relative references.
Use EGM for requested design/specification review. Goldfish review needs native fresh-context subagents. If unavailable, say the review is incomplete; do not simulate a Goldfish in the parent context.

## Code review

Read `/workspace/.0c-tools/review-skills/code-review-codex/SKILL.md` when code review is requested. This is a Codex adaptation of Anthropic's code-review workflow. Preserved official sources and Apache license are in `vendor/claude-code-review/`; `sources.json` records exact commits and hashes. The adaptation uses available Codex agents, not Anthropic models, and never grants permission to post GitHub comments.

Preinstalled does not mean always running: choose the workflow appropriate to the requested task. Do not run expensive full reviews on every edit or require permission gates beyond the user's authorized scope. Review reports do not replace functional tests. Existing repo startup/build instructions remain authoritative for the project.

## Verification and update

Run `bash /workspace/.0c-tools/review-skills/install-review-skills.sh` to restore/check pinned files. The same complete script is appended to the environment's saved install_script; it can reconstruct this directory when missing. Updates are explicit: do not fetch latest automatically or change expected hashes to accept an unverified download. No repository source or personal configuration is changed by this installer.

File installation, future-task startup instructions, skill-picker registration, successful end-to-end review and publication/restoration are separate states. The file installer checks sources and all required workflow references; the new adapted code-review workflow has not yet completed a live review. Publication and a fresh task must still verify restored startup and discovery.
