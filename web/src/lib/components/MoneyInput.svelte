<script lang="ts">
	let { label = 'Amount', value = $bindable(''), name = 'amount', required = true, disabled = false } = $props<{ label?: string; value?: string; name?: string; required?: boolean; disabled?: boolean }>();
</script>
<label class="field">{label}<input {name} inputmode="decimal" autocomplete="off" {required} {disabled} bind:value aria-label={label} /></label>
