<script lang="ts">
	import { jsonLd } from '$lib/seo';
	let { data } = $props();
	const faqs = $derived(
		data.copy.sections.map((section: { heading: string; body: string }) => ({ q: section.heading, a: section.body }))
	);

	const faqSchema = $derived({
		'@context': 'https://schema.org',
		'@type': 'FAQPage',
		mainEntity: faqs.map((item) => ({
			'@type': 'Question',
			name: item.q,
			acceptedAnswer: { '@type': 'Answer', text: item.a }
		}))
	});
</script>

<svelte:head>
	<!-- eslint-disable-next-line svelte/no-at-html-tags -- jsonLd() JSON-encodes and escapes '<', so no markup can be produced. -->
	{@html `<script type="application/ld+json">${jsonLd(faqSchema)}<\/script>`}
</svelte:head>

<main class="shell prose-page">
	<p class="eyebrow">Common questions</p>
	<h1>{data.copy.title}<br />{data.copy.accent}</h1>
	<p>{data.copy.introduction}</p>
	<dl class="faq">
		{#each faqs as item, i (i)}<div>
				<dt>{item.q}</dt>
				<dd>{item.a}</dd>
			</div>{/each}
	</dl>
	<p>
		Still not sure? <a href="/demo">Try a sample sale</a>, see <a href="/pricing">what it costs</a>, or read the
		<a href="/legal/privacy">privacy notice</a>.
	</p>
</main>
