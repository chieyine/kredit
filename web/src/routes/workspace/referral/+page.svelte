<script lang="ts">
	import { onMount } from 'svelte';
	import { read, Mutation } from '$lib/consumer';
	import { record } from '$lib/api/reliable';
	import { organization, type Organization } from '$lib/records';
	let code = $state(''),
		org = $state(''),
		name = $state(''),
		businesses = $state<Organization[]>([]),
		consent = $state(false),
		busy = $state(false),
		error = $state(''),
		message = $state('');
	const mutation = new Mutation();
	function clearSavedReferral() {
		try {
			sessionStorage.removeItem('kredit_dsa_referral');
		} catch {
			// Local storage cleanup cannot change the confirmed server result.
		}
	}
	async function load() {
		busy = true;
		error = '';
		name = '';
		consent = false;
		try {
			const v = record(await read('/api/v1/organizations'));
			if (!Array.isArray(v.organizations)) throw new Error('Could not load businesses.');
			businesses = v.organizations.map(organization);
			org = businesses[0]?.id ?? '';
			if (code) name = String(record(await read('/api/v1/dsa/code/' + encodeURIComponent(code))).name);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not load referral.';
		} finally {
			busy = false;
		}
	}
	async function claim() {
		busy = true;
		error = '';
		try {
			await mutation.send(`/api/v1/organizations/${encodeURIComponent(org)}/dsa`, { code, consent }, (value) => {
				const result = record(value);
				if (result.saved !== true) throw new Error('Referral confirmation was incomplete.');
				return result;
			});
			message = 'Referral confirmed. Finish your business setup; Kredit checks reward qualification automatically.';
			clearSavedReferral();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Referral was not confirmed.';
		} finally {
			busy = false;
		}
	}
	async function retry() {
		busy = true;
		try {
			await mutation.retry();
			message = 'Referral confirmed.';
			clearSavedReferral();
			error = '';
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not confirm referral.';
		} finally {
			busy = false;
		}
	}
	onMount(() => {
		const linkedCode = new URLSearchParams(location.search).get('code');
		code = linkedCode ?? '';
		try {
			code = linkedCode ?? sessionStorage.getItem('kredit_dsa_referral') ?? '';
			if (code) sessionStorage.setItem('kredit_dsa_referral', code);
		} catch {
			// The URL or a manually entered code works without browser storage.
		}
		void load();
	});
</script>

<svelte:head><title>Confirm your referral — Kredit</title></svelte:head>
<main class="shell workspace feature-page referral-confirmation">
	<header class="feature-heading">
		<div>
			<p class="eyebrow">Your business · Introduction</p>
			<h1>Confirm your introduction.</h1>
			<p class="lede">Connect your business to the person who helped you discover Kredit.</p>
		</div>
	</header>
	<section>
		<p class="eyebrow">One quick confirmation</p>
		<h2>Who introduced your business?</h2>
		<p>
			Only the business owner can confirm this. Your agent sees the business name and referral progress and may earn
			rewards. Your customer records stay private.
		</p>
		{#if error}<p role="alert">{error}</p>
			{#if mutation.pending}<button onclick={retry} disabled={busy}>Retry the same confirmation</button
				>{/if}{/if}{#if message}<p role="status">{message}</p>
			<a href={`/workspace/onboarding?organization=${encodeURIComponent(org)}`}>Finish business setup</a>{:else}<form
				onsubmit={(e) => {
					e.preventDefault();
					void claim();
				}}
			>
				<fieldset disabled={busy}>
					<label
						>Referral code<input
							bind:value={code}
							oninput={() => {
								name = '';
								consent = false;
							}}
							required
							maxlength="40"
						/></label
					><button type="button" onclick={load}>Check code and refresh businesses</button>{#if name}<p>
							Agent: {name}
						</p>{/if}<label
						>Business<select bind:value={org} required
							>{#each businesses as b (b.id)}<option value={b.id}>{b.legal_name}</option>{/each}</select
						></label
					><label
						><input type="checkbox" bind:checked={consent} required />This agent introduced my business to Kredit. I
						agree to referral attribution and progress sharing.</label
					><button class="primary" disabled={!org || !name}>Confirm introduction</button>
				</fieldset>
			</form>{/if}
		<p>
			<a href="/workspace/today">Create your business if it is not listed</a>. Return here before accepting your first
			sale.
		</p>
		<a href="/account/security">Set up or confirm extra sign-in safety</a>
	</section>
</main>
