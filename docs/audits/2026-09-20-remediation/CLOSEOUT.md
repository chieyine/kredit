# CI closeout changes — 21 September 2026

Scope: draft PR #16 on `audit/2026-09-20-integrity-remediation`. These changes
continue the scoped audit; they are not a whole-repository certification or a
production release approval. Actual run conclusions must be checked against the
candidate commit, not inferred from this implementation note.

## Buyer payment claims

The restricted-role regression could read an obligation, but creation failed at
`SELECT ... FOR UPDATE`: the buyer has SELECT policies, not UPDATE policies, on
the supplier's obligation. Replacing the lock with an unlocked read would create
a race with collections/payments, and granting buyers debt UPDATE rights would
break the authorization boundary.

Migration 203 instead supplies `app.lock_buyer_payment_claim(uuid)`. It verifies
the original buyer, matching request/obligation identity, active actor, current
purchasing authority and branch restrictions before locking the obligation. It
uses the existing authority lock and does not change a row, disable RLS, or
switch the buyer into the supplier tenant. Its supplementary UPDATE policy is
usable only by the trusted function owner and has `WITH CHECK(false)`; ordinary
buyer connections do not match that policy. Workers have no execute grant.
Claim insertion then runs under the buyer's ordinary policies. Any authenticated
context/input buyer mismatch is rejected, and incoming supplier scope is cleared.

The PostgreSQL regression covers buyer read access without direct debt UPDATE,
unauthorized/spoofed actors, absence of a worker capability grant, cancellation
while waiting for an existing obligation lock, no residual claim after failure,
confirmation/restart replay, conflicting reviewers/reasons, one financial journal
and one payment, correct schedule allocation, and removed-owner rejection.

## Scanner boundaries

The outbound-host check excludes `_test.go` negative fixtures, normalizes DNS
case and fails on source-read errors, missing registers and new production
hosts. Paystack's documentation reference is distinguished from its API endpoint;
known hosted authorization destinations are recorded without claiming that this
code review verifies their processing locations or contractual terms. Dynamic
provider endpoints still need deployment review.

The historical launch-readiness patch contains two explicit dummy rotation-test
values. Exact, anchored Trivy value exceptions preserve that archived evidence;
no file path, detector class or credential prefix is disabled. A real-scanner
regression checks that altered and other unlisted synthetic values still produce
findings. Independent security scans still fail on actual findings. Existing
narrow gosec path exceptions remain disclosed in `scripts/security.sh`.

## Release boundary

The compatible binary now requires migration **203** and the new SQL capability;
this supersedes the older migration-floor references in the initial implementation
note. Apply the existing reviewed maintenance cutover procedure. The new migration
adds no stored fields and does not itself authorize a production migration.

No merge, deployment, production database operation or live provider transaction
is part of this closeout. Import review still must not be represented as applying
opening balances to the ledger. Recorded schema coverage is not completion of
control/legal review, provider certification or restore/deployment rehearsal.
