<script lang="ts">
	import type { SaleView } from '$lib/records';
	import { timeLabel } from '$lib/records';
	import { saleNextStep } from '$lib/sale-progress';
	let { view, audience }: { view: SaleView; audience: 'buyer' | 'seller' } = $props();
	const next = $derived(saleNextStep(view));
	const events = $derived(view.timeline ?? []);
	// Name whoever acts from where the reader sits: "You" for their own step,
	// the other side by their role in this sale.
	const actor = $derived.by(() => {
		const own = audience === 'seller' ? 'Seller' : 'Customer';
		if (next.actor === own) return 'You';
		if (next.actor === 'Seller') return 'The seller';
		if (next.actor === 'Customer') return 'Your customer';
		if (next.actor === 'Customer / bank')
			return audience === 'buyer' ? 'You and your bank' : 'Your customer and their bank';
		return next.actor;
	});
</script>

<section class="sale-progress" aria-label="Sale progress">
	<p class="eyebrow">Next step · {actor}</p>
	<h2>{next.title}</h2>
	<p>{next.detail}</p>
	<details>
		<summary>Sale timeline</summary>
		<ol>
			{#each events as event, i (i)}<li><strong>{event.label}</strong><span>{timeLabel(event.at)}</span></li>{:else}<li>
					Recorded dates are unavailable. Refresh the sale before acting.
				</li>{/each}
		</ol>
	</details>
</section>

<style>
	.sale-progress {
		border: 1px solid var(--color-border, var(--color-border));
		border-left: 4px solid var(--color-primary, var(--color-primary));
		padding: 1.25rem;
		margin: 1rem 0;
		background: var(--color-surface, var(--color-surface));
	}
	h2 {
		margin: 0.35rem 0;
		font-size: 1.3rem;
	}
	p {
		line-height: 1.6;
	}
	summary {
		cursor: pointer;
		padding: 0.8rem 0;
	}
	ol {
		padding-left: 1.3rem;
	}
	li {
		padding: 0.45rem 0;
	}
	li span {
		display: block;
		color: var(--color-muted, var(--color-primary));
		font-size: 0.9rem;
	}
</style>
