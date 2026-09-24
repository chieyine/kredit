<script lang="ts">
	import { onMount } from 'svelte';
	import { adminGet } from '$lib/admin-client';
	import { record, rows, text } from '$lib/api/reliable';
	type AttentionItem = { label: string; count: number; href: string; action: string };
	function attentionItem(value: unknown): AttentionItem {
		const item = record(value);
		const href = text(item.href);
		// Links come from the server but are rendered as navigation; only an
		// admin-relative path is followed, never another origin or scheme.
		if (!href.startsWith('/admin') || href.startsWith('//')) throw new Error('Attention link could not be verified.');
		if (!Number.isSafeInteger(item.count) || Number(item.count) < 0)
			throw new Error('Attention count could not be verified.');
		return { label: text(item.label), count: Number(item.count), href, action: text(item.action) };
	}
	let items: AttentionItem[] = $state([]),
		error = $state('');
	async function load() {
		error = '';
		items = [];
		try {
			const b = await adminGet('/api/v1/ops/attention');
			if (!Array.isArray(b.items)) throw new Error('Attention list could not be verified.');
			items = rows('items', attentionItem)(b);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Attention list unavailable';
		}
	}
	onMount(load);
</script>

<section aria-label="Priority work">
	<h2>Work needing attention</h2>
	{#if error}<p role="alert">{error} <button onclick={load}>Try again</button></p>{/if}
	<div>
		{#each items as item (item.label)}<a href={item.href}
				><strong>{item.count}</strong><b>{item.label}</b><span>{item.action} →</span></a
			>{/each}
	</div>
</section>

<style>
	section {
		margin: 2rem 0;
	}
	section > div {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
		gap: 1rem;
	}
	a {
		display: grid;
		gap: 0.6rem;
		border: 1px solid var(--color-border);
		background: var(--color-surface);
		padding: 1rem;
		color: var(--color-primary);
		text-decoration: none;
	}
	strong {
		font: 2rem var(--font-serif);
	}
	span {
		font-size: 0.85rem;
	}
</style>
