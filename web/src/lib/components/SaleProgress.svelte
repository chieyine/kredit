<script lang="ts">
 import type { SaleView } from '$lib/records';
 import { timeLabel } from '$lib/records';
 import { saleNextStep } from '$lib/sale-progress';
 let {view,audience}:{view:SaleView;audience:'buyer'|'seller'}=$props();
 const next=$derived(saleNextStep(view));
 const events=$derived(view.timeline??[]);
</script>
<section class="sale-progress" aria-label="Sale progress">
 <p class="eyebrow">Next step · {next.actor}</p><h2>{next.title}</h2><p>{next.detail}</p>
 {#if audience==='buyer' && ['VERIFICATION_PENDING','SENT','BUYER_REVIEWING'].includes(view.request.state)}<a href={`/buyer?business_id=${encodeURIComponent(view.request.buyer_business_id)}`}>Check identity and business verification</a>{/if}
 <details><summary>Sale timeline</summary><ol>{#each events as event}<li><strong>{event.label}</strong><span>{timeLabel(event.at)}</span></li>{:else}<li>Recorded dates are unavailable. Refresh the sale before acting.</li>{/each}</ol></details>
</section>
<style>.sale-progress{border:1px solid var(--color-border,#d8e1df);border-left:4px solid var(--color-primary,#303caa);border-radius:.75rem;padding:1.25rem;margin:1rem 0;background:var(--color-surface,#fff)}h2{margin:.35rem 0;font-size:1.3rem}p{line-height:1.6}summary{cursor:pointer;padding:.8rem 0}ol{padding-left:1.3rem}li{padding:.45rem 0}li span{display:block;color:var(--color-muted,#626977);font-size:.9rem}</style>
