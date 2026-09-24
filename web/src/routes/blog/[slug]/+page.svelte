<script lang="ts">
	import { articleCategoryDetails } from '$lib/blog/articles';
	import { jsonLd, SITE_URL } from '$lib/seo';
	let { data } = $props();
	let article = $derived(data.article);
	let articleURL = $derived(`${SITE_URL}/blog/${article.slug}`);
	let category = $derived(articleCategoryDetails[article.category]);
	let categoryURL = $derived(`${SITE_URL}/blog/topic/${category.slug}`);
	let updatedDate = $derived(
		new Intl.DateTimeFormat('en-NG', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' }).format(
			new Date(`${article.modified}T00:00:00Z`)
		)
	);
	let faqSchema = $derived({
		'@context': 'https://schema.org',
		'@type': 'FAQPage',
		mainEntity: article.faq.map((item) => ({
			'@type': 'Question',
			name: item.question,
			acceptedAnswer: { '@type': 'Answer', text: item.answer }
		}))
	});
	let breadcrumbSchema = $derived({
		'@context': 'https://schema.org',
		'@type': 'BreadcrumbList',
		itemListElement: [
			{ '@type': 'ListItem', position: 1, name: 'Home', item: `${SITE_URL}/` },
			{ '@type': 'ListItem', position: 2, name: 'Helpful guides', item: `${SITE_URL}/blog` },
			{ '@type': 'ListItem', position: 3, name: article.category, item: categoryURL },
			{ '@type': 'ListItem', position: 4, name: article.title, item: articleURL }
		]
	});
</script>

<svelte:head>
	<!-- eslint-disable-next-line svelte/no-at-html-tags -- jsonLd() JSON-encodes and escapes '<', so no markup can be produced. -->
	{@html `<script type="application/ld+json">${jsonLd(faqSchema)}<\/script>`}
	<!-- eslint-disable-next-line svelte/no-at-html-tags -- jsonLd() JSON-encodes and escapes '<', so no markup can be produced. -->
	{@html `<script type="application/ld+json">${jsonLd(breadcrumbSchema)}<\/script>`}
</svelte:head>

<nav class="crumbs" aria-label="Breadcrumb">
	<a href="/">Home</a><span>/</span><a href="/blog">Helpful guides</a><span>/</span><a
		href={`/blog/topic/${category.slug}`}>{article.category}</a
	><span>/</span><span>Guide</span>
</nav>
<article class="guide">
	<header>
		<p class="eyebrow">{article.category}</p>
		<h1>{article.title}</h1>
		<p class="lede">{article.intro}</p>
		<div class="byline"><span>Updated {updatedDate}</span><span>{article.readingMinutes} minute read</span></div>
	</header>
	{#each article.sections as section, index (index)}
		<section>
			<h2>{section.heading}</h2>
			{#each section.paragraphs as paragraph, i (i)}<p>{paragraph}</p>{/each}{#if section.points}<ul>
					{#each section.points as point, i (i)}<li>{point}</li>{/each}
				</ul>{/if}
		</section>
		{#if index === 1 || index === 3}
			{@const related = article.related[index === 1 ? 0 : 1]}
			{#if related}<aside class="inside-link" aria-label="Related guide">
					<span>Read this next</span><a href={`/blog/${related.slug}`}>{related.title} <b aria-hidden="true">→</b></a>
				</aside>{/if}
		{/if}
	{/each}
	<nav class="guide-links" aria-label="Useful Kredit pages">
		<header>
			<p class="eyebrow">Explore further</p>
			<h2>What to read next.</h2>
		</header>
		<div class="resource-grid">
			<a href={`/blog/topic/${category.slug}`}
				><span
					><strong>More on {article.category.toLowerCase()}</strong><small>Everything else on this topic.</small></span
				><b aria-hidden="true">↗</b></a
			>
			<a href="/glossary"
				><span><strong>Terms, explained</strong><small>Look up a word from this guide.</small></span><b
					aria-hidden="true">↗</b
				></a
			>
			<a href="/how-it-works"
				><span><strong>How Kredit works</strong><small>Follow a sale from agreement to payment.</small></span><b
					aria-hidden="true">↗</b
				></a
			>
			<a href="/faq"
				><span><strong>Questions about Kredit</strong><small>Read about fees, payments and your account.</small></span
				><b aria-hidden="true">↗</b></a
			>
		</div>
	</nav>
	<section class="faq">
		<p class="eyebrow">Questions people ask</p>
		<h2>Frequently asked questions</h2>
		{#each article.faq as item, i (i)}<details>
				<summary>{item.question}</summary>
				<p>{item.answer}</p>
			</details>{/each}
	</section>
	{#if article.sources.length}<section class="sources">
			<h2>Further reading</h2>
			<ul>
				{#each article.sources as source, i (i)}<li>
						<a href={source.url} rel="noreferrer">{source.name}</a><span>{source.note}</span>
					</li>{/each}
			</ul>
		</section>{/if}
	<section class="next-step">
		<div>
			<p class="eyebrow">Put it into practice</p>
			<h2>Keep the deal and every payment together.</h2>
			<p>Kredit helps Nigerian sellers and buyers see the same goods, amount, dates and payment record.</p>
		</div>
		<a href="/signin">Start with Kredit →</a>
	</section>
</article>

<aside class="related">
	<p class="eyebrow">Keep learning</p>
	<h2>Related guides</h2>
	<div>
		{#each article.related as item (item.slug)}<a href={`/blog/${item.slug}`}
				><span>{item.category}</span><strong>{item.title}</strong><b>Read guide →</b></a
			>{/each}
	</div>
</aside>

<style>
	.crumbs {
		display: flex;
		flex-wrap: wrap;
		gap: 0.55rem;
		margin-bottom: 2rem;
		color: var(--color-muted);
		font-size: 0.78rem;
	}
	.crumbs a {
		color: inherit;
	}
	.guide {
		max-width: 50rem;
	}
	.guide > header {
		padding-bottom: 2rem;
		border-bottom: 3px solid var(--color-foreground);
	}
	.guide h1 {
		max-width: 16ch;
		margin: 0.8rem 0;
		font-family: var(--font-serif);
		font-size: clamp(2.8rem, 7vw, 5.4rem);
		font-weight: 500;
		line-height: 0.93;
		letter-spacing: -0.055em;
	}
	.guide .lede {
		max-width: 44rem;
		color: var(--color-muted);
		font-size: 1.15rem;
		line-height: 1.75;
	}
	.byline {
		display: flex;
		flex-wrap: wrap;
		gap: 0.6rem 1.2rem;
		margin-top: 1.5rem;
		color: var(--color-muted);
		font-size: 0.78rem;
	}
	.guide section {
		margin-top: 3rem;
	}
	.guide section h2 {
		max-width: 22ch;
		font-family: var(--font-serif);
		font-size: clamp(1.8rem, 4vw, 2.6rem);
		font-weight: 500;
		line-height: 1.05;
	}
	.guide section p,
	.guide section li {
		color: var(--color-muted);
		font-size: 1.02rem;
		line-height: 1.82;
	}
	.guide section li {
		margin: 0.65rem 0;
		padding-left: 0.35rem;
	}
	.inside-link {
		display: grid;
		gap: 0.45rem;
		margin: 2.2rem 0;
		padding: 1.1rem 1.25rem;
		border-top: 1px solid var(--color-muted);
		border-bottom: 1px solid var(--color-muted);
	}
	.inside-link span {
		color: var(--color-primary);
		font-size: 0.68rem;
		font-weight: 850;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	.inside-link a {
		color: var(--color-foreground);
		font-family: var(--font-serif);
		font-size: 1.25rem;
		font-weight: 600;
		text-decoration: none;
	}
	.inside-link a:hover {
		text-decoration: underline;
	}
	.inside-link b {
		color: var(--color-accent-ink);
	}
	.guide-links {
		margin-top: 3.5rem;
		padding-top: 1.5rem;
		border-top: 2px solid var(--color-foreground);
	}
	.guide-links header {
		margin-bottom: 1.5rem;
	}
	.guide-links header .eyebrow {
		margin: 0 0 0.65rem;
	}
	.guide-links h2 {
		margin: 0;
		font-family: var(--font-serif);
		font-size: clamp(1.8rem, 4vw, 2.35rem);
		font-weight: 500;
		line-height: 1.1;
		letter-spacing: -0.035em;
	}
	.resource-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		column-gap: 2rem;
	}
	.resource-grid a {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1rem;
		min-height: 5.5rem;
		box-sizing: border-box;
		padding: 1.25rem 0;
		border-top: 1px solid var(--color-border);
		color: var(--color-foreground);
		text-decoration: none;
	}
	.resource-grid a > span {
		display: grid;
		gap: 0.45rem;
	}
	.resource-grid strong {
		font-size: 0.98rem;
		font-weight: 700;
		line-height: 1.4;
	}
	.resource-grid small {
		color: var(--color-muted);
		font-size: 0.83rem;
		line-height: 1.5;
	}
	.resource-grid b {
		flex: none;
		color: var(--color-primary);
		font-size: 1.15rem;
		font-weight: 500;
		line-height: 1.3;
	}
	.resource-grid a:hover strong {
		color: var(--color-primary);
		text-decoration: underline;
		text-underline-offset: 0.2em;
	}
	.resource-grid a:focus-visible {
		outline: 2px solid var(--color-primary);
		outline-offset: 5px;
	}
	.resource-grid a:nth-last-child(-n + 2) {
		border-bottom: 1px solid var(--color-border);
	}
	.faq details {
		border-top: 1px solid var(--color-border);
	}
	.faq details:last-child {
		border-bottom: 1px solid var(--color-border);
	}
	.faq summary {
		padding: 1rem 0;
		font-weight: 780;
		cursor: pointer;
	}
	.faq details p {
		margin-top: 0;
	}
	.sources {
		padding: 1.3rem;
		background: var(--color-border);
	}
	.sources ul {
		padding: 0;
		list-style: none;
	}
	.sources li {
		display: grid;
		gap: 0.25rem;
		padding: 0.8rem 0;
		border-bottom: 1px solid var(--color-border);
	}
	.sources li:last-child {
		border: 0;
	}
	.sources a {
		color: var(--color-primary);
		font-weight: 750;
	}
	.sources span {
		color: var(--color-muted);
		font-size: 0.88rem;
	}
	.next-step {
		display: flex;
		justify-content: space-between;
		align-items: end;
		gap: 2rem;
		padding: 2rem !important;
		color: var(--color-foreground);
		background: var(--color-surface-muted);
		border: 1px solid var(--color-border);
	}
	.next-step h2 {
		margin: 0.35rem 0;
	}
	.next-step p {
		color: var(--color-muted) !important;
	}
	.next-step > a {
		flex: none;
		padding: 0.8rem 1rem;
		background: var(--color-accent-ink);
		color: var(--color-on-primary);
		font-weight: 800;
		text-decoration: none;
	}
	.related {
		margin-top: 6rem;
		padding-top: 1.5rem;
		border-top: 3px solid var(--color-foreground);
	}
	.related > div {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 1rem;
	}
	.related a {
		display: grid;
		gap: 1rem;
		min-height: 12rem;
		padding: 1.2rem;
		border: 1px solid var(--color-border);
		color: var(--color-foreground);
		text-decoration: none;
	}
	.related a span,
	.related a b {
		color: var(--color-primary);
		font-size: 0.72rem;
	}
	.related a strong {
		font-family: var(--font-serif);
		font-size: 1.25rem;
		font-weight: 500;
	}
	.related a b {
		align-self: end;
	}
	@media (max-width: 700px) {
		.resource-grid {
			grid-template-columns: 1fr;
		}
		.resource-grid a:nth-last-child(2) {
			border-bottom: 0;
		}
		.resource-grid a {
			min-height: 5rem;
		}
		.next-step {
			align-items: stretch;
			flex-direction: column;
		}
		.next-step > a {
			text-align: center;
		}
		.related > div {
			grid-template-columns: 1fr;
		}
		.related a {
			min-height: 8rem;
		}
	}
</style>
