<script lang="ts">import { productLabel } from '$lib/product-language'; const props = $props<{ status: string; label?: string }>(); const status = $derived(props.status); const label = $derived(props.label ?? productLabel(status));</script>
<span class={`status status-${status.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`} aria-label={`Status: ${label}`}>{label}</span>
