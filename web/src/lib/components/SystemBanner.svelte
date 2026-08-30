<script lang="ts">let { message, tone = 'info' } = $props<{ message: string; tone?: string }>();</script>
<aside class={`system-banner ${tone}`} role="status">{message}</aside>
