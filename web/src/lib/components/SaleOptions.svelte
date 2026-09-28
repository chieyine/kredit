<script lang="ts">
	import type { SaleOptions } from '$lib/sale-options';
	let { options = $bindable(), disabled = false }: { options: SaleOptions; disabled?: boolean } = $props();
</script>

<details>
	<summary>Add an invoice or split payments</summary>
	<fieldset {disabled}>
		<label>Invoice reference (optional)<input bind:value={options.invoiceReference} maxlength="200" /></label>
		<label
			>Invoice file (optional)<input
				type="file"
				accept="application/pdf,image/jpeg,image/png"
				onchange={(event) => (options.invoiceFile = event.currentTarget.files?.[0] ?? null)}
			/><small>PDF, JPG or PNG, up to 2 MB.</small></label
		>
		<label
			>Payment plan<select bind:value={options.scheduleType}
				><option value="one_time">One payment</option><option value="equal">Equal payments</option><option
					value="custom">Different amounts or dates</option
				></select
			></label
		>
		{#if options.scheduleType === 'equal'}
			<label>Number of payments<input type="number" min="2" max="60" bind:value={options.count} /></label>
			<label
				>How often?<select bind:value={options.cadence}
					><option value="weekly">Weekly</option><option value="fortnightly">Every two weeks</option><option
						value="monthly">Monthly</option
					></select
				></label
			>
			<p>The date above is the first payment date. Monthly dates stop at the last day of a shorter month.</p>
		{:else if options.scheduleType === 'custom'}
			<label
				>Payments<textarea rows="4" bind:value={options.custom} placeholder="50000, 2026-10-01&#10;100000, 2026-11-01"
				></textarea></label
			>
			<p>One amount and date per line. They must add up to the sale amount and start on the date above.</p>
		{/if}
	</fieldset>
</details>

<style>
	fieldset {
		border: 0;
		padding: 1rem 0;
		display: grid;
		gap: 1rem;
		min-width: 0;
	}
	label {
		display: grid;
		gap: 0.4rem;
	}
	input,
	select,
	textarea {
		width: 100%;
		min-height: 44px;
	}
	summary {
		cursor: pointer;
		min-height: 44px;
		padding-block: 0.75rem;
	}
</style>
