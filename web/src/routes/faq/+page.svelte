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
			a: 'Kredit sends reminders first. After the agreed grace period, it can request a debit under valid bank authorization, subject to payment and dispute checks. A debit can fail; repayment is not guaranteed.'
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
			a: 'Adding a draft sale is free. The seller owes the agreed base fee when the accepted sale becomes active. Successful collection adds the agreed collection fee on the amount collected. Check the pricing page for current rates; an existing offer keeps its recorded rates.'
		},
		{
			q: 'Is my business data private?',
			a: 'Access depends on your account and staff permissions. The buyer can see their sale, and authorized Kredit staff may access records to provide support or investigate a problem. Shared receipts omit names and bank details. The privacy notice explains how information is used.'
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

<main class="shell prose-page"><p class="eyebrow">Common questions</p><h1>Simple answers before you start.</h1><dl class="faq">{#each faqs as item}<div><dt>{item.q}</dt><dd>{item.a}</dd></div>{/each}</dl><p>See <a href="/pricing">current pricing</a> or read the <a href="/legal/privacy">privacy notice</a> for details.</p></main>
