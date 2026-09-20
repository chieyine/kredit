<script lang="ts">
  import NetworkChain from '$lib/components/NetworkChain.svelte';
  import { network, tradeSteps } from '$lib/network-content';
  let { data } = $props();
</script>

<main class="network-page shell">
  <section class="network-hero home-hero" aria-labelledby="home-title">
    <div>
      <p class="eyebrow">Trade credit, connected</p>
      <h1 id="home-title">{data.copy.title}<br /><em>{data.copy.accent}</em></h1>
      <p class="network-lede">{data.copy.introduction}</p>
      <div class="network-actions">
        <a class="primary" href="/signin?next=%2Fstart">Get started <span aria-hidden="true">→</span></a>
        <a href="/demo">Explore the demo</a>
      </div>
      <p class="network-caption">Use your browser. Start with one partner. Grow from there.</p>
    </div>

    <!-- The workspace itself rather than a picture of one. This is a layout
         example, not platform data: no figures are shown that would read as a
         claim about volume or performance. -->
    <aside class="network-preview" aria-label="How your workspace is organised">
      <div class="ledger-head"><p class="eyebrow">One business workspace</p><h2>Both sides of your trade</h2></div>
      <div>
        <div class="ledger-row"><div><span>Sales</span><strong>What customers owe you</strong></div></div>
        <p>Terms, deliveries and receivables.</p>
      </div>
      <div>
        <div class="ledger-row"><div><span>Purchases</span><strong>What you owe suppliers</strong></div></div>
        <p>Agreements, payment dates and balances.</p>
      </div>
      <div>
        <div class="ledger-row">
          <div><span>Settlement</span><strong>When a trade completes</strong></div>
          <span class="ledger-seal">Settled</span>
        </div>
        <p>The record stays open to both sides afterwards.</p>
      </div>
      <footer>Separate balances. Clear responsibilities.</footer>
    </aside>
  </section>

  <section class="network-section" aria-labelledby="network-heading">
    <p class="eyebrow">The whole trading journey</p>
    <h2 id="network-heading">Start upstream. Grow together.</h2>
    <p class="network-lede">A manufacturer brings distributors onto Kredit. Each distributor can invite retailers, and each retailer can serve individual consumers.</p>
    <NetworkChain />
  </section>

  <section class="network-section" aria-labelledby="fits-heading">
    <h2 id="fits-heading">One platform. A clear place for you.</h2>
    <div class="network-cards role-cards">
      {#each network as stage}
        <article><h3>{stage.label}</h3><p>{stage.description}</p><a href={`/${stage.key}`}>See your journey <span aria-hidden="true">→</span></a></article>
      {/each}
    </div>
  </section>

  <section class="network-section" aria-labelledby="simple-heading">
    <p class="eyebrow">How a trade works</p>
    <h2 id="simple-heading">Connect. Agree. Deliver. Settle.</h2>
    <ol class="steps">
      {#each tradeSteps as [title, body]}
        <li><div><h3>{title}</h3><span>{body}</span></div></li>
      {/each}
    </ol>
    <a class="network-text-link" href="/how-it-works">Walk through the full journey →</a>
  </section>

  <section class="network-note">
    <h2>Connected does not mean shared with everyone.</h2>
    <p>Your business controls its customer relationships. An upstream supplier does not gain access to your downstream sales. Each trade has its own agreement, balance and authorised participants.</p>
    <div class="network-actions"><a href="/security">Privacy and account safety →</a><a href="/pricing">See how fees work →</a></div>
  </section>

  <section class="network-cta">
    <div><h2>Make your next trade clearer.</h2><p>Set up your business, or open the invitation your trading partner sent you.</p></div>
    <a class="primary" href="/signin?next=%2Fstart">Get started →</a>
  </section>
</main>
