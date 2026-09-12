<script lang="ts">
 import type { SaleView } from '$lib/records';
 import { baseFeeForKobo,feeDisclosure } from '$lib/fee-terms';
 import { dateLabel,timeLabel } from '$lib/records';
 import Money from './Money.svelte';
 let {view}:{view:SaleView}=$props();
 const sellerFee=$derived(view.request.fee_terms?baseFeeForKobo(view.request.principal_kobo,view.request.fee_terms):null);
</script>
<section class="sale-costs" aria-label="Sale amount and fees"><h2>What this sale costs</h2><dl><div><dt>Customer's agreed total</dt><dd><Money amountKobo={view.request.principal_kobo}/></dd></div><div><dt>Seller's fee when the sale becomes active</dt><dd><Money amountKobo={sellerFee}/></dd></div><div><dt>Original payment date</dt><dd>{dateLabel(view.request.due_date)}</dd></div><div><dt>Earliest agreed bank-debit time</dt><dd>{timeLabel(view.request.collection_at)}</dd></div></dl><p>{feeDisclosure(view.request.fee_terms)}</p><p>Bank permission is separate from accepting the sale. The payment schedule and any recorded changes remain available below.</p></section>
<style>.sale-costs{border:1px solid var(--color-border,#d8e1df);border-radius:.75rem;padding:1.25rem;margin:1rem 0}h2{font-size:1.25rem}dl{display:grid;grid-template-columns:repeat(auto-fit,minmax(14rem,1fr));gap:1rem}dt{color:var(--color-muted,#626977);font-size:.9rem}dd{margin:.35rem 0;font-weight:650}p{line-height:1.6}</style>
