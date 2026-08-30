<script lang="ts">
	import { jsonLd } from '$lib/seo';
	const faqs = [
		{
			q: 'Who gives the goods on credit?',
			a: 'The seller. Kredit does not give loans. We only help both sides keep the sale and payment clear.'
		},
		{
			q: 'Is Kredit a loan app?',
			a: 'No. The seller gives the goods and chooses the customer. Kredit keeps the amount, date and payments.'
		},
		{
			q: 'What happens if a customer does not pay?',
			a: 'We remind them first. We wait for the extra days to end. Then Kredit can debit only the money still owed. If there is a problem, we stop and check.'
		},
		{
			q: 'What if a completed payment is reversed?',
			a: 'We show that the money came back. The amount may be added to the money still owed.'
		},
		{
			q: 'Does Kredit show a credit score?',
			a: 'No. Kredit shows only what happened in your sales. It does not use a secret number.'
		},
		{
			q: 'Does my customer need to install an app?',
			a: 'No. They open a private link. Reminders and receipts can come through WhatsApp.'
		},
		{
			q: 'How much does it cost?',
			a: 'If your customer pays ₦100,000, Kredit takes ₦500. If Kredit has to debit ₦100,000 after it is late, Kredit takes ₦1,000. Adding a sale is free.'
		},
		{
			q: 'Is my business data private?',
			a: 'Yes. A shared receipt does not show names or bank details. Only you and the customer see the full sale.'
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

<main class="shell prose-page"><p class="eyebrow">Common questions</p><h1>Simple answers before you start.</h1><dl class="faq">{#each faqs as item}<div><dt>{item.q}</dt><dd>{item.a}</dd></div>{/each}</dl></main>
