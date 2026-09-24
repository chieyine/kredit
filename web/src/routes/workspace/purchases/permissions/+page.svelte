<script lang="ts">
	import { nairaInput } from '$lib/money';
	import { onMount } from 'svelte';
	import { checkedJSON, LatestRequest, record, rows, text } from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	import { requestedWorkspace } from '$lib/workspace-context';
	import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
	type Grant = {
		active: boolean;
		user_id: string;
		name: string;
		actions: string[];
		ceiling_kobo: number;
		drawdown_ceiling_kobo: number;
		expires_at: string;
		version: number;
		limit: string;
		drawdown_limit: string;
		expiry: string;
	};
	let organization = $state(''),
		businesses = $state<{ id: string; name: string }[]>([]),
		grants = $state<Grant[]>([]),
		owner = $state(false),
		loading = $state(true),
		busy = $state(false),
		error = $state(''),
		message = $state('');
	const reads = new LatestRequest();
	const base = () => `/api/v1/organizations/${encodeURIComponent(organization)}/purchasing-authority`;
	function grant(value: unknown): Grant {
		const g = record(value);
		if (
			!Array.isArray(g.actions) ||
			!g.actions.every((a) => typeof a === 'string') ||
			!Number.isSafeInteger(g.ceiling_kobo) ||
			!Number.isSafeInteger(g.version)
		)
			throw new Error('Permissions could not be verified.');
		const expiry = text(g.expires_at);
		const ddCeiling = Number(g.drawdown_ceiling_kobo ?? 0);
		return {
			active: g.active === true,
			user_id: text(g.user_id),
			name: text(g.name),
			actions: g.actions as string[],
			ceiling_kobo: g.ceiling_kobo as number,
			drawdown_ceiling_kobo: ddCeiling,
			version: g.version as number,
			expires_at: expiry,
			limit: nairaInput(g.ceiling_kobo as number),
			drawdown_limit: nairaInput(ddCeiling),
			expiry:
				Date.parse(expiry) > Date.now()
					? expiry.slice(0, 10)
					: new Date(Date.now() + 90 * 86400000).toISOString().slice(0, 10)
		};
	}
	async function load() {
		const request = reads.begin();
		loading = true;
		error = '';
		grants = [];
		owner = false;
		try {
			const available = await checkedJSON(
				'/api/v1/organizations',
				rows('organizations', (value) => {
					const b = record(value);
					return {
						id: text(b.id),
						name: typeof b.trading_name === 'string' && b.trading_name ? b.trading_name : text(b.legal_name)
					};
				}),
				{ signal: request.signal }
			);
			if (!request.current()) return;
			businesses = available;
			organization = requestedWorkspace(businesses);
			if (!organization) return;
			const result = await checkedJSON(
				base(),
				(value) => {
					const b = record(value);
					if (typeof b.can_manage !== 'boolean') throw new Error('Permission status unavailable.');
					return { owner: b.can_manage, grants: rows('grants', grant)(b) };
				},
				{ signal: request.signal }
			);
			if (!request.current()) return;
			owner = result.owner;
			grants = result.grants;
		} catch (cause) {
			if (request.current()) error = cause instanceof Error ? cause.message : 'Permissions could not be loaded.';
		} finally {
			if (request.current()) loading = false;
		}
	}
	function toggle(g: Grant, action: string, enabled: boolean) {
		g.actions = enabled
			? [...new Set([...g.actions, 'read', action])]
			: action === 'read'
				? []
				: g.actions.filter((a) => a !== action);
	}
	async function save(g: Grant, revoke = false) {
		if (busy || !owner) return;
		error = '';
		message = '';
		let kobo = g.ceiling_kobo,
			ddKobo = g.drawdown_ceiling_kobo,
			expiry = new Date(g.expires_at);
		if (!revoke) {
			if (!/^\d+(\.\d{1,2})?$/.test(g.limit)) {
				error = 'Enter a purchase limit in naira with at most two decimal places.';
				return;
			}
			const [whole, fraction = ''] = g.limit.split('.');
			kobo = Number(whole) * 100 + Number(fraction.padEnd(2, '0'));
			if (!Number.isSafeInteger(kobo) || kobo < 0) {
				error = 'Enter a valid purchase limit.';
				return;
			}

			if (!/^\d+(\.\d{1,2})?$/.test(g.drawdown_limit)) {
				error = 'Enter a drawdown limit in naira with at most two decimal places.';
				return;
			}
			const [dWhole, dFraction = ''] = g.drawdown_limit.split('.');
			ddKobo = Number(dWhole) * 100 + Number(dFraction.padEnd(2, '0'));
			if (!Number.isSafeInteger(ddKobo) || ddKobo < 0) {
				error = 'Enter a valid drawdown limit.';
				return;
			}

			expiry = new Date(`${g.expiry}T23:59:59+01:00`);
		}
		if (!Number.isFinite(expiry.getTime())) {
			error = 'Choose an expiry date.';
			return;
		}
		busy = true;
		try {
			await new MutationIntent(
				`purchasing:${organization}:${g.user_id}`,
				`${base()}/${encodeURIComponent(g.user_id)}`
			).run(
				{
					actions: revoke ? [] : g.actions,
					ceiling_kobo: kobo,
					drawdown_ceiling_kobo: ddKobo,
					expires_at: expiry.toISOString(),
					version: g.version
				},
				record,
				'PUT'
			);
			message = revoke ? 'Purchasing access removed.' : 'Purchasing permissions saved.';
			await load();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'The result could not be confirmed. Refresh before retrying.';
		} finally {
			busy = false;
		}
	}

	onMount(() => {
		void load();
		return () => reads.cancel();
	});
</script>

<svelte:head><title>Purchasing permissions — Kredit</title></svelte:head>
<main class="shell workspace permissions">
	<p class="eyebrow">Purchases</p>
	<h1>Choose who can buy for your business</h1>
	<p>
		Give current team members specific purchasing permissions, a limit for each accepted purchase and drawdown, and an
		expiry date. Personal purchases stay private.
	</p>
	<VerifyIdentity />
	{#if error}<p role="alert">{error}</p>
		<button disabled={busy} onclick={load}>Refresh permissions</button>{/if}{#if message}<p role="status">
			{message}
		</p>{/if}

	{#if loading}<p role="status">Loading purchasing permissions…</p>{:else if !organization && !error}<p>
			<a href="/workspace/onboarding">Set up a business first.</a>
		</p>{:else if !error}
		<p>
			Only an owner can change these permissions. The recorded purchasing owner keeps their existing authority.
			Accepting terms still requires current identity and authority verification. Bank mandates remain under the
			recorded owner’s control. Removing staff membership or revoking a grant blocks future delegated actions.
		</p>
		{#if !grants.length}<p>
				No current team members. <a href={`/workspace/team?organization=${encodeURIComponent(organization)}`}
					>Open your team</a
				>.
			</p>{/if}
		{#each grants as g (g.user_id)}<section class="card">
				<h2>{g.name}</h2>
				<p>
					{g.active
						? 'Delegated access expires ' + new Date(g.expires_at).toLocaleDateString('en-NG')
						: 'No current delegated purchasing access'}
				</p>
				<fieldset disabled={busy || !owner}>
					<div class="actions-grid">
						{#each [['read', 'View business purchases'], ['review', 'Review or decline offers'], ['accept', 'Accept verified purchase terms'], ['receive', 'Confirm delivery or report an issue'], ['drawdown', 'Draw down trade line credit'], ['dispute', 'Open or manage purchase disputes'], ['claim', 'Submit or confirm payment claims'], ['amend', 'Request or review amendments']] as [action, label] (action)}<label
								class="check"
								><input
									type="checkbox"
									checked={g.actions.includes(action)}
									onchange={(e) => toggle(g, action, e.currentTarget.checked)}
								/>{label}</label
							>{/each}
					</div>
					<label
						>Maximum per accepted purchase (₦)<input
							bind:value={g.limit}
							inputmode="decimal"
							autocomplete="off"
						/></label
					>
					<label
						>Maximum per trade line drawdown (₦)<input
							bind:value={g.drawdown_limit}
							inputmode="decimal"
							autocomplete="off"
						/></label
					>
					<label>Access expiry<input type="date" bind:value={g.expiry} /></label>
					{#if owner}<div class="actions">
							<button onclick={() => save(g)}>Save permissions for {g.name}</button><button
								onclick={() => save(g, true)}>Remove delegated access</button
							>
						</div>{/if}
				</fieldset>
			</section>{/each}{/if}
</main>

<style>
	.permissions {
		max-width: 58rem;
	}
	.card {
		padding: 1.4rem;
		margin: 1rem 0;
		border: 1px solid var(--color-border);
		border-radius: 1rem;
		background: var(--color-surface);
	}
	fieldset {
		border: 0;
		padding: 0;
		min-width: 0;
	}
	label {
		display: grid;
		gap: 0.4rem;
		margin: 0.8rem 0;
	}
	.check {
		display: flex;
		align-items: center;
		gap: 0.6rem;
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.7rem;
	}
	input,
	button {
		font: inherit;
		padding: 0.7rem;
		max-width: 100%;
		box-sizing: border-box;
		border: 1px solid var(--color-border);
		border-radius: 0.5rem;
		background: var(--color-surface);
		color: inherit;
	}
	button:disabled {
		opacity: 0.6;
	}
	p {
		line-height: 1.6;
	}
</style>
