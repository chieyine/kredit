<script lang="ts">
	import AuthGate from '$lib/components/AuthGate.svelte';
	import { signOut } from '$lib/api/client';
	let leaving = $state(false),
		error = $state('');
	async function leave() {
		leaving = true;
		try {
			await signOut();
		} catch {
			error = 'Sign-out was not confirmed. Please try again.';
		} finally {
			leaving = false;
		}
	}
</script>

<svelte:head><title>Welcome to Kredit</title></svelte:head>
<AuthGate area="account">
	<main class="shell account-entry">
		<header>
			<a class="wordmark" href="/" aria-label="Kredit home"><span>K</span><b>Kredit</b></a><button
				disabled={leaving}
				onclick={leave}>{leaving ? 'Signing out…' : 'Sign out'}</button
			>
		</header>
		<section>
			<p class="eyebrow">One account. Your choice.</p>
			<h1>Where would you like to go?</h1>
			<p>Your business workspace for trade, or the purchases you made for yourself.</p>
			<div class="entry-options">
				<a href="/workspace/today"
					><span class="eyebrow">Manufacturers · Distributors · Retailers</span>
					<h2>Business workspace</h2>
					<p>
						Your customers, what you bought from suppliers, the terms you set and the payments against them. New here?
						Add your business first.
					</p>
					<strong>Open workspace →</strong></a
				><a href="/personal/purchases"
					><span class="eyebrow">Individual consumers</span>
					<h2>Personal purchases</h2>
					<p>
						Read what a retailer sold you, follow the payments and the delivery, or report a problem. No business setup
						needed.
					</p>
					<strong>View purchases →</strong></a
				>
			</div>
			<p class="entry-note">
				Been sent an invitation? Open the private link from your supplier or retailer. It takes you straight to that
				trade.
			</p>
			{#if error}<p role="alert">{error}</p>{/if}
		</section>
	</main>
</AuthGate>

<style>
	.account-entry {
		max-width: 66rem;
		min-height: 100svh;
		padding-block: 1.5rem;
	}
	.account-entry > header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.account-entry > section {
		padding-block: clamp(2rem, 7vw, 5rem);
	}
	h1 {
		font-size: clamp(2rem, 4vw, 3rem);
		line-height: 1.15;
		letter-spacing: -0.04em;
	}
	.account-entry p {
		line-height: 1.7;
		color: var(--color-muted);
	}
	.entry-options {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
		margin: 2rem 0;
	}
	.entry-options > a {
		padding: 2rem;
		border: 1px solid var(--color-border);
		border-radius: 1rem;
		background: var(--color-surface);
		text-decoration: none;
	}
	.entry-options > a:hover {
		border-color: var(--color-primary);
	}
	.entry-options strong {
		display: block;
		margin-top: 1.5rem;
		color: var(--color-primary);
	}
	.entry-options h2 {
		font-size: 1.5rem;
	}
	.entry-note {
		max-width: 70ch;
	}
	button {
		padding: 0.7rem 1rem;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 0.4rem;
		font: inherit;
	}
	@media (max-width: 640px) {
		.entry-options {
			grid-template-columns: 1fr;
		}
		.entry-options > a {
			padding: 1.5rem;
		}
	}
</style>
