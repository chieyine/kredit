# Local development quick start

The project specification lives in `README.md`; the implementation plan lives in
`IMPLEMENTATION_PLAN.md`. The repository contains the product implementation. The file-by-file audit is
in progress; production launch also requires provider and approval evidence.

1. Copy `.env.example` to `.env` and change development secrets if needed.
2. Start dependencies with `docker compose up -d`.
3. Run `bash scripts/bootstrap.sh`.
4. Start the API and worker with `bash scripts/dev.sh`.

If the `task` binary is installed, the equivalent commands are `task bootstrap`,
`task dev`, and `task ci`.
