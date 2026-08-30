<script lang="ts">import DateTime from './DateTime.svelte'; let { value, timezone = 'Africa/Lagos' } = $props<{ value: string | Date; timezone?: string }>();</script>
<span class="due-date"><DateTime {value} {timezone} /></span>
