<script lang="ts">
	const terms: [string, string, string][] = [
		['A', 'Ageing', 'How late the money is. Not due yet, one to thirty days late, or past thirty days.'],
		[
			'D',
			'Dispute',
			'When either side reports a problem with the goods or with the amount owed. While it is being looked at, new bank debits leave the disputed amount alone. A debit already sent to the bank may still go through.'
		],
		[
			'D',
			'Drawdown',
			'A fresh sale taken against a limit you already gave the customer. He does not have to come back and negotiate terms again.'
		],
		[
			'G',
			'Grace period',
			'The extra days you allow after the payment day. Reminders keep going out, but no debit can start yet.'
		],
		[
			'M',
			'Mandate',
			'The permission your customer signs so an approved payment company can debit his account, up to an agreed amount and not a naira more.'
		],
		[
			'O',
			'Obligation',
			'A sale that has gone live. It carries what is owed, the days it falls due, and every payment made against it so far.'
		],
		[
			'P',
			'Payment claim',
			'When your customer says he has sent a transfer. You check the money actually landed before the balance comes down. Cash is recorded separately by you.'
		],
		['P', 'Principal', 'The money for the goods themselves, before any fee is added.'],
		[
			'P',
			'Public receipt',
			'A receipt you can forward to anybody without worry. Names, bank details and private records are stripped out of it.'
		],
		[
			'T',
			'Trade credit',
			'Goods now, money later. You supply the credit yourself. A bank or payment provider may handle the collection afterwards.'
		]
	];
</script>

<main class="glossary-page shell">
	<header>
		<div>
			<p class="eyebrow">What the words mean</p>
			<h1>The words,<br />in plain language.</h1>
		</div>
		<p>
			Every term you will meet on Kredit, in one line. Where we cannot avoid a finance word, it is explained here rather
			than left to guesswork.
		</p>
	</header>

	<nav class="letter-index" aria-label="Glossary letters">
		{#each ['A', 'D', 'G', 'M', 'O', 'P', 'T'] as letter, i (i)}<a href={'#letter-' + letter}>{letter}</a>{/each}
	</nav>

	<dl class="term-list">
		{#each terms as [letter, term, definition], index (index)}
			<div id={index === 0 || terms[index - 1][0] !== letter ? 'letter-' + letter : undefined}>
				<dt><span>{letter}</span>{term}</dt>
				<dd>{definition}</dd>
			</div>
		{/each}
	</dl>

	<aside>
		<p>Still have a question?</p>
		<a href="/faq">Read the common questions <span>↗</span></a>
	</aside>
</main>

<style>
	.glossary-page {
		padding-top: clamp(5rem, 10vw, 9rem);
		padding-bottom: clamp(6rem, 11vw, 10rem);
		color: var(--color-foreground);
	}
	header {
		display: grid;
		grid-template-columns: 1.35fr 0.65fr;
		gap: clamp(4rem, 11vw, 11rem);
		align-items: end;
	}
	h1 {
		margin: 1.2rem 0 0;
		font-family: var(--font-serif);
		font-size: clamp(4rem, 8vw, 7.5rem);
		font-weight: 500;
		line-height: 0.88;
		letter-spacing: -0.065em;
	}
	header > p {
		margin: 0;
		padding-top: 1.2rem;
		border-top: 1px solid var(--color-foreground);
		color: var(--color-muted);
		font-size: 1.05rem;
		line-height: 1.7;
	}
	.letter-index {
		display: flex;
		gap: 0;
		margin-top: clamp(4rem, 8vw, 7rem);
		border-top: 1px solid var(--color-border);
		border-bottom: 1px solid var(--color-border);
	}
	.letter-index a {
		display: grid;
		place-items: center;
		width: 3.4rem;
		min-height: 3.4rem;
		border-right: 1px solid var(--color-border);
		color: var(--color-muted);
		text-decoration: none;
		font-size: 0.78rem;
		font-weight: 760;
	}
	.letter-index a:hover {
		color: var(--color-on-primary);
		background: var(--color-primary);
	}
	.term-list {
		margin: 0;
	}
	.term-list > div {
		display: grid;
		grid-template-columns: 0.85fr 1.15fr;
		gap: clamp(3rem, 9vw, 9rem);
		padding: 2.7rem 0;
		border-bottom: 1px solid var(--color-border);
		scroll-margin-top: 6rem;
	}
	.term-list dt {
		display: grid;
		grid-template-columns: 3rem 1fr;
		align-items: baseline;
		font-family: var(--font-serif);
		font-size: clamp(1.6rem, 3vw, 2.5rem);
		font-weight: 500;
		letter-spacing: -0.025em;
	}
	.term-list dt span {
		color: var(--color-primary);
		font-family: Inter, sans-serif;
		font-size: 0.65rem;
		font-weight: 800;
		letter-spacing: 0.1em;
	}
	.term-list dd {
		max-width: 39rem;
		margin: 0;
		color: var(--color-muted);
		font-size: 1rem;
		line-height: 1.75;
	}
	aside {
		display: flex;
		justify-content: space-between;
		gap: 2rem;
		align-items: center;
		margin-top: 5rem;
		padding-top: 1.3rem;
		border-top: 3px solid var(--color-foreground);
	}
	aside p {
		margin: 0;
		font-family: var(--font-serif);
		font-size: 1.4rem;
	}
	aside a {
		display: inline-flex;
		gap: 2rem;
		color: var(--color-primary);
		font-weight: 760;
		text-decoration: none;
	}
	@media (max-width: 760px) {
		header {
			grid-template-columns: 1fr;
			gap: 2.5rem;
		}
		.term-list > div {
			grid-template-columns: 1fr;
			gap: 1rem;
		}
		.term-list dt {
			grid-template-columns: 2rem 1fr;
		}
		.letter-index {
			overflow-x: auto;
		}
	}
	@media (max-width: 520px) {
		.glossary-page {
			padding-top: 4rem;
		}
		.letter-index a {
			flex: 1 0 3rem;
		}
		.term-list > div {
			padding: 2rem 0;
		}
		aside {
			align-items: flex-start;
			flex-direction: column;
		}
	}
</style>
