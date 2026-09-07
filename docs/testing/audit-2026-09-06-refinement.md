# Audit refinement

The advanced invoice/instalment form now shares authenticated business scoping,
exact kobo, durable invoice/create operation identities, customer-fetch errors,
server-verified Lagos timing and an explicit review step. Legacy unscoped browser
records are removed rather than migrated between people. Browser tests target API
URLs only; they cannot accidentally replace application HTML with mock JSON.
Native account menus also wrap Tab/Shift-Tab across the browser focus boundary.
The public pricing API retains its established three-field contract.

No existing accepted agreement is rewritten. No live collection gate is enabled.
Test outcomes must be read from the checks on the final commit; this note itself
is not evidence that a test passed. External gates #5 and #7 remain open.
