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

The example deployment enables the complete admin application:

```
ADMIN_SURFACES=all
```

This includes `platform-settings`, `business-policies`, `commands` and owner
controls. Their role checks, recent MFA and audit requirements remain active.
If you use a restricted list, include every surface needed by your admin
workflows; an omitted surface returns 404 even to the owner.

## Reviewing it

The enabled list belongs in the same review as platform role assignments. When
a surface is added, record who asked for it, what workflow needed it, and who
approved it — the same evidence any other privileged grant carries.
