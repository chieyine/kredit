# Admin surface enablement

The product ships around twenty operations surfaces under `/api/v1/ops/`. Each
one is a privileged path that has to be access-reviewed, audit-logged, and
defended, and a five-to-ten supplier pilot needs far fewer of them than the
product ships.

`ADMIN_SURFACES` enumerates the surfaces a deployment actually operates.
Anything not listed answers `404` and is logged. Production refuses to start
without the setting (`internal/config`), so the enumeration is a decision
someone makes rather than a default someone inherits.

## How it behaves

- **Unset**: every surface is available. This is the development and CI
  behaviour and keeps the test suites working against the full product.
- **`all`**: every surface is available, explicitly. Production may use this,
  but it is then a recorded choice rather than an accident.
- **A list**: only the named surfaces are reachable. The name is the first path
  segment after `/api/v1/ops/`, so `team` covers
  `/api/v1/ops/team/{userID}/roles` as well as the listing.

A disabled surface answers `404` rather than `403`. A caller with no reason to
know the surface exists is not told that it does.

Disabling a surface does not remove its route, its handler, or its permission
checks. It is a reachability control layered on top of them, not a replacement
for them.

## Choosing the list

The launch owners choose this, not the codebase. Two rules make the choice
sound:

1. **Never disable a safety control to reduce surface.** The dual-control admin
   change workflow (`admin-changes`, `review-assignments`, `change-context`,
   `change-history`) exists so that no single operator can make a privileged
   change alone. Turning it off shrinks the surface and enlarges the risk.
2. **Enable on demand, not in advance.** A surface nobody has needed yet is a
   surface nobody has been trained on, and it can be added the day it is first
   required.

A defensible starting point for a pilot, to be reviewed rather than copied:

```
ADMIN_SURFACES=overview,attention,capabilities,approval-inbox,cases,disputes,money,search,users,organizations,jobs,provider-events,audit,analytics,account-recovery,privacy-requests,admin-changes,review-assignments,change-context,change-history,financial-reconciliation,team,metrics,diagnostics
```

Surfaces most often deferred past a first pilot are `business-policies` and
`commands`: policy proposal workflows and the operations command path both
assume an operating scale a first pilot does not have, and the command path in
particular is the highest-privilege surface in the product.

## Reviewing it

The enabled list belongs in the same review as platform role assignments. When
a surface is added, record who asked for it, what workflow needed it, and who
approved it — the same evidence any other privileged grant carries.
