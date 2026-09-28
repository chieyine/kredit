<script lang="ts">
	import type { PersonalOptions } from '$lib/personal-sale-options';
	let { options = $bindable(), disabled = false }: { options: PersonalOptions; disabled?: boolean } = $props();
</script>

<details>
	<summary>Payment and delivery options</summary>
	<fieldset {disabled}>
		<label
			>Invite by<select bind:value={options.channel}
				><option value="phone">Phone number</option><option value="email">Email</option></select
			></label
		>
		<label>Quantity<input type="number" min="1" max="10000" bind:value={options.quantity} /></label>
		<label>Number of payments<input type="number" min="1" max="60" bind:value={options.count} /></label>
		{#if options.count > 1}<label
				>How often?<select bind:value={options.cadence}
					><option value="weekly">Weekly</option><option value="fortnightly">Every two weeks</option><option
						value="monthly">Monthly</option
					></select
				></label
			>{/if}
		<label>Deposit (₦, optional)<input inputmode="decimal" bind:value={options.deposit} /></label>
		{#if options.deposit && options.deposit !== '0'}<label
				>Deposit date<input type="date" bind:value={options.depositDate} /></label
			>{/if}
		<label
			>When will you hand over the goods?<select bind:value={options.fulfillment}
				><option value="immediate">After acceptance</option><option value="on_full_payment">After full payment</option
				><option value="on_percentage">After a part-payment</option></select
			></label
		>
		{#if options.fulfillment === 'on_percentage'}<label
				>Percentage to pay first<input type="number" min="1" max="99" bind:value={options.threshold} /></label
			>{/if}
		<label>Delivery within (days)<input type="number" min="1" max="90" bind:value={options.days} /></label>
		<label
			>Return and refund terms<textarea rows="4" minlength="20" maxlength="2000" bind:value={options.returns}
			></textarea></label
		>
		<p>
			The date above is the first payment date. Monthly payments stop at month-end. Deposits must be due before the
			first payment.
		</p>
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
