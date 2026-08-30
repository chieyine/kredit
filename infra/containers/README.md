# OCI images

`Dockerfile.api` builds the API, worker, migration, and reconciliation
binaries in a pinned multi-stage build and runs them as the non-root
`distroless` user. The named `api` and `worker` targets provide unambiguous
entrypoints; migration, reconciliation, and simulator jobs may override the
entrypoint on the `api` target:

- `/app/kredit-api`
- `/app/kredit-worker`
- `/app/kredit-migrate`
- `/app/kredit-reconcile`
- `/app/kredit-provider-simulator` (development and contract tests only)

The image does not contain secrets and expects all production configuration
through the deployment environment. It must remain behind the application’s
durability/readiness gate until every domain aggregate is PostgreSQL-backed.

`Dockerfile.web` builds the SvelteKit application with the official Node
adapter, precompresses static assets, and runs as the non-root `node` user.
The root Compose stack starts migrations, API, worker, and web in dependency
order alongside PostgreSQL, object storage, and the local mail sink.
