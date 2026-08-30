<script lang="ts">
	import { onMount } from 'svelte';
	let scorecard: any = null;
	let error = '';
	let loading = true;
	let to = new Date().toISOString().slice(0, 10);
	let from = new Date(Date.now() - 7 * 86400000).toISOString().slice(0, 10);
	let organization = '';
	const format = (metric: any) => metric.unit === 'percent' ? `${metric.value.toFixed(1)}%` : metric.unit === 'kobo' ? new Intl.NumberFormat('en-NG',{style:'currency',currency:'NGN'}).format(metric.value/100) : metric.unit === 'cases_per_100_active_suppliers' ? `${metric.value.toFixed(1)} per 100` : `${metric.value.toFixed(metric.unit === 'hours' || metric.unit === 'days' ? 1 : 0)} ${metric.unit}`;
	async function load() {
		loading = true; error = '';
		const params = new URLSearchParams({from,to});
		if (organization.trim()) params.set('organization_id', organization.trim());
		const response = await fetch(`/api/v1/ops/analytics/scorecard?${params}`, {credentials:'include'});
		const body = await response.json().catch(()=>({}));
		if (!response.ok) error = body.detail ?? 'The pilot scorecard is unavailable.';
		else scorecard = body.scorecard;
		loading = false;
	}
	onMount(load);
</script>

<svelte:head><title>Pilot scorecard — Kredit</title></svelte:head>
<main class="shell workspace analytics">
	<p class="eyebrow">Operations / Pilot scorecard</p>
	<h1>Know whether the pilot is healthy.</h1>
	<p class="lede">Live product KPIs come from authoritative trade, payment and dispute records. Privacy-minimised events prove the funnel and expose reconciliation gaps.</p>
	<form onsubmit={(event)=>{event.preventDefault();load()}} aria-label="Scorecard filters">
		<label>From<input type="date" bind:value={from} required /></label>
		<label>To<input type="date" bind:value={to} required /></label>
		<label>Supplier organisation UUID (optional)<input bind:value={organization} autocomplete="off" /></label>
		<button disabled={loading}>{loading?'Refreshing…':'Apply filters'}</button>
	</form>
	{#if error}<p class="error" role="alert">{error}</p>{:else if loading}<p role="status">Calculating the live scorecard…</p>{:else if scorecard}
		<div class="status" class:healthy={scorecard.reconciliation_ok}><strong>{scorecard.reconciliation_ok?'Reconciled':'Review required'}</strong><span>{scorecard.refresh_mode} · generated {new Date(scorecard.generated_at).toLocaleString()}</span></div>
		<h2>Primary pilot KPIs</h2>
		<section class="metrics" aria-label="Primary pilot KPIs">{#each scorecard.kpis as metric}<article><strong>{format(metric)}</strong><h3>{metric.label}</h3><p>{metric.definition}</p><small>Source: {metric.source} · target: baseline required</small></article>{/each}</section>
		<h2>Drivers</h2>
		<section class="metrics" aria-label="KPI drivers">{#each scorecard.drivers as metric}<article><strong>{format(metric)}</strong><h3>{metric.label}</h3><p>{metric.definition}</p><small>Source: {metric.source}</small></article>{/each}</section>
		<h2>Safety guardrails</h2>
		<section class="metrics" aria-label="Pilot guardrails">{#each scorecard.guardrails as metric}<article><strong>{format(metric)}</strong><h3>{metric.label}</h3><p>{metric.definition}</p><small>Source: {metric.source}</small></article>{/each}</section>
		<h2>Event reconciliation</h2>
		<div class="table-wrap"><table><caption>Zero tolerance between reconstructable source facts and product events</caption><thead><tr><th>Event</th><th>Source</th><th>Events</th><th>Status</th></tr></thead><tbody>{#each scorecard.reconciliation as row}<tr><th>{row.event}</th><td>{row.source_count}</td><td>{row.event_count}</td><td><span class:ok={row.status==='reconciled'}>{row.status}</span></td></tr>{/each}</tbody></table></div>
	{/if}
</main>

<style>
	.analytics{padding-bottom:4rem}.analytics form{display:grid;grid-template-columns:repeat(auto-fit,minmax(11rem,1fr));gap:1rem;align-items:end;margin:2rem 0;padding:1rem;border:1px solid var(--color-border);border-radius:0}.analytics label{display:grid;gap:.4rem;font-weight:700}.analytics input,.analytics button{min-height:2.75rem;border:1px solid var(--color-border);border-radius:0;padding:.6rem}.analytics button{background:#2738d6;color:white;font-weight:800}.status{display:flex;justify-content:space-between;gap:1rem;flex-wrap:wrap;padding:1rem;border-left:.35rem solid #ad641b;background:#fff4e6}.status.healthy{border-color:#267054;background:#eaf7f1}.status span{color:var(--color-muted)}.metrics{display:grid;grid-template-columns:repeat(auto-fit,minmax(15rem,1fr));gap:1rem;margin-bottom:2rem}.metrics article{border:1px solid var(--color-border);border-radius:0;padding:1.15rem;background:white}.metrics article>strong{font-size:1.55rem;color:#2738d6}.metrics h3{margin:.45rem 0}.metrics p{min-height:3.8rem}.metrics small{color:var(--color-muted)}.table-wrap{overflow-x:auto}table{border-collapse:collapse;width:100%;background:white}caption{text-align:left;padding:.75rem 0}th,td{text-align:left;padding:.75rem;border-bottom:1px solid var(--color-border)}td span{color:#9b351f;font-weight:800}.ok{color:#267054}@media(max-width:40rem){.metrics p{min-height:0}}
</style>
