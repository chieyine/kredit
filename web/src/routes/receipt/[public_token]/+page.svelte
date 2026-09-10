<script lang="ts">
    import { page } from '$app/state';
    import Money from '$lib/components/Money.svelte';
    import { checkedJSON, LatestRequest, publicError, record, RequestError, text } from '$lib/api/reliable';
    import { kobo, timeLabel } from '$lib/records';
    let receipt: any = $state(null), error = $state('');
    const requests = new LatestRequest();
    async function load(token: string) {
        const request = requests.begin(); receipt = null; error = '';
        try {
            const result = await checkedJSON(`/api/v1/public/receipts/${encodeURIComponent(token)}`, value => {
                const row = record(record(value).receipt);
                if (!['recognized', 'reversed'].includes(text(row.state))) throw new RequestError('Receipt status could not be verified.');
                if (!text(row.reference).trim() || !Number.isFinite(Date.parse(text(row.paid_at)))) throw new RequestError('Receipt date could not be verified.');
                return { reference: text(row.reference), amount_kobo: kobo(row.amount_kobo), state: row.state, paid_at: text(row.paid_at) };
            }, { signal: request.signal });
            if (request.current()) receipt = result;
        } catch (cause) { if (request.current()) error = publicError(cause, 'this receipt'); }
    }
    $effect(() => { void load(page.params.public_token!); return () => requests.cancel(); });
</script>
<svelte:head><title>Receipt — Kredit</title></svelte:head>
<main class="shell prose-page"><p class="eyebrow">Payment receipt</p>{#if receipt}<h1>{receipt.state === 'reversed' ? 'This payment was reversed.' : 'This money was received.'}</h1>{#if receipt.state === 'reversed'}<p role="status">This receipt is a historical record. The payment no longer reduces the balance owed.</p>{/if}<dl><div><dt>Money paid</dt><dd><Money amountKobo={receipt.amount_kobo} /></dd></div><div><dt>Day it was paid</dt><dd>{timeLabel(receipt.paid_at)}</dd></div><div><dt>Receipt number</dt><dd>{receipt.reference}</dd></div></dl><p>You can share this receipt with anybody. Names and bank details are hidden on it.</p><div class="support-card"><p><strong>Something wrong with this receipt?</strong> Talk to a real person.</p><a class="whatsapp-support" href="mailto:hello@kredit.com.ng?subject=Receipt%20question">Email Kredit support</a></div>{:else if error}<h1>This receipt is not working.</h1><p role="alert">{error}</p><button type="button" onclick={() => load(page.params.public_token!)}>Try again</button>{:else}<p>Opening your receipt…</p>{/if}</main>
<style>dl{display:grid;gap:.75rem;padding:1.25rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}dl div{display:flex;justify-content:space-between;gap:2rem}dt{color:var(--color-muted)}dd{font-weight:750;text-align:right;overflow-wrap:anywhere}.support-card{margin:1.5rem 0;padding:1rem;background:#f0fdf4;border:1px solid #bbf7d0;border-radius:.75rem;display:flex;align-items:center;justify-content:space-between;gap:1rem;flex-wrap:wrap}.support-card p{margin:0;font-size:.9rem;color:#166534}.whatsapp-support{display:inline-flex;padding:.5rem 1rem;background:#15803d;color:#fff;font-weight:700;border-radius:.5rem;text-decoration:none;font-size:.85rem}</style>
