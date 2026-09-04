# Kredit

## Production V1 Product, Architecture, Engineering, Security, Operations, and AI-Agent Build Specification

> **Canonical product name:** Kredit  
> **Product category:** Supplier-funded B2B trade-credit infrastructure  
> **Initial market:** Nigeria  
> **Primary users:** Suppliers, wholesalers, distributors, and their business buyers  
> **Primary interfaces:** Mobile-first web application and WhatsApp  
> **Default currency:** NGN  
> **Money storage unit:** Kobo  
> **Default business timezone:** Africa/Lagos  
> **Architecture:** Modular monolith  
> **Frontend:** SvelteKit + Svelte 5 + TypeScript  
> **Backend:** Go 1.26  
> **Database:** PostgreSQL 18  
> **Document status:** Binding source of truth for the first production release  
> **Baseline verified:** 16 August 2026

---

Mono Sweep sandbox backend: see the [setup and certification runbook](docs/runbooks/mono-sweep.md). Migration 052 and local tests are implemented; real Mono sandbox authorization and provider proof remain pending. Production Sweep remains disabled.

## Table of contents

- [0. How to use this document](#0-how-to-use-this-document)
- [1. Product definition](#1-product-definition)
- [2. Product goals and non-goals](#2-product-goals-and-non-goals)
- [3. Users, organisations, and roles](#3-users-organisations-and-roles)
- [4. Jobs to be done](#4-jobs-to-be-done)
- [5. Canonical terminology](#5-canonical-terminology)
- [6. Non-negotiable business rules](#6-non-negotiable-business-rules)
- [7. Pricing and fee policy](#7-pricing-and-fee-policy)
- [8. Complete production v1 scope](#8-complete-production-v1-scope)
- [9. Technology baseline](#9-technology-baseline)
- [10. Architecture overview](#10-architecture-overview)
- [11. Repository structure](#11-repository-structure)
- [12. Local development](#12-local-development)
- [13. Frontend architecture](#13-frontend-architecture)
- [14. Backend architecture](#14-backend-architecture)
- [15. OpenAPI and REST contract](#15-openapi-and-rest-contract)
- [16. PostgreSQL design](#16-postgresql-design)
- [17. Core data model](#17-core-data-model)
- [18. State machines](#18-state-machines)
- [19. Agreement and evidence model](#19-agreement-and-evidence-model)
- [20. Money and ledger architecture](#20-money-and-ledger-architecture)
- [21. Authentication, sessions, and MFA](#21-authentication-sessions-and-mfa)
- [22. Identity, KYC, KYB, and authority](#22-identity-kyc-kyb-and-authority)
- [23. Mandate and collection-provider abstraction](#23-mandate-and-collection-provider-abstraction)
- [24. Payment recording and reconciliation](#24-payment-recording-and-reconciliation)
- [25. Collection engine](#25-collection-engine)
- [26. Instalments and recurring trade lines](#26-instalments-and-recurring-trade-lines)
- [27. Disputes and complaints](#27-disputes-and-complaints)
- [28. Factual trade history and reputation](#28-factual-trade-history-and-reputation)
- [29. WhatsApp integration](#29-whatsapp-integration)
- [30. Notification system](#30-notification-system)
- [31. Background jobs and River](#31-background-jobs-and-river)
- [32. Documents and object storage](#32-documents-and-object-storage)
- [33. Admin, support, and compliance console](#33-admin-support-and-compliance-console)
- [34. Risk and fraud controls](#34-risk-and-fraud-controls)
- [35. Security architecture](#35-security-architecture)
- [36. Privacy and compliance engineering](#36-privacy-and-compliance-engineering)
- [37. Observability and service objectives](#37-observability-and-service-objectives)
- [38. Deployment and infrastructure](#38-deployment-and-infrastructure)
- [39. CI/CD](#39-cicd)
- [40. Testing strategy](#40-testing-strategy)
- [41. Operational runbooks](#41-operational-runbooks)
- [42. Product analytics](#42-product-analytics)
- [43. Design and content specification](#43-design-and-content-specification)
- [44. Build plan](#44-build-plan)
- [45. Definition of done](#45-definition-of-done)
- [46. AI coding-agent operating rules](#46-ai-coding-agent-operating-rules)
- [47. Launch limits](#47-launch-limits)
- [48. Demo and acceptance dataset](#48-demo-and-acceptance-dataset)
- [49. Open questions that must remain explicit](#49-open-questions-that-must-remain-explicit)
- [50. Source and standards baseline](#50-source-and-standards-baseline)
- [51. Final product standard](#51-final-product-standard)

---

## 0. How to use this document

This README is the master build contract for Kredit. It is written for the AI coding agent, engineers, designers, security reviewers, operations staff, compliance advisers, and future technical partners.

The first release is not a throwaway prototype. It is a narrow but complete production product. It must be safe enough to process genuine supplier-funded trade-credit agreements and real repayment events, while avoiding features that would turn Kredit into a bank, lender, wallet, insurance company, marketplace, ERP, or accounting suite.

The AI agent must use the following priority order whenever instructions conflict:

1. Financial correctness and prevention of wrongful debit.
2. Explicit buyer consent and preservation of evidence.
3. Security, privacy, and tenant isolation.
4. This README and approved architecture decisions.
5. Product simplicity for suppliers and buyers.
6. Delivery speed.

A feature belongs in production v1 when it is required to safely:

- create supplier-funded trade credit;
- present and accept its exact terms;
- verify the relevant parties;
- attach a compliant repayment mandate;
- prove that goods were supplied;
- track full, partial, early, scheduled, and collected repayments;
- reconcile balances and fees;
- resolve disputes;
- preserve an audit trail;
- report factual repayment history.

A feature does **not** belong in production v1 merely because it could be useful later. Lending capital, insurance, public credit scores, a buyer marketplace, cards, wallets, inventory management, cross-border settlement, consumer lending, and full accounting remain out of scope.

### 0.1 Interpretation rules

The words **must**, **shall**, and **required** define non-negotiable requirements.

The word **should** defines a strong default that may be changed only through an Architecture Decision Record.

The word **may** defines an optional implementation detail.

Anything that can move, debit, record, reverse, allocate, or calculate money is a **financially material action** and must have:

- validation;
- authorization;
- idempotency;
- concurrency control;
- audit logging;
- tests for the normal path and failure paths.

### 0.2 Required living documents

The repository must contain and maintain:

- `README.md` — this specification;
- `IMPLEMENTATION_STATUS.md` — feature-by-feature completion status;
- `CHANGELOG.md` — user-visible changes;
- `docs/adr/` — architecture decisions;
- `api/openapi.yaml` — canonical API contract (indexed from `docs/api/openapi.yaml`);
- `docs/threat-model.md` — threats, controls, and open risks;
- `docs/data-map.md` — personal and business data inventory;
- `docs/runbooks/` — incident and operational procedures;
- `docs/product/open-questions.md` — unresolved legal, provider, and commercial questions;
- `docs/testing/test-matrix.md` — required scenario coverage;
- `docs/release/readiness-checklist.md` — launch gates.

The coding agent must update these documents as the implementation evolves. Code and generated artifacts that contradict this README are defects until an approved change updates the specification.

---

## 1. Product definition

Kredit digitises trade credit that businesses already extend to one another.

A supplier already knows a business buyer and is willing to let that buyer take goods now and pay later. The supplier uses its own inventory and carries the commercial exposure. Kredit does not initially provide the financing and does not replace the supplier's decision about whom to trust.

Kredit turns the informal statement:

> “Take the goods and pay me next Friday.”

into a verified, buyer-accepted, evidenced, trackable, and collectable commercial obligation.

### 1.1 The canonical transaction

1. A business buyer asks a supplier for goods on credit.
2. The supplier independently decides to consider the request.
3. The supplier creates a Kredit request containing the buyer, goods, amount, repayment schedule, due date, grace period, and evidence.
4. Kredit generates a secure invitation link.
5. The buyer opens the link and sees the exact terms before accepting anything.
6. The buyer verifies identity, business authority, and account ownership as required.
7. The buyer activates an approved repayment mandate.
8. The buyer explicitly accepts the specific trade-credit agreement.
9. Kredit tells the supplier that the required pre-release conditions are complete.
10. The supplier releases the goods.
11. The buyer confirms receipt or raises a structured issue.
12. The obligation becomes active.
13. The buyer may pay fully or partially before the agreed collection date.
14. At the collection time, Kredit recalculates the true outstanding amount.
15. If money remains due and collection is permitted, Kredit initiates a debit only for that amount.
16. The resulting payment, failure, partial recovery, dispute, or mandate event becomes part of a factual trade history.

### 1.2 The simple customer promise

Supplier-facing promise:

> **Give goods on credit with confidence.**

Product explanation:

> Kredit turns the credit you already give your business customers into verified, trackable payment commitments.

Operational instruction:

> **They ask for credit. You send one link.**

### 1.3 What makes Kredit different

Kredit is not conventional BNPL. In conventional BNPL, a lender or fintech finances the purchase. In Kredit's initial model, the supplier provides its own goods on deferred payment terms.

Kredit is not an invoice tool. An invoice requests payment; Kredit records the buyer's acceptance, mandate, supply evidence, repayment schedule, ledger, and collection workflow.

Kredit is not merely debt collection. It enters the transaction before goods are released and helps prevent weak or undocumented credit decisions.

Kredit is not a marketplace. The supplier and buyer may already trade in a shop, warehouse, market, phone call, WhatsApp conversation, sales-representative route, website, or existing ERP.

Kredit owns the **credit commitment and repayment workflow**, not the sale channel.

---

## 2. Product goals and non-goals

### 2.1 Production v1 goals

Production v1 must allow verified businesses to:

- create one-time trade-credit agreements;
- create equal or custom instalment schedules;
- establish recurring trade lines;
- invite buyers by secure mobile link;
- complete buyer KYC/KYB and authority verification;
- attach an approved collection mandate;
- preserve immutable accepted terms;
- confirm release and receipt of goods;
- record voluntary full and partial payments;
- calculate outstanding principal exactly;
- schedule and perform provider-backed collections;
- handle partial collection and permitted retries;
- detect mandate cancellation or failure;
- raise and resolve structured disputes;
- calculate Kredit fees correctly;
- show supplier receivables and ageing;
- show buyer obligations and repayment history;
- communicate through WhatsApp, email, and SMS fallback;
- export statements and reports;
- support authorised supplier teams;
- provide a secure support and compliance console;
- operate with a replaceable payment-provider adapter.

### 2.2 Explicit non-goals

Do not build the following in v1:

- loans funded by Kredit;
- lending marketplace;
- credit insurance or repayment guarantee;
- public blacklist;
- consumer personal loans;
- peer-to-peer debts;
- wallet balances;
- cards;
- POS hardware;
- inventory management;
- purchase-order marketplace;
- full bookkeeping;
- payroll;
- tax filing;
- cross-border payments;
- multi-country support;
- cryptocurrency;
- native mobile applications;
- opaque numeric AI credit scores;
- automatic credit-bureau reporting;
- automatic legal enforcement.

### 2.3 Product boundary

Kredit may describe a transaction as:

- verified;
- accepted;
- active;
- mandate-enabled;
- tracked;
- due;
- paid;
- overdue;
- partially collected;
- disputed.

Kredit must not describe a supplier-funded obligation as:

- guaranteed;
- insured;
- risk-free;
- impossible to cancel;
- certain to be recovered;

unless a later, separately approved product genuinely provides that protection.

---

## 3. Users, organisations, and roles

### 3.1 Supplier organisations

Typical launch users include:

- pharmaceutical distributors;
- medical-consumables suppliers;
- auto-parts wholesalers;
- textile and fashion wholesalers;
- building-material suppliers;
- manufacturers supplying distributors or retailers;
- general wholesalers;
- any verified Nigerian business that routinely gives goods to known business buyers on deferred terms.

Supported supplier legal forms may include:

- limited companies;
- registered business names;
- partnerships;
- verified sole proprietors.

### 3.2 Buyer businesses

Typical buyers include:

- pharmacies;
- clinics and diagnostic centres;
- retailers;
- workshops;
- boutiques;
- spare-parts dealers;
- contractors;
- distributors;
- other verified commercial buyers.

A buyer business may be represented by one or more authorised individuals. Personal identity, business identity, authority to bind the business, account ownership, and mandate consent are separate facts and must be modelled separately.

### 3.3 Supplier roles

#### Owner

Can:

- manage organisation identity;
- manage billing and settlement settings;
- invite and remove users;
- assign roles;
- approve high-value adjustments;
- approve write-offs;
- configure default policies;
- enable provider-backed collection;
- view all data.

#### Administrator

Can manage normal organisation settings and team members but cannot transfer ownership or silently alter high-risk settlement details.

#### Finance

Can:

- view all obligations;
- record and reconcile payments;
- manage collection attempts;
- export reports;
- manage fees;
- propose adjustments and write-offs.

Sensitive actions require step-up authentication.

#### Sales

Can:

- create credit requests;
- select customers;
- upload invoices;
- send invitations;
- confirm release of goods when authorised.

Cannot:

- change settlement accounts;
- waive fees;
- write off balances;
- view restricted organisation settings;
- alter accepted terms.

#### Collections

Can:

- view due and overdue items;
- send reminders;
- manage permitted retry requests;
- handle payment claims;
- participate in disputes.

#### Viewer/Auditor

Read-only access to specifically granted business information.

### 3.4 Platform roles

#### Support Agent

Can view limited support context after a case is opened. Sensitive fields remain masked. Every access must be tied to a support case.

#### Compliance Reviewer

Can review KYC/KYB, provider eligibility, suspicious patterns, data requests, and regulatory holds.

#### Dispute Reviewer

Can inspect agreement and evidence timelines and issue review outcomes within approved authority limits.

#### Platform Administrator

Has no routine unrestricted access. Break-glass use requires:

- step-up authentication;
- a ticket or incident reference;
- a written reason;
- automatic high-severity audit event;
- review by another authorised person.

---

## 4. Jobs to be done

### 4.1 Supplier jobs

A supplier needs to:

- create a credit arrangement quickly;
- know whether the buyer completed the required steps;
- know exactly what is outstanding;
- avoid collecting money already paid;
- see which customers are due or overdue;
- automate reminders and agreed debit attempts;
- prove what was accepted and supplied;
- manage repeated credit relationships;
- prevent employees from giving excessive or unauthorised credit;
- build factual repayment history for customers;
- export an ageing report and customer ledger.

### 4.2 Buyer jobs

A buyer needs to:

- see the exact offer before accepting;
- understand the due dates and collection rules;
- use an existing bank account rather than open a wallet;
- complete verification once and reuse it when appropriate;
- pay early or partially;
- see what remains outstanding;
- receive receipts;
- dispute incorrect or undelivered goods;
- build a positive trade history that may improve future terms;
- control consent to any cross-supplier sharing.

### 4.3 Platform jobs

Kredit needs to:

- preserve evidence;
- enforce domain rules;
- process provider events safely;
- prevent double collection;
- calculate balances and fees exactly;
- detect fraud and operational anomalies;
- support customers without silently changing financial history;
- remain provider-neutral;
- prove every material action through an audit trail.

---

## 5. Canonical terminology

Use these terms in code, APIs, database schema, product copy, support scripts, and reports.

| Term | Meaning |
|---|---|
| Supplier | The business that supplies goods and funds the trade credit itself. |
| Buyer | The business receiving goods and agreeing to pay later. |
| Credit request | A supplier proposal that has not yet become an accepted agreement. |
| Agreement version | An immutable snapshot of the exact terms presented to the buyer. |
| Obligation | An activated amount owed after goods are released and the required evidence exists. |
| Trade relationship | The continuing commercial link between a supplier and buyer. |
| Trade line | A recurring relationship with a maximum outstanding exposure and repayment rules. |
| Drawdown | A new supply of goods added under a trade line. |
| Principal | The value of goods supplied on credit, excluding Kredit fees unless expressly agreed otherwise. |
| Schedule item | One required repayment amount and date. |
| Due date | Date by which the buyer agreed to pay. |
| Grace period | Additional agreed time before automated collection is permitted. |
| Collection time | Earliest instant Kredit may initiate debit for an unpaid amount. |
| Mandate | Provider-backed buyer authorisation for account debit. |
| Voluntary payment | Payment made or confirmed before Kredit initiates collection for that amount. |
| Collection | Debit initiated by Kredit under an active mandate. |
| Settlement | Confirmed arrival of money to the supplier or approved destination. |
| Outstanding principal | Activated principal minus recognised principal-reducing payments and adjustments. |
| Dispute | Formal challenge to all or part of an obligation. |
| Trade history | Factual record created by Kredit transactions and repayment events. |
| Provider event | Signed message from a KYC, banking, collection, messaging, or storage provider. |
| Platform fee | Amount payable to Kredit for its service. |

Do not use “borrower” or “lender” in the ordinary supplier-funded workflow unless legal counsel and the final contract structure require those terms.

---

## 6. Non-negotiable business rules

1. A supplier cannot create a debit merely by entering that a buyer owes money.
2. The buyer must see and accept the exact obligation before goods are released.
3. The buyer must authorise the relevant mandate through a compliant provider flow.
4. A materially changed agreement requires a new immutable version and fresh acceptance.
5. A supplier must confirm goods release.
6. The buyer must be asked to confirm receipt or raise an issue.
7. Principal becomes active only under the approved activation policy.
8. The application must never debit more than the authoritative outstanding amount.
9. An accepted obligation must never be silently deleted.
10. A payment reversal must be represented by a reversal event, not deletion.
11. A mandate cancellation does not erase an underlying accepted obligation.
12. A collection attempt must be idempotent.
13. Duplicate provider events must not post money twice.
14. A dispute may block only the contested amount where the rules permit the undisputed amount to remain payable.
15. A trade-line drawdown must not exceed the available limit.
16. A new drawdown must be blocked when the required mandate is not active.
17. Kredit's collection uplift applies only to money successfully collected through Kredit.
18. Exact supplier-to-supplier exposure information must not be exposed without a lawful basis and buyer consent.
19. The product must distinguish “verified identity” from “approved creditworthiness.”
20. The product must never imply that KYC or a mandate guarantees repayment.
21. Buyer silence may activate an obligation only where a delivered notice proves the buyer was told, and never on a buyer's first trade credit.

---

## 7. Pricing and fee policy

### 7.1 Standard pricing

Kredit's initial supplier pricing is:

- **0.5% base trade-credit service fee** on activated principal;
- **additional 0.5% collection fee** on amounts successfully collected by Kredit at or after the collection time.

### 7.2 Examples

#### Fully paid before collection

Principal: ₦1,000,000  
Voluntary payment: ₦1,000,000  
Kredit collection: ₦0

Base fee: ₦5,000  
Collection fee: ₦0  
Total fee: **₦5,000**

#### Fully collected by Kredit

Principal: ₦1,000,000  
Voluntary payment: ₦0  
Kredit collection: ₦1,000,000

Base fee: ₦5,000  
Collection fee: ₦5,000  
Total fee: **₦10,000**

#### Partially paid and partially collected

Principal: ₦20,000,000  
Voluntary payment: ₦15,000,000  
Kredit collection: ₦5,000,000

Base fee: ₦100,000  
Collection fee: ₦25,000  
Total fee: **₦125,000**

### 7.3 Fee rules

- All calculations use integer kobo.
- Floating-point money is prohibited.
- **Rounding is always down, to the whole kobo, in the payer's favour.** A fee is
  computed as `amount x basis points / 10000` with the remainder discarded, so
  the supplier is never charged a fraction of a kobo it did not incur. The
  maximum effect is one kobo per fee. This is a contractual term, not an
  implementation detail: it is stated in the supplier agreement and enforced by
  `TestFeeRoundingAlwaysFavoursTheSupplier`.
- The base fee accrues only when principal becomes active.
- A request cancelled before goods are released does not attract the base fee.
- A trade-line limit itself is not billable; actual activated drawdowns are billable.
- Collection fees accrue only on confirmed successful collected amounts.
- Failed debit attempts do not attract the collection uplift.
- Provider charges are separate internal costs unless commercial policy later passes them through transparently.
- Fee waivers and refunds require an authorised role, reason code, note, and audit event.
- Tax handling must remain configurable and must not be guessed by the application.

### 7.4 Supplier billing methods

Preferred order:

1. compliant split settlement or fee deduction supported by a licensed provider;
2. supplier-authorised billing debit after settlement;
3. consolidated supplier invoice on an approved billing cycle.

Do not introduce a Kredit wallet merely to collect Kredit's fees.

---

## 8. Complete production v1 scope

### 8.1 Supplier onboarding

Required:

- phone verification;
- email verification;
- legal and trading name;
- business type;
- registration information where applicable;
- principal business address;
- industry;
- authorised representative;
- settlement destination;
- billing method;
- KYB state;
- team creation;
- notification preferences;
- default credit policy;
- terms and privacy acceptance;
- mandatory MFA for owner and finance roles.

Real collections remain disabled until the supplier reaches the required compliance state.

### 8.2 Buyer onboarding

Required:

- secure invitation;
- phone verification;
- personal identity verification;
- business identity verification where applicable;
- authority-to-bind-business verification;
- provider-hosted mandate authorisation;
- consent versioning;
- accepted agreement version;
- bank-account token or provider reference;
- masked account metadata only where needed;
- buyer portal access.

Do not ask a buyer to send a BVN, OTP, PIN, or online-banking credential through WhatsApp.

### 8.3 One-time credit

Must support:

- buyer selection or invitation;
- principal;
- goods description;
- invoice upload;
- due date;
- grace period;
- collection date and time;
- buyer acceptance;
- mandate activation;
- release confirmation;
- receipt confirmation;
- voluntary payments;
- automatic collection;
- disputes;
- statement and closure.

#### 8.3.1 Receipt confirmation and deemed acceptance

Activation normally follows an explicit buyer answer: confirm receipt, or raise
an issue. Where a buyer neither confirms nor objects, a sale may be activated by
silence only under all of the following conditions, which are enforced together
by `internal/credit` and by the database trigger in
`db/migrations/070_deemed_acceptance_evidence.sql`:

- goods release is recorded;
- the buyer business has previously answered a goods-release notice itself, on
  another sale, so a buyer's first trade credit is never activated by silence;
- a goods-release notice was delivered to the buyer, evidenced by an
  authenticated delivery receipt, and has been with them for the full waiting
  period;
- the waiting period is `DEEMED_ACCEPTANCE_MIN_HOURS`, at least 24 hours and by
  default 72, so goods released before a weekend closure cannot be deemed
  accepted before the buyer reopens.

Elapsed time alone is never sufficient. Where the evidence cannot be read, the
sale stays in `RECEIPT_CONFIRMATION_PENDING` and waits for an explicit answer:
delaying a supplier is recoverable, debiting a buyer who was never told is not.

The goods-release notice states the deadline and what silence will be taken to
mean, and links the buyer to the screen where they can object.

### 8.4 Instalment credit

Must support:

- equal instalments;
- custom amounts;
- weekly, fortnightly, monthly, or custom dates;
- separate schedule-item states;
- partial payment;
- early settlement;
- allocation rules;
- collection for only unpaid instalments or the authorised overdue amount;
- total mandate ceiling.

### 8.5 Recurring trade lines

Must support:

- approved limit;
- current exposure;
- available amount;
- repayment cadence;
- default grace period;
- start and optional end date;
- active mandate;
- multiple drawdowns;
- buyer confirmation of each drawdown in v1;
- payment allocation;
- limit blocking;
- suspension after mandate or risk issues.

### 8.6 Payments

Must support:

- integrated voluntary bank payment;
- supplier-recorded off-platform transfer;
- cash or other offline settlement records;
- full payment;
- partial payment;
- multiple payments;
- allocation to schedule items;
- reversal;
- reconciliation;
- receipts;
- buyer notification.

### 8.7 Collection

Must support:

- one-time variable collection;
- recurring scheduled collection;
- partial provider success;
- configurable retries;
- provider status reconciliation;
- multi-account sweep only when explicitly approved and capability-enabled;
- mandate cancellation detection;
- direct-to-beneficiary settlement only when approved;
- no duplicate debit.

### 8.8 Disputes

Must support:

- full or partial disputed amount;
- standard reasons;
- evidence upload;
- supplier response;
- human review;
- resolution outcome;
- ledger adjustment;
- collection blocking for contested amounts;
- complete decision audit.

### 8.9 Trade history

Must display factual, interpretable events rather than an unexplained score:

- verified-since date;
- completed obligations;
- total completed principal;
- current active principal;
- largest completed amount;
- on-time count and percentage;
- average days late;
- unresolved overdue obligations;
- dispute counts and outcomes;
- mandate cancellations while owing;
- partial recovery history;
- repeat supplier relationships.

### 8.10 Reports

Supplier reports must include:

- receivables summary;
- due today;
- due this week;
- overdue;
- ageing buckets;
- voluntary payments;
- Kredit collections;
- fees;
- largest exposures;
- mandate issues;
- disputes;
- customer ledger;
- transaction statement.

### 8.11 Admin and operations

Must include:

- customer search;
- agreement timeline;
- KYC/KYB state;
- mandate state;
- provider events;
- ledger view;
- collection attempts;
- disputes;
- support notes;
- suspicious activity flags;
- failed jobs;
- failed webhooks;
- controlled retries;
- fee waiver workflow;
- write-off workflow;
- correction workflow;
- audit search.

---

## 9. Technology baseline

### 9.1 Selected stack

| Layer | Selection |
|---|---|
| Frontend framework | SvelteKit with Svelte 5 and TypeScript |
| Frontend runtime | Node.js 24 LTS |
| Package manager | pnpm workspace |
| Styling | Tailwind CSS 4 |
| Component foundation | shadcn-svelte and Bits UI, customised into Kredit's own design system |
| Backend language | Go 1.26, latest patched point release |
| HTTP stack | Standard `net/http` ServeMux with hand-written handlers and explicit middleware ([ADR 0005](docs/adr/0005-hand-written-http-and-sql.md)) |
| API contract | REST, OpenAPI 3.1, problem-details errors; `api/openapi.yaml` is enforced against the routes by `scripts/product-contract-sync.mjs` |
| Database | PostgreSQL 18, latest patched point release |
| Database driver | `pgx/v5`, patched release at or above the version fixing known 2026 advisories |
| Query generation | None. Financial SQL is written explicitly against `pgx` ([ADR 0005](docs/adr/0005-hand-written-http-and-sql.md)) |
| Migrations | Goose, SQL-first migrations |
| Durable jobs | River using PostgreSQL and `pgx/v5` |
| Cache | None by default; Redis only when a measured need exists |
| Object storage | S3-compatible object storage |
| Telemetry | OpenTelemetry traces and metrics, structured `slog` logs |
| Browser tests | Playwright |
| Go tests | Standard testing, race detector, fuzzing, integration database |
| Architecture | Modular monolith with separate API and worker processes |
| Deployment unit | OCI containers for Go; SvelteKit adapter selected per web host |

### 9.2 Why this stack

SvelteKit provides a fast mobile application shell, server rendering, form and route primitives, and flexible deployment while keeping the user interface lighter than a typical React application.

Go provides explicit concurrency, simple deployment, strong standard-library networking, predictable performance, and a good fit for webhooks, collection orchestration, schedules, reconciliation, and background jobs.

PostgreSQL is the source of truth because Kredit's data is relational and transactional. Agreements, limits, obligations, payments, allocations, ledger postings, disputes, mandates, users, and audit records require strong constraints and ACID transactions.

`pgx` and `sqlc` keep database behaviour visible. The project must not hide core financial queries behind a general-purpose ORM.

River gives Kredit durable PostgreSQL-backed jobs and transactionally safe job insertion, allowing the initial platform to avoid Redis as a mandatory operational dependency.

### 9.3 Version policy

The lockfile and Go module files, not this README, define exact dependency patches.

Rules:

- use supported stable releases;
- apply Go and PostgreSQL security point releases promptly;
- pin production dependencies;
- use Renovate or Dependabot for controlled updates;
- run `govulncheck`, OSV scanning, and frontend dependency scanning in CI;
- never upgrade a financial dependency automatically without tests and review;
- keep `pgx/v5` above versions affected by published security advisories;
- do not deploy PostgreSQL beta versions.

### 9.4 Official technical references

- Go release history: <https://go.dev/doc/devel/release>
- SvelteKit documentation: <https://svelte.dev/docs/kit>
- Svelte 5 documentation: <https://svelte.dev/docs/svelte>
- PostgreSQL documentation: <https://www.postgresql.org/docs/current/>
- pgx: <https://pkg.go.dev/github.com/jackc/pgx/v5>
- sqlc: <https://docs.sqlc.dev/>
- River: <https://riverqueue.com/docs>
- Tailwind CSS: <https://tailwindcss.com/docs>
- shadcn-svelte: <https://www.shadcn-svelte.com/docs>
- OpenAPI TypeScript: <https://openapi-ts.dev/>
- oapi-codegen: <https://github.com/oapi-codegen/oapi-codegen>
- OpenTelemetry Go: <https://opentelemetry.io/docs/languages/go/>

---

## 10. Architecture overview

```text
                           ┌─────────────────────────┐
                           │     Marketing site      │
                           │  SvelteKit, SSR/static  │
                           └────────────┬────────────┘
                                        │
                                        ▼
┌──────────────────┐        ┌─────────────────────────┐
│ WhatsApp / Email │        │   Kredit Web / PWA     │
│ / SMS channels   │◄──────►│ SvelteKit + Svelte 5    │
└────────┬─────────┘        └────────────┬────────────┘
         │                                │ same-origin /api proxy
         ▼                                ▼
┌───────────────────────────────────────────────────────────┐
│                         Go API                            │
│ Auth · Organisations · Credit · Agreements · Ledger      │
│ Trade Lines · Payments · Mandates · Collections · Fees   │
│ Disputes · Reputation · Reports · Admin · Audit           │
└──────────────┬────────────────────┬───────────────────────┘
               │                    │
               ▼                    ▼
      ┌────────────────┐   ┌────────────────────┐
      │ PostgreSQL 18  │   │ S3 object storage  │
      │ source of truth│   │ documents/evidence │
      └───────┬────────┘   └────────────────────┘
              │
              ▼
      ┌────────────────┐
      │ River job queue│
      └───────┬────────┘
              ▼
┌───────────────────────────────────────────────────────────┐
│                       Go Worker                           │
│ Reminders · Collection schedules · Retries · Reconcile   │
│ Reports · File scanning · Provider polling · Notifications│
└──────────────┬────────────────────────────────────────────┘
               ▼
┌───────────────────────────────────────────────────────────┐
│ External provider adapters                               │
│ KYC/KYB · Mandates · Debit · Settlement · WhatsApp       │
│ Email · SMS · Storage scan · Fraud signals                │
└───────────────────────────────────────────────────────────┘
```

### 10.1 Process boundaries

Production starts with three deployable processes:

- `kredit-web` — SvelteKit frontend;
- `kredit-api` — synchronous Go HTTP API;
- `kredit-worker` — River workers and scheduled processing.

The API and worker share the same Go domain modules and database. This is still a modular monolith, not microservices.

### 10.2 Source-of-truth rules

- PostgreSQL is authoritative for domain state.
- The ledger is authoritative for money-derived balances.
- External providers are authoritative for their own mandate, debit, and settlement states.
- Provider states must be normalised into Kredit states.
- WhatsApp messages are not authoritative financial records.
- Svelte client state is never authoritative.
- Cached or denormalised balances must be reproducible.

### 10.3 Same-origin browser architecture

The browser should call `/api/*` on the application origin. The web host or edge proxy forwards that path to the Go API.

Benefits:

- simpler secure cookie handling;
- reduced CORS complexity;
- easier CSRF protections;
- consistent request tracing;
- no exposure of internal provider endpoints.

A separate `api.kredit.com.ng` may later serve approved third-party API clients, but the first-party web application should use same-origin requests.

### 10.4 Why no microservices

Microservices would add:

- distributed transactions;
- more deployment and secret surfaces;
- event-consistency problems;
- more expensive observability;
- harder local development;
- premature organisational boundaries.

The domain must be modular in code, but it remains one database and one coordinated release until scale, compliance isolation, or team ownership provides a concrete reason to split it.

---

## 11. Repository structure

Use one repository so the OpenAPI contract, database schema, Go code, Svelte client, infrastructure, tests, and documentation evolve together.

```text
kredit/
├── cmd/
│   ├── api/                  # Go HTTP service entry point
│   ├── worker/               # River workers and schedulers
│   ├── migrate/              # Embedded Goose migration runner
│   ├── seed/                 # Deterministic development/demo data
│   └── reconcile/            # Controlled operational reconciliation command
├── internal/
│   ├── access/               # Roles, permissions, tenant context
│   ├── admin/                # Platform operations and support
│   ├── agreements/           # Immutable terms and acceptance evidence
│   ├── audit/                # Immutable audit events
│   ├── auth/                 # OTP, sessions, MFA, passkeys
│   ├── buyers/               # Buyer profiles and authority
│   ├── collections/          # Eligibility, attempts, retry policy
│   ├── config/               # Typed configuration
│   ├── credit/               # Requests and obligations
│   ├── disputes/             # Dispute domain
│   ├── documents/            # Uploads, evidence, malware state
│   ├── fees/                 # Pricing and fee accrual
│   ├── identity/             # KYC/KYB abstractions
│   ├── ledger/               # Journal, postings, invariants
│   ├── mandates/             # Mandate lifecycle
│   ├── notifications/        # Channel-neutral messages
│   ├── organizations/        # Supplier organisations and teams
│   ├── payments/             # Voluntary payments and allocation
│   ├── providers/            # External adapter implementations
│   ├── reputation/           # Factual trade history
│   ├── reports/              # Read models and exports
│   ├── risk/                 # Factual flags and manual review
│   ├── settlements/          # Supplier settlement observations
│   ├── tradelines/           # Limits and drawdowns
│   └── web/                  # HTTP middleware and hand-written handlers
├── api/
│   ├── openapi.yaml          # Canonical OpenAPI 3.1 specification
│   ├── components/           # Optional split OpenAPI components
│   └── examples/             # Request and response fixtures
├── db/
│   ├── migrations/           # Goose SQL migrations
│   ├── fixtures/             # Integration-test data
│   └── seeds/                # Local/staging demo seeds
├── web/
│   ├── src/
│   │   ├── lib/
│   │   │   ├── api/          # Generated types and client wrappers
│   │   │   ├── components/   # Kredit design-system components
│   │   │   ├── features/     # Feature modules
│   │   │   ├── forms/        # Shared validation/form helpers
│   │   │   ├── money/        # Display-only money utilities
│   │   │   ├── stores/       # Minimal global UI state
│   │   │   └── utils/
│   │   ├── routes/
│   │   ├── hooks.server.ts
│   │   ├── app.html
│   │   └── service-worker.ts
│   ├── static/
│   ├── tests/
│   └── package.json
├── docs/
│   ├── adr/
│   ├── api/
│   ├── compliance/
│   ├── product/
│   ├── release/
│   ├── runbooks/
│   └── testing/
├── infra/
│   ├── containers/
│   ├── terraform-or-opentofu/
│   ├── monitoring/
│   └── environments/
├── scripts/
├── tests/
│   ├── contract/
│   ├── e2e/
│   ├── provider-simulators/
│   └── performance/
├── .github/workflows/
├── .env.example
├── .golangci.yml
├── docker-compose.yml
├── go.mod
├── go.sum
├── package.json
├── pnpm-lock.yaml
├── pnpm-workspace.yaml
├── Taskfile.yml
├── README.md
└── IMPLEMENTATION_STATUS.md
```

### 11.1 Generated code policy

Go server code and Go database access are hand-written; see
[ADR 0005](docs/adr/0005-hand-written-http-and-sql.md).

The following are generated and must never be edited manually:

- `web/src/lib/api/generated/schema.d.ts`;
- any generated OpenAPI client file.

`api/openapi.yaml` remains the canonical transport contract. Because no Go
server code is generated from it, drift is caught by comparing the document to
the implemented routes instead: `scripts/product-contract-sync.mjs` fails when
the OpenAPI operations and the backend routes disagree, and
`scripts/frontend-api-coverage.mjs` fails when a route has no frontend surface
and no recorded exemption. CI must fail when generation produces an uncommitted
diff.

### 11.2 Module structure

Each substantial Go module should use a predictable structure:

```text
internal/credit/
├── domain.go          # Entities and value objects
├── commands.go        # Write use cases
├── queries.go         # Read use cases
├── service.go         # Domain orchestration
├── repository.go      # Interfaces owned by the domain
├── events.go          # Domain events
├── policy.go          # Business rules
├── errors.go          # Stable domain errors
├── handler.go         # HTTP adapter only
└── *_test.go
```

Avoid generic dumping-ground packages called `utils`, `helpers`, `common`, or `services` in Go. Shared code must have a clear domain or platform purpose.

---

## 12. Local development

### 12.1 Required tools

- Go 1.26, latest security patch;
- Node.js 24 LTS;
- pnpm;
- Docker and Docker Compose;
- PostgreSQL client tools;
- sqlc;
- Goose;
- Task;
- Playwright browser dependencies;
- optional `mise` or equivalent version manager.

### 12.2 One-command start

The repository must support:

```bash
task bootstrap
task dev
```

`task bootstrap` must:

- verify tool versions;
- install frontend dependencies;
- download Go modules;
- start local dependencies;
- apply database migrations;
- generate sqlc code;
- generate OpenAPI code and TypeScript types;
- load deterministic development seed data;
- install Playwright browsers if missing.

`task dev` must run:

- SvelteKit development server;
- Go API with reload where practical;
- Go worker;
- PostgreSQL;
- local S3-compatible storage such as MinIO;
- provider simulators;
- local mail viewer.

### 12.3 Local services

Recommended Docker Compose services:

- PostgreSQL 18;
- MinIO;
- Mailpit or equivalent;
- optional OpenTelemetry collector;
- optional Grafana/Tempo/Prometheus local stack;
- provider webhook simulator.

Redis is not required for the normal local stack.

### 12.4 Task commands

Required commands:

```text
task dev
task build
task test
task test:unit
task test:integration
task test:e2e
task test:race
task test:fuzz
task lint
task security
task generate
task api:lint
task db:migrate
task db:rollback
task db:reset
task db:seed
task db:check
task openapi:generate
task sqlc:generate
task web:check
task web:test
task ci
```

`task ci` must reproduce the required CI checks locally.

For a release gate, run `SECURITY_STRICT=1 bash scripts/security.sh`. Strict
mode fails when a required scanner is unavailable instead of silently treating
the check as complete.

### 12.5 Environment configuration

All configuration must be typed and validated on startup. The application must fail fast when a required production variable is absent or malformed.

Use environment variables for deployment-specific configuration and secrets. Do not commit secrets.

Database credentials are separated by workload. `DATABASE_URL` authenticates
with an unprivileged API login and the API enters the `kredit_app` role at
connection startup. `RIVER_DATABASE_URL` uses a distinct worker login and the
worker enters `kredit_worker`. `DATABASE_DIRECT_URL` is reserved for migrations,
development seed fixtures, controlled backups, resets, and rollbacks; it must
never be supplied to an API or worker deployment. Apply
`infra/postgres/roles.sql` after migrations. Local and Compose environments can
provision the development-only login wrappers with
`scripts/configure-development-database.sh`; production credentials come from
the deployment secret manager.
`BACKUP_DATABASE_URL` uses a separate read-only login that enters
`kredit_backup`; that role may bypass RLS solely so `pg_dump` can read every
tenant and forced-RLS audit row, but it has no write or administration grants.

Categories:

```text
APP_ENV
APP_VERSION
PUBLIC_BASE_URL
APP_BASE_URL
API_INTERNAL_URL
DATABASE_URL
DATABASE_DIRECT_URL
RIVER_DATABASE_URL
OBJECT_STORAGE_ENDPOINT
OBJECT_STORAGE_BUCKET
OBJECT_STORAGE_REGION
OBJECT_STORAGE_ACCESS_KEY
OBJECT_STORAGE_SECRET_KEY
SESSION_SIGNING_KEY
FIELD_ENCRYPTION_KEY_ID
OTP_HMAC_KEY
TOKEN_HASH_KEY
WHATSAPP_PROVIDER
EMAIL_PROVIDER
SMS_PROVIDER
IDENTITY_PROVIDER
COLLECTION_PROVIDER
OTEL_EXPORTER_OTLP_ENDPOINT
SENTRY_DSN
FEATURE_*
```

Do not expose private environment values through SvelteKit public variables.

### 12.6 Seed accounts

The local seed must create:

- ABC Pharmaceuticals Ltd;
- Royal Pharmacy Ltd;
- supplier owner;
- finance officer;
- sales representative;
- buyer authorised representative;
- one-time credit demo;
- instalment demo;
- recurring trade-line demo;
- overdue demo;
- mandate-cancellation demo;
- dispute demo.

All seed credentials and OTPs must be visibly marked as non-production.

---

## 13. Frontend architecture

### 13.1 Svelte rules

Use Svelte 5 syntax and runes for new code.

- Use `$state` for local reactive state.
- Use `$derived` for computed UI state.
- Use `$effect` only for genuine side effects, not as a substitute for derivation.
- Prefer server load functions for initial data.
- Prefer SvelteKit navigation and form primitives.
- Avoid large global stores.
- Never duplicate authoritative financial calculations in the browser.

### 13.2 Data access

The Go OpenAPI schema is the contract.

Generation flow:

```text
api/openapi.yaml
      ├── oapi-codegen → Go request/response types and handler interface
      └── openapi-typescript → TypeScript path/schema types
                              + openapi-fetch client
```

The frontend must not hand-write duplicate API response interfaces.

Use SvelteKit's provided `fetch` in server `load` functions so cookies and internal routing behave correctly.

### 13.3 State ownership

| State | Owner |
|---|---|
| Money, balances, statuses | Go API/PostgreSQL |
| Logged-in user and permissions | Go session plus server load |
| Route data | SvelteKit load data |
| Form draft | Local component state |
| Toasts and temporary UI | Small client store |
| Theme and display preferences | Local preference store |
| Provider financial state | Provider, normalised by Go backend |

Do not use a global client store as a shadow database.

### 13.4 Routes

Recommended route map:

```text
/
/how-it-works
/pricing
/security
/for-suppliers
/for-buyers
/faq
/legal/terms
/legal/privacy
/legal/complaints

/app
/app/overview
/app/credit/new
/app/credit/[id]
/app/customers
/app/customers/[id]
/app/trade-lines
/app/trade-lines/[id]
/app/payments
/app/collections
/app/overdue
/app/disputes
/app/reports
/app/team
/app/settings
/app/settings/billing
/app/settings/settlement
/app/settings/security

/buyer
/buyer/requests
/buyer/obligations
/buyer/obligations/[id]
/buyer/trade-lines
/buyer/history
/buyer/mandates
/buyer/disputes
/buyer/settings

/c/[token]                    # secure buyer invitation
/pay/[token]                  # optional payment flow
/receipt/[public-token]       # privacy-safe receipt

/admin
/admin/users
/admin/organizations
/admin/money
/admin/search
/admin/cases
/admin/cases/[id]
/admin/disputes
/admin/disputes/[id]
/admin/provider-events
/admin/jobs
/admin/team
/admin/audit
```

### 13.5 Layouts

- Marketing layout: public, fast, SEO-friendly.
- Supplier app layout: authenticated navigation and organisation context.
- Buyer layout: simpler obligations-focused navigation.
- Public acceptance layout: distraction-free and mobile-first.
- Admin layout: separate role gate and stronger session policy.

### 13.6 Component system

Use shadcn-svelte and Bits UI as editable foundations. Do not ship an unmodified generic shadcn dashboard.

Required Kredit primitives:

```text
Money
MoneyInput
Percentage
DateTime
DueDate
StatusPill
RiskFact
ReferenceCode
Timeline
AuditTimeline
AgreementSummary
MandateStatus
PaymentBreakdown
OutstandingBalance
FeeBreakdown
TradeLineMeter
ScheduleTable
CollectionAttemptCard
DisputePanel
CustomerIdentityCard
BusinessVerificationCard
DocumentUploader
DocumentViewer
ConfirmFinancialAction
StepUpAuthDialog
EmptyState
InlineError
SystemBanner
```

### 13.7 Design tokens

Define CSS variables for:

- background layers;
- foreground text;
- muted text;
- borders;
- primary brand colour;
- positive state;
- warning state;
- overdue state;
- destructive state;
- disputed state;
- focus ring;
- charts;
- spacing;
- radii;
- shadows;
- type scale.

Never use colour alone to communicate financial status.

### 13.8 Accessibility

Target WCAG 2.2 AA.

Required:

- keyboard navigation;
- visible focus;
- semantic headings;
- labelled controls;
- accessible dialogs;
- screen-reader announcements for async status;
- sufficient contrast;
- reduced-motion support;
- touch targets suitable for mobile;
- meaningful error summaries;
- table alternatives on narrow screens.

### 13.9 Money display

The browser may format amounts using `Intl.NumberFormat`, but it receives exact integer-kobo values or decimal strings from the API.

Rules:

- display NGN with `₦` and thousands separators;
- show full amount on agreement and confirmation screens;
- never abbreviate a financially material confirmation as `₦1.2M`;
- allow abbreviated display only in charts or summary cards with accessible full value;
- show `₦0.00` only where decimals are relevant; normal trade amounts may display whole naira;
- never calculate fees in TypeScript for authoritative use.

### 13.10 Date and timezone display

- Store instants in UTC.
- Store contractual local dates explicitly where needed.
- Display commercial due dates in Africa/Lagos by default.
- Show an exact date, not only “tomorrow”.
- Confirmation screens must show date, time, and timezone when collection timing matters.
- Do not let browser timezone silently alter contractual dates.

### 13.11 PWA policy

Kredit should be installable as a PWA, but financial actions are online-only.

May cache:

- application shell;
- static assets;
- public marketing content;
- non-sensitive help content.

Must not cache:

- API responses containing financial or identity data;
- invitation pages;
- agreement details;
- documents;
- receipts with sensitive content;
- admin pages;
- authenticated HTML snapshots.

Do not queue financial mutations for background sync. When offline, the app must state clearly that the action was not submitted.

### 13.12 Browser support

Support current and recent versions of:

- Chrome on Android;
- Safari on iOS;
- Chrome, Edge, Firefox, and Safari desktop.

Tailwind 4's browser baseline must be checked against actual target devices before launch. If field testing shows material use of unsupported old browsers, adopt a documented fallback rather than silently breaking the product.

---

## 14. Backend architecture

### 14.1 HTTP layer

Use Go's standard `net/http` stack and current `ServeMux` method/wildcard routing. Handlers are hand-written and checked against `api/openapi.yaml` by the contract-sync gate rather than generated from it ([ADR 0005](docs/adr/0005-hand-written-http-and-sql.md)).

The HTTP layer is responsible for:

- request parsing;
- schema validation;
- authentication context;
- authorisation gate;
- idempotency extraction;
- domain command invocation;
- response mapping;
- problem-details errors;
- trace metadata.

It must not contain:

- SQL;
- ledger arithmetic;
- provider-specific business rules;
- direct state mutation;
- fee calculations.

### 14.2 Middleware order

Recommended order:

1. request ID;
2. panic recovery;
3. trusted-proxy and client-IP handling;
4. security headers where API responses require them;
5. request size limits;
6. rate limiting;
7. authentication/session load;
8. CSRF validation for cookie-authenticated writes;
9. tenant context;
10. idempotency handling;
11. OpenTelemetry tracing;
12. structured access logging;
13. route handler.

Webhook routes use a specialised chain that preserves the raw request body and verifies provider signatures before acting.

### 14.3 Command and query separation

Do not over-engineer full CQRS. Use a practical separation:

- commands perform state changes in transactions;
- queries build read models efficiently;
- financial commands use locking and ledger writes;
- reporting queries may use denormalised views or materialised read tables.

### 14.4 Transactions

Every financially material command must define its transaction boundary explicitly.

Examples requiring one database transaction:

- activate obligation plus post principal plus accrue base fee plus enqueue notifications;
- record payment plus allocate payment plus update schedule state plus enqueue receipt;
- create collection attempt plus reserve amount plus enqueue provider call;
- process successful provider event plus post collection plus allocate plus accrue collection fee;
- resolve dispute plus post adjustment plus update collection eligibility.

Use `pgx.Tx`. Do not pass a global repository that silently opens nested transactions.

### 14.5 Unit of work pattern

A lightweight unit-of-work abstraction may provide:

```go
type UnitOfWork interface {
    WithinTx(ctx context.Context, opts TxOptions, fn func(ctx context.Context, tx Tx) error) error
}
```

The transaction object must expose generated sqlc queries and River transactional insertion where required.

### 14.6 Domain errors

Define stable errors:

- `ErrNotFound`;
- `ErrForbidden`;
- `ErrConflict`;
- `ErrInvalidTransition`;
- `ErrIdempotencyConflict`;
- `ErrMandateInactive`;
- `ErrCollectionBlocked`;
- `ErrInsufficientAvailableLimit`;
- `ErrAgreementVersionMismatch`;
- `ErrPaymentExceedsOutstanding`;
- `ErrDisputedAmount`;
- `ErrProviderUnavailable`;
- `ErrStepUpRequired`.

Map them to stable problem-details response codes. Do not expose raw database or provider errors to clients.

### 14.7 Logging

Use `log/slog` with structured JSON in production.

Every request log should include where available:

- request ID;
- trace ID;
- authenticated user ID;
- organisation ID;
- resource public reference;
- provider;
- operation;
- outcome;
- duration;
- safe error code.

Never log:

- raw BVN;
- OTP;
- PIN;
- provider access token;
- full bank account number;
- identity document;
- complete provider webhook body;
- agreement access token.

### 14.8 Graceful shutdown

API and worker must:

- stop accepting new work;
- allow bounded in-flight work to finish;
- release River leadership cleanly;
- flush telemetry;
- close database pools;
- terminate before platform hard timeout.

---

## 15. OpenAPI and REST contract

### 15.1 Contract-first approach

`api/openapi.yaml` is the canonical transport contract.

Changes must follow this order:

1. update OpenAPI;
2. lint it;
3. regenerate the TypeScript client types;
4. implement the domain/handler;
5. add contract and end-to-end tests;
6. commit the generated TypeScript and run `pnpm run audit` so the contract-sync gate confirms the document and the routes still agree.

### 15.2 API versioning

Use `/v1` for first public API routes.

Backward-compatible additions may remain in v1. Breaking changes require:

- a new API version;
- migration plan;
- deprecation period;
- consumer communication.

### 15.3 Error format

Use `application/problem+json` compatible with RFC 9457.

Example:

```json
{
  "type": "https://docs.kredit.com.ng/problems/mandate-inactive",
  "title": "Payment mandate is not active",
  "status": 409,
  "code": "MANDATE_INACTIVE",
  "detail": "This trade line cannot receive a new drawdown until the buyer restores the mandate.",
  "instance": "/v1/trade-lines/tl_.../drawdowns",
  "request_id": "req_...",
  "errors": []
}
```

### 15.4 Public resource identifiers

Do not expose sequential database IDs.

Use PostgreSQL 18 UUIDv7 values for primary identifiers or generate UUIDv7 in Go. Human-facing references must be separate and readable.

Examples:

- `TCR-7H4M-92QK` — credit request;
- `TCA-4F8N-1K6P` — agreement;
- `TCL-8D3S-77YX` — trade line;
- `TCP-2A9V-40MR` — payment;
- `TCC-5G2J-13ZW` — collection.

References must include a checksum or enough entropy to reduce transcription mistakes and guessing.

### 15.5 Idempotency

Require `Idempotency-Key` for:

- credit request creation;
- invitation send;
- buyer acceptance;
- goods release confirmation;
- payment recording;
- collection initiation;
- adjustment;
- dispute resolution;
- fee waiver;
- settlement change.

Store:

- key hash;
- actor scope;
- route and method;
- request hash;
- response status and body;
- creation and expiry.

If the same key is reused with a different request hash, return `IDEMPOTENCY_CONFLICT`.

### 15.6 Optimistic concurrency

Mutable resources must include a version or ETag.

Use `If-Match` for changes to:

- draft requests;
- organisation settings;
- trade-line policies;
- disputes under review;
- team roles.

Financial posting operations use row locks and idempotency rather than only optimistic concurrency.

### 15.7 Pagination

Use cursor pagination, not offset pagination, for large event and transaction lists.

Response:

```json
{
  "data": [],
  "page": {
    "next_cursor": "...",
    "has_more": true
  }
}
```

### 15.8 Representative endpoints

```text
POST   /v1/credit-requests
GET    /v1/credit-requests/{credit_request_id}
PATCH  /v1/credit-requests/{credit_request_id}
POST   /v1/credit-requests/{credit_request_id}/send
POST   /v1/credit-requests/{credit_request_id}/cancel

GET    /v1/public/credit-invitations/{token}
POST   /v1/public/credit-invitations/{token}/verify-phone
POST   /v1/public/credit-invitations/{token}/accept
POST   /v1/public/credit-invitations/{token}/decline
POST   /v1/public/credit-invitations/{token}/confirm-receipt

POST   /v1/obligations/{obligation_id}/confirm-release
GET    /v1/obligations/{obligation_id}
GET    /v1/obligations/{obligation_id}/statement
POST   /v1/obligations/{obligation_id}/payments
POST   /v1/obligations/{obligation_id}/payment-claims
POST   /v1/obligations/{obligation_id}/disputes

POST   /v1/trade-lines
GET    /v1/trade-lines/{trade_line_id}
PATCH  /v1/trade-lines/{trade_line_id}
POST   /v1/trade-lines/{trade_line_id}/drawdowns
POST   /v1/trade-lines/{trade_line_id}/suspend
POST   /v1/trade-lines/{trade_line_id}/resume

GET    /v1/customers
GET    /v1/customers/{customer_id}
GET    /v1/customers/{customer_id}/history
GET    /v1/customers/{customer_id}/exposure

GET    /v1/mandates
POST   /v1/mandates/{mandate_id}/authorization-session
POST   /v1/mandates/{mandate_id}/restore

POST   /v1/collections/{collection_id}/retry
GET    /v1/collections/{collection_id}

GET    /v1/reports/receivables
GET    /v1/reports/ageing
GET    /v1/reports/fees
POST   /v1/reports/exports

POST   /v1/webhooks/identity/{provider}
POST   /v1/webhooks/collection/{provider}
POST   /v1/webhooks/messaging/{provider}
POST   /v1/webhooks/storage/{provider}
```

### 15.9 Create-credit request example

```json
{
  "buyer": {
    "phone": "+2348012345678",
    "business_name": "Royal Pharmacy Ltd"
  },
  "principal_kobo": 120000000,
  "currency": "NGN",
  "goods_description": "Pharmaceutical supplies under invoice ABC-1288",
  "invoice_document_id": "019...",
  "repayment": {
    "type": "ONE_TIME",
    "due_date": "2026-09-30",
    "due_time": "17:00:00",
    "timezone": "Africa/Lagos",
    "grace_period_hours": 48
  },
  "supplier_note": "Goods released after mandate activation and acceptance"
}
```

### 15.10 Response projection

Responses should include action-oriented information:

```json
{
  "id": "019...",
  "reference": "TCR-7H4M-92QK",
  "lifecycle_status": "SENT",
  "payment_status": "UNPAID",
  "mandate_status": "NOT_STARTED",
  "principal_kobo": 120000000,
  "outstanding_kobo": 0,
  "next_action": {
    "actor": "BUYER",
    "code": "REVIEW_AND_ACCEPT"
  },
  "links": {
    "self": "/v1/credit-requests/019..."
  }
}
```

---

## 16. PostgreSQL design

### 16.1 Database version

Use PostgreSQL 18 on the latest security patch. Do not use beta or release-candidate database builds in production.

### 16.2 Database schemas

Recommended logical schemas:

```text
app       # main domain tables
ledger    # journal and account tables
jobs      # River-managed tables
reporting # optional views/materialised read models
```

Do not place application tables casually in `public`.

### 16.3 Extensions

Use the minimum required extensions. Likely:

- `pgcrypto` where cryptographic database functions are specifically needed;
- `citext` only if case-insensitive fields justify it.

PostgreSQL 18 supports UUIDv7 generation. Prefer UUIDv7 for locality-friendly identifiers.

### 16.4 Money types

- Database: `bigint` kobo.
- Go: named `int64` money type with checked arithmetic.
- API: integer kobo for first-party clients, accompanied by currency.
- Reports: derive naira display from kobo.

Do not use PostgreSQL `money`, floating point, or JavaScript number arithmetic for authoritative values.

### 16.5 Time types

- Instants: `timestamptz` in UTC.
- Contractual local date: `date`.
- Contractual local time: `time` plus IANA timezone.
- Created/updated timestamps: `timestamptz`.

A due date is a contractual business date and must not change because the user's device timezone changes.

### 16.6 Common columns

Mutable operational tables normally include:

```sql
id uuid primary key default uuidv7(),
created_at timestamptz not null default now(),
updated_at timestamptz not null default now(),
version bigint not null default 1
```

Immutable tables omit `updated_at` and version where changes are prohibited.

Financial and agreement records are never hard-deleted. Use explicit cancellation, void, reversal, supersession, or retention workflows.

### 16.7 Constraints before application checks

Enforce at database level where possible:

- positive principal;
- valid currency;
- non-negative balance;
- fee basis not negative;
- due date not before agreement date unless imported under a controlled workflow;
- drawdown not exceeding line limit;
- allocation not exceeding payment;
- unique provider event;
- unique reference;
- valid enum checks;
- balanced journal transaction;
- one active membership per user/organisation;
- one active processing collection reservation for the same amount scope.

### 16.8 Index policy

Every index must correspond to an observed query or a known integrity requirement.

Likely indexes:

- organisation and status;
- buyer business and status;
- due/collection time for open schedule items;
- overdue obligations;
- provider reference;
- public reference;
- phone normalised hash;
- open dispute;
- mandate state;
- created-at cursor indexes;
- audit target and time;
- ledger account and sequence.

Review index bloat and unused indexes operationally.

### 16.9 Migration policy

Use Goose SQL migrations.

Rules:

- migrations are timestamped;
- production migrations are forward-safe;
- destructive operations use expand/migrate/contract;
- do not combine long data backfills with blocking schema changes;
- large indexes use concurrent creation where appropriate;
- migrations run as a dedicated release job, not automatically in every app instance;
- every migration is tested against a production-like schema copy;
- down migrations are provided for development where safe, but production rollback may use forward fixes;
- accepted agreement records must remain readable after every migration.

### 16.10 Row-level security

Use defence-in-depth tenant isolation for supplier-owned tables.

Pattern:

1. API authenticates user and selects organisation.
2. A database transaction begins.
3. The application sets `SET LOCAL app.current_organization_id`.
4. RLS policies restrict rows to the current organisation.
5. SQL queries still include explicit organisation predicates.
6. Platform jobs and controlled admin workflows use separate database roles.

RLS does not replace application authorisation. It reduces the blast radius of a missing predicate.

Network-level buyer identity and trade-history tables require carefully designed policies because they may be referenced by multiple suppliers. Access must flow through relationship and consent tables.

### 16.11 Locking strategy

Use `SELECT ... FOR UPDATE` for:

- obligation outstanding changes;
- payment allocation;
- collection reservation;
- trade-line available-limit changes;
- dispute resolution affecting collectability;
- fee reversal.

Use PostgreSQL advisory locks only when a row lock cannot represent the coordination key. Document every advisory-lock key scheme.

Avoid `SERIALIZABLE` globally. Use it only for commands that genuinely require it and implement retry on serialization failure.

---

## 17. Core data model

This section defines the minimum tables and their responsibilities. Exact names may be refined through an ADR, but the concepts must not be collapsed in ways that destroy auditability.

### 17.1 Authentication and access

#### `app.users`

- `id`;
- normalised email;
- normalised phone;
- display name;
- account status;
- last authenticated time;
- created time.

#### `app.sessions`

- user;
- hashed opaque token;
- device label;
- IP metadata;
- user agent;
- authentication level;
- expires time;
- revoked time;
- rotation lineage.

#### `app.otp_challenges`

- target type and hash;
- purpose;
- code HMAC/hash;
- attempt count;
- expiry;
- consumed time;
- risk metadata.

#### `app.mfa_methods`

- user;
- method type;
- encrypted configuration or credential reference;
- verified time;
- last used time;
- revoked time.

#### `app.organizations`

- legal name;
- trading name;
- business type;
- registration identifiers;
- industry;
- status;
- default timezone;
- default currency.

#### `app.memberships`

- organisation;
- user;
- role;
- status;
- invited by;
- accepted time.

### 17.2 Identity and business entities

#### `app.persons`

Represents a verified or invited individual without making the user account itself the legal identity.

#### `app.businesses`

Represents supplier and buyer legal/business entities.

#### `app.business_representatives`

Links a person to a business and records:

- role/title;
- authority type;
- authority verification status;
- start/end dates;
- supporting evidence.

#### `app.verification_cases`

Normalised KYC/KYB state:

- subject type;
- provider;
- provider reference;
- verification level;
- state;
- reasons;
- started/completed/expired times;
- safe result metadata.

Store provider tokens and masked facts, not raw identity data unless strictly required.

#### `app.bank_account_references`

- owner subject;
- provider;
- token/reference;
- bank code;
- masked account;
- account-name result;
- account type;
- ownership state;
- active state.

### 17.3 Supplier-buyer network

#### `app.trade_relationships`

- supplier organisation;
- buyer business;
- relationship status;
- first transaction time;
- last transaction time;
- supplier-private customer code;
- supplier-private notes;
- default policy reference.

Supplier-private notes must never leak into buyer-visible or cross-supplier views.

#### `app.relationship_consents`

Records what information the buyer permits to be shared and with whom.

### 17.4 Credit creation

#### `app.credit_requests`

Mutable only while draft or awaiting acceptance.

- supplier organisation;
- relationship/buyer;
- request type;
- proposed principal;
- currency;
- description;
- current agreement-version ID;
- lifecycle state;
- invitation state;
- expiry;
- created by.

#### `app.agreement_versions`

Immutable.

- credit request;
- version number;
- canonical JSON terms;
- rendered document hash;
- principal;
- goods description;
- due/schedule terms;
- grace terms;
- mandate disclosure;
- terms version;
- privacy version;
- created by;
- created time.

#### `app.agreement_acceptances`

Immutable acceptance evidence:

- agreement version;
- accepting person;
- represented business;
- acceptance method;
- timestamp;
- IP/device metadata where lawful;
- OTP or authentication assurance level;
- signed evidence hash;
- provider mandate reference at acceptance.

### 17.5 Obligations and schedules

#### `app.obligations`

- accepted agreement;
- supplier;
- buyer;
- activated principal;
- currency;
- lifecycle status;
- payment status;
- outstanding cached value;
- outstanding version;
- activation time;
- closed time;
- trade-line ID where applicable.

The cached outstanding value is validated against the ledger and may be rebuilt.

#### `app.repayment_schedules`

- obligation;
- schedule type;
- timezone;
- allocation policy;
- status.

#### `app.schedule_items`

- schedule;
- sequence;
- principal due;
- local due date;
- local due time;
- due instant;
- grace hours;
- collection instant;
- allocated amount;
- collected amount;
- status;
- disputed amount;
- collection-block reason.

### 17.6 Trade lines

#### `app.trade_lines`

- relationship;
- approved limit;
- cached current exposure;
- cached available amount;
- cadence;
- default grace period;
- start/end;
- lifecycle state;
- mandate;
- suspension reason;
- terms version.

#### `app.drawdowns`

Each new supply under a line:

- trade line;
- principal;
- goods description;
- invoice;
- agreement/confirmation version;
- release and receipt states;
- activation state;
- resulting obligation.

### 17.7 Goods and documents

#### `app.goods_releases`

- obligation or drawdown;
- supplier actor;
- release time;
- delivery method;
- delivery-note document;
- notes.

#### `app.receipt_confirmations`

- obligation or drawdown;
- buyer actor;
- state;
- received time;
- issue reason;
- evidence.

#### `app.documents`

- owner organisation;
- uploader;
- purpose;
- object key;
- original filename;
- media type;
- byte size;
- checksum;
- malware state;
- availability state;
- retention class;
- created time.

### 17.8 Mandates

#### `app.payment_mandates`

- buyer subject;
- provider;
- provider mandate ID;
- mandate type;
- amount ceiling;
- frequency ceiling;
- start/end;
- state;
- primary account token;
- capability snapshot;
- accepted disclosure version;
- provider-updated time.

#### `app.mandate_events`

Append-only normalised events:

- mandate;
- provider event;
- old state;
- new state;
- reason code;
- event time;
- processed time.

### 17.9 Payments and collection

#### `app.payments`

- obligation/buyer/supplier;
- source type;
- amount;
- currency;
- provider/reference;
- state;
- paid time;
- recognised time;
- recorded by;
- evidence document;
- reversal linkage.

#### `app.payment_allocations`

- payment;
- schedule item or principal target;
- amount;
- allocation order;
- created time.

#### `app.collection_reservations`

Prevents concurrent over-collection:

- obligation;
- schedule item where applicable;
- outstanding snapshot version;
- reserved amount;
- state;
- expiry;
- idempotency key.

#### `app.collection_attempts`

- reservation;
- provider;
- provider collection ID;
- requested amount;
- succeeded amount;
- state;
- attempt number;
- request time;
- final time;
- retry classification;
- safe failure code.

#### `app.settlement_events`

- payment/collection;
- supplier destination;
- provider settlement reference;
- gross amount;
- fee amount;
- net amount;
- state;
- expected/actual time.

### 17.10 Fees

#### `app.fees`

- supplier organisation;
- obligation/drawdown/payment link;
- fee type;
- basis amount;
- rate basis points;
- amount;
- currency;
- state;
- accrued time;
- paid time;
- waived/refunded amount.

### 17.11 Disputes

#### `app.disputes`

- obligation;
- opened by;
- total disputed amount;
- reason;
- state;
- collection effect;
- assigned reviewer;
- opened/resolved times;
- outcome.

#### `app.dispute_evidence`

- dispute;
- party;
- document;
- statement;
- submitted time.

#### `app.dispute_decisions`

Append-only decision history:

- dispute;
- reviewer;
- outcome;
- valid principal;
- adjustment amount;
- reason;
- decision time.

### 17.12 Operations

#### `app.provider_events`

- provider;
- provider event ID;
- type;
- signature state;
- safe payload hash;
- received time;
- processed time;
- processing state;
- failure code;
- retry count.

Unique `(provider, provider_event_id)`.

#### `app.idempotency_records`

- actor scope;
- key hash;
- method/path;
- request hash;
- response status;
- encrypted or safe response snapshot;
- expiry.

#### `app.audit_events`

Append-only:

- actor type and ID;
- organisation;
- action;
- target type and ID;
- request ID;
- support/incident reason;
- safe before/after metadata;
- time.

#### `app.notifications`

- recipient;
- channel;
- template;
- event reference;
- state;
- provider message ID;
- scheduled/sent/delivered/read/failed times.

#### `app.support_cases`

- requester;
- organisation;
- category;
- severity;
- status;
- assigned agent;
- linked resources;
- notes.

---

## 18. State machines

Do not use one giant status field. Kredit has independent lifecycle, payment, mandate, dispute, collection, and settlement dimensions.

### 18.1 Credit lifecycle

```text
DRAFT
  → SENT
  → BUYER_REVIEWING
  → BUYER_ACCEPTED
  → VERIFICATION_PENDING
  → READY_TO_RELEASE
  → GOODS_RELEASED
  → RECEIPT_CONFIRMATION_PENDING
  → ACTIVE
  → CLOSED
```

Alternative terminal paths:

```text
DRAFT/SENT → CANCELLED_BEFORE_ACTIVATION
SENT → EXPIRED
BUYER_REVIEWING → DECLINED
```

No path may jump directly from `DRAFT` to `ACTIVE`.

### 18.2 Payment status

- `UNPAID`;
- `PARTIALLY_PAID`;
- `PAID`;
- `DUE`;
- `OVERDUE`;
- `PARTIALLY_COLLECTED`;
- `WRITTEN_OFF`.

Payment status is derived from schedule, ledger, and time. It must not be manually set without a domain command.

### 18.3 Mandate status

- `NOT_STARTED`;
- `PENDING`;
- `ACTIVE`;
- `PAUSED`;
- `CANCELLED`;
- `EXPIRED`;
- `FAILED`.

### 18.4 Collection status

- `NOT_SCHEDULED`;
- `SCHEDULED`;
- `BLOCKED`;
- `RESERVED`;
- `QUEUED`;
- `SUBMITTED`;
- `PROCESSING`;
- `SUCCEEDED`;
- `PARTIALLY_SUCCEEDED`;
- `FAILED_RETRYABLE`;
- `FAILED_FINAL`;
- `CANCELLED_BEFORE_SUBMISSION`.

### 18.5 Dispute status

- `NONE`;
- `OPEN`;
- `AWAITING_SUPPLIER`;
- `AWAITING_BUYER`;
- `UNDER_REVIEW`;
- `PARTIALLY_RESOLVED`;
- `RESOLVED`;
- `WITHDRAWN`.

### 18.6 Trade-line status

- `PROPOSED`;
- `PENDING_BUYER_ACCEPTANCE`;
- `PENDING_MANDATE`;
- `ACTIVE`;
- `SUSPENDED`;
- `EXPIRED`;
- `CLOSED`.

### 18.7 Valid transition enforcement

Every transition must:

- be defined in a domain policy;
- validate actor permission;
- validate current state;
- validate required evidence;
- run in a transaction;
- update version;
- append audit event;
- enqueue required notifications.

Controllers, SQL callers, and admin tools may not assign status fields directly.

---

## 19. Agreement and evidence model

### 19.1 Immutable agreement version

The buyer must accept an immutable canonical representation containing at minimum:

- supplier legal and trading identity;
- buyer identity and represented business;
- goods description;
- invoice reference and hash where present;
- principal;
- currency;
- payment schedule;
- due dates and times;
- grace period;
- collection time;
- mandate scope and ceiling;
- early-payment treatment;
- dispute process;
- supplier and Kredit fee disclosures relevant to the parties;
- terms version;
- privacy notice version;
- complaint path.

### 19.2 Acceptance assurance

Record:

- authenticated user/person;
- phone or email verification assurance;
- KYC/KYB state;
- authority-to-bind-business state;
- timestamp;
- IP and device metadata where lawful;
- agreement hash;
- mandate reference and state;
- exact acceptance action.

A checkbox preselected by default is not valid acceptance.

### 19.3 Material changes

Material changes include:

- principal;
- goods;
- due date;
- instalment amount;
- grace period;
- collection scope;
- mandate ceiling;
- buyer or supplier legal party;
- fees charged to the buyer;
- dispute terms.

A material change requires:

1. new agreement version;
2. buyer review;
3. fresh acceptance;
4. audit link to the superseded version.

### 19.4 Generated agreement PDF

After activation, generate a PDF or printable document containing:

- human-readable terms;
- transaction reference;
- acceptance details;
- goods evidence list;
- payment schedule;
- mandate summary;
- support and dispute instructions;
- cryptographic document hash.

The PDF is a representation of the canonical stored agreement, not the canonical source itself.

### 19.5 Invitation token security

- Generate at least 256 bits of cryptographically secure randomness.
- Store only a keyed hash of the token.
- Expire the token.
- Bind it to the specific request and intended buyer contact.
- Require phone verification before sensitive details or acceptance.
- Rotate the token after a resend where risk warrants it.
- Rate-limit token lookup attempts.
- Never place raw token values in logs or analytics.

---

## 20. Money and ledger architecture

### 20.1 Principle

The ledger is an append-only balanced subledger. Kredit may not own the supplier's receivable, but it needs an exact journal to reproduce:

- activated principal;
- returns and reductions;
- voluntary repayments;
- collected repayments;
- reversals;
- write-offs;
- fees;
- provider costs;
- settlements.

### 20.2 Ledger tables

#### `ledger.accounts`

- account ID;
- owner scope;
- account code;
- account type;
- currency;
- normal balance;
- created time.

#### `ledger.transactions`

- transaction ID;
- event type;
- reference type and ID;
- idempotency key;
- effective time;
- recorded time;
- description;
- reversal-of transaction where applicable.

#### `ledger.postings`

- transaction;
- account;
- debit kobo;
- credit kobo;
- sequence;
- metadata.

Constraint: for each ledger transaction and currency, total debits equal total credits.

### 20.3 Account examples

Per obligation or supplier as appropriate:

- `TRADE_RECEIVABLE_CONTROL`;
- `PRINCIPAL_ORIGINATED_CONTROL`;
- `VOLUNTARY_SETTLEMENT_CONTROL`;
- `COLLECTION_SETTLEMENT_CONTROL`;
- `RETURNS_ADJUSTMENT_CONTROL`;
- `WRITE_OFF_CONTROL`;
- `SUPPLIER_FEE_RECEIVABLE`;
- `PLATFORM_SERVICE_REVENUE`;
- `PLATFORM_COLLECTION_REVENUE`;
- `PROVIDER_FEE_EXPENSE`;
- `PROVIDER_PAYABLE`;
- `SETTLEMENT_CLEARING`.

### 20.4 Example postings

#### Activate ₦2,000,000 principal

```text
Dr Trade Receivable Control                 ₦2,000,000
Cr Principal Originated Control             ₦2,000,000
```

#### Recognise ₦500,000 voluntary payment

```text
Dr Voluntary Settlement Control               ₦500,000
Cr Trade Receivable Control                    ₦500,000
```

#### Recognise ₦300,000 collected by Kredit

```text
Dr Collection Settlement Control              ₦300,000
Cr Trade Receivable Control                    ₦300,000
```

#### Accrue 0.5% base fee on ₦2,000,000

```text
Dr Supplier Fee Receivable                      ₦10,000
Cr Platform Service Revenue                     ₦10,000
```

#### Accrue 0.5% collection fee on ₦300,000

```text
Dr Supplier Fee Receivable                       ₦1,500
Cr Platform Collection Revenue                   ₦1,500
```

### 20.5 Outstanding calculation

```text
Outstanding principal
= activated principal
- recognised voluntary payments allocated to principal
- recognised Kredit collections allocated to principal
- approved goods-return reductions
- approved principal adjustments
- approved write-offs
+ payment reversals
```

Fees are not included in buyer principal unless the buyer's accepted agreement expressly and lawfully makes them payable by the buyer. The initial model charges the supplier.

### 20.6 Ledger invariants

- Every ledger transaction balances.
- A posting amount is non-negative.
- Exactly one of debit or credit is positive per posting.
- Outstanding principal never becomes negative.
- Allocations never exceed payment amount.
- Collection never exceeds outstanding principal.
- A reversed transaction cannot be reversed twice.
- A provider event cannot post the same collection twice.
- Cached balances equal a ledger rebuild in reconciliation tests.

### 20.7 Financial event service

Only the ledger domain may create ledger postings.

Other modules issue typed commands such as:

```go
type ActivatePrincipal struct {
    ObligationID uuid.UUID
    Amount       Money
    EffectiveAt  time.Time
    Idempotency  string
}
```

The ledger service validates and posts the correct balanced journal.

### 20.8 Reconciliation

Run automated reconciliation:

- after every provider event;
- daily for open obligations;
- daily for provider settlements;
- before statement generation for high-value accounts;
- on demand through controlled operations.

Reconciliation results must identify:

- ledger mismatch;
- provider mismatch;
- settlement mismatch;
- duplicate provider reference;
- missing webhook;
- outstanding cache mismatch;
- fee mismatch.

A mismatch creates an operational case and may block further collection on the affected obligation.

---

## 21. Authentication, sessions, and MFA

### 21.1 Authentication model

Use passwordless authentication initially:

- phone OTP;
- email OTP;
- passkeys/WebAuthn where supported;
- TOTP or passkey as MFA for privileged users.

Avoid passwords unless a later requirement justifies the additional credential-recovery and breach risk.

### 21.2 Session tokens

Browser sessions use opaque random tokens, not long-lived JWTs.

Rules:

- generate at least 256 bits of randomness;
- store only a cryptographic hash in the database;
- send through `Secure`, `HttpOnly`, `SameSite=Lax` cookies;
- scope cookies narrowly;
- rotate after login, MFA, privilege change, and sensitive recovery;
- revoke on logout and suspicious activity;
- maintain device/session management;
- expire inactive and absolute lifetimes;
- do not store tokens in local storage.

### 21.3 Authentication assurance levels

Suggested internal levels:

- `AAL0` — unauthenticated;
- `AAL1` — verified OTP session;
- `AAL2` — OTP plus TOTP/passkey or equivalent MFA;
- `AAL3` — controlled high-assurance review for exceptional operations.

Actions requiring AAL2 include:

- change settlement destination;
- change billing method;
- add owner/admin/finance role;
- approve large adjustment;
- approve write-off;
- manual collection retry above configured threshold;
- export highly sensitive data;
- platform admin break-glass use.

### 21.4 OTP controls

- short validity window;
- single purpose;
- maximum attempts;
- resend cooldown;
- per-contact, per-device, and per-IP limits;
- code stored as HMAC/hash, never plaintext;
- consumed atomically;
- generic response to avoid account enumeration;
- provider delivery event tracked;
- step-up checks for high-risk changes.

### 21.5 Account recovery

Recovery must not rely solely on an easily ported phone number for privileged accounts.

Supplier owner recovery may require:

- verified email;
- existing MFA or recovery code;
- business identity evidence;
- manual support review;
- cooling-off period for settlement changes;
- notifications to existing contacts.

### 21.6 CSRF

Because browser authentication uses cookies:

- same-origin API proxy is preferred;
- state-changing routes require CSRF protection;
- verify `Origin` and `Sec-Fetch-Site`;
- use synchroniser or double-submit CSRF tokens as appropriate;
- webhook routes are excluded and use provider signatures instead.

### 21.7 Authorisation

Permissions are checked in the Go application before every protected operation.

Do not rely on:

- hidden frontend buttons;
- route visibility;
- role text from the client;
- provider metadata alone.

The database RLS layer is defence-in-depth, not the primary permission engine.

---

## 22. Identity, KYC, KYB, and authority

### 22.1 Provider abstraction

```go
type IdentityProvider interface {
    Name() string
    CreatePersonVerification(ctx context.Context, in PersonVerificationInput) (VerificationSession, error)
    CreateBusinessVerification(ctx context.Context, in BusinessVerificationInput) (VerificationSession, error)
    GetVerification(ctx context.Context, providerID string) (ProviderVerification, error)
    VerifyWebhook(ctx context.Context, headers http.Header, body []byte) (VerifiedIdentityEvent, error)
}
```

### 22.2 Data minimisation

Prefer storing:

- provider verification reference;
- verification state;
- verified name;
- masked identifier;
- verification level;
- completion/expiry date;
- safe reason codes.

Avoid storing:

- raw BVN;
- raw identity-document images outside provider-hosted storage;
- face biometrics;
- bank credentials;
- OTPs.

If raw regulated identifiers become unavoidable, complete a DPIA, encrypt the field, minimise access, and define retention.

### 22.3 Person versus business

The following must remain independent:

1. The individual is who they claim to be.
2. The business exists.
3. The individual is associated with the business.
4. The individual has authority to accept the transaction for the business.
5. The bank account belongs to the relevant person or business.
6. The mandate is active.

A successful personal KYC does not prove authority to bind a company.

### 22.4 Verification levels

Implement configurable levels rather than hard-coded amount assumptions.

Example:

- Level 0: invitation only, no activation;
- Level 1: phone and basic identity;
- Level 2: provider KYC and account ownership;
- Level 3: business KYB and authorised representative;
- Enhanced review: high-risk or high-value case.

The compliance policy maps transaction types and values to required levels.

### 22.5 Expiry and refresh

Verification may expire or become stale. Store expiry where supplied and create refresh workflows.

Block or review new obligations when:

- KYC expired;
- business status changed;
- representative authority ended;
- provider flags mismatch;
- suspicious activity requires review.

Existing accepted obligations remain preserved even if later verification expires.

---

## 23. Mandate and collection-provider abstraction

### 23.1 Capability-based interface

```go
type ProviderCapabilities struct {
    OneTimeDebit             bool
    RecurringDebit           bool
    VariableAmountDebit      bool
    PartialDebit             bool
    MultiAccountSweep        bool
    BusinessAccounts         bool
    PersonalAccounts         bool
    DirectToBeneficiary      bool
    MandateStatusWebhooks    bool
    CollectionStatusWebhooks bool
    SettlementWebhooks       bool
    Reversals                bool
}

type CollectionProvider interface {
    Name() string
    Capabilities(ctx context.Context) (ProviderCapabilities, error)

    CreateAuthorizationSession(
        ctx context.Context,
        in CreateAuthorizationSessionInput,
    ) (AuthorizationSession, error)

    GetMandate(
        ctx context.Context,
        providerMandateID string,
    ) (ProviderMandate, error)

    RequestCollection(
        ctx context.Context,
        in RequestCollectionInput,
    ) (ProviderCollectionRequest, error)

    GetCollection(
        ctx context.Context,
        providerCollectionID string,
    ) (ProviderCollection, error)

    CancelPendingCollection(
        ctx context.Context,
        providerCollectionID string,
    ) error

    VerifyWebhook(
        ctx context.Context,
        headers http.Header,
        rawBody []byte,
    ) (VerifiedProviderEvent, error)

    NormaliseEvent(
        ctx context.Context,
        event VerifiedProviderEvent,
    ) (NormalisedProviderEvent, error)
}
```

### 23.2 Required implementations

- `MockCollectionProvider` — deterministic development and tests;
- `SandboxCollectionProvider` — optional provider sandbox wrapper;
- first approved production provider adapter.

Do not build an adapter against undocumented behaviour.

### 23.3 Provider selection

A provider routing service may choose based on:

- supplier eligibility;
- buyer account type;
- mandate type;
- transaction amount;
- provider capability;
- availability;
- cost;
- regulatory approval;
- existing active mandate.

Provider routing must not silently create multiple overlapping mandates without buyer disclosure.

### 23.4 Feature flags

Required flags:

```text
REAL_COLLECTIONS_ENABLED
MULTI_ACCOUNT_SWEEP_ENABLED
PARTIAL_SWEEP_ENABLED
DIRECT_TO_BENEFICIARY_ENABLED
BUSINESS_ACCOUNT_MANDATES_ENABLED
RECURRING_MANDATES_ENABLED
PROVIDER_ROUTING_ENABLED
NETWORK_HISTORY_SHARING_ENABLED
OFF_PLATFORM_PAYMENT_CLAIMS_ENABLED
```

A disabled feature must fail safely and clearly. Never fall back from an approved multi-account workflow to an unapproved debit mode without explicit policy.

### 23.5 Provider webhooks

Workflow:

1. Receive raw body with strict size limit.
2. Authenticate source/signature.
3. Store safe event metadata and encrypted raw payload only if required.
4. Deduplicate by provider event ID.
5. Return provider-required acknowledgement promptly.
6. Process asynchronously where allowed.
7. Normalise event.
8. lock relevant resource;
9. apply idempotent domain transition;
10. append audit event;
11. reconcile with provider API when necessary.

### 23.6 Polling and reconciliation

Webhooks may be delayed, duplicated, or lost. The worker must poll uncertain provider states according to a bounded schedule.

Poll when:

- submission timed out;
- provider returned pending;
- expected webhook did not arrive;
- settlement is late;
- mandate state is stale;
- provider and internal states conflict.

### 23.7 No provider lock-in

Provider-specific fields must live in adapter metadata, not core tables or user copy.

Core language uses:

- mandate;
- authorisation session;
- collection;
- settlement;
- partial result;
- provider failure.

It must not use product names such as “Sweep” as the domain model.

---

## 24. Payment recording and reconciliation

### 24.1 Payment sources

- `INTEGRATED_VOLUNTARY`;
- `SUPPLIER_RECORDED_TRANSFER`;
- `BUYER_PAYMENT_CLAIM`;
- `CASH_RECORDED`;
- `KREDIT_COLLECTION`;
- `ADJUSTMENT`;
- `REVERSAL`.

### 24.2 Integrated voluntary payment

Preferred flow:

1. Kredit generates a reference or provider payment instruction.
2. Buyer pays.
3. Provider sends verified event.
4. Kredit records and allocates payment.
5. Outstanding amount reduces.
6. scheduled collection is reduced or suppressed.
7. both parties receive receipt.

### 24.3 Supplier-recorded payment

When a buyer pays directly to the supplier outside an integrated reference:

1. authorised supplier user records amount, date, method, and reference;
2. optional evidence is uploaded;
3. Kredit posts the payment immediately or marks it pending under the supplier's policy;
4. buyer is notified;
5. buyer may report an error;
6. reversal requires a controlled command and reason.

A supplier recording that it received money reduces the amount it may collect. This is safer than allowing a supplier to omit a known payment.

### 24.4 Buyer payment claim

A buyer may claim payment before the supplier confirms it.

The claim includes:

- amount;
- date;
- source account masked details where appropriate;
- transfer reference;
- evidence.

Policy may place a temporary collection hold on the claimed amount for a bounded period. Abuse controls must prevent last-minute unsupported claims from indefinitely blocking collection.

### 24.5 Allocation policy

Default allocation order:

1. oldest due undisputed principal;
2. currently due undisputed principal;
3. future principal in schedule order;
4. supplier-payable fees only when contract and policy allow.

The supplier and buyer must see the allocation result.

Do not automatically allocate supplier-paid Kredit fees to buyer principal.

### 24.6 Concurrency and double-payment prevention

A payment and collection may arrive concurrently.

Required pattern:

- lock obligation;
- identify active collection reservation;
- post payment once;
- update outstanding version;
- cancel or reduce unsubmitted collection;
- if provider collection is already processing, mark reconciliation risk;
- if overpayment occurs, create refund/credit case rather than negative outstanding.

### 24.7 Reversals

A reversal must reference the original payment and create inverse ledger postings.

Reasons include:

- provider reversal;
- bank return;
- duplicate payment correction;
- supplier recording error;
- dispute decision.

Reversal may make an obligation due again. Notify both parties and recalculate collection eligibility.

---

## 25. Collection engine

### 25.1 Eligibility calculation

An amount is eligible only when all conditions pass:

- obligation active;
- schedule item due;
- grace period expired;
- outstanding amount positive;
- mandate active;
- amount within mandate remaining ceiling;
- no blocking dispute for that amount;
- no pending buyer payment claim under hold;
- supplier collection enabled;
- provider supports requested account and amount;
- no active collection reservation;
- compliance/risk hold absent;
- feature flag enabled.

The engine returns both a boolean and explicit reason codes.

### 25.2 Collection reservation

Before calling a provider:

1. begin transaction;
2. lock obligation and relevant schedule rows;
3. rebuild or verify outstanding;
4. create reservation for the exact amount;
5. store outstanding version;
6. create attempt record;
7. enqueue provider-submission job transactionally;
8. commit.

The provider call happens outside the database transaction.

### 25.3 Submission idempotency

Provider request must use a stable external reference derived from the collection attempt.

If submission times out:

- do not assume failure;
- query provider using reference;
- wait for webhook or reconciliation;
- never submit a new attempt until the previous state is known or explicitly abandoned under policy.

### 25.4 Partial collection

When provider reports partial success:

- post only confirmed amount;
- allocate it;
- accrue collection fee on confirmed amount;
- release or update reservation;
- leave remaining amount open;
- schedule permitted retry;
- notify both parties with exact remaining balance.

### 25.5 Retry policy

Retry policy is provider- and agreement-aware.

Configuration may define:

- maximum attempts;
- minimum interval;
- business-day rules;
- partial retry behaviour;
- stop conditions;
- buyer reminder before retry;
- high-value manual review threshold.

Do not retry:

- cancelled mandate;
- final provider refusal;
- identity mismatch;
- unresolved blocking dispute;
- amount outside mandate;
- suspected duplicate;
- closed obligation.

### 25.6 Multi-account collection

Enable only after written provider/compliance approval.

The product must disclose the mandate scope clearly. The application must not market this as unrestricted access to “all accounts”. It is provider-managed collection across accounts that the customer has authorised under the supported flow.

### 25.7 Collection success

A provider “debit succeeded” event and a supplier “settlement received” event may be different states.

Track:

- debit succeeded;
- funds pending settlement;
- settled to supplier;
- reversed;
- settlement delayed.

Do not tell the supplier “money received” until the settlement state justifies that statement.

### 25.8 Collection cancellation

A pending unsubmitted collection must be cancelled when:

- outstanding becomes zero;
- dispute blocks entire amount;
- supplier cancels before submission under allowed policy;
- mandate becomes inactive;
- risk hold is applied.

If already submitted, follow provider capabilities. Never claim cancellation until provider confirms it.

---

## 26. Instalments and recurring trade lines

### 26.1 Instalment schedule generation

Support:

- fixed equal amounts;
- custom amount per date;
- weekly;
- fortnightly;
- monthly;
- explicit custom dates.

Validate:

- sum of instalment principal equals total principal;
- dates are ordered;
- all amounts positive;
- collection schedule remains within mandate duration and ceiling;
- timezone is explicit.

### 26.2 Monthly date edge cases

For schedules created on the 29th, 30th, or 31st, the supplier and buyer must choose or see a documented policy:

- last day of month;
- fixed day capped to month end;
- custom schedule.

Do not silently shift dates.

### 26.3 Trade-line exposure

```text
Current exposure
= sum of activated drawdown outstanding principal
```

```text
Available limit
= approved limit - current exposure - reserved pending drawdowns
```

### 26.4 Drawdown workflow

1. Supplier selects active trade line.
2. Enters amount and goods.
3. System locks line and checks available limit.
4. Creates pending drawdown reservation.
5. Buyer confirms the specific supply.
6. Supplier releases goods.
7. Buyer confirms receipt.
8. Drawdown activates and exposure increases.
9. Reservation converts to activated amount.

Pending reservations expire if not completed.

### 26.5 Trade-line suspension

Suspend automatically or manually when:

- mandate inactive;
- buyer overdue beyond policy;
- active high-risk dispute;
- verification expired;
- supplier chooses to pause;
- platform compliance hold;
- line expired.

Suspension blocks new drawdowns but does not erase existing obligations.

### 26.6 Limit changes

A limit increase is material and requires buyer acceptance if it alters mandate scope or agreement terms.

A supplier may reduce the unused available limit without increasing the buyer's obligation. Existing activated principal remains unchanged.

### 26.7 Rolling payment cadence

For “settle every Friday” relationships:

- drawdowns accumulate during the period;
- schedule cut-off policy determines which drawdowns are due;
- statement shows opening balance, new supplies, payments, and closing balance;
- collection targets only the amount due under accepted terms.

Do not assume every trade line is revolving credit under banking law. Final contract language requires review.

---

## 27. Disputes and complaints

### 27.1 Dispute reasons

- goods not received;
- wrong quantity;
- damaged goods;
- returned goods;
- incorrect amount;
- payment already made;
- unauthorised transaction;
- duplicate obligation;
- identity/business mismatch;
- other.

### 27.2 Opening a dispute

The buyer must specify:

- disputed amount;
- reason;
- explanation;
- supporting evidence where available.

The system records the undisputed amount separately.

### 27.3 Collection effect

The dispute policy returns:

- full block;
- block contested amount only;
- no automatic block but mandatory human review;
- fraud/security escalation.

This is configurable and legally reviewed. It is not a UI guess.

### 27.4 Evidence timeline

Reviewers see:

- all agreement versions;
- acceptance evidence;
- mandate state;
- invoice and document hashes;
- release confirmation;
- receipt confirmation;
- messages and reminders;
- payments;
- provider events;
- previous decisions.

### 27.5 Resolution outcomes

- supplier fully upheld;
- buyer fully upheld;
- partial adjustment;
- payment already made;
- goods return recognised;
- duplicate removed through reversal;
- insufficient evidence;
- referred outside platform.

Every outcome requires a reason and ledger effect where monetary values change.

### 27.6 Complaints

A complaint is not always a financial dispute. Support complaint categories include:

- service quality;
- privacy/data rights;
- unauthorised access;
- notification problem;
- payment delay;
- provider behaviour;
- staff conduct;
- accessibility.

Maintain service-level targets and escalation procedures.

### 27.7 No abusive collection

Kredit must not:

- shame buyers publicly;
- message unrelated contacts;
- threaten criminal consequences inaccurately;
- repeatedly harass users;
- expose debts in WhatsApp groups;
- use deceptive urgency.

Collections communication must be factual, proportionate, and reviewable.

---

## 28. Factual trade history and reputation

### 28.1 V1 principle

Kredit reports facts generated by its own workflows. It does not launch with a single opaque score.

### 28.2 Buyer-visible history

A buyer can see:

- active obligations;
- completed obligations;
- total completed principal;
- on-time performance;
- late payments;
- disputes;
- mandate events;
- corrections;
- which information is shareable.

### 28.3 Supplier-visible history

A supplier always sees its own relationship history.

Cross-supplier facts require:

- buyer consent;
- approved legal basis;
- privacy-preserving presentation;
- purpose limitation;
- access audit.

### 28.4 Possible consented summary

```text
Verified business: Yes
Completed Kredit obligations: 24
Total completed principal band: ₦10m–₦25m
Paid by due date: 22 of 24
Unresolved overdue obligations: 0
Mandate cancellations while owing: 0
Active exposure band: Moderate
```

Do not disclose another supplier's name, exact invoice, or exact exposure without appropriate permission.

### 28.5 Corrections and appeals

Buyers must be able to challenge inaccurate history.

Correction workflow:

1. user identifies fact;
2. Kredit locates source event;
3. evidence reviewed;
4. append correction/reversal if needed;
5. recalculate derived history;
6. notify affected user;
7. preserve audit trail.

### 28.6 Future score

A future model may estimate risk only after:

- sufficient representative data;
- fairness and bias review;
- explainability design;
- DPIA;
- human review and appeal path;
- legal approval;
- monitoring for drift.

No current code should present a placeholder random or heuristic number as a credit score.

---

## 29. WhatsApp integration

### 29.1 Role of WhatsApp

WhatsApp is a convenient command, invitation, reminder, and notification interface. It is not the ledger and not the secure location for sensitive verification.

### 29.2 Supplier commands

Examples:

```text
Create credit
Royal Pharmacy, ₦1.2m, due 30 September
How much am I owed?
Who is overdue?
What is due this week?
Check Royal Pharmacy
Remind overdue customers
Royal Pharmacy paid ₦500,000
Show TCR-7H4M-92QK
```

Natural-language input must resolve to structured fields. Before a financial action, show a confirmation summary.

### 29.3 Buyer messages

Invitation:

> ABC Pharmaceuticals is offering Royal Pharmacy ₦1,200,000 in trade credit for pharmaceutical supplies. Payment is due 30 September 2026. Review the exact terms securely.

Ready-to-release notice to supplier:

> Royal Pharmacy has verified its details, accepted the terms, and activated the required payment mandate. Review the status before releasing the goods.

Payment reminder:

> Your ₦300,000 instalment to ABC Pharmaceuticals is due on 30 September 2026.

Partial payment receipt:

> ₦500,000 received. Remaining principal: ₦700,000.

Collection notice:

> ₦700,000 remained unpaid after the agreed grace period. A collection request has been submitted under your authorised mandate. Reference: TCC-5G2J-13ZW.

### 29.4 WhatsApp safety

- never request BVN in chat;
- never request OTP, PIN, or online-banking password;
- mask account data;
- avoid sensitive detail in message previews;
- use secure expiring links;
- use approved message templates;
- honour opt-out rules where applicable;
- fall back to email/SMS for critical events;
- keep group-chat use out of v1.

### 29.5 Webhook handling

WhatsApp delivery/read events are normalised and stored. A message marked sent is not necessarily delivered.

### 29.6 Command permissions

A phone number associated with a user is insufficient for high-risk action. The bot may require:

- secure login link;
- OTP;
- step-up authentication;
- web confirmation.

Do not allow settlement-account changes, write-offs, or high-value collections entirely through unauthenticated chat.

---

## 30. Notification system

### 30.1 Channel-neutral events

Domain modules emit events such as:

- `CreditRequestSent`;
- `BuyerAccepted`;
- `MandateActivated`;
- `ReadyToRelease`;
- `GoodsReleased`;
- `ReceiptConfirmed`;
- `PaymentRecorded`;
- `PaymentReversed`;
- `PaymentDueSoon`;
- `GracePeriodEnding`;
- `CollectionSubmitted`;
- `CollectionSucceeded`;
- `CollectionPartiallySucceeded`;
- `CollectionFailed`;
- `MandateCancelled`;
- `DisputeOpened`;
- `DisputeResolved`;
- `ObligationClosed`.

Notification policy maps these events to channels and templates.

### 30.2 Template rules

Templates must contain:

- exact amount where relevant;
- exact date;
- reference;
- next action;
- support link;
- no unsupported promise.

### 30.3 Delivery priority

- Critical financial/security: WhatsApp plus email or SMS fallback.
- Routine reminders: preferred channel, fallback after failure.
- Marketing: separate consent and preference.

### 30.4 Deduplication

A notification event has a stable deduplication key. Retries must not send duplicate “payment succeeded” messages.

### 30.5 Quiet hours

Routine reminders respect configurable quiet hours in Africa/Lagos. Critical security or time-sensitive financial events may bypass quiet hours according to policy.

---

## 31. Background jobs and River

### 31.1 Why River

River uses PostgreSQL and supports transactional job insertion with `pgx`. This allows domain state and required asynchronous work to commit together.

### 31.2 Job categories

- invitation expiry;
- incomplete onboarding reminder;
- verification polling;
- mandate refresh;
- due reminder;
- grace-period reminder;
- collection eligibility;
- provider submission;
- collection polling;
- retry scheduling;
- settlement polling;
- payment reconciliation;
- daily ledger reconciliation;
- report generation;
- file scan;
- notification delivery;
- trade-history projection;
- retention/deletion workflow;
- support SLA escalation.

### 31.3 Transactional enqueue

When a command requires a job, insert it using the same `pgx.Tx` before commit.

Example:

```go
_, err := riverClient.InsertTx(ctx, tx, SendNotificationArgs{
    EventID: eventID,
}, &river.InsertOpts{Queue: "notifications"})
```

### 31.4 Job requirements

Every job type defines:

- stable kind;
- versioned arguments;
- uniqueness rule;
- queue;
- priority;
- retry policy;
- timeout;
- safe idempotency behaviour;
- observability fields;
- final failure action.

### 31.5 Queue separation

Suggested queues:

- `critical-financial`;
- `provider-webhooks`;
- `collections`;
- `reconciliation`;
- `notifications`;
- `documents`;
- `reports`;
- `maintenance`.

Critical financial jobs must not be starved by large report exports.

### 31.6 Scheduled work

Do not rely only on external cron for per-obligation schedules.

Use River scheduled jobs or a database scheduler service that scans indexed due times and transactionally enqueues unique work.

### 31.7 Dead-letter handling

Final job failures create:

- operational alert;
- support/incident case where relevant;
- visible admin status;
- safe manual retry action;
- audit event.

Do not silently discard a failed collection reconciliation job.

---

## 32. Documents and object storage

### 32.1 Upload flow

1. Authenticated client requests upload slot.
2. API validates purpose, type, and size.
3. API creates quarantined document row.
4. API returns short-lived presigned upload URL.
5. Client uploads directly to object storage.
6. storage event or callback enqueues scan.
7. worker verifies checksum, media type, and malware result.
8. document becomes available or rejected.

### 32.2 Security rules

- private bucket;
- server-side encryption;
- customer-managed key where available;
- no public ACL;
- short-lived signed downloads;
- per-resource authorisation on every download;
- malware scan;
- file-signature validation;
- size limits;
- content-disposition protection;
- no active HTML/SVG rendering from untrusted upload;
- checksum and audit event.

### 32.3 Allowed initial formats

- PDF;
- JPEG;
- PNG;
- optionally HEIC with safe conversion;
- CSV/XLSX only for controlled imports, not general evidence rendering.

### 32.4 Retention classes

- agreement evidence;
- KYC reference evidence;
- dispute evidence;
- temporary upload;
- report export;
- support attachment.

Each class has a retention policy and deletion/hold rules.

---

## 33. Admin, support, and compliance console

### 33.1 Search

Search by:

- public reference;
- masked phone;
- organisation;
- buyer business;
- provider reference;
- payment reference;
- collection reference;
- support case.

Search results must respect role and purpose.

### 33.2 Case view

A case view combines:

- identity summary;
- agreement timeline;
- mandate timeline;
- goods evidence;
- payment and ledger timeline;
- provider events;
- notifications;
- disputes;
- support notes;
- audit events.

### 33.3 Manual actions

Manual actions must be commands with validation, not direct database edits.

Examples:

- retry reconciliation;
- resend invitation;
- waive fee;
- approve adjustment;
- suspend user;
- restore access;
- close support case;
- place risk hold.

Each action requires:

- permission;
- reason code;
- free-text note where material;
- step-up authentication where required;
- audit event.

### 33.4 Impersonation

Routine impersonation is prohibited.

If a controlled support-assist mode is implemented:

- it is read-only by default;
- user sees an indicator;
- session is short;
- every action is audited;
- financial actions remain blocked.

### 33.5 Provider operations

Admin dashboard must show:

- webhook backlog;
- provider latency;
- error rate;
- pending collections;
- unknown submissions;
- settlement delays;
- mandate-state drift;
- reconciliation mismatches.

---

## 34. Risk and fraud controls

### 34.1 Threats

- fake supplier and fake buyer fabricate history;
- circular trades inflate reputation;
- invoice duplication;
- inflated invoice value;
- stolen identity;
- representative lacks authority;
- mule or mismatched bank account;
- supplier receives payment but attempts collection;
- buyer falsely claims payment;
- employee creates unauthorised credit;
- buyer cancels mandate after receiving goods;
- rapid exposure growth across suppliers;
- provider webhook forgery;
- account takeover;
- document malware;
- support-agent abuse.

### 34.2 V1 controls

- KYC/KYB;
- authority verification;
- account ownership result;
- explicit acceptance;
- goods release and receipt evidence;
- invoice hash and duplicate checks;
- device/IP velocity signals;
- supplier team permissions;
- step-up authentication;
- cross-transaction anomaly flags;
- provider signature verification;
- immutable ledger and audit;
- manual-review queues;
- per-supplier exposure policies;
- feature flags and limits.

### 34.3 Risk facts, not fake certainty

V1 may show facts such as:

- business newly verified;
- first transaction;
- exposure increased rapidly;
- active overdue amount;
- mandate cancelled while owing;
- multiple disputes;
- account ownership mismatch;
- document duplication.

Avoid labels such as “fraudster” without a formal, reviewed basis.

### 34.4 Velocity rules

Configurable examples:

- maximum invitation attempts per buyer/day;
- maximum new principal per new supplier/day;
- maximum drawdown increase percentage;
- maximum OTP attempts;
- maximum settlement-account changes;
- maximum failed collections before review;
- maximum identical invoice hash across parties.

### 34.5 High-value controls

Above configurable thresholds:

- enhanced verification;
- dual supplier approval;
- mandatory invoice and delivery note;
- mandatory AAL2;
- manual compliance review;
- collection retry review.

Thresholds must be configuration and reviewed policy, not code constants.

---

## 35. Security architecture

### 35.1 Standard

Target OWASP ASVS 5.0 Level 2 across the application, with selected Level 3 controls for authentication, sensitive data, financial operations, administrative access, and cryptographic key management.

### 35.2 Data classification

#### Restricted

- raw identity identifiers;
- bank-account tokens and sensitive account metadata;
- identity documents;
- authentication secrets;
- provider secrets;
- encryption keys;
- private agreement evidence.

#### Confidential

- exact obligations;
- repayment history;
- supplier-customer relationships;
- settlement details;
- disputes;
- internal risk flags.

#### Internal

- operational metrics;
- non-sensitive configuration;
- staff procedures.

#### Public

- marketing content;
- public legal documents;
- privacy-safe status pages.

### 35.3 Encryption

- TLS 1.2 minimum, prefer TLS 1.3;
- managed disk/database encryption;
- S3 server-side encryption;
- application field-level envelope encryption for selected restricted fields;
- managed KMS/HSM-backed keys;
- key rotation;
- no keys in source control;
- separate keys per environment.

### 35.4 Secret management

Use managed secret storage. Access via workload identity where possible.

Rotate:

- provider API keys;
- webhook secrets;
- session signing keys;
- encryption keys according to policy;
- database credentials.

Every secret has an owner, purpose, environment, and rotation date.

### 35.5 Web security headers

Use:

- Content Security Policy;
- HSTS;
- `X-Content-Type-Options: nosniff`;
- frame restrictions;
- Referrer Policy;
- Permissions Policy;
- cross-origin protections where compatible.

### 35.6 Input and output controls

- schema validation;
- parameterised SQL only;
- no dynamic SQL from user input;
- contextual escaping;
- safe file rendering;
- request body limits;
- strict media types;
- normalised phone and email;
- canonical amount parsing.

### 35.7 Dependency security

CI must run:

- `govulncheck`;
- `gosec`;
- `staticcheck`;
- `go test -race`;
- frontend lockfile audit;
- OSV scanner;
- container scanner;
- secret scanner;
- CodeQL or equivalent static analysis.

### 35.8 Audit integrity

Audit events are append-only. Restrict update/delete at database permission level.

For higher assurance, periodically hash audit batches and store the digest in a separate protected location.

### 35.9 Backups

- encrypted backups;
- point-in-time recovery;
- periodic restore test;
- restricted restore access;
- documented RPO/RTO;
- backup deletion aligned with retention.

### 35.10 Security testing

Before broad launch:

- threat model review;
- ASVS control review;
- penetration test;
- provider webhook tests;
- tenant isolation tests;
- authorisation tests;
- incident simulation;
- backup restore exercise.

---

## 36. Privacy and compliance engineering

### 36.1 Governing posture

Kredit must be designed under the Nigeria Data Protection Act and current NDPC guidance. Legal and data-protection professionals must review the final implementation and documents.

### 36.2 Privacy by design

- collect only necessary data;
- state purposes clearly;
- separate operational consent from marketing consent;
- minimise raw identifiers;
- define retention;
- log access;
- support correction and access rights;
- assess profiling before launch;
- complete a DPIA before real financial/identity processing;
- maintain processor and subprocessor records.

### 36.3 Data map

`docs/data-map.md` must identify for every field:

- subject;
- purpose;
- legal basis to be confirmed;
- source;
- processor/provider;
- storage location;
- encryption;
- role access;
- retention;
- deletion/hold rule;
- cross-border transfer consideration.

### 36.4 Data subject requests

Build operational support for:

- access;
- correction;
- deletion where legally possible;
- restriction;
- objection;
- consent withdrawal;
- portability where applicable;
- complaint escalation.

Financial and agreement records may need to be retained; deletion requests may result in restriction, pseudonymisation, or legally required retention rather than destruction.

### 36.5 Profiling

Factual trade-history calculation is still consequential data processing. Document:

- inputs;
- derivations;
- sharing;
- explanations;
- correction route;
- human review.

Do not add automated adverse scoring without a separate approved design.

### 36.6 Payment regulation

Kredit must use appropriately licensed providers for regulated payment functions. The exact supplier-funded trade-credit and multi-account collection use case must receive written provider/compliance approval before real collection is enabled.

### 36.7 Contract and legal review gates

Before live money:

- supplier terms;
- buyer agreement;
- mandate disclosure;
- privacy notice;
- fee disclosure;
- complaints policy;
- dispute role;
- data-processing agreements;
- provider contracts;
- marketing claims;
- record-retention policy.

---

## 37. Observability and service objectives

### 37.1 OpenTelemetry

Instrument API and worker with OpenTelemetry traces and metrics. Export through OTLP so the observability backend remains replaceable.

### 37.2 Required traces

- request lifecycle;
- credit creation;
- buyer acceptance;
- obligation activation;
- payment posting;
- collection submission;
- provider webhook processing;
- settlement reconciliation;
- dispute resolution;
- notification delivery.

Never attach restricted values to trace attributes.

### 37.3 Metrics

#### Technical

- request rate/latency/error;
- database pool saturation;
- transaction retries;
- River queue depth and oldest age;
- job success/failure;
- webhook verification failures;
- provider latency and status;
- reconciliation mismatches;
- file scan failures;
- notification delivery.

#### Financial correctness

- duplicate collection prevented;
- collection amount mismatch;
- negative-balance invariant attempt;
- ledger imbalance;
- overpayment case;
- settlement discrepancy;
- stale mandate state.

#### Product

- request-to-view conversion;
- view-to-accept conversion;
- verification completion;
- mandate activation;
- time to ready-to-release;
- voluntary repayment rate;
- collection success rate;
- dispute rate;
- repeat supplier/buyer rate.

### 37.4 Initial service objectives

Pilot targets:

- monthly API availability: 99.9% excluding announced maintenance;
- p95 normal read API latency: under 500 ms excluding provider time;
- p95 normal write API latency: under 900 ms excluding provider time;
- signed webhook persisted: under 5 seconds p95;
- financial webhook applied or queued: under 60 seconds p95;
- critical notification submitted: under 2 minutes p95;
- duplicate debit caused by Kredit: zero;
- unreconciled ledger imbalance: zero.

### 37.5 Alerting

Page immediately for:

- possible duplicate debit;
- ledger imbalance;
- provider event signature anomaly;
- database unavailable;
- collection queue stalled;
- settlement mismatch above threshold;
- unauthorised admin access;
- encryption or secret failure.

Create ticket, not page, for lower-severity trends.

---

## 38. Deployment and infrastructure

### 38.1 Platform-neutral containers

The Go API and worker must build as OCI images with:

- multi-stage build;
- non-root runtime user;
- minimal/distroless or scratch-compatible image;
- read-only filesystem where practical;
- health endpoints;
- SBOM;
- image signing;
- no build secrets in layers.

### 38.2 Web deployment

SvelteKit may deploy to Vercel using its official adapter or another approved platform. Pin the adapter explicitly rather than relying indefinitely on auto-detection.

The web deployment must support:

- SSR;
- same-origin `/api` proxy;
- security headers;
- preview environments;
- environment separation;
- no caching of authenticated responses.

### 38.3 API and worker deployment

Requirements:

- always-available API process;
- independently scalable worker;
- private database networking where possible;
- graceful rolling deploy;
- controlled concurrency;
- outbound provider egress;
- central secrets;
- autoscaling that does not create unbounded provider calls.

### 38.4 PostgreSQL hosting

Managed PostgreSQL must provide:

- PostgreSQL 18 support;
- encryption;
- automated backups;
- point-in-time recovery;
- high availability appropriate to stage;
- direct connection for River coordinator `LISTEN/NOTIFY`;
- monitoring;
- maintenance controls;
- connection limits.

If PgBouncer is used, River worker coordinator requires session-compatible/direct connectivity. Do not route all River coordination through statement-pooling mode.

### 38.5 Object storage

Use AWS S3, Cloudflare R2, or another approved S3-compatible private store with encryption, lifecycle rules, and signed URLs.

### 38.6 Environment separation

- separate accounts/projects where practical;
- separate databases;
- separate buckets;
- separate provider credentials;
- separate encryption keys;
- no production data copied to development;
- masked or synthetic staging data.

### 38.7 Infrastructure as code

Use Terraform or OpenTofu for production infrastructure.

Changes require:

- plan review;
- protected state;
- least-privilege credentials;
- drift detection;
- documented rollback.

### 38.8 Release strategy

- build immutable artifacts;
- apply backward-compatible migration;
- deploy API and worker;
- verify health and metrics;
- enable feature flags gradually;
- run smoke tests;
- observe provider flows;
- remove old schema only in later release.

Do not run destructive migration automatically during app startup.

---

## 39. CI/CD

### 39.1 Pull-request checks

Every PR must run:

- formatting;
- Go compile;
- Go vet/staticcheck;
- golangci-lint;
- govulncheck;
- gosec;
- Go unit tests;
- race detector for relevant packages;
- frontend lint;
- `svelte-check`;
- frontend unit tests;
- OpenAPI lint;
- generated-code diff check;
- sqlc generation and vet;
- migration apply on fresh DB;
- migration compatibility test;
- integration tests;
- container build;
- secret scan;
- dependency scan.

### 39.2 Main-branch checks

Additionally:

- full end-to-end test;
- Playwright browser matrix;
- provider simulator tests;
- performance smoke test;
- image scan and signing;
- staging deploy;
- staging smoke tests.

### 39.3 Production deployment gates

- reviewed change;
- green CI;
- migration reviewed;
- release notes;
- readiness checklist;
- no unresolved critical/high security issue;
- provider sandbox test for affected flows;
- rollback/forward-fix plan;
- human approval.

### 39.4 Branch protection

- no direct production branch push;
- required reviews;
- CODEOWNERS for ledger, auth, provider, database, and security code;
- signed commits/tags where practical;
- protected environment approval.

---

## 40. Testing strategy

### 40.1 Unit tests

Required domains:

- money arithmetic;
- fee calculation;
- schedule generation;
- due/grace calculations;
- payment allocation;
- state transitions;
- trade-line availability;
- collection eligibility;
- mandate ceilings;
- dispute blocking;
- permissions;
- history derivation.

### 40.2 Invariant and property tests

- outstanding never negative;
- ledger balances;
- allocations never exceed payment;
- schedule principal sums correctly;
- collection never exceeds outstanding;
- collection fee equals 0.5% of confirmed collected amount;
- idempotent duplicate event has no second effect;
- trade-line available amount never negative;
- reversal restores expected balance;
- random event sequences preserve invariants.

### 40.3 Go fuzzing

Fuzz:

- amount parsing;
- reference parsing;
- webhook parsers after signature layer;
- schedule generation;
- allocation;
- problem-details mapping;
- CSV import parser;
- phone normalisation.

### 40.4 Integration tests

Use a real ephemeral PostgreSQL instance.

Test:

- SQL constraints;
- transactions;
- locks;
- RLS;
- River job insertion;
- sqlc queries;
- migrations;
- provider event deduplication;
- object storage;
- OpenTelemetry propagation where practical.

### 40.5 Provider contract tests

Every provider adapter has a common contract suite:

- authorization session;
- mandate state normalisation;
- collection success;
- pending then webhook success;
- partial success;
- retryable failure;
- final failure;
- duplicate webhook;
- invalid signature;
- out-of-order events;
- settlement;
- reversal.

### 40.6 End-to-end scenarios

Minimum:

1. one-time credit paid fully before due date;
2. partial early payment plus remaining collection;
3. full collection at maturity;
4. four instalments with one late instalment;
5. recurring trade line with three drawdowns;
6. mandate cancelled while amount remains;
7. buyer payment claim before collection;
8. provider timeout followed by success webhook;
9. duplicate success webhook;
10. partial collection and retry;
11. partial dispute with undisputed amount payable;
12. payment reversal reopens balance;
13. supplier role violation;
14. settlement account change with step-up;
15. tenant isolation attempt;
16. expired invitation;
17. malware upload rejection;
18. buyer history correction;
19. worker restart during collection processing;
20. database failover/reconnect simulation.

### 40.7 Performance tests

Test:

- 100k obligations;
- due-date scan;
- 10k webhook burst;
- report generation;
- supplier portfolio query;
- concurrent payment and collection;
- trade-line drawdown contention;
- River queue recovery.

### 40.8 Acceptance tests

A feature is incomplete unless tests cover:

- happy path;
- validation failure;
- authorisation failure;
- duplicate request;
- concurrent request;
- provider timeout;
- provider duplicate event;
- notification failure;
- audit event;
- rollback/no partial state.

---

## 41. Operational runbooks

Required runbooks:

- suspected duplicate debit;
- buyer says already paid;
- supplier says settlement missing;
- provider webhook outage;
- provider API outage;
- mandate state mismatch;
- collection stuck pending;
- ledger reconciliation mismatch;
- database unavailable;
- queue backlog;
- object-storage malware event;
- compromised supplier account;
- settlement destination changed fraudulently;
- data breach;
- secret leak;
- bad deployment rollback;
- backup restore;
- privacy request;
- high-severity dispute.

Each runbook includes:

- trigger;
- severity;
- immediate containment;
- evidence preservation;
- customer communication;
- provider escalation;
- resolution;
- reconciliation;
- post-incident review.

---

## 42. Product analytics

### 42.1 North-star and financial metrics

- Gross Trade Credit Volume activated;
- active supplier organisations;
- active buyer businesses;
- active principal;
- voluntary repayment amount;
- Kredit-collected amount;
- on-time repayment rate;
- recovery rate after maturity;
- dispute rate;
- supplier repeat rate;
- buyer repeat rate;
- platform revenue;
- provider cost;
- contribution margin.

### 42.2 Funnel

- request created;
- invitation delivered;
- invitation viewed;
- buyer phone verified;
- KYC/KYB completed;
- mandate active;
- terms accepted;
- goods released;
- receipt confirmed;
- first payment;
- obligation closed.

### 42.3 Privacy rules

- analytics IDs are not raw phone/BVN;
- financial values are access-controlled;
- avoid sending sensitive transaction details to generic analytics vendors;
- prefer first-party or privacy-respecting analytics;
- document every analytics event and purpose;
- respect consent for non-essential tracking.

---

## 43. Design and content specification

### 43.1 Visual direction

Use Mercury as inspiration for calm premium B2B finance, Stripe for progressive explanation of infrastructure, and TreviPay for the depth expected in B2B trade-credit products. Do not clone their layouts.

Kredit should feel:

- serious enough for ₦50m transactions;
- simple enough for a trader using an Android phone;
- warm rather than cold-bank blue;
- precise rather than flashy;
- modern without looking like a generic dark shadcn dashboard.

### 43.2 Homepage

Navigation:

```text
Kredit | How it works | For businesses | Pricing | Security | Sign in | Get started
```

Hero:

> **Give goods on credit. Get paid with confidence.**
>
> Kredit turns the credit you already give your business customers into verified, trackable payment commitments.

Primary CTA:

> Create your first Kredit

Hero visual should demonstrate:

```text
₦1,200,000 credit request
→ buyer reviewed
→ identity verified
→ terms accepted
→ mandate active
→ goods released
→ paid
```

### 43.3 Pricing section

Display plainly:

> **0.5% when your customer pays before collection.**
>
> **1% on the amount Kredit has to collect.**

Include accurate examples and clarify that the supplier provides the goods and carries the underlying trade credit unless a future financed product is explicitly selected.

### 43.4 Product copy rules

Prefer:

- “Payment due 30 September 2026.”
- “₦700,000 remains outstanding.”
- “Mandate inactive. New credit is paused.”
- “Collection submitted. Settlement is pending.”

Avoid:

- “Your financial journey.”
- “Unlock limitless possibilities.”
- “Guaranteed repayment.”
- “We have debited all BVN accounts.”
- “AI-powered credit revolution.”

### 43.5 Confirmation screens

Every financially material confirmation displays:

- party names;
- principal or payment amount;
- due date;
- collection date;
- outstanding after action;
- fee effect;
- reference;
- explicit button label.

Buttons must say what happens, such as:

- `Accept ₦1,200,000 credit`;
- `Record ₦500,000 payment`;
- `Submit ₦700,000 collection`;

not generic `Continue` for the final step.

---

## 44. Build plan

The implementation is complete only when all production-v1 requirements pass, but development proceeds as vertical slices.

### Milestone 0 — Repository and guardrails

- repository structure;
- CI;
- toolchain pins;
- Postgres/MinIO local stack;
- OpenAPI generation;
- sqlc generation;
- migration runner;
- logging and telemetry skeleton;
- docs structure;
- implementation status tracker.

**Exit:** one command starts the system; CI prevents generated-code drift.

### Milestone 1 — Authentication and organisations

- OTP login;
- sessions;
- supplier onboarding;
- organisations;
- memberships;
- RBAC;
- MFA;
- audit events;
- RLS baseline.

**Exit:** supplier owner can create an organisation and invite a sales user; tenant isolation tests pass.

### Milestone 2 — Buyer and identity skeleton

- buyer invitation;
- person/business entities;
- verification provider abstraction and mock;
- authority records;
- secure public token;
- buyer portal account.

**Exit:** invited buyer completes mock verification securely.

### Milestone 3 — One-time credit vertical slice

- draft;
- immutable agreement version;
- invitation;
- buyer review/accept;
- mock mandate;
- ready-to-release;
- release and receipt;
- obligation activation;
- base fee ledger posting.

**Exit:** complete request-to-active flow with audit and PDF summary.

### Milestone 4 — Ledger and voluntary payments

- balanced ledger;
- payment recording;
- allocations;
- partial payment;
- reversal;
- statements;
- receipts;
- fee engine.

**Exit:** outstanding rebuild matches cached balance across tests.

### Milestone 5 — Instalments

- schedule generation;
- due/grace calculations;
- early allocation;
- separate schedule states;
- reminders.

**Exit:** six-instalment demo works end to end.

### Milestone 6 — Trade lines

- limit;
- drawdown reservation;
- buyer confirmation;
- exposure;
- suspension;
- recurring statement.

**Exit:** concurrent drawdowns cannot exceed line limit.

### Milestone 7 — Collection engine with mock provider

- eligibility;
- reservation;
- provider submission;
- webhooks;
- partial result;
- retries;
- reconciliation;
- collection fee.

**Exit:** provider simulator passes contract suite, including timeout and duplicate webhook.

### Milestone 8 — Disputes and operational controls

- disputes;
- evidence;
- partial block;
- review;
- adjustments;
- support console;
- manual operations.

**Exit:** dispute demo produces correct ledger and collection result.

### Milestone 9 — WhatsApp and notifications

- template management;
- invitation;
- reminders;
- status messages;
- command parsing;
- secure confirmation links;
- fallbacks.

**Exit:** supplier creates a structured request from WhatsApp and buyer completes web flow.

### Milestone 10 — Reporting and history

- dashboards;
- ageing;
- exports;
- buyer factual history;
- correction workflow;
- analytics.

**Exit:** reports reconcile to ledger.

### Milestone 11 — Real provider adapter

- written approval;
- adapter;
- sandbox tests;
- mandate flow;
- one-time/recurring/variable collection as approved;
- settlement;
- operations runbook;
- feature flags.

**Exit:** approved sandbox scenarios pass; no production flag yet.

### Milestone 12 — Production hardening and pilot

- security review;
- DPIA and legal review;
- penetration test;
- backup restore;
- load test;
- provider certification;
- support training;
- pilot limits;
- production readiness review.

**Exit:** signed launch checklist and limited pilot enablement.

---

## 45. Definition of done

Production v1 is not done until:

- supplier onboarding and team roles work;
- buyer verification and authority are represented correctly;
- one-time, instalment, and trade-line flows work;
- accepted terms are immutable;
- mandate state is provider-neutral;
- goods release and receipt are evidenced;
- money uses integer kobo;
- ledger is balanced and rebuildable;
- full and partial early payments work;
- collection is idempotent;
- duplicate webhooks do not duplicate money;
- partial collections and retries work;
- mandate cancellation alerts and blocks new credit;
- disputes affect only approved amounts;
- fees calculate correctly;
- reports reconcile;
- WhatsApp and fallback notifications work;
- mobile UX is polished;
- WCAG 2.2 AA checks pass for critical flows;
- tenant isolation tests pass;
- restricted data is absent from logs;
- backups and restore are tested;
- required runbooks exist;
- legal and provider approvals exist for enabled capabilities;
- no unresolved critical/high security defect remains;
- pilot limits and feature flags are configured.

---

## 46. AI coding-agent operating rules

The coding agent must follow these rules throughout the project.

### 46.1 Before coding

1. Read this README.
2. Read relevant ADRs.
3. Check `IMPLEMENTATION_STATUS.md`.
4. Inspect existing code before introducing a new pattern.
5. Identify domain rules, permissions, financial effects, audit events, notifications, and tests.
6. Update OpenAPI first when changing the API.

### 46.2 During implementation

- Implement one coherent vertical slice at a time.
- Keep domain logic out of handlers and Svelte components.
- Use database constraints in addition to Go validation.
- Use transactions for material commands.
- Use idempotency for financial writes.
- Use row locks for balance-sensitive operations.
- Post money only through the ledger module.
- Enqueue jobs transactionally.
- Add audit events.
- Add permission tests.
- Add failure-path tests.
- Update documentation and implementation status.

### 46.3 Prohibited shortcuts

The agent must not:

- use floats for money;
- use an ORM for core financial tables;
- edit generated code;
- directly update statuses from handlers;
- directly modify ledger rows in admin tools;
- store raw tokens;
- log restricted data;
- disable CSRF to fix a request;
- remove idempotency to simplify a test;
- treat provider timeout as failure;
- claim money settled from a debit-only event;
- add a fake credit score;
- create a wallet;
- hard-code Mono or another provider into the domain;
- leave TODO placeholders in a financially material path;
- silently weaken a requirement in this README.

### 46.4 Every feature deliverable

For each feature, provide:

- schema/migration;
- domain types;
- domain rules;
- repository queries;
- command/query service;
- API contract;
- handler;
- permission rules;
- audit events;
- jobs;
- notifications;
- frontend flow;
- unit tests;
- integration tests;
- end-to-end tests where relevant;
- documentation.

### 46.5 Unavailable external dependency

When provider approval, credentials, legal wording, or a human decision is missing:

- implement an interface and deterministic mock;
- place real capability behind a disabled feature flag;
- document the open question;
- do not invent provider behaviour;
- do not block unrelated core work.

### 46.6 Completion report

After each milestone, update `IMPLEMENTATION_STATUS.md` with:

- completed work;
- tests run;
- known limitations;
- open questions;
- migration notes;
- next milestone.

---

## 47. Launch limits

The first pilot must use configurable conservative limits, including:

- maximum supplier organisations;
- maximum buyer businesses;
- maximum principal per obligation;
- maximum active exposure per buyer;
- maximum drawdowns per line/day;
- maximum collection retries;
- permitted provider and account types;
- enhanced-review threshold;
- industries allowed in pilot.

Limits must be changeable through controlled configuration, not code deployment.

---

## 48. Demo and acceptance dataset

### Supplier

**ABC Pharmaceuticals Ltd**

### Buyer

**Royal Pharmacy Ltd**

### Scenario A — one-time credit

- principal: ₦1,200,000;
- due: 30 September 2026;
- grace: 48 hours;
- voluntary payment: ₦500,000;
- Kredit collection: ₦700,000;
- base fee: ₦6,000;
- collection fee: ₦3,500;
- total fee: ₦9,500.

### Scenario B — instalments

- principal: ₦3,000,000;
- six monthly instalments of ₦500,000;
- instalment 2 paid early;
- instalment 4 partially collected;
- final statement balances.

### Scenario C — recurring trade line

- limit: ₦5,000,000;
- settlement cadence: Friday;
- drawdowns: ₦1.2m, ₦900k, ₦650k;
- voluntary payment: ₦1m;
- available limit recalculated;
- new drawdown blocked above limit.

### Scenario D — mandate cancellation

- active outstanding: ₦1,800,000;
- provider cancellation event;
- supplier alerted;
- new drawdown blocked;
- obligation preserved;
- trade history event recorded.

### Scenario E — partial dispute

- outstanding: ₦1,000,000;
- disputed: ₦200,000;
- undisputed: ₦800,000;
- policy allows collection of undisputed amount;
- decision reduces disputed amount by ₦100,000;
- final ledger correct.

### Scenario F — duplicate provider webhook

- collection succeeds once;
- same provider event delivered three times;
- exactly one ledger transaction;
- exactly one collection fee;
- one customer receipt.

---

## 49. Open questions that must remain explicit

Track answers in `docs/product/open-questions.md`.

1. Which collection provider formally approves supplier-funded B2B trade credit?
2. Which mandate structures support one-time, variable, recurring, and instalment collection?
3. Is multi-account BVN-linked collection approved for this use case?
4. Is partial recovery across authorised accounts available?
5. What happens technically and contractually when a mandate is cancelled while money remains due?
6. Can mandate authorisation occur before goods are released?
7. Can collected money settle directly to the supplier's existing account?
8. Can Kredit fees be split or deducted compliantly?
9. Which personal and business account types are supported?
10. What amount bands trigger enhanced KYC/KYB?
11. What provider webhook guarantees and reconciliation APIs exist?
12. What reversal and dispute rules apply to account debit?
13. What licence or regulated-partner structure applies to Kredit's role?
14. When exactly does activated principal legally arise?
15. What evidence is required for business authority?
16. What retention periods apply to agreement and payment records?
17. What wording may Kredit use about trade history and risk?
18. How should VAT and other taxes apply to Kredit fees?
19. Which initial industry produces the best pilot economics and lowest dispute complexity?
20. What pilot amount limits will providers approve?

---

## 50. Source and standards baseline

This specification relies on current primary documentation available on 16 August 2026:

- Go 1.26 and security point releases: <https://go.dev/doc/devel/release>
- Go standard HTTP routing: <https://go.dev/blog/routing-enhancements>
- Go vulnerability management: <https://go.dev/doc/security/vuln/>
- SvelteKit: <https://svelte.dev/docs/kit>
- Svelte 5: <https://svelte.dev/docs/svelte>
- SvelteKit deployment adapters: <https://svelte.dev/docs/kit/adapters>
- PostgreSQL 18: <https://www.postgresql.org/docs/current/>
- pgx/v5: <https://pkg.go.dev/github.com/jackc/pgx/v5>
- sqlc 1.31: <https://docs.sqlc.dev/>
- Goose migrations: <https://pressly.github.io/goose/>
- River jobs: <https://riverqueue.com/docs>
- Tailwind CSS 4: <https://tailwindcss.com/docs>
- shadcn-svelte: <https://www.shadcn-svelte.com/docs>
- OpenAPI TypeScript/openapi-fetch: <https://openapi-ts.dev/>
- oapi-codegen: <https://github.com/oapi-codegen/oapi-codegen>
- OpenTelemetry Go: <https://opentelemetry.io/docs/languages/go/>
- OWASP ASVS 5.0: <https://owasp.org/www-project-application-security-verification-standard/>
- Nigeria Data Protection Commission: <https://ndpc.gov.ng/>
- NIBSS Direct Debit: <https://nibss-plc.com.ng/nibss-direct-debit-ndd/>
- CBN payment-service-provider register: <https://www.cbn.gov.ng/PaymentsSystem/PSPs.html>

These references inform engineering choices but do not replace provider contracts, legal advice, regulatory approvals, or a formal security assessment.

---

## 51. Final product standard

A wholesaler should be able to explain Kredit in one sentence:

> **“If you want goods on credit, complete this Kredit link first.”**

The buyer should see a clear, fair commercial agreement rather than an intimidating banking product.

The supplier should always know:

- what was agreed;
- whether the required mandate is active;
- whether goods may be released;
- what has been paid;
- what remains outstanding;
- what is due next;
- whether collection is pending, successful, or failed;
- whether a dispute or mandate event changes the risk.

The system underneath that simple interaction must withstand duplicate webhooks, provider outages, partial payments, concurrent events, staff mistakes, account compromise, disputes, and financial reconciliation.

Kredit's first release succeeds when it is both:

- **simple enough to become a habit**, and
- **precise enough to be trusted with real trade credit**.
