<script lang="ts">
 import {page} from '$app/state';
 let {buyer=false}:{buyer?:boolean}=$props();
 const personal=$derived(page.url.pathname.includes('/consumer-sales')||page.url.pathname.includes('/purchases'));
 const trade=$derived(buyer?/^\/buyer\/(requests|obligations|trade-lines|history|payments|mandates|amendments)(\/|$)/.test(page.url.pathname):page.url.pathname.startsWith('/app/credit'));
</script>
<nav class="shell flow-switch" aria-label={buyer?'Choose what you are buying':'Choose who you are selling to'}>
 <a href={buyer?'/buyer/requests':'/app/credit'} aria-current={trade?'location':undefined} class:chosen={trade}>
  <span>Wholesaler → Retailer</span><strong>{buyer?'Stock for your business':'Sell to a business'}</strong><small>Goods bought for resale or business use.</small>
 </a>
 <a href={buyer?'/buyer/purchases':'/app/consumer-sales'} aria-current={personal?'location':undefined} class:chosen={personal}>
  <span>Retailer → Consumer</span><strong>{buyer?'Your personal purchases':'Sell to an individual'}</strong><small>Goods bought for personal use, on installments or layaway.</small>
 </a>
</nav>
<style>
 .flow-switch{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.75rem;padding-top:1.5rem;max-width:68rem}
 .flow-switch a{display:grid;gap:.35rem;padding:1rem 1.2rem;border:1px solid var(--color-border);background:var(--color-surface);color:var(--color-foreground);text-decoration:none;min-width:0;border-top:3px solid var(--color-border)}
 .flow-switch a.chosen{border-top-color:var(--color-primary);background:#eef0ff}
 .flow-switch a:hover{border-color:var(--color-primary)}
 .flow-switch span{font-size:.75rem;font-weight:700;color:var(--color-primary)}
 .flow-switch strong{font-size:1rem}.flow-switch small{font-size:.85rem;line-height:1.5;color:var(--color-muted)}
 @media(max-width:560px){.flow-switch{gap:.5rem}.flow-switch a{padding:.8rem}.flow-switch small{display:none}.flow-switch strong{font-size:.9rem}}
 @media print{.flow-switch{display:none}}
</style>
