<script lang="ts">
	import { jsonLd } from '$lib/seo';
	const faqs = [
		{
			q: 'Who gives the goods on credit?',
			a: 'The seller does. Kredit does not lend money. We only help both sides keep the sale and the payment clear.'
		},
		{
			q: 'Is Kredit a loan app?',
			a: 'No. The seller gives the goods and chooses who gets them. Kredit only keeps the amount, the payment day and every payment.'
		},
		{
			q: 'What if my customer does not pay?',
			a: 'Kredit sends reminders first. After the extra days you agreed, Kredit can ask their bank for the money, but only if that customer gave permission and there is no open problem. A bank debit can still fail. Nobody can promise you the money.'
		},
		{
			q: 'What if money enters and then goes back?',
			a: 'We show clearly that the money was reversed, and we add that amount back to what the customer still owes.'
		},
		{
			q: 'Does Kredit show a credit score?',
			a: 'No. Kredit only shows what actually happened in your own sales. There is no secret number deciding anything.'
		},
		{
			q: 'Must my customer install an app?',
			a: 'No. They open a private link on their phone. Reminders and receipts can come on WhatsApp.'
		},
		{
			q: 'How much does it cost?',
			a: 'Writing a sale down is free. You pay the base fee only when your customer accepts the sale and it starts. If Kredit later collects money from their bank, a collection fee applies on the money actually collected. There is no monthly fee. A sale keeps the rates it was agreed on, even if rates change later.'
		},
		{
			q: 'Can other sellers see my business?',
			a: 'No. Only your own staff, and the permission you give them, decides who sees what. Your customer sees their own sale. Kredit staff may open a record to help you or check a problem. A receipt that gets shared hides names and bank details. The privacy notice explains everything.'
		}
	];

	const faqSchema = {
		'@context': 'https://schema.org',
		'@type': 'FAQPage',
		mainEntity: faqs.map((item) => ({
			'@type': 'Question',
			name: item.q,
			acceptedAnswer: { '@type': 'Answer', text: item.a }
		}))
	};
</script>

<svelte:head>
	{@html `<script type="application/ld+json">${jsonLd(faqSchema)}<\/script>`}
</svelte:head>

<main class="shell prose-page"><p class="eyebrow">Common questions</p><h1>Straight answers,<br />before you start.</h1><dl class="faq">{#each faqs as item}<div><dt>{item.q}</dt><dd>{item.a}</dd></div>{/each}</dl><p>Still not sure? <a href="/demo">Try a sample sale</a>, see <a href="/pricing">what it costs</a>, or read the <a href="/legal/privacy">privacy notice</a>.</p></main>
