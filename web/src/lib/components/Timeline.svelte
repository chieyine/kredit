<script lang="ts">let { items = [] } = $props<{ items?: Array<{ title: string; detail?: string; at?: string }> }>();</script>
<ol class="timeline" aria-label="Timeline">{#each items as item}<li><strong>{item.title}</strong>{#if item.detail}<span>{item.detail}</span>{/if}{#if item.at}<time>{item.at}</time>{/if}</li>{/each}</ol>
