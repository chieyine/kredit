<script lang="ts">
	let { rows = 3, tall = false }: { rows?: number; tall?: boolean } = $props();
</script>

<div class="skeleton" role="status" aria-label="Loading">
	{#each Array(rows) as _}
		<span class:tall></span>
	{/each}
	<span class="sr-only">Loading…</span>
</div>

<style>
	.skeleton { display: grid; gap: 0.75rem; }
	.skeleton span {
		height: 3.25rem;
		border-radius: var(--radius-md);
		background: linear-gradient(90deg, var(--color-surface-muted) 25%, var(--color-border) 45%, var(--color-surface-muted) 65%);
		background-size: 200% 100%;
		animation: shimmer 1.4s ease-in-out infinite;
	}
	.skeleton span.tall { height: 7rem; }
	@keyframes shimmer {
		from { background-position: 200% 0; }
		to { background-position: -200% 0; }
	}
	@media (prefers-reduced-motion: reduce) {
		.skeleton span { animation: none; }
	}
</style>
