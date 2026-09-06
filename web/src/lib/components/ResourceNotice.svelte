<script lang="ts">
  import type { Resource } from '$lib/api/reliable';
  let { resource, label, retry }: { resource: Resource<unknown>; label: string; retry?: () => void } = $props();
</script>
{#if resource.state === 'loading'}
  <p class="resource-loading" role="status">Checking {label}…</p>
{:else if resource.state === 'error'}
  <div class="resource-error" role="alert"><div><strong>{label} unavailable</strong><p>{resource.message}</p></div>{#if retry}<button type="button" onclick={retry}>Try again</button>{/if}</div>
{/if}
<style>
  .resource-loading{color:var(--color-muted);padding:.75rem 0;line-height:1.5}.resource-error{display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:.75rem;padding:1rem;background:#fff3e0;border:1px solid #d4b485;border-radius:.4rem;color:#603f16;margin:1rem 0}.resource-error p{margin:.25rem 0 0;font-size:.9rem;line-height:1.5}.resource-error button{background:transparent;color:inherit;border:1px solid currentColor;min-height:2.75rem;padding:.55rem .8rem;border-radius:.3rem;cursor:pointer}
</style>
