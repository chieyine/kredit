<script lang="ts">
	import { articleCategoryDetails } from '$lib/blog/articles';
	import { jsonLd, SITE_URL } from '$lib/seo';
	let { data } = $props();
	const { article } = data;
	const articleURL = `${SITE_URL}/blog/${article.slug}`;
	const category = articleCategoryDetails[article.category];
	const categoryURL = `${SITE_URL}/blog/topic/${category.slug}`;
	const checkedDate = new Intl.DateTimeFormat('en-NG', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' }).format(new Date(`${article.modified}T00:00:00Z`));
	const faqSchema = { '@context': 'https://schema.org', '@type': 'FAQPage', mainEntity: article.faq.map((item:any) => ({ '@type': 'Question', name: item.question, acceptedAnswer: { '@type': 'Answer', text: item.answer } })) };
	const breadcrumbSchema = { '@context': 'https://schema.org', '@type': 'BreadcrumbList', itemListElement: [
		{ '@type': 'ListItem', position: 1, name: 'Home', item: `${SITE_URL}/` },
		{ '@type': 'ListItem', position: 2, name: 'Helpful guides', item: `${SITE_URL}/blog` },
		{ '@type': 'ListItem', position: 3, name: article.category, item: categoryURL },
		{ '@type': 'ListItem', position: 4, name: article.title, item: articleURL }
	]};
</script>

<svelte:head>
	<meta name="keywords" content={`${article.keyphrase}, trade credit Nigeria, credit sales, customer payments`} />
	{@html `<script type="application/ld+json">${jsonLd(faqSchema)}<\/script>`}
	{@html `<script type="application/ld+json">${jsonLd(breadcrumbSchema)}<\/script>`}
</svelte:head>

<nav class="crumbs" aria-label="Breadcrumb"><a href="/">Home</a><span>/</span><a href="/blog">Helpful guides</a><span>/</span><a href={`/blog/topic/${category.slug}`}>{article.category}</a><span>/</span><span>Guide</span></nav>
<article class="guide">
	<header><p class="eyebrow">{article.category}</p><h1>{article.title}</h1><p class="lede">{article.intro}</p><div class="byline"><span>Checked {checkedDate}</span><span>{article.readingMinutes} minute read</span><span>{article.wordCount.toLocaleString('en-NG')} words</span></div></header>
	<aside class="quick-note"><strong>Before you use this guide</strong><p>This is practical general information, not personal legal, tax or financial advice. Check important decisions with a qualified Nigerian professional.</p></aside>
	{#each article.sections as section, index}
		<section><h2>{section.heading}</h2>{#each section.paragraphs as paragraph}<p>{paragraph}</p>{/each}{#if section.points}<ul>{#each section.points as point}<li>{point}</li>{/each}</ul>{/if}</section>
		{#if index === 2 || index === 6}
			{@const related = article.related[index === 2 ? 0 : 1]}
			<aside class="inside-link" aria-label="Related guide"><span>Read this next</span><a href={`/blog/${related.slug}`}>{related.title} <b aria-hidden="true">→</b></a><p>It explains the next part of this topic in simple steps.</p></aside>
		{/if}
	{/each}
	<nav class="guide-links" aria-label="Useful Kredit pages"><strong>Continue with useful pages</strong><a href={`/blog/topic/${category.slug}`}>See all {article.category.toLowerCase()} guides</a><a href="/glossary">Check the meaning of a word</a><a href="/how-it-works">See how Kredit works</a><a href="/faq">Read common questions</a></nav>
	<section class="faq"><p class="eyebrow">Questions people ask</p><h2>Frequently asked questions</h2>{#each article.faq as item}<details><summary>{item.question}</summary><p>{item.answer}</p></details>{/each}</section>
	<section class="sources"><h2>Official sources used for this guide</h2><p>Rules and services can change. These official pages are the best place to confirm current information.</p><ul>{#each article.sources as source}<li><a href={source.url} rel="noreferrer">{source.name}</a><span>{source.note}</span></li>{/each}</ul></section>
	<section class="next-step"><div><p class="eyebrow">Put it into practice</p><h2>Keep the deal and every payment together.</h2><p>Kredit helps Nigerian sellers and buyers see the same goods, amount, dates and payment record.</p></div><a href="/app">Start with Kredit →</a></section>
</article>

<aside class="related"><p class="eyebrow">Keep learning</p><h2>Related guides</h2><div>{#each article.related as item}<a href={`/blog/${item.slug}`}><span>{article.category}</span><strong>{item.title}</strong><b>Read guide →</b></a>{/each}</div></aside>

<style>
	.crumbs{display:flex;flex-wrap:wrap;gap:.55rem;margin-bottom:2rem;color:var(--color-muted);font-size:.78rem}.crumbs a{color:inherit}.guide{max-width:50rem}.guide>header{padding-bottom:2rem;border-bottom:3px solid #17181b}.guide h1{max-width:16ch;margin:.8rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(2.8rem,7vw,5.4rem);font-weight:500;line-height:.93;letter-spacing:-.055em}.guide .lede{max-width:44rem;color:#41433f;font-size:1.15rem;line-height:1.75}.byline{display:flex;flex-wrap:wrap;gap:.6rem 1.2rem;margin-top:1.5rem;color:var(--color-muted);font-size:.78rem}.quick-note{margin:2rem 0;padding:1.1rem;border-left:4px solid #2738d6;background:#eef0ff}.quick-note p{margin:.35rem 0 0}.guide section{margin-top:3rem}.guide section h2{max-width:22ch;font-family:Georgia,'Times New Roman',serif;font-size:clamp(1.8rem,4vw,2.6rem);font-weight:500;line-height:1.05}.guide section p,.guide section li{color:#41433f;font-size:1.02rem;line-height:1.82}.guide section li{margin:.65rem 0;padding-left:.35rem}.inside-link{display:grid;gap:.45rem;margin:2.2rem 0;padding:1.1rem 1.25rem;border-top:1px solid #b9b3a8;border-bottom:1px solid #b9b3a8}.inside-link span{color:#2738d6;font-size:.68rem;font-weight:850;letter-spacing:.1em;text-transform:uppercase}.inside-link a{color:#17181b;font-family:Georgia,'Times New Roman',serif;font-size:1.25rem;font-weight:600;text-decoration:none}.inside-link a:hover{text-decoration:underline}.inside-link b{color:#ff6848}.inside-link p{margin:0;color:#67645e;font-size:.82rem}.guide-links{display:grid;grid-template-columns:1fr 1fr;gap:.2rem 1rem;margin-top:3rem;padding:1.25rem;background:#eef0ff}.guide-links strong{grid-column:1/-1;margin-bottom:.4rem}.guide-links a{padding:.55rem 0;color:#2738d6;font-weight:700}.faq details{border-top:1px solid var(--color-border)}.faq details:last-child{border-bottom:1px solid var(--color-border)}.faq summary{padding:1rem 0;font-weight:780;cursor:pointer}.faq details p{margin-top:0}.sources{padding:1.3rem;background:#ebe7de}.sources ul{padding:0;list-style:none}.sources li{display:grid;gap:.25rem;padding:.8rem 0;border-bottom:1px solid #c8c1b5}.sources li:last-child{border:0}.sources a{color:#2738d6;font-weight:750}.sources span{color:var(--color-muted);font-size:.88rem}.next-step{display:flex;justify-content:space-between;align-items:end;gap:2rem;padding:2rem!important;color:white;background:#17181b}.next-step h2{margin:.35rem 0}.next-step p{color:#d4d3cf!important}.next-step>a{flex:none;padding:.8rem 1rem;background:#ff6848;color:#17181b;font-weight:800;text-decoration:none}.related{margin-top:6rem;padding-top:1.5rem;border-top:3px solid #17181b}.related>div{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:1rem}.related a{display:grid;gap:1rem;min-height:12rem;padding:1.2rem;border:1px solid var(--color-border);color:#17181b;text-decoration:none}.related a span,.related a b{color:#2738d6;font-size:.72rem}.related a strong{font-family:Georgia,'Times New Roman',serif;font-size:1.25rem;font-weight:500}.related a b{align-self:end}@media(max-width:700px){.guide-links{grid-template-columns:1fr}.next-step{align-items:stretch;flex-direction:column}.next-step>a{text-align:center}.related>div{grid-template-columns:1fr}.related a{min-height:8rem}}
</style>
