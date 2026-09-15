# Field agent referral programme

Kredit supports commission-based introductions by current or former DSAs and other authorised field representatives. Agents sign in at `/agents`; Super Admin operates `/admin/agents`.

## Default rewards

- ₦1,000 once for verified CAC onboarding, an authorised owner confirmation, verified business contact and receiving bank.
- ₦2,000 once after an accepted sale and at least ₦5,000 in eligible Kredit fees received.
- 10% of additional eligible fees for six calendar months after activation. The fees already counted at activation are excluded from the share.
- Seven full days of qualification hold, then the following Monday in Africa/Lagos. Payout details must also remain unchanged for seven days.
- Initial limit of 20 onboarding rewards per agent. Additional qualified businesses wait for a limit increase; their activation and revenue-share eligibility remains independent.

Super Admin can change future reward settings and pause new enrolments/referrals. Existing referral terms are saved permanently with the owner's confirmation. No salary, guaranteed income or recruitment/downline commission is offered.

## Owner-confirmed attribution

Each agent receives a random code and a `https://kredit.ng/join?ref=...` link. The merchant signs in personally, creates a business if needed, and confirms the introduction at `/app/referral`. A browser-local pending code helps the merchant return after business creation; it does not attribute anything without confirmation. A code can also be entered manually.

Only the current business owner can confirm attribution, within 30 days of the business joining Kredit and before its first accepted sale. One agent per organization; one CAC identity can qualify only once. Businesses the agent belongs to cannot qualify. Neither acceptance nor a merchant-confirmed payment earns the activation bounty by itself.

## Qualification and accounting

The background worker refreshes referrals without dashboard visits. A provider-confirmed CAC result, current approved business verification, contact verification and bank setup are required for onboarding. Native Mono and other providers use the stored `cac_status=verified` evidence contract; a generic approved business status alone does not count as CAC verification.

Eligible revenue includes only the existing `base_service` and `collection` fee lines. Customer principal, taxes and provider pass-through charges are excluded. Invoice bank receipts count only up to supported fee balances. Provider collections count only after separately reconciled bank receipts, capped by supported allocations. Provider-held fees alone do not count. Current waivers, refunds, reversals and disputed debit evidence reduce eligibility.

A referral freezes its fee total at activation as the revenue-share baseline. After expiry, new receipts are excluded while later refunds and reversals can still reduce rewards. Provider-level bank evidence is conservatively capped against supported allocations; it does not invent a fee-level bank settlement link.

Reward changes are immutable signed entries, posted to separate DSA expense/payable accounts. Negative adjustments immediately reduce the agent's available balance. Repeated or concurrent refreshes do not duplicate rewards. Prior completed payouts remain in history when a later reversal creates a negative balance.

## Payout operations

Preparing a payout refreshes that agent's fee evidence, checks account status and bank holds, then reserves the matured unpaid balance. One pending payout per agent prevents duplicate reservations. Its bank destination is frozen; bank edits are blocked while a payout is pending.

Super Admin checks the receiving name in the bank, completes the transfer separately, then records its unique reference and evidence. The platform records the completed transfer; it does not call a bank-transfer API. Cancel an unsent reservation when no transfer was made. Check current balances again before sending if time has passed since preparation.

## Permissions and operation

Agents see their own profile, referral progress, reward history and payouts. They do not receive access to merchant customer records or verification documents. All writes require CSRF, idempotency protection and recent MFA; financial administration also requires current Super Admin authority. Personal privacy exports include agent records. Financial history follows the existing retention and legal-hold process.

The programme terms prohibit fabricated transactions, self-referrals, sharing merchant sign-in codes, handling customer money, importing confidential bank customer lists and implying bank endorsement. Currently employed agents attest that participation is permitted.

Deploy migration **153**, the updated PostgreSQL role template, API and worker together. No new payment or messaging provider is required. The worker's existing recurring collection-work discovery runs reward refreshes.
