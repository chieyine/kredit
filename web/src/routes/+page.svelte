<script lang="ts">
	import NetworkChain from '$lib/components/NetworkChain.svelte';
	import TradeRecord from '$lib/components/TradeRecord.svelte';
	import { tradeSteps } from '$lib/network-content';
	let { data } = $props();
</script>

<main class="network-page shell">
	<section class="network-hero home-hero" aria-labelledby="home-title">
		<div class="hero-copy">
			<h1 id="home-title">
				<span class="line">{data.copy.title}</span>
				<em class="line">{data.copy.accent}</em>
			</h1>
			<p class="network-lede">{data.copy.introduction}</p>
			<div class="network-actions">
				<a class="primary" href="/signin?next=%2Fstart">Get started <span aria-hidden="true">→</span></a>
				<a href="/demo">Try the demo</a>
			</div>
			<p class="network-caption">It runs in your phone’s browser. There is nothing to install.</p>
		</div>
		<TradeRecord />
	</section>

	<section class="network-section" aria-labelledby="network-heading">
		<h2 id="network-heading">Built for everyone between the factory and the customer.</h2>
		<p class="network-lede">
			A manufacturer invites its distributors. A distributor can then give credit to the shops it supplies, and a shop
			can do the same for the people who buy from it. Each business sees its own trades and nobody else’s.
		</p>
		<NetworkChain />
	</section>

	<section class="network-section" aria-labelledby="simple-heading">
		<h2 id="simple-heading">How a sale on credit works.</h2>
		<ol class="steps">
			{#each tradeSteps as [title, body], i (i)}
				<li>
					<div>
						<h3>{title}</h3>
						<span>{body}</span>
					</div>
				</li>
			{/each}
		</ol>
		<a class="network-text-link" href="/how-it-works">See the whole process <span aria-hidden="true">→</span></a>
	</section>

	<section class="network-note">
		<h2>Your customers stay yours.</h2>
		<p>
			Joining your supplier on Kredit does not let them see who you sell to. Each sale has its own agreement and its own
			balance, and only the two businesses on it can open it.
		</p>
		<div class="network-actions">
			<a href="/security">How we protect your account</a><a href="/pricing">What it costs</a>
		</div>
	</section>

	<section class="network-cta">
		<div>
			<h2>Start with one customer.</h2>
			<p>Set up your business, or open the invitation a supplier sent you.</p>
		</div>
		<a class="primary" href="/signin?next=%2Fstart">Get started <span aria-hidden="true">→</span></a>
	</section>
</main>
