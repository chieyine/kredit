<script lang="ts">
	import { checkedJSON, LatestRequest, publicError, record, text } from '$lib/api/reliable';
	let { organizationID }: { organizationID: string } = $props();
	let ready = $state(false),
		loading = $state(true),
		error = $state(''),
		missing = $state<{ label: string; href: string }[]>([]);
	const reads = new LatestRequest();
	async function load(scope: string) {
		const request = reads.begin();
		loading = true;
		error = '';
		missing = [];
		ready = false;
		try {
			const result = await checkedJSON(
				`/api/v1/organizations/${encodeURIComponent(scope)}/onboarding`,
				(value) => {
					const body = record(record(value).readiness);
					if (typeof body.ready !== 'boolean' || !Array.isArray(body.missing)) throw new Error('Incomplete readiness');
					const steps = body.missing.map((value) => {
						const item = record(value),
							href = text(item.manage_path);
						// eslint-disable-next-line no-control-regex -- the control characters are the input being rejected
						if (!(href.startsWith('/workspace/') || href.startsWith('/account/')) || /[\\\x00-\x1f]/.test(href))
							throw new Error('Invalid setup path');
						return { label: text(item.label), href };
					});
					if (body.ready !== (steps.length === 0)) throw new Error('Unconfirmed readiness');
					return { ready: body.ready, steps };
				},
				{ signal: request.signal }
			);
			if (request.current()) {
				ready = result.ready;
				missing = result.steps;
			}
		} catch (cause) {
			if (request.current()) error = publicError(cause, 'your business setup');
		} finally {
			if (request.current()) loading = false;
		}
	}
	$effect(() => {
		if (organizationID) void load(organizationID);
		return () => reads.cancel();
	});
	const query = $derived(`?organization=${encodeURIComponent(organizationID)}`);
</script>

<section class="next-steps" aria-label="Your next step">
	{#if loading}<p role="status">Checking your next setup step…</p>
	{:else if error}<h2>Check your business setup</h2>
		<p>{error}</p>
		<button onclick={() => load(organizationID)}>Check again</button><a href={`/workspace/onboarding${query}`}
			>Open setup →</a
		>
	{:else if !ready}<p class="eyebrow">Before your first live sale</p>
		<h2>Finish your selling setup.</h2>
		<p>
			{missing.length} step{missing.length === 1 ? '' : 's'} remaining. You can prepare your customer network while completing
			the required checks.
		</p>
		{#if missing[0]}<a
				class="primary"
				href={`${missing[0].href}${missing[0].href.includes('?') ? '&' : '?'}organization=${encodeURIComponent(organizationID)}`}
				>{missing[0].label} →</a
			>{/if}<a href={`/workspace/onboarding${query}`}>See all setup steps</a>
	{:else}<p class="eyebrow">Next customer</p>
		<h2>Connect your next customer.</h2>
		<p>
			Invite a distributor or retailer, or create a personal offer for an individual consumer. Each trade still has its
			own acceptance and payment checks.
		</p>
		<a class="primary" href={`/workspace/partners/customers/new${query}`}>Invite a business →</a><a
			href={`/workspace/sales/consumers${query}`}>Sell to a consumer</a
		>{/if}
</section>

<style>
	.next-steps {
		padding: 1.5rem;
		border: 1px solid var(--color-border);
		border-radius: 0.8rem;
		background: var(--color-surface);
		margin-block: 1.5rem;
	}
	.next-steps h2 {
		font-size: 1.3rem;
		margin: 0.4rem 0;
	}
	.next-steps p {
		max-width: 65ch;
		line-height: 1.6;
		color: var(--color-muted);
	}
	.next-steps a {
		display: inline-flex;
		align-items: center;
		min-height: 44px;
		margin-right: 1rem;
	}
	.next-steps button {
		padding: 0.6rem 1rem;
		font: inherit;
		margin-right: 1rem;
	}
</style>
