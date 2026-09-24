<script lang="ts">
	import { network } from '$lib/network-content';
	let { compact = false }: { compact?: boolean } = $props();
</script>

<ol class:compact class="network-chain" aria-label="From manufacturer to consumer">
	{#each network as stage, index (stage.key)}
		<li>
			<span class="step">{String(index + 1).padStart(2, '0')}</span>
			<a href={`/${stage.key}`}>
				<strong>{stage.label}</strong>
				<span class="short">{stage.short}</span>
				<span class="arrow" aria-hidden="true">→</span>
			</a>
		</li>
	{/each}
</ol>

<style>
	/* A chain, read left to right: each link separated by a hairline rather than
     boxed, so the four stages read as one run instead of four cards. */
	.network-chain {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 0;
		list-style: none;
		padding: 0;
		margin: 2.25rem 0 0;
		border-top: 1px solid var(--color-border);
	}
	li {
		display: flex;
		flex-direction: column;
		padding: 1.6rem 1.5rem 1.6rem 0;
		border-right: 1px solid var(--color-border);
		background: transparent;
	}
	li:last-child {
		border-right: 0;
	}
	li + li {
		padding-left: 1.5rem;
	}
	.step {
		font-family: var(--font-display);
		font-variation-settings: 'opsz' 40;
		font-size: 1.15rem;
		font-weight: 450;
		letter-spacing: -0.02em;
		color: var(--color-accent-ink);
	}
	a {
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
		margin-top: 1.5rem;
		text-decoration: none;
		color: inherit;
	}
	strong {
		font-size: 1.02rem;
		font-weight: 600;
		letter-spacing: -0.012em;
	}
	.short {
		color: var(--color-muted);
		font-size: 0.9rem;
	}
	.arrow {
		margin-top: 0.75rem;
		color: var(--color-muted);
		transition: color 220ms cubic-bezier(0.16, 0.7, 0.2, 1);
	}
	li:hover .arrow,
	a:focus-visible .arrow {
		color: var(--color-accent-ink);
	}

	.compact li {
		padding-block: 1.25rem;
	}
	.compact a {
		margin-top: 1rem;
	}

	@media (max-width: 760px) {
		.network-chain {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
		li {
			padding: 1.25rem 1rem 1.25rem 0;
		}
		li:nth-child(2) {
			border-right: 0;
		}
		li:nth-child(n + 3) {
			border-top: 1px solid var(--color-border);
		}
		li + li {
			padding-left: 0;
		}
		li:nth-child(even) {
			padding-left: 1rem;
		}
	}
	@media (max-width: 420px) {
		.network-chain {
			grid-template-columns: 1fr;
		}
		li,
		li:nth-child(even) {
			border-right: 0;
			border-top: 1px solid var(--color-border);
			padding-left: 0;
		}
		li:first-child {
			border-top: 0;
		}
	}
</style>
