# Admin connection controls

Open **Admin → Platform settings → Connections** as the platform owner, with a fresh authenticator verification.

Email, SMS and WhatsApp each support an encrypted connection override. Choose **Connect or replace credentials**, enter the HTTPS connector address and access token, and record a reason. Both fields are replaced together. The next delivery in staging or production reads the saved configuration; API and worker processes do not require a restart. Development deliberately retains mock delivery.

The screen distinguishes no admin override, disabled, enabled but delivery unverified, and unreadable configuration. Saving proves neither vendor acceptance nor delivery. Review delivery failures under **Needs attention** and **System diagnostics** after an authorized test to your own destination.

Choose **Disable this channel** to stop it, including sign-in codes. A disabled channel does not fall back to deployment credentials. A database or decryption failure also does not reactivate old credentials. When no override exists, existing deployment credentials remain the fallback.

## Supported connector contract

These are Kredit-compatible connectors, not native adapters for arbitrary SMS or email vendors. A vendor API key alone is insufficient. The configured service must accept an HTTPS POST with a Bearer token and these JSON fields:

```json
{
  "channel": "sms",
  "destination": "recipient",
  "event_id": "unique-event",
  "template": "template-name",
  "template_version": "v1",
  "body": "message content",
  "secure_link": ""
}
```

The connector must deduplicate the `Idempotency-Key` header (`event_id:channel`) and return a successful HTTP status with `{"message_id":"provider-reference"}`. Redirects are refused. Delivery requests have a ten-second deadline.

## Identity, Mono, document scanning and other bank connectors

Connections now also includes **Identity verification**, **Mono bank collections**, **Document scanning**, **Other bank collection connector**, and **Launch approvals and provider limits**. The ordinary fields are prefilled; saved passwords and signing secrets stay hidden. Leave a password field blank to retain its current value, or explicitly choose removal. Pausing Mono collections preserves the credentials needed for reconciliation.

These connections apply when the API and worker restart. Both load the same encrypted settings before constructing their services. The page shows when a saved version still needs a restart and when the API has applied it. Confirm the worker has restarted as well; API status does not prove worker activation or provider success.

Save approval references and pilot limits before enabling the relevant service. Configuration changes are validated with the other saved settings inside the same transaction lock. Invalid configurations and stale versions are rejected. An owner can replace supported provider credentials without editing application code or environment files.

Mono has a native adapter. Identity, document scanning and the other collection integration use the existing Kredit connector contracts. An arbitrary vendor API still requires a compatible adapter; this screen cannot convert an unsupported vendor protocol automatically.

## Initial deployment and verification

Apply database migrations through 096 before running this application version. Keep `SETTINGS_ENCRYPTION_KEY` in deployment secrets; changing it without a planned re-encryption makes existing credentials unreadable. Backups must preserve access to the matching key.

The first production owner sign-in requires a working notification service configured during deployment. Admin cannot configure a first sign-in channel before the owner can sign in. Preserve another working sign-in channel before disabling one.

Database connections, storage infrastructure and root encryption/signing keys remain deployment configuration. These are needed before an administrator can sign in and read encrypted settings. Provider credentials now have admin controls. Native adapters for unselected vendors and actual live delivery, identity checks and bank collections still need provider-specific verification. This document is not launch sign-off.

Release certification now includes saved admin settings. For an explicit configuration check, use `go run ./cmd/configcheck --stored` against the intended environment; it reads settings without contacting delivery or bank providers.

Use `ADMIN_SURFACES=all` for the full admin application, as in `.env.example`.
A restricted list must include `platform-settings`, `business-policies`,
`commands` and the owner workflow surfaces you operate.

## Automatic acceptance and bank registration recovery

Platform settings includes `features.system_acceptance` (off by default) and `automation.system_acceptance_hours` (72–720 hours, subject to the deployment minimum). These govern only requests with qualifying saved evidence; enabling the switch does not bypass missing delivery evidence or a buyer-reported issue.

Open **Admin → Customer registrations** for an uncertain Mono registration. Check the provider record before resolving it. Linking verifies the provider identity against the original server-side fingerprint. “Not created” requires an explanation of the provider evidence checked and permits the buyer to try again; do not choose it merely because the first request timed out. Recent MFA is required.
