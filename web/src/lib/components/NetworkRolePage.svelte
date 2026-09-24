<script lang="ts">
	import { network, type NetworkRole } from '$lib/network-content';
	import NetworkChain from './NetworkChain.svelte';
	let { role }: { role: NetworkRole } = $props();
	const stage = $derived(network.find((item) => item.key === role)!);
	const consumer = $derived(role === 'consumers');
</script>

<!-- Public title and social metadata come from the shared route SEO in +layout. -->

<main class="network-page shell">
	<header class="network-hero home-hero">
		<div class="hero-copy">
			<h1><span class="line">{stage.title}</span></h1>
			<p class="network-lede">{stage.description}</p>
			<div class="network-actions">
				<a class="primary" href={stage.href}>{stage.action} <span aria-hidden="true">→</span></a>
				<a href={consumer ? '/demo/consumer' : '/demo'}>Try the demo</a>
			</div>
		</div>
		<!-- What this person's account puts in front of them. A layout example
		     only: no figures, so nothing here reads as a claim. -->
		<aside class="role-panel" aria-label={consumer ? 'What your purchase page shows' : 'What your account shows'}>
			<p>{consumer ? 'Your purchase page' : `Your account as a ${stage.label.toLowerCase().replace(/s$/, '')}`}</p>
			<ul>
				{#each stage.panel as [title, body], i (title)}
					<li style={`--i: ${i}`}><strong>{title}</strong><span>{body}</span></li>
				{/each}
			</ul>
		</aside>
	</header>

	<section class="network-section" aria-labelledby="role-steps">
		<h2 id="role-steps">{consumer ? 'What happens when you buy.' : 'How you get started.'}</h2>
		<ol class="steps">
			{#each stage.steps as [title, body], i (i)}
				<li>
					<div>
						<h3>{title}</h3>
						<span>{body}</span>
					</div>
				</li>
			{/each}
		</ol>
	</section>

	<section class="network-note">
		<h2>{consumer ? 'Your purchases are your own.' : 'Where you fit.'}</h2>
		<p>{stage.next}</p>
		<NetworkChain compact current={role} />
	</section>

	<section class="network-cta">
		<div>
			<h2>{consumer ? 'Did a seller send you a link?' : 'Start with one customer.'}</h2>
			<p>
				{consumer
					? 'Open it to go straight to the purchase. Once you accept, the record stays in your account.'
					: 'One business account covers the credit you give and the credit you receive. Each customer keeps their own terms and balance.'}
			</p>
		</div>
		<a class="primary" href={stage.href}>{stage.action} <span aria-hidden="true">→</span></a>
	</section>
</main>

<style>
	.role-panel {
		border: 1px solid var(--color-border-strong);
		background: linear-gradient(176deg, #171b24, #101319 60%);
		box-shadow: var(--shadow-lg), var(--edge-lit);
		animation: panel-in 800ms cubic-bezier(0.16, 0.7, 0.2, 1) 250ms both;
	}
	.role-panel > p {
		margin: 0;
		padding: 1rem 1.5rem;
		border-bottom: 1px solid var(--color-border);
		background: rgb(255 255 255 / 0.02);
		color: var(--color-muted);
		font-size: 0.78rem;
		font-weight: 550;
	}
	ul {
		margin: 0;
		padding: 0;
		list-style: none;
	}
	li {
		display: grid;
		gap: 0.3rem;
		padding: 1.3rem 1.5rem;
		border-top: 1px solid var(--color-border);
		animation: row-in 600ms cubic-bezier(0.16, 0.7, 0.2, 1) both;
		animation-delay: calc(650ms + var(--i) * 160ms);
	}
	li:first-child {
		border-top: 0;
	}
	li strong {
		font-size: 1rem;
		font-weight: 600;
		letter-spacing: -0.012em;
	}
	li span {
		color: var(--color-muted);
		font-size: 0.88rem;
	}
	@keyframes panel-in {
		from {
			opacity: 0;
			transform: translateY(24px);
		}
	}
	@keyframes row-in {
		from {
			opacity: 0;
			transform: translateX(-10px);
		}
	}
</style>
