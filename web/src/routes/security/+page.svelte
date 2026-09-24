<script lang="ts">
	const safeguards = [
		[
			'A code before anything serious',
			'Signing in needs a code. So does anything that moves money. Keep those codes to yourself, including when you are on the phone to our support.'
		],
		[
			'Amounts that do not drift',
			'Every figure is held in kobo, not rounded naira. Payment references and balance checks stop the same payment being counted twice.'
		],
		[
			'Your staff see their own work',
			'What a member of staff can open depends on the role you gave him. Private document links expire.'
		],
		[
			'Nothing changes quietly',
			'Account and money actions leave a history behind them. When something looks wrong, it can be traced instead of argued over.'
		],
		[
			'We ask before we debit',
			'A bank collection needs your customer’s authorisation and an eligible payment behind it. A reported payment or a dispute is treated according to what is on record. A debit already sent to the bank may still finish.'
		],
		[
			'Somewhere to take a problem',
			'Help and complaints is there for an account, a sale or a payment that has gone wrong. Send us the Kredit reference and what happened. Never send a password, a one-time code or your bank PIN.'
		]
	];
</script>

<main class="trust-page">
	<section class="trust-hero shell">
		<div>
			<p class="eyebrow">Your account and records</p>
			<h1>What we do to protect you.</h1>
		</div>
		<div class="hero-side">
			<p>
				Sign-in codes, staff roles and a record of every money action. If you see something on your account you did not
				do, tell us and it will be looked into.
			</p>
			<dl>
				<div>
					<dt>Your staff</dt>
					<dd>See only their own work</dd>
				</div>
				<div>
					<dt>Your sale</dt>
					<dd>Every change is recorded</dd>
				</div>
				<div>
					<dt>A problem</dt>
					<dd>Report it for review</dd>
				</div>
			</dl>
		</div>
	</section>

	<section class="safeguards shell">
		<div class="section-head">
			<p class="eyebrow">How we keep it safe</p>
			<h2>What is in place.</h2>
		</div>
		<div class="safeguard-list">
			{#each safeguards as [title, body], index (index)}<article>
					<span>0{index + 1}</span>
					<h3>{title}</h3>
					<p>{body}</p>
				</article>{/each}
		</div>
	</section>

	<section class="boundaries-section">
		<div class="shell boundaries">
			<article>
				<span>01 / LINKS YOU SHARE</span>
				<h2>Shared receipts hide your details.</h2>
				<p>
					A receipt you forward carries no full names, no bank details and no private notes. But a private invitation or
					document link is a different thing. Send those to one person only, the person they were meant for.
				</p>
			</article>
			<article>
				<span>02 / WHEN A DEBIT IS SLOW</span>
				<h2>Check before retrying.</h2>
				<p>
					If a bank debit is slow, we do not just try again and hope. We go back and ask the bank what happened to the
					first one.
				</p>
			</article>
		</div>
	</section>

	<section class="reporting shell">
		<div>
			<p class="eyebrow">Tell us about a problem</p>
			<h2>Seen something you did not do?</h2>
		</div>
		<div>
			<p>
				Tell us what happened. But never send your password, your one-time code, your recovery code or your bank
				details. Not to us. Not to anybody who rings you claiming to be us.
			</p>
			<a href="/legal/complaints">Report a problem <span>↗</span></a>
		</div>
	</section>

	<aside class="release shell">
		<strong>Before you act</strong>
		<p>
			Look at the sale, the amount and where the payment stands. Private links and sign-in codes are yours alone. Our
			support will never ask you for your bank PIN or a one-time code.
		</p>
	</aside>
</main>

<style>
	.trust-page {
		color: var(--color-foreground);
		background: var(--color-surface);
	}
	.trust-hero {
		display: grid;
		grid-template-columns: 1.35fr 0.65fr;
		gap: clamp(4rem, 10vw, 10rem);
		align-items: end;
		padding-top: clamp(5rem, 10vw, 9rem);
		padding-bottom: clamp(6rem, 10vw, 9rem);
	}
	.trust-hero h1 {
		max-width: 11ch;
		margin: 1.1rem 0 0;
		font-family: var(--font-serif);
		font-size: clamp(4rem, 8vw, 7.5rem);
		font-weight: 500;
		line-height: 0.88;
		letter-spacing: -0.065em;
	}
	.hero-side {
		padding-top: 1.2rem;
		border-top: 1px solid var(--color-foreground);
	}
	.hero-side > p {
		margin: 0 0 2rem;
		color: var(--color-muted);
		font-size: 1.05rem;
		line-height: 1.7;
	}
	.hero-side dl {
		margin: 0;
	}
	.hero-side dl div {
		display: flex;
		justify-content: space-between;
		gap: 2rem;
		padding: 0.75rem 0;
		border-top: 1px solid var(--color-border);
	}
	.hero-side dt {
		color: var(--color-muted);
		font-size: 0.72rem;
	}
	.hero-side dd {
		margin: 0;
		font-size: 0.76rem;
		font-weight: 760;
	}
	.safeguards {
		padding-top: 8rem;
		padding-bottom: 9rem;
		border-top: 1px solid var(--color-border);
	}
	.section-head {
		display: grid;
		grid-template-columns: 0.55fr 1.45fr;
		gap: 3rem;
	}
	.section-head h2 {
		max-width: 12ch;
		margin: 0;
		font-family: var(--font-serif);
		font-size: clamp(3.2rem, 6vw, 6rem);
		font-weight: 500;
		line-height: 0.92;
		letter-spacing: -0.055em;
	}
	.safeguard-list {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		margin-top: 5rem;
		border-top: 1px solid var(--color-border);
		border-left: 1px solid var(--color-border);
	}
	.safeguard-list article {
		min-height: 15rem;
		padding: 1.6rem;
		border-right: 1px solid var(--color-border);
		border-bottom: 1px solid var(--color-border);
	}
	.safeguard-list span {
		color: var(--color-primary);
		font-size: 0.68rem;
		font-weight: 800;
	}
	.safeguard-list h3 {
		margin: 3.5rem 0 0.7rem;
		font-family: var(--font-serif);
		font-size: 1.35rem;
		font-weight: 500;
	}
	.safeguard-list p {
		margin: 0;
		color: var(--color-muted);
		line-height: 1.65;
	}
	.boundaries-section {
		padding: 8rem 0;
		color: var(--color-foreground);
		background: var(--color-surface-muted);
	}
	.boundaries {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0;
		border-top: 1px solid var(--color-border);
		border-left: 1px solid var(--color-border);
	}
	.boundaries article {
		min-height: 23rem;
		padding: clamp(2rem, 5vw, 4rem);
		border-right: 1px solid var(--color-border);
		border-bottom: 1px solid var(--color-border);
	}
	.boundaries span {
		color: var(--color-muted);
		font-size: 0.68rem;
		font-weight: 800;
		letter-spacing: 0.1em;
	}
	.boundaries h2 {
		max-width: 13ch;
		margin: 5rem 0 1.2rem;
		font-family: var(--font-serif);
		font-size: clamp(2.4rem, 4vw, 4rem);
		font-weight: 500;
		line-height: 0.98;
		letter-spacing: -0.045em;
	}
	.boundaries p {
		max-width: 31rem;
		margin: 0;
		color: var(--color-muted);
		line-height: 1.7;
	}
	.reporting {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: clamp(4rem, 10vw, 10rem);
		padding-top: 9rem;
		padding-bottom: 8rem;
	}
	.reporting h2 {
		max-width: 13ch;
		margin: 1rem 0 0;
		font-family: var(--font-serif);
		font-size: clamp(3rem, 5vw, 5rem);
		font-weight: 500;
		line-height: 0.95;
		letter-spacing: -0.05em;
	}
	.reporting > div:last-child {
		padding-top: 1.2rem;
		border-top: 1px solid var(--color-foreground);
	}
	.reporting p {
		margin: 0 0 2rem;
		color: var(--color-muted);
		line-height: 1.7;
	}
	.reporting a {
		display: inline-flex;
		gap: 3rem;
		padding: 1rem 1.2rem;
		color: var(--color-on-primary);
		background: var(--color-primary);
		font-weight: 760;
		text-decoration: none;
	}
	.reporting a span {
		color: var(--color-accent-ink);
	}
	.release {
		display: grid;
		grid-template-columns: 0.4fr 1.6fr;
		gap: 3rem;
		margin-bottom: 8rem;
		padding-top: 1.3rem;
		border-top: 3px solid var(--color-accent-ink);
	}
	.release p {
		max-width: 47rem;
		margin: 0;
		color: var(--color-muted);
		line-height: 1.7;
	}
	@media (max-width: 800px) {
		.trust-hero,
		.section-head,
		.reporting {
			grid-template-columns: 1fr;
			gap: 2.5rem;
		}
		.hero-side {
			max-width: 32rem;
		}
		.safeguard-list {
			grid-template-columns: 1fr 1fr;
		}
	}
	@media (max-width: 560px) {
		.trust-hero {
			padding-top: 4rem;
		}
		.safeguards,
		.boundaries-section,
		.reporting {
			padding-top: 6rem;
			padding-bottom: 6rem;
		}
		.safeguard-list,
		.boundaries,
		.release {
			grid-template-columns: 1fr;
		}
		.safeguard-list article {
			min-height: auto;
		}
		.safeguard-list h3 {
			margin-top: 2rem;
		}
		.boundaries article {
			min-height: auto;
		}
		.boundaries h2 {
			margin-top: 3rem;
		}
		.release {
			gap: 1rem;
			margin-bottom: 6rem;
		}
	}
</style>
