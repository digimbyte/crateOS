# Contributing to CrateOS

Thanks for helping shape CrateOS. This project aims for predictable, panel-first server ops; contributions should reduce ambiguity and improve determinism.

## How to contribute
1. Open an issue describing the user-facing problem and the desired “one true path.”
2. Keep changes scoped; small PRs are easier to review.
3. Include tests or a manual verification note when possible.

## Development basics
- Go 1.24+
- `go fmt ./...` before sending code
- Keep `/srv/crateos` paths canonical; avoid scattering files
- Prefer idempotent, declarative flows over ad-hoc scripts

## Binaries to keep building
- `crateos`
- `crateos-agent`
- `crateos-policy`

## Pull requests
- Describe the user-facing change and risk.
- Note any docs updated (README, GUIDE, docs/*).
- Add co-author line for external contributors if applicable.

## Code style
- Standard Go formatting and linting.
- No hidden side effects; functions should be explicit about filesystem/network mutations.

## Security & access
- Do not introduce default credentials.
- Keep SSH forced-command behavior intact; break-glass remains explicit and audited.
