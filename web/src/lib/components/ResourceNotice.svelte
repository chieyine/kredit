<script lang="ts">
	import type { Resource } from '$lib/api/reliable';
	let { resource, label, retry }: { resource: Resource<unknown>; label: string; retry?: () => void } = $props();
</script>

{#if resource.state === 'loading'}
	<p class="resource-loading" role="status">Checking {label}…</p>
{:else if resource.state === 'error'}
	<div class="resource-error" role="alert">
		<div>
			<strong>{label} unavailable</strong>
			<p>{resource.message}</p>
		</div>
		{#if retry}<button type="button" onclick={retry}>Try again</button>{/if}
	</div>
{/if}

<style>
	.resource-loading {
		color: var(--color-muted);
		padding: 0.75rem 0;
		line-height: 1.5;
	}
	.resource-error {
		display: flex;
		align-items: center;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: 0.75rem;
		/* the same shape as every other notice: a rule on the left, a tint */
		padding: 0.85rem 1rem;
		border-left: 2px solid var(--color-warning);
		background: rgb(138 82 16 / 0.07);
		color: var(--color-warning);
		margin: 1rem 0;
	}
	.resource-error p {
		margin: 0.25rem 0 0;
		font-size: 0.9rem;
		line-height: 1.5;
	}
	.resource-error button {
		background: transparent;
		color: inherit;
		border: 1px solid currentColor;
		min-height: 2.75rem;
		padding: 0.55rem 0.8rem;
		cursor: pointer;
	}
</style>
