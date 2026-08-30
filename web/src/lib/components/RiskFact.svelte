<script lang="ts">let { label, value, tone = 'neutral' } = $props<{ label: string; value: string; tone?: string }>();</script>
<div class={`risk-fact ${tone}`}><span>{label}</span><strong>{value}</strong></div>
