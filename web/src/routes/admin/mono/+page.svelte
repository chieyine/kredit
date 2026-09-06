<script lang="ts">
	import { onMount } from 'svelte';

	type MonoStatus = {
		provider: string;
		environment: string;
		mode: string;
		sweep_enabled: boolean;
		partial_sweep_enabled: boolean;
		automatic_collection_enabled: boolean;
		automatic_retry_enabled: boolean;
		secret_key_configured: boolean;
		webhook_secret_configured: boolean;
		redirect_url_configured: boolean;
		redirect_url: string;
		provider_certification_recorded: boolean;
		ready_for_configured_environment: boolean;
		blockers: string[];
	};

	let mono = $state<MonoStatus | null>(null);
	let loading = $state(true);
	let error = $state('');

	async function load() {
		loading = true;
		error = '';
		const response = await fetch('/api/v1/ops/business-policies', { credentials: 'include' });
		const body = await response.json().catch(() => ({}));
		loading = false;
		if (!response.ok) {
			error = body.detail ?? 'Mono configuration could not be loaded.';
			return;
		}
		mono = body.mono ?? null;
	}

	onMount(load);
</script>

<svelte:head><title>Mono integration — Kredit admin</title></svelte:head>
<main class="shell workspace mono-page">
	<header>
		<div><p class="eyebrow">Administration / Integrations</p><h1>Mono operations.</h1><p class="lede">See exactly what Kredit is configured to do with Mono. Secret values never leave the server; Admin only shows whether they are present.</p></div>
		<button onclick={load} disabled={loading}>Refresh</button>
	</header>
	{#if error}<p class="error" role="alert">{error}</p>{:else if loading}<p>Loading Mono configuration…</p>{:else if mono}
		<section class:ready={mono.ready_for_configured_environment} class="status-card">
			<div><span>Current mode</span><strong>{mono.mode}</strong><small>{mono.environment} · {mono.provider}</small></div>
			<div><span>Technical readiness</span><strong>{mono.ready_for_configured_environment ? 'Ready' : 'Needs setup'}</strong><small>{mono.ready_for_configured_environment ? 'No configuration blocker detected.' : `${mono.blockers.length} item${mono.blockers.length === 1 ? '' : 's'} remain.`}</small></div>
		</section>
		<section class="grid" aria-label="Mono configuration status">
			{#each [
				['Sweep', mono.sweep_enabled],
				['Partial sweep', mono.partial_sweep_enabled],
				['Automatic collection', mono.automatic_collection_enabled],
				['Automatic retry', mono.automatic_retry_enabled],
				['Secret key', mono.secret_key_configured],
				['Webhook secret', mono.webhook_secret_configured],
				['Redirect URL', mono.redirect_url_configured],
				['Provider certification', mono.provider_certification_recorded]
			] as item}
				<article><span>{item[0]}</span><strong>{item[1] ? 'Configured' : 'Not configured'}</strong></article>
			{/each}
		</section>
		{#if mono.redirect_url}<section class="card"><h2>Redirect</h2><code>{mono.redirect_url}</code></section>{/if}
		<section class="card"><h2>What still needs attention</h2>{#if mono.blockers.length}<ul>{#each mono.blockers as blocker}<li>{blocker}</li>{/each}</ul>{:else}<p>No technical configuration blocker is currently reported for this environment.</p>{/if}</section>
		<section class="actions"><div><h2>Operational controls</h2><p>Collections, automatic collection, retry policy, notice periods, fees and pilot limits are managed through the audited settings workflow.</p></div><a class="primary" href="/admin/settings">Open business settings →</a></section>
		<section class="security"><strong>Why keys are not displayed here</strong><p>Provider credentials and webhook secrets are write-only deployment secrets. Showing them in a browser would turn an admin account compromise into a provider-credential compromise. Admin therefore reports configured/missing state while the secret itself stays outside page data, logs and API responses.</p></section>
	{/if}
</main>

<style>
	.mono-page>header{display:flex;justify-content:space-between;align-items:end;gap:2rem;padding:2.5rem 0 2rem;border-bottom:3px solid #17181b}.mono-page h1{margin:.5rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3rem,7vw,5.5rem);font-weight:500;line-height:.92;letter-spacing:-.055em}.mono-page header button{padding:.75rem 1rem;border:1px solid #17181b;background:#fff;font-weight:800}.status-card{display:grid;grid-template-columns:1fr 1fr;margin:2rem 0;background:#17181b;color:#fff}.status-card>div{display:grid;gap:.45rem;padding:1.5rem;border-right:1px solid #555}.status-card.ready{background:#153f2f}.status-card span,.grid span{font-size:.72rem;font-weight:800;letter-spacing:.08em;text-transform:uppercase}.status-card strong{font-family:Georgia,serif;font-size:2rem;font-weight:500;text-transform:capitalize}.status-card small{color:#d6d3cc}.grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));border-left:1px solid var(--color-border);border-top:1px solid var(--color-border)}.grid article{display:grid;gap:.75rem;min-height:6.5rem;padding:1rem;border-right:1px solid var(--color-border);border-bottom:1px solid var(--color-border);background:#fffdf8}.grid strong{align-self:end}.card,.actions,.security{margin-top:1.5rem;padding:1.25rem;border:1px solid var(--color-border);background:#fffdf8}.card code{overflow-wrap:anywhere}.actions{display:flex;justify-content:space-between;align-items:center;gap:2rem}.actions p{max-width:40rem;color:var(--color-muted)}.security{border-left:4px solid #2738d6;background:#eef1ff}.security p{max-width:55rem;margin-bottom:0;line-height:1.65}.error{color:#b42318}@media(max-width:800px){.grid{grid-template-columns:1fr 1fr}.actions,.mono-page>header{align-items:flex-start;flex-direction:column}}@media(max-width:480px){.status-card,.grid{grid-template-columns:1fr}.status-card>div{border-right:0;border-bottom:1px solid #555}}
</style>
