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
<main class="shell workspace workspace-hub">
	<header>
		<p class="eyebrow">{eyebrow}</p>
		<h1>{title}</h1>
		<p class="lede">{description}</p>
	</header>
	{#each groups as group, groupIndex (groupIndex)}<section>
			<h2>{group.title}</h2>
			<p>{group.description}</p>
			<div class="hub-cards">
				{#each group.links as [label, href, body], linkIndex (linkIndex)}<a href={workspaceHref(href, page.url)}
						><h3>{label}</h3>
						<p>{body}</p>
						<span aria-hidden="true">→</span></a
					>{/each}
			</div>
		</section>{/each}
</main>

<style>
	.workspace-hub {
		max-width: 66rem;
		padding-bottom: 3rem;
	}
	header {
		padding-block: 1.5rem;
	}
	h1 {
		font-size: clamp(2rem, 4vw, 3rem);
		letter-spacing: -0.035em;
		line-height: 1.1;
		margin: 0.6rem 0;
	}
	h2 {
		font-size: 1.3rem;
		margin: 0 0 0.5rem;
	}
	h3 {
		font-size: 1.1rem;
		margin: 0;
	}
	p {
		color: var(--color-muted);
		line-height: 1.65;
	}
	.lede {
		max-width: 65ch;
	}
	.workspace-hub section {
		padding-block: 1.5rem;
		border-top: 1px solid var(--color-border);
	}
	.hub-cards {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 1rem;
		margin-top: 1.25rem;
	}
	.hub-cards a {
		padding: 1.5rem;
		border: 1px solid var(--color-border);
		border-radius: 0.8rem;
		background: var(--color-surface);
		text-decoration: none;
	}
	.hub-cards a:hover {
		border-color: var(--color-primary);
	}
	.hub-cards span {
		color: var(--color-primary);
		font-size: 1.2rem;
	}
	.hub-cards p {
		font-size: 0.95rem;
		margin: 0.6rem 0 1rem;
	}
	@media (max-width: 760px) {
		.hub-cards {
			grid-template-columns: 1fr;
		}
	}
</style>
