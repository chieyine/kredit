# Kredit product and engineering principles

These rules are architectural constraints for Kredit. They describe how new product work should extend the platform without weakening financial correctness, customer control, tenant isolation or operational explainability.

## 1. Modular monolith first

Kredit remains one Go application, PostgreSQL database and River worker system until measured load, deployment independence or team ownership proves that a service boundary is necessary. Code should still be organised by domain: identity, organisations, customers, credit sales, obligations, mandates, collections, payments, disputes, notifications, reporting and operations.

Do not introduce a network boundary merely to make a domain look like a microservice. A split must include a plan for transactional consistency, idempotency, reconciliation, observability and failure recovery.

## 2. Integer money only

The backend stores and calculates NGN in integer kobo. The browser uses the shared `web/src/lib/money.ts` helpers. Financial calculations must not use floating point amounts. Formatting is a presentation concern and must never change the authoritative amount.

## 3. Financial history is append-only in meaning

A recognized payment, ledger posting, accepted agreement or collection result is never edited to make history look different. Corrections use explicit reversal, write-off, waiver, adjustment or correction workflows with actor, reason, time and audit evidence. The original event remains visible.

## 4. Explicit state transitions

Credit requests, obligations, mandates, collection attempts, payments, disputes, support cases and administrative changes must move through explicit permitted transitions. Do not expose generic `status = ...` mutation APIs. Unknown provider outcomes remain unknown until reconciliation resolves them.

## 5. Provider adapters are replaceable

Kredit owns the domain language: mandate, collection request, collection result, account discovery and reconciliation. Mono is one provider adapter. Provider-specific request/response fields must not become the source of truth for the Kredit financial model.

A new provider must pass the same provider-contract, persistence, lost-response, duplicate-callback, out-of-order-event and reconciliation tests before it can move real money.

## 6. No automatic retry of financial mutations in the browser

Safe GET requests may retry bounded transient network/5xx failures. Mutations are retried only through an explicit idempotency contract. The browser must never assume a timed-out debit, payment, acceptance, release or administrative action failed.

## 7. Poor connectivity is a product requirement

Seller forms should preserve non-secret unfinished input locally where appropriate. Connectivity loss must be visible. Kredit may retain already-loaded information while offline, but must not claim that a financial or legal action succeeded until the server confirms it. Buyer acceptance, mandate actions and money movement never execute offline.

## 8. Canonical identity before convenience

Phone numbers and other identifiers are normalized before matching. Duplicate warnings and supported merge/correction workflows must be preferred to silently creating parallel customer identities. Financial records must never be reassigned by an ad-hoc database update.

## 9. Operational explainability

Support and finance operators must be able to reconstruct a transaction lifecycle from durable evidence: sale created, buyer decision, goods release/receipt, mandate, notice, debit attempt, provider result, payment recognition, ledger posting, reconciliation and balance change. Technical logs supplement this timeline; they do not replace it.

## 10. Admin is a control plane

Admin surfaces must be permission-scoped, auditable and fail closed. High-risk actions require recent MFA and the existing preview/reason/idempotency controls. Provider secrets are never returned to the browser after configuration; Admin may show only configured/missing state and non-secret readiness information.

Production operations should make queue backlog, dead letters, notification failures, provider events, reconciliation differences and unresolved money states easy to find.

## 11. Product analytics use business events

The server-side product scorecard is authoritative. Prefer events such as seller activated, first sale created, buyer opened sale, sale accepted, payment recognized, repeat sale created and retained seller over click-stream noise. Analytics metadata must not contain direct identifiers or sensitive financial payloads.

## 12. WhatsApp is a delivery channel, not an authority

Transactional WhatsApp messages may deliver secure links, reminders and receipts. The authoritative agreement, mandate, payment and dispute state remains in Kredit. A message delivery receipt must never substitute for buyer acceptance or provider money evidence.

## 13. AI cannot authorize money movement

AI may later summarize queues, extract draft invoice fields or assist support, but an LLM must never determine the authoritative balance, approve a debit, alter a ledger, decide a dispute, accept an agreement on behalf of a customer or bypass deterministic eligibility rules. Any AI output affecting a financial workflow is advisory until a deterministic or authorized human action records the decision.

## 14. Search remains authorization-scoped

Global search is useful only when it preserves the same tenant and platform-role boundaries as direct reads. Search results must not become an alternate path around RLS, role permissions or sensitive-data minimization.

## 15. Staging must look like production before launch

Release evidence should include production-shaped staging concurrency, notification/provider failure simulations, reconciliation after load, backup/restore and incident exercises. CI microbenchmarks are regression signals, not capacity claims.

## 16. Accessibility is part of financial correctness

Critical journeys must remain usable with keyboard navigation, screen readers, high text zoom and narrow mobile screens. Automated checks are required regression protection; manual VoiceOver/TalkBack and representative-device testing remain human release evidence and must never be marked complete by code alone.

## 17. External evidence stays external

CAC approval, provider certification, legal/privacy approval, independent penetration testing, production backup evidence and human launch approval cannot be manufactured by repository code. The application may track their non-secret references and block capabilities until required evidence exists.
