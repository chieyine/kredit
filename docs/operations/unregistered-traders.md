# Unregistered traders and sales types

Sales type and CAC registration are separate:

- **Wholesaler → Retailer:** goods for resale or business use, in Business sales.
- **Retailer → Consumer:** goods for personal use, in Consumer sales. Customer payments go to the retailer’s receiving account; this flow does not automatically repay a wholesaler.

An organization created as `unregistered_business` can activate selling without a CAC lookup. Its active owner starts a personal check from business setup, using the saved full owner name. The configured identity adapter verifies that person with the organization ID as the verification subject. Registered organizations continue to use business verification.

Personal approval requires provider-confirmed NIN or BVN evidence, verification level 2 or higher, a future expiry and a matching full name. The saved approval reason is `owner_identity_approved`; it never asserts CAC registration. Bank registration must return the same full name. Readiness also checks that name match, including when bank evidence existed before personal approval. Name order, case and whitespace may differ; missing or additional names require corrected evidence. An approved owner name cannot be replaced through the ordinary representative form.

Contact checks, MFA, verified receiving-account setup, fee setup, terms and sales settings still apply. Payment-provider availability remains a launch dependency. Existing identity review controls handle mismatched identity evidence; they cannot make a mismatched bank account eligible.

The agent onboarding reward still requires CAC-verified registration. Creating an unregistered trader account does not qualify for that reward.

Mono uses the existing consent-based phone/NIN integration: https://docs.mono.co/api/phone/verify . Other identity adapters must return the same normalized evidence contract; this route adds no vendor-specific activation endpoint.
