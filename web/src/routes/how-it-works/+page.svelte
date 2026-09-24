<script lang="ts">
	import NetworkChain from '$lib/components/NetworkChain.svelte';
	import { tradeSteps } from '$lib/network-content';
</script>

<main class="network-page shell">
	<header class="network-hero">
		<h1><span class="line">From the factory</span> <span class="line">to the last shop.</span></h1>
		<p class="network-lede">
			Start with one customer. Write down the terms, record what was delivered, then track the payments. It works the
			same way whether you supply one distributor or forty.
		</p>
		<div class="network-actions">
			<a class="primary" href="/demo">Try the demo <span aria-hidden="true">→</span></a><a href="/signin?next=%2Fstart"
				>Get started</a
			>
		</div>
	</header>
	<NetworkChain />
	<section class="network-section">
		<h2>The same four steps for every sale.</h2>
		<ol class="steps">
			{#each tradeSteps as [title, body], index (index)}
				<li>
					<div>
						<h3>{title}</h3>
						<span>{body}</span>
					</div>
				</li>
			{/each}
		</ol>
	</section>
	<section class="network-section">
		<h2>Today’s customer can be tomorrow’s supplier.</h2>
		<div class="network-cards">
			<article>
				<h3>The manufacturer starts</h3>
				<p>
					Bring in the distributors you already work with, or invite one and see how it goes. Each distributor gets
					their own limit and their own terms.
				</p>
			</article>
			<article>
				<h3>The distributor sells on</h3>
				<p>
					The same account handles what they buy from the manufacturer and what they sell to retailers. After a few
					checks, they can give credit to their own customers.
				</p>
			</article>
			<article>
				<h3>The retailer serves the street</h3>
				<p>
					Retailers track their own stock purchases, sell to other businesses, and send a private purchase link to
					anyone buying for themselves. That person gets a simpler personal account.
				</p>
			</article>
		</div>
	</section>
	<section class="network-note">
		<h2>Every sale has its own balance.</h2>
		<p>
			What your customers owe you and what you owe your suppliers are kept in separate books. Nothing is netted off
			between them, no limit is shared, and repayment is never guaranteed. Kredit can only request a payment from a
			customer’s bank with that customer’s permission, and only after the payment checks pass.
		</p>
	</section>
	<section class="network-section">
		<h2>When something goes wrong.</h2>
		<div class="network-cards">
			<article>
				<h3>Some goods were missing</h3>
				<p>
					Report the shortage on that sale and attach any proof you have. While it is being reviewed, you can see the
					amount in question and whether collection has been paused.
				</p>
			</article>
			<article>
				<h3>The customer paid you directly</h3>
				<p>
					Your customer records the transfer. You check that the money has actually arrived, then confirm it. Only then
					does the balance go down.
				</p>
			</article>
			<article>
				<h3>A payment is late</h3>
				<p>
					Reminders go out first, then the grace days you agreed run out. If there is a valid bank permission, Kredit
					can request the payment from your customer’s bank. That request can still fail.
				</p>
			</article>
		</div>
	</section>
	<section class="network-cta">
		<div>
			<h2>Start with one customer.</h2>
			<p>Get one account working well before you bring the rest on.</p>
		</div>
		<a class="primary" href="/signin?next=%2Fstart">Open Kredit <span aria-hidden="true">→</span></a>
	</section>
</main>
