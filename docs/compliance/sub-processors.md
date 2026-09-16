# Sub-processor register

Every external host that Kredit's own code sends personal or transaction data to,
and the basis on which it does so.

`docs/compliance/data-inventory.tsv` covers data **at rest**, column by column, and
`scripts/data-inventory-check.sh` proves it matches the live schema. It cannot see
data **in transit** to a third party, because a new outbound call adds no column.
This register closes that gap, and `scripts/sub-processor-check.sh` fails the build
when Go source references a production host that is not listed here.

A row here records where a decision lives; it is not the decision. Adding a host to
this file does not create a lawful basis for the transfer.

| Host | Processor | Capability | Data sent | Enabled by | Transfer basis |
|---|---|---|---|---|---|
| `api.withmono.com` | Mono | Bank authorisation, direct debit, identity lookup | Customer reference, BVN or NIN-linked phone for verification, mandate and debit amounts, transaction references | `FEATURE_REAL_COLLECTIONS`, `MONO_SWEEP_ENABLED`, `FEATURE_REAL_IDENTITY` | Nigerian processor; data remains in Nigeria. Contracted under the Mono agreement referenced by `PROVIDER_CERTIFICATION_REFERENCE`. |
| `authorise.mono.co` | Mono | Hosted bank-authorisation page the buyer is redirected to | Buyer completes authorisation directly with Mono; Kredit sends only the mandate reference | `FEATURE_REAL_COLLECTIONS` | As above. The host is pinned in `internal/providers/mono/hosted_url.go`. |
| `api.paystack.co` | Paystack | Bank authorization, tokenized recurring debits, transaction verification | Customer reference, email, account details for authorization, transaction amounts in kobo | `COLLECTION_ADAPTER=paystack` | Nigerian processor; data remains in Nigeria. Contracted under Paystack merchant terms. |
| `link.paystack.com` | Paystack | Hosted bank authorization and mandate enrollment page | Buyer completes bank authorization directly with Paystack | `COLLECTION_ADAPTER=paystack` | As above. Pinned in Paystack authorization redirect flow. |
| `api.flutterwave.com` | Flutterwave | Bank debit initiation, reference-based verification, webhooks | Customer reference, bank details, mandate amounts, debit references | `COLLECTION_ADAPTER=flutterwave` | Nigerian processor; data remains in Nigeria. Contracted under Flutterwave merchant terms. |
| `api.monnify.com` | Monnify | Bank direct debit mandates and collections | Customer reference, bank account details, mandate amounts and schedules | `COLLECTION_ADAPTER=monnify` | Nigerian processor; data remains in Nigeria. Contracted under Monnify merchant terms. |
| `sandbox.monnify.com` | Monnify | Direct debit mandate and collection sandbox testing | Test account references and test amounts | `COLLECTION_ADAPTER=monnify` | Non-production sandbox environment. |
| `api.sendlyai.com` | Sendly | Transactional email delivery | Recipient email address, template name, notification body including amounts and dates | `NOTIFICATION_EMAIL_ENDPOINT` | Recorded in the environment retention register. Sender domain verified at `kredit.ng`. |
| `api.mesaj.cloud` | Mesaj | SMS delivery | Recipient phone number, message body | `NOTIFICATION_SMS_ENDPOINT` | Nigerian processor; data remains in Nigeria. |
| `graph.facebook.com` | Meta | WhatsApp Business messaging and media retrieval | Recipient phone number, template name, message body, media identifiers | `FEATURE_WHATSAPP` | Cross-border transfer to Meta. Covered by the Meta processor terms recorded against the WhatsApp Business account. |
| `generativelanguage.googleapis.com` | Google (Gemini API) | Reads an inbound WhatsApp message or voice note back to the seller as structured fields | Message text or voice-note audio. In practice this carries customer names, goods descriptions, amounts and payment dates. No identifier, credential or account number is sent deliberately, but the seller controls the message content. | `FEATURE_WHATSAPP_ASSISTANT` (off by default) | **Cross-border transfer outside Nigeria.** Requires `WHATSAPP_ASSISTANT_TRANSFER_REFERENCE` naming the completed transfer assessment; the API refuses to start in production without it. |
| `r2.cloudflarestorage.com` | Cloudflare (R2) | Private document storage and offsite database backups | Uploaded invoices and dispute evidence; full database dumps from `cmd/backup-r2` | `OBJECT_STORAGE_ENDPOINT` | Cross-border transfer. Bucket is private, downloads are short-lived signed URLs, and backups carry a recorded SHA-256. |

## Notes on the assistant

The WhatsApp assistant is the only entry here that sends free-text customer data
to a general-purpose model provider, so it carries the tightest controls:

- off unless `FEATURE_WHATSAPP_ASSISTANT` is explicitly true;
- production startup fails without both a valid `GEMINI_API_KEY` and a recorded
  transfer reference;
- the key travels in the `x-goog-api-key` header, never a URL;
- a per-sender budget of 12 requests per 10 minutes bounds both spend and volume;
- voice notes are not downloaded at all when the assistant is unavailable;
- provider error bodies are never logged, because they can echo the message;
- it holds no write capability. It cannot create a sale, confirm one or record a
  payment, and its replies say so.

## Hosts that are not sub-processors

`opentelemetry.io` and `docs.mono.co` appear in source as documentation links.
`*.kredit.ng` are Kredit's own services. `*.example`, `*.test` and `*.invalid`
hosts appear only in tests and fixtures. The check script ignores all of these.
