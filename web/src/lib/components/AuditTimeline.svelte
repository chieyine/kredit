<script lang="ts">import Timeline from './Timeline.svelte'; let { items = [] } = $props<{ items?: Array<{ action: string; actor?: string; at?: string }> }>();</script>
<Timeline items={items.map((item: { action: string; actor?: string; at?: string }) => ({ title: item.action, detail: item.actor, at: item.at }))} />
