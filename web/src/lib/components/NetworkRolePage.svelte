<script lang="ts">
	import { network, type NetworkRole } from '$lib/network-content';
	import NetworkChain from './NetworkChain.svelte';
	let { role }: { role: NetworkRole } = $props();
	const stage = $derived(network.find((item) => item.key === role)!);
</script>

<!-- Public title and social metadata come from the shared route SEO in +layout. -->

<main class="network-page shell">
	<header class="network-hero">
		<p class="eyebrow">Kredit for {stage.label.toLowerCase()}</p>
		<h1>{stage.title}</h1>
		<p class="network-lede">{stage.description}</p>
		<div class="network-actions">
			<a class="primary" href={stage.href}>{stage.action} <span aria-hidden="true">→</span></a>
			<a href={role === 'consumers' ? '/demo/consumer' : '/demo'}>Explore the demo</a>
		</div>
	</header>

	<section class="network-section" aria-labelledby="role-steps">
		<p class="eyebrow">Getting started</p>
		<h2 id="role-steps">A clear next step, every time.</h2>
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
		<h2>Part of a connected network.</h2>
		<p>{stage.next}</p>
		<NetworkChain compact />
	</section>

	<section class="network-cta">
		<div>
			<h2>
				{role === 'consumers' ? 'Already received a purchase link?' : 'Start with your next trading relationship.'}
			</h2>
			<p>
				{role === 'consumers'
					? 'Open that link to go straight to the purchase. Your account keeps the accepted record available afterwards.'
					: 'One business account for the credit you give and receive. Each relationship keeps its own terms and balance.'}
			</p>
		</div>
		<a class="primary" href={stage.href}>{stage.action} →</a>
	</section>
</main>
