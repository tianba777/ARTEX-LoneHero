# ARTEX 孤胆英豪

RuoJi6/ARTEX is the upstream mainline. Selectively ported BaaRai features and
source identities are documented in `integration/README.md` and `integration/SOURCES.json`.

- Preserve upstream ancestry, AGPL attribution, existing features, persisted data,
  custom prompts, scheduler/active-run clocks, approval rules and strict Worker bindings.
- Sync using `scripts/sync-upstream.sh` on a review branch. Never force-push,
  replace history, automatically merge upstream or deploy as part of synchronization.
- Keep self-update and image sources on `tianba777/ARTEX-LoneHero`; do not replace
  this application with upstream release binaries.
- Keep lessons candidate-only by default; injection requires explicit adoption
  and task/company scoping. Preserve the opt-in defaults for quiet planning,
  consecutive-failure escalation and the goal gate.
- Do not add excluded exploit/evasion assets or unrelated forks. Do not run real
  target scans, benchmarks that launch targets or deployments for routine validation.
- Verify backend with `go build ./...`, `go vet ./...` and PostgreSQL-backed
  `go test -p 1 -timeout 180s ./...`. Set `ARTEX_PG_DSN` to an isolated disposable
  database; tests use per-package database suffixes and require create-database permission.
- Verify frontend with `npm ci`, `npx tsc --noEmit` and `npm run build:static` in `web`.
  Verify benchmark helpers with `python3 -m unittest discover -s scripts/benchmark -p 'test_*.py'`.
- Use `toolbox-smoke` for both Docker architectures; successful compilation does
  not establish scanner effectiveness or real-target results.
- Never commit runtime configuration, credentials, databases, backups or generated
  application bundles. Private update tokens belong only in the deployment environment.
