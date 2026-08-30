<script lang="ts">
	let { amountKobo = 0, currency = 'NGN', abbreviated = false } = $props<{ amountKobo?: number | string; currency?: string; abbreviated?: boolean }>();
	const format = (value: number) => new Intl.NumberFormat('en-NG', { style: 'currency', currency, maximumFractionDigits: 2 }).format(value / 100);
	const full = $derived(format(Number(amountKobo) || 0));
	const compact = $derived(new Intl.NumberFormat('en-NG', { notation: 'compact', maximumFractionDigits: 1 }).format((Number(amountKobo) || 0) / 100));
</script>
<span title={full}>{abbreviated ? `${currency === 'NGN' ? '₦' : currency + ' '}${compact}` : full}</span>
