<script lang="ts">
	import NetworkChain from '$lib/components/NetworkChain.svelte';
	import { tradeSteps } from '$lib/network-content';
</script>

<main class="network-page shell">
	<header class="network-hero">
		<p class="eyebrow">How Kredit works</p>
		<h1>From the factory to the last shop.</h1>
		<p class="network-lede">
			Start with one customer. Put the terms down, record what was delivered, then follow the money. The same four steps
			hold whether you have one distributor or forty.
		</p>
		<div class="network-actions">
			<a class="primary" href="/demo">Try the sample journey →</a><a href="/signin?next=%2Fstart">Get started</a>
		</div>
	</header>
	<NetworkChain />
	<section class="network-section">
		<h2>Four steps, every time.</h2>
		<div class="network-cards four">
			{#each tradeSteps as [title, body], index (index)}<article>
					<span class="eyebrow">0{index + 1}</span>
					<h3>{title}</h3>
					<p>{body}</p>
				</article>{/each}
		</div>
	</section>
	<section class="network-section">
		<h2>Today’s customer is tomorrow’s supplier.</h2>
		<div class="network-cards">
			<article>
				<h3>The manufacturer starts it</h3>
				<p>
					Bring in the roster you already have, or invite one distributor and see how it runs. Each one gets his own
					limit and his own terms.
				</p>
			</article>
			<article>
				<h3>The distributor grows into it</h3>
				<p>
					One workspace handles what he buys from the manufacturer and what he sells on to retailers. A few selling
					checks have to be done before he can give credit himself.
				</p>
			</article>
			<article>
				<h3>The retailer serves the street</h3>
				<p>
					He follows his stock purchases, sells on to other businesses, and sends an individual customer a personal
					purchase link. That customer keeps a simpler account.
				</p>
			</article>
		</div>
	</section>
	<section class="network-note">
		<h2>Every balance sits on its own trade.</h2>
		<p>
			What your customers owe you is one book. What you owe your suppliers is another. Nothing is netted off between
			them, no limit is shared, and repayment is not guaranteed. A bank collection needs the customer’s permission and
			has to pass the payment checks before it goes out.
		</p>
	</section>
	<section class="network-section">
		<h2>When something goes wrong.</h2>
		<div class="network-cards">
			<article>
				<h3>The goods were short</h3>
				<p>
					Report it against that trade and attach what you have. While it is being looked at, you can see the amount in
					question and whether collection is still running.
				</p>
			</article>
			<article>
				<h3>He paid you directly</h3>
				<p>
					Your customer reports the transfer. You check that the money actually landed, then confirm it. Only then does
					the balance come down.
				</p>
			</article>
			<article>
				<h3>The money is late</h3>
				<p>
					Reminders go out, then the grace days you agreed run down. With valid permission the debit can be presented to
					his bank. It can still fail.
				</p>
			</article>
		</div>
	</section>
	<section class="network-cta">
		<div>
			<h2>Start with one customer.</h2>
			<p>Get one account running properly before you bring the rest on.</p>
		</div>
		<a class="primary" href="/signin?next=%2Fstart">Open Kredit →</a>
	</section>
</main>
