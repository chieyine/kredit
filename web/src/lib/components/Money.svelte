<script lang="ts">
 import { exactKobo, formatKobo, type KoboValue } from '$lib/money';
 let { amountKobo = 0, currency = 'NGN', abbreviated = false } = $props<{ amountKobo?: KoboValue; currency?: string; abbreviated?: boolean }>();
 const full = $derived(formatKobo(amountKobo,currency));
 const amount = $derived(exactKobo(amountKobo));
 const compact = $derived(amount===null ? full : new Intl.NumberFormat('en-NG',{notation:'compact',maximumFractionDigits:1}).format(amount/100n));
</script>
<span title={full}>{abbreviated && amount!==null ? `${currency==='NGN'?'₦':currency+' '}${compact}` : full}</span>
