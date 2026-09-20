<script lang="ts">
  import { tick } from 'svelte';
  import Money from '$lib/components/Money.svelte';
  const steps=[
    {title:'Review your purchase',body:'A retailer offers a standing fan for ₦60,000, paid in three instalments of ₦20,000. Delivery is due within seven days after full payment.',action:'Accept sample terms'},
    {title:'Follow each payment',body:'The first ₦20,000 payment has been confirmed. Two instalments remain. Reporting a transfer alone would not change the balance.',action:'Complete sample payments'},
    {title:'Check delivery',body:'The sample balance is now zero and the retailer has dispatched the fan. The consumer can confirm receipt or report a problem.',action:'Confirm sample delivery'},
    {title:'Keep the record',body:'Payment and delivery are recorded. A return request or complaint starts from this same purchase, with the agreed terms kept available.',action:''}
  ];
  let stage=$state(0), heading:HTMLHeadingElement|undefined=$state();
  const current=$derived(steps[stage]);
  async function next(){stage=Math.min(stage+1,steps.length-1);await tick();heading?.focus()}
</script>
<svelte:head><title>Try a personal purchase — Kredit demo</title></svelte:head>
<main class="network-page shell">
  <header class="network-hero"><p class="eyebrow">Retailer → Consumer · Illustrative demo</p><h1>A personal purchase, made clear.</h1><p class="network-lede">No sign-in, bank connection or real money. Explore the consumer journey separately from business trade credit.</p><div class="network-actions"><a href="/demo">Business trade demo</a><a href="/consumers">About personal purchases</a></div></header>
  <section class="consumer-demo" aria-label="Sample personal purchase">
    <p class="eyebrow">Step {stage+1} of {steps.length}</p><h2 bind:this={heading} tabindex="-1">{current.title}</h2><p>{current.body}</p>
    <dl><div><dt>Sample item</dt><dd>Standing fan</dd></div><div><dt>Total agreed price</dt><dd><Money amountKobo={6000000}/></dd></div><div><dt>Confirmed payments</dt><dd><Money amountKobo={stage===0?0:stage===1?2000000:6000000}/></dd></div><div><dt>Left to pay</dt><dd><Money amountKobo={stage===0?6000000:stage===1?4000000:0}/></dd></div></dl>
    {#if current.action}<button class="primary" onclick={next}>{current.action} →</button>{:else}<p role="status">Sample complete. No real agreement or payment was created.</p><button onclick={()=>stage=0}>Restart sample</button><a class="primary" href="/signin?next=%2Fpersonal%2Fpurchases">Open my purchases →</a>{/if}
  </section>
  <section class="network-cta"><div><h2>Buying for your business?</h2><p>Business purchases live in your workspace, with your supplier relationships and customer sales.</p></div><a href="/distributors">Explore business purchases →</a></section>
</main>
<style>
 .consumer-demo{max-width:44rem;padding:clamp(1.5rem,4vw,3rem);border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}dl{margin:1.5rem 0}dl>div{display:flex;justify-content:space-between;gap:1rem;padding:.8rem 0;border-bottom:1px solid var(--color-border)}dd{margin:0;font-weight:700;text-align:right}button,a.primary{margin:.5rem .5rem .5rem 0}button{padding:.8rem 1rem;font:inherit}
</style>
