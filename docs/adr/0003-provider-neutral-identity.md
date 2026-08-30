# ADR 0003: Provider-neutral identity verification

- Status: accepted for Milestone 2
- Date: 2026-08-16

## Decision

Person, business, authority, consent, and account-ownership facts remain
separate domain records. KYC/KYB providers implement a small capability-based
interface and return normalized verification state plus safe result metadata.
The development runtime uses a deterministic mock provider. Real provider
adapters remain behind approval and feature flags.

## Consequences

- Personal KYC never implies business authority or repayment approval.
- Provider tokens, raw identity documents, BVNs, PINs, and credentials are not
  stored in the domain runtime.
- Provider state must be normalized and auditable without coupling core domain
  rules to a vendor.
- Verification expiry and refresh can block new obligations without deleting
  accepted obligations.

