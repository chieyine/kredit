<script lang="ts">
    import { page } from '$app/state';
    import Money from '$lib/components/Money.svelte';
    import { checkedJSON, LatestRequest, publicError, record, text } from '$lib/api/reliable';
    import { kobo } from '$lib/records';
    import { exactKobo } from '$lib/money';
    let intent: any = $state(null), error = $state('');
    const requests = new LatestRequest();
    async function load(token: string) {
        const request = requests.begin(); intent = null; error = '';
        try {
            const result = await checkedJSON(`/api/v1/public/payment-intents/${encodeURIComponent(token)}`, value => {
                const row = record(record(value).payment_intent);
                return { reference: text(row.reference), supplier_name: text(row.supplier_name), description: text(row.description), amount_kobo: kobo(row.amount_kobo), provider_action: text(row.provider_action) };
            }, { signal: request.signal });
            if (request.current()) intent = result;
        } catch (cause) { if (request.current()) error = publicError(cause, 'this payment link'); }
    }
    $effect(() => { void load(page.params.token!); return () => requests.cancel(); });
</script>
<svelte:head><title>Sale balance — Kredit</title></svelte:head>
<main class="shell prose-page"><p class="eyebrow">Sale balance</p>{#if intent}<h1>{exactKobo(intent.amount_kobo) === 0n ? 'There is nothing left to pay.' : 'Check what is left to pay.'}</h1><dl><div><dt>Seller</dt><dd>{intent.supplier_name||'Seller'}</dd></div><div><dt>For</dt><dd>{intent.description}</dd></div><div><dt>Money left to pay</dt><dd><Money amountKobo={intent.amount_kobo} /></dd></div></dl><p>{intent.provider_action}</p>{#if exactKobo(intent.amount_kobo)! > 0n}<a class="primary" href={`/buyer/credit-requests/${encodeURIComponent(intent.reference)}`}>Open your sale</a>{/if}<div class="support-card"><p><strong>Not sure about this payment?</strong> Talk to a real person before you pay.</p><a class="whatsapp-support" href="mailto:hello@kredit.ng?subject=Payment%20question">Email Kredit support</a></div><p class="privacy">This page never shows your bank details or your private account information.</p>{:else if error}<h1>This payment link is not working.</h1><p role="alert">{error}</p><button type="button" onclick={() => load(page.params.token!)}>Try again</button>{:else}<p>Opening the sale balance…</p>{/if}</main>
<style>dl{display:grid;gap:.8rem;padding:1.25rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}dl div{display:flex;justify-content:space-between;gap:2rem}dt{color:var(--color-muted)}dd{font-weight:750;text-align:right}.privacy{font-size:.9rem;color:var(--color-muted)}.support-card{margin:1.5rem 0;padding:1rem;background:#f0fdf4;border:1px solid #bbf7d0;border-radius:.75rem;display:flex;align-items:center;justify-content:space-between;gap:1rem;flex-wrap:wrap}.support-card p{margin:0;font-size:.9rem;color:#166534}.whatsapp-support{display:inline-flex;padding:.5rem 1rem;background:#15803d;color:#fff;font-weight:700;border-radius:.5rem;text-decoration:none;font-size:.85rem}</style>
