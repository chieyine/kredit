<script lang="ts">
	const terms: [string, string, string][] = [
		['A', 'Ageing', 'How late a payment is. For example: not due yet, 1\u201330 days late, or more than 30 days late.'],
		['D', 'Dispute', 'When a seller or customer reports a problem with the goods or the amount owed. The money being argued about cannot be debited while it is being sorted out.'],
		['D', 'Drawdown', 'A new sale taken from a limit the seller already gave the customer. Like taking a bag of rice from a store you already agreed on.'],
		['G', 'Grace period', 'Extra days after the payment day. Reminders continue, but no bank debit can start yet.'],
		['M', 'Mandate', 'The permission a customer gives so an approved payment company can debit their account, up to an agreed amount and no more.'],
		['O', 'Obligation', 'A sale that has started. It shows the money owed, the payment days and every payment made so far.'],
		['P', 'Payment claim', 'When a customer says they paid by transfer or cash. The seller must check their bank and confirm it before the balance drops.'],
		['P', 'Principal', 'The money owed for the goods themselves, before any fee.'],
		['P', 'Public receipt', 'A receipt you can forward to anybody. It hides names, bank details and private records.'],
		['T', 'Trade credit', 'When a seller gives goods now and lets the customer pay later. It is not a bank loan. No bank is involved.']
	];
</script>


<main class="glossary-page shell">
	<header>
		<div><p class="eyebrow">What the words mean</p><h1>Business words,<br />in plain language.</h1></div>
		<p>Plain meanings for words you may see on Kredit. If we ever have to use a big finance word, we explain it here.</p>
	</header>

	<nav class="letter-index" aria-label="Glossary letters">
		{#each ['A','D','G','M','O','P','T'] as letter}<a href={'#letter-' + letter}>{letter}</a>{/each}
	</nav>

	<dl class="term-list">
		{#each terms as [letter, term, definition], index}
			<div id={index === 0 || terms[index - 1][0] !== letter ? 'letter-' + letter : undefined}>
				<dt><span>{letter}</span>{term}</dt><dd>{definition}</dd>
			</div>
		{/each}
	</dl>

	<aside><p>Still have a question?</p><a href="/faq">Read the common questions <span>↗</span></a></aside>
</main>

<style>
	.glossary-page { padding-top: clamp(5rem, 10vw, 9rem); padding-bottom: clamp(6rem, 11vw, 10rem); color: #17181b; }
	header { display: grid; grid-template-columns: 1.35fr .65fr; gap: clamp(4rem, 11vw, 11rem); align-items: end; }
	h1 { margin: 1.2rem 0 0; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(4rem, 8vw, 7.5rem); font-weight: 500; line-height: .88; letter-spacing: -.065em; }
	header > p { margin: 0; padding-top: 1.2rem; border-top: 1px solid #17181b; color: #686a66; font-size: 1.05rem; line-height: 1.7; }
	.letter-index { display: flex; gap: 0; margin-top: clamp(4rem, 8vw, 7rem); border-top: 1px solid #cec9bf; border-bottom: 1px solid #cec9bf; }
	.letter-index a { display: grid; place-items: center; width: 3.4rem; min-height: 3.4rem; border-right: 1px solid #cec9bf; color: #5f615e; text-decoration: none; font-size: .78rem; font-weight: 760; }.letter-index a:hover { color: white; background: #2738d6; }
	.term-list { margin: 0; }.term-list > div { display: grid; grid-template-columns: .85fr 1.15fr; gap: clamp(3rem, 9vw, 9rem); padding: 2.7rem 0; border-bottom: 1px solid #cec9bf; scroll-margin-top: 6rem; }.term-list dt { display: grid; grid-template-columns: 3rem 1fr; align-items: baseline; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(1.6rem, 3vw, 2.5rem); font-weight: 500; letter-spacing: -.025em; }.term-list dt span { color: #2738d6; font-family: Inter, sans-serif; font-size: .65rem; font-weight: 800; letter-spacing: .1em; }.term-list dd { max-width: 39rem; margin: 0; color: #686a66; font-size: 1rem; line-height: 1.75; }
	aside { display: flex; justify-content: space-between; gap: 2rem; align-items: center; margin-top: 5rem; padding-top: 1.3rem; border-top: 3px solid #17181b; }aside p { margin: 0; font-family: Georgia, 'Times New Roman', serif; font-size: 1.4rem; }aside a { display: inline-flex; gap: 2rem; color: #2738d6; font-weight: 760; text-decoration: none; }
	@media (max-width: 760px) { header { grid-template-columns: 1fr; gap: 2.5rem; }.term-list > div { grid-template-columns: 1fr; gap: 1rem; }.term-list dt { grid-template-columns: 2rem 1fr; }.letter-index { overflow-x: auto; } }
	@media (max-width: 520px) { .glossary-page { padding-top: 4rem; }.letter-index a { flex: 1 0 3rem; }.term-list > div { padding: 2rem 0; }aside { align-items: flex-start; flex-direction: column; } }
</style>
