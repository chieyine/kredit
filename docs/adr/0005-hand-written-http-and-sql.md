# ADR 0005: Hand-written HTTP handlers and SQL, with the OpenAPI document as the enforced contract

- Status: accepted
- Date: 3 September 2026
- Supersedes: the generated-handler and sqlc statements in README sections 9.1, 11, 11.1, 14.1 and 15.1

## Context

README section 9.1 selected "generated OpenAPI handlers" for the HTTP stack and
`sqlc` for query generation, and section 11.1 required `api/generated/*` and
`db/generated/*` to be committed generated artifacts that CI keeps in sync.

The implementation went a different way, and the gap was only found by a
full-repository audit:

- Nothing imports `kredit/api/generated`. The 191 routes in `internal/web` are
  hand-written `net/http` `ServeMux` handlers.
- Nothing imports `kredit/db/generated`. All 292 query sites in `internal/*` are
  hand-written SQL executed through `pgx`.
- `api/generated/types.gen.go` (89 KB), `db/generated/` (18 files), the 14 query
  files in `db/queries/`, `sqlc.yaml`, `scripts/sqlc-generate.sh` and
  `scripts/sqlc-check.sh` were therefore maintained, and gated in CI, for code
  that never compiled into the binaries.

That is not a neutral cost. It makes the README untrustworthy as a build
contract, which section 0.2 says it must be, and it invites a future change to
edit generated files that have no effect.

Two things are worth keeping, and are unaffected:

- `api/openapi.yaml` is real and enforced. `scripts/product-contract-sync.mjs`
  verifies that its 190 operations and the 191 backend routes agree, and
  `scripts/frontend-api-coverage.mjs` checks the frontend against the same list.
- Frontend pages define the response shapes they consume and validate received data.
  The unused generated TypeScript client has also been retired.

## Decision

The Go HTTP layer and the Go persistence layer stay hand-written.

1. `api/openapi.yaml` remains the canonical transport contract. It is enforced by
   the contract-sync and coverage gates rather than by generating Go server
   interfaces from it.
2. Frontend response types are maintained alongside their consumers; the
   OpenAPI document remains the contract checked against frontend API usage.
3. Go type generation from OpenAPI, and `sqlc`, are removed along with their
   orphaned output and their CI gate.
4. Financial SQL stays explicit and visible, which is what README section 9.2
   actually asks for: "the project must not hide core financial queries behind a
   general-purpose ORM."

## Consequences

- The repository no longer carries roughly 240 KB of generated Go that nothing
  compiles, nor a `sqlc` gate for it.
- Route and schema drift is still caught, because it is caught by comparing the
  OpenAPI document to the routes rather than by compiling generated stubs.
- Hand-written SQL now has no generated cross-check, so migrations and query
  sites must be reviewed together. The integration suite against a real
  PostgreSQL instance is the compensating control.
- Reintroducing `sqlc` later remains possible: `db/migrations` is unchanged and
  is the schema source it would read. That would be a new ADR.

## Alternatives considered

**Wire the generated code in.** Adopting `oapi-codegen` server interfaces across
191 hand-written routes and moving 292 query sites to `sqlc` is a multi-week
refactor of the entire request and persistence path, for a repository that is
otherwise feature-complete and awaiting provider certification. The risk is not
justified by the benefit at this stage.

**Leave both in place and document the drift.** Rejected: it keeps the
maintenance cost and the CI gate while leaving the README describing an
architecture the code does not have.

## September 9 audit follow-through

The orphaned Go output, SQL query templates and `sqlc.yaml` remained despite
this decision. Their complete source review confirmed zero runtime imports
and substantial schema drift. They are now removed. Database migrations and
the handwritten persistence code remain the active implementation.
