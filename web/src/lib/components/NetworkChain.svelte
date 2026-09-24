<script lang="ts">
	import { network, type NetworkRole } from '$lib/network-content';
	let { compact = false, current }: { compact?: boolean; current?: NetworkRole } = $props();
</script>

<!-- One line runs through the four stages, because it is one chain of trade:
     goods move right, money comes back left. The line draws itself in when the
     chain scrolls into view; without script or with reduced motion it is simply
     there. -->
<ol class:compact class="network-chain" aria-label="From manufacturer to consumer" data-reveal>
	{#each network as stage, index (stage.key)}
		<li class:current={stage.key === current} style={`--i: ${index}`}>
			<span class="node" aria-hidden="true"></span>
			<a href={`/${stage.key}`} aria-current={stage.key === current ? 'page' : undefined}>
				<strong>{stage.label}</strong>
				{#if !compact}<span class="short">{stage.short}</span>{/if}
				<span class="more">{stage.key === current ? 'You are here' : 'Read more'}</span>
			</a>
		</li>
	{/each}
</ol>

<style>
	.network-chain {
		--line-y: 0.45rem;
		position: relative;
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 2rem;
		list-style: none;
		padding: 0;
		margin: 2.75rem 0 0;
	}
	/* the line itself, drawn from left to right */
	.network-chain::before {
		content: '';
		position: absolute;
		top: var(--line-y);
		left: 0.45rem;
		right: 0;
		height: 1px;
		background: linear-gradient(90deg, var(--color-accent-ink), var(--color-border-strong) 70%);
		transform-origin: left;
		transition: transform 1100ms cubic-bezier(0.65, 0, 0.35, 1);
	}
	li {
		position: relative;
		display: flex;
		flex-direction: column;
	}
	.node {
		width: 0.9rem;
		height: 0.9rem;
		border: 1px solid var(--color-accent-ink);
		background: var(--color-background);
		transform: rotate(45deg);
		transition:
			background-color 300ms ease,
			transform 500ms cubic-bezier(0.16, 0.7, 0.2, 1);
		transition-delay: calc(250ms + var(--i) * 180ms);
	}
	.current .node,
	li:hover .node {
		background: var(--color-accent-ink);
	}
	a {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
		margin-top: 1.5rem;
		text-decoration: none;
		color: inherit;
	}
	strong {
		font-family: var(--font-display);
		font-variation-settings: 'opsz' 40;
		font-size: 1.4rem;
		font-weight: 450;
		letter-spacing: -0.025em;
	}
	.short {
		max-width: 26ch;
		color: var(--color-muted);
		font-size: 0.92rem;
		line-height: 1.55;
	}
	.more {
		margin-top: 0.6rem;
		color: var(--color-muted);
		font-size: 0.82rem;
		font-weight: 550;
		transition: color 220ms ease;
	}
	.current .more,
	li:hover .more,
	a:focus-visible .more {
		color: var(--color-accent-ink);
	}

	.compact {
		margin-top: 2rem;
	}
	.compact strong {
		font-size: 1.15rem;
	}
	.compact a {
		margin-top: 1.1rem;
	}

	/* Before the chain is revealed: line undrawn, nodes small. */
	:global(.motion-ready) .network-chain:not(:global(.is-revealed))::before {
		transform: scaleX(0);
	}
	:global(.motion-ready) .network-chain:not(:global(.is-revealed)) .node {
		transform: rotate(45deg) scale(0.2);
	}

	/* Narrow: the line turns and runs down the left edge. */
	@media (max-width: 760px) {
		.network-chain {
			grid-template-columns: 1fr;
			gap: 1.75rem;
			padding-left: 2rem;
		}
		.network-chain::before {
			top: 0.45rem;
			bottom: 0.45rem;
			left: 0.45rem;
			right: auto;
			width: 1px;
			height: auto;
			background: linear-gradient(180deg, var(--color-accent-ink), var(--color-border-strong) 70%);
			transform-origin: top;
		}
		:global(.motion-ready) .network-chain:not(:global(.is-revealed))::before {
			transform: scaleY(0);
		}
		.node {
			position: absolute;
			left: -2rem;
			top: 0;
		}
		a,
		.compact a {
			margin-top: -0.2rem;
		}
	}
</style>
