# Active page inventory

Generated from the final page tree. This inventories routes; it does not claim every retained transaction screen or operations workflow was rewritten. All public and signed-in entry points use the shared network/account structure described in `current-user-flow.md`.

| Page | Area |
|---|---|
| `/` | Public |
| `/account` | Shared account |
| `/account/display` | Shared account |
| `/account/messages` | Shared account |
| `/account/notifications` | Shared account |
| `/account/privacy` | Shared account |
| `/account/security` | Shared account |
| `/account/suppliers` | Shared account |
| `/admin` | Operations |
| `/admin/agents` | Operations |
| `/admin/analytics` | Operations |
| `/admin/approvals` | Operations |
| `/admin/attention` | Operations |
| `/admin/audit` | Operations |
| `/admin/billing` | Operations |
| `/admin/cases` | Operations |
| `/admin/cases/[id]` | Operations |
| `/admin/consumer-sales` | Operations |
| `/admin/consumer-sales/[saleID]` | Operations |
| `/admin/controls` | Operations |
| `/admin/customer-registrations` | Operations |
| `/admin/diagnostics` | Operations |
| `/admin/disputes` | Operations |
| `/admin/disputes/[id]` | Operations |
| `/admin/history` | Operations |
| `/admin/inbox` | Operations |
| `/admin/jobs` | Operations |
| `/admin/mandate-authorizations` | Operations |
| `/admin/message-submissions` | Operations |
| `/admin/money` | Operations |
| `/admin/mono` | Operations |
| `/admin/organizations` | Operations |
| `/admin/platform-settings` | Operations |
| `/admin/privacy` | Operations |
| `/admin/provider-events` | Operations |
| `/admin/provider-work` | Operations |
| `/admin/reconciliation` | Operations |
| `/admin/recovery` | Operations |
| `/admin/search` | Operations |
| `/admin/seller-settlements` | Operations |
| `/admin/settings` | Operations |
| `/admin/settlement-review` | Operations |
| `/admin/setup` | Operations |
| `/admin/team` | Operations |
| `/admin/users` | Operations |
| `/admin/verification-requests` | Operations |
| `/admin/website` | Operations |
| `/agents` | Field agents |
| `/blog` | Public |
| `/blog/[slug]` | Public |
| `/blog/topic/[topic]` | Public |
| `/buyer-invitations/[token]` | Private invitation |
| `/c/[token]` | Account entry and access |
| `/consumers` | Public |
| `/contact` | Public |
| `/demo` | Public |
| `/demo/consumer` | Public |
| `/distributors` | Public |
| `/faq` | Public |
| `/glossary` | Public |
| `/how-it-works` | Public |
| `/join` | Public |
| `/legal/complaints` | Public |
| `/legal/privacy` | Public |
| `/legal/terms` | Public |
| `/manufacturers` | Public |
| `/pay/[token]` | Private payment |
| `/personal/purchases` | Personal purchases |
| `/personal/purchases/[saleID]` | Personal purchases |
| `/pricing` | Public |
| `/receipt/[public_token]` | Private receipt |
| `/recover` | Account entry and access |
| `/retailers` | Public |
| `/secure` | Account entry and access |
| `/security` | Public |
| `/signin` | Account entry and access |
| `/start` | Account entry and access |
| `/workspace/activity` | Business workspace |
| `/workspace/disputes` | Business workspace |
| `/workspace/disputes/[id]` | Business workspace |
| `/workspace/help` | Business workspace |
| `/workspace/money` | Business workspace |
| `/workspace/money/collections` | Business workspace |
| `/workspace/money/received` | Business workspace |
| `/workspace/onboarding` | Business workspace |
| `/workspace/overdue` | Business workspace |
| `/workspace/partners` | Business workspace |
| `/workspace/partners/customers` | Business workspace |
| `/workspace/partners/customers/[id]` | Business workspace |
| `/workspace/partners/customers/new` | Business workspace |
| `/workspace/partners/import` | Business workspace |
| `/workspace/partners/invitations` | Business workspace |
| `/workspace/purchases` | Business workspace |
| `/workspace/purchases/access` | Business workspace |
| `/workspace/purchases/permissions` | Business workspace |
| `/workspace/purchases/amendments` | Business workspace |
| `/workspace/purchases/bank-authorization/[reference]` | Business workspace |
| `/workspace/purchases/disputes` | Business workspace |
| `/workspace/purchases/disputes/[id]` | Business workspace |
| `/workspace/purchases/history` | Business workspace |
| `/workspace/purchases/mandates` | Business workspace |
| `/workspace/purchases/obligations` | Business workspace |
| `/workspace/purchases/obligations/[id]` | Business workspace |
| `/workspace/purchases/orders` | Business workspace |
| `/workspace/purchases/orders/[requestID]` | Business workspace |
| `/workspace/purchases/payments` | Business workspace |
| `/workspace/purchases/trade-lines` | Business workspace |
| `/workspace/referral` | Business workspace |
| `/workspace/reports` | Business workspace |
| `/workspace/sales` | Business workspace |
| `/workspace/sales/[id]` | Business workspace |
| `/workspace/sales/consumers` | Business workspace |
| `/workspace/sales/consumers/[saleID]` | Business workspace |
| `/workspace/sales/limits` | Business workspace |
| `/workspace/sales/limits/[id]` | Business workspace |
| `/workspace/sales/new` | Business workspace |
| `/workspace/sales/quick` | Business workspace |
| `/workspace/search` | Business workspace |
| `/workspace/settings` | Business workspace |
| `/workspace/settings/billing` | Business workspace |
| `/workspace/settings/credit-policy` | Business workspace |
| `/workspace/settings/settlement` | Business workspace |
| `/workspace/team` | Business workspace |
| `/workspace/today` | Business workspace |

`/workspace` and `/personal` open their corresponding overview. Old `/app`, `/buyer`, `/for-suppliers` and `/for-buyers` routes are removed. API, sitemap, robots, manifest and server endpoints are outside this page inventory.

- `/workspace/partners/access`: owner-managed branch scopes, current-membership status, empty-scope restriction and staff self-view. Company-wide specialist workflows remain explicitly restricted.
