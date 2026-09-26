<script lang="ts">
	import { page } from '$app/state';
	import { workspaceHref } from '$lib/workspace-navigation';
	let {
		title,
		description,
		groups,
		eyebrow = 'Your business workspace'
	}: {
		title: string;
		description: string;
		eyebrow?: string;
		groups: { title: string; description: string; links: [string, string, string][] }[];
	} = $props();
</script>

<svelte:head><title>{title} — Kredit</title></svelte:head>
<main class="shell k-page workspace-hub">
	<header class="k-head">
		<div>
			{#if eyebrow}<p class="k-eyebrow">{eyebrow}</p>{/if}
			<h1>{title}</h1>
			{#if description}<p>{description}</p>{/if}
		</div>
	</header>
	{#each groups as group, groupIndex (groupIndex)}<section class="k-section">
			<div class="k-section-head">
				<h2>{group.title}</h2>
			</div>
			{#if group.description}<p class="group-note">{group.description}</p>{/if}
			<div class="k-ledger hub-cards">
				{#each group.links as [label, href, body], linkIndex (linkIndex)}<a
						class="hub-row"
						href={workspaceHref(href, page.url)}
						><span class="text"
							><h3>{label}</h3>
							<small>{body}</small></span
						><span aria-hidden="true">→</span></a
					>{/each}
			</div>
		</section>{/each}
</main>

<style>
	.workspace-hub .k-section:first-of-type {
		margin-top: 0;
	}
	.group-note {
		margin: -0.4rem 0 1rem;
		color: var(--color-muted);
		line-height: 1.55;
	}
	.hub-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		min-height: 4.5rem;
		padding: 1rem 1.25rem;
		border-top: 1px solid var(--color-border);
		color: inherit;
		text-decoration: none;
		transition: background-color 160ms ease;
	}
	.hub-row:first-child {
		border-top: 0;
	}
	.hub-row:hover {
		background: var(--color-surface-muted);
	}
	.text {
		display: grid;
		gap: 0.25rem;
	}
	h3 {
		margin: 0;
		font-size: 1.02rem;
	}
	small {
		color: var(--color-muted);
		font-size: 0.9rem;
		line-height: 1.45;
	}
	.hub-row > span:last-child {
		color: var(--color-primary);
		font-weight: 600;
	}
</style>
