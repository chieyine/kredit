<script lang="ts">
	import { articleCategoryDetails, articlesBySlug } from '$lib/blog/articles';
	import { jsonLd, SITE_URL } from '$lib/seo';
	let { data } = $props();
	let article = $derived(data.article);
	let articleURL = $derived(`${SITE_URL}/blog/${article.slug}`);
	let category = $derived(articleCategoryDetails[article.category]);
	let categoryURL = $derived(`${SITE_URL}/blog/topic/${category.slug}`);
	let updatedDate = $derived(new Intl.DateTimeFormat('en-NG', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' }).format(new Date(`${article.modified}T00:00:00Z`)));
	let faqSchema = $derived({ '@context': 'https://schema.org', '@type': 'FAQPage', mainEntity: article.faq.map((item:any) => ({ '@type': 'Question', name: item.question, acceptedAnswer: { '@type': 'Answer', text: item.answer } })) });
	let breadcrumbSchema = $derived({ '@context': 'https://schema.org', '@type': 'BreadcrumbList', itemListElement: [
		{ '@type': 'ListItem', position: 1, name: 'Home', item: `${SITE_URL}/` },
		{ '@type': 'ListItem', position: 2, name: 'Helpful guides', item: `${SITE_URL}/blog` },
		{ '@type': 'ListItem', position: 3, name: article.category, item: categoryURL },
		{ '@type': 'ListItem', position: 4, name: article.title, item: articleURL }
	]});
</script>

<svelte:head>
	{@html `<script type="application/ld+json">${jsonLd(faqSchema)}<\/script>`}
	{@html `<script type="application/ld+json">${jsonLd(breadcrumbSchema)}<\/script>`}
</svelte:head>

<nav class="crumbs" aria-label="Breadcrumb"><a href="/">Home</a><span>/</span><a href="/blog">Helpful guides</a><span>/</span><a href={`/blog/topic/${category.slug}`}>{article.category}</a><span>/</span><span>Guide</span></nav>
<article class="guide">
	<header><p class="eyebrow">{article.category}</p><h1>{article.title}</h1><p class="lede">{article.intro}</p><div class="byline"><span>Updated {updatedDate}</span><span>{article.readingMinutes} minute read</span></div></header>
	{#each article.sections as section, index}
		<section><h2>{section.heading}</h2>{#each section.paragraphs as paragraph}<p>{paragraph}</p>{/each}{#if section.points}<ul>{#each section.points as point}<li>{point}</li>{/each}</ul>{/if}</section>
		{#if index === 1 || index === 3}
			{@const related = article.related[index === 1 ? 0 : 1]}
			<aside class="inside-link" aria-label="Related guide"><span>Read this next</span><a href={`/blog/${related.slug}`}>{related.title} <b aria-hidden="true">→</b></a></aside>
		{/if}
	{/each}
	<nav class="guide-links" aria-label="Useful Kredit pages">
		<header><p class="eyebrow">Explore further</p><h2>Find what you need next.</h2></header>
		<div class="resource-grid">
			<a href={`/blog/topic/${category.slug}`}><span><strong>More on {article.category.toLowerCase()}</strong><small>Browse the guides in this topic.</small></span><b aria-hidden="true">↗</b></a>
			<a href="/glossary"><span><strong>Terms, explained</strong><small>Look up a word from this guide.</small></span><b aria-hidden="true">↗</b></a>
			<a href="/how-it-works"><span><strong>How Kredit works</strong><small>Follow a sale from agreement to payment.</small></span><b aria-hidden="true">↗</b></a>
			<a href="/faq"><span><strong>Questions about Kredit</strong><small>Read about fees, payments and your account.</small></span><b aria-hidden="true">↗</b></a>
		</div>
	</nav>
	<section class="faq"><p class="eyebrow">Questions people ask</p><h2>Frequently asked questions</h2>{#each article.faq as item}<details><summary>{item.question}</summary><p>{item.answer}</p></details>{/each}</section>
	{#if article.sources.length}<section class="sources"><h2>Further reading</h2><ul>{#each article.sources as source}<li><a href={source.url} rel="noreferrer">{source.name}</a><span>{source.note}</span></li>{/each}</ul></section>{/if}
	<section class="next-step"><div><p class="eyebrow">Put it into practice</p><h2>Keep the deal and every payment together.</h2><p>Kredit helps Nigerian sellers and buyers see the same goods, amount, dates and payment record.</p></div><a href="/app">Start with Kredit →</a></section>
</article>

<aside class="related"><p class="eyebrow">Keep learning</p><h2>Related guides</h2><div>{#each article.related as item}<a href={`/blog/${item.slug}`}><span>{articlesBySlug.get(item.slug)?.category}</span><strong>{item.title}</strong><b>Read guide →</b></a>{/each}</div></aside>

<style>
	.crumbs{display:flex;flex-wrap:wrap;gap:.55rem;margin-bottom:2rem;color:var(--color-muted);font-size:.78rem}.crumbs a{color:inherit}.guide{max-width:50rem}.guide>header{padding-bottom:2rem;border-bottom:3px solid #17181b}.guide h1{max-width:16ch;margin:.8rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(2.8rem,7vw,5.4rem);font-weight:500;line-height:.93;letter-spacing:-.055em}.guide .lede{max-width:44rem;color:#41433f;font-size:1.15rem;line-height:1.75}.byline{display:flex;flex-wrap:wrap;gap:.6rem 1.2rem;margin-top:1.5rem;color:var(--color-muted);font-size:.78rem}.guide section{margin-top:3rem}.guide section h2{max-width:22ch;font-family:Georgia,'Times New Roman',serif;font-size:clamp(1.8rem,4vw,2.6rem);font-weight:500;line-height:1.05}.guide section p,.guide section li{color:#41433f;font-size:1.02rem;line-height:1.82}.guide section li{margin:.65rem 0;padding-left:.35rem}.inside-link{display:grid;gap:.45rem;margin:2.2rem 0;padding:1.1rem 1.25rem;border-top:1px solid #b9b3a8;border-bottom:1px solid #b9b3a8}.inside-link span{color:#2738d6;font-size:.68rem;font-weight:850;letter-spacing:.1em;text-transform:uppercase}.inside-link a{color:#17181b;font-family:Georgia,'Times New Roman',serif;font-size:1.25rem;font-weight:600;text-decoration:none}.inside-link a:hover{text-decoration:underline}.inside-link b{color:#ff6848}.guide-links{margin-top:3.5rem;padding-top:1.5rem;border-top:2px solid #17181b}.guide-links header{margin-bottom:1.5rem}.guide-links header .eyebrow{margin:0 0 .65rem}.guide-links h2{margin:0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(1.8rem,4vw,2.35rem);font-weight:500;line-height:1.1;letter-spacing:-.035em}.resource-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));column-gap:2rem}.resource-grid a{display:flex;align-items:flex-start;justify-content:space-between;gap:1rem;min-height:5.5rem;box-sizing:border-box;padding:1.25rem 0;border-top:1px solid #cbc6bc;color:#17181b;text-decoration:none}.resource-grid a>span{display:grid;gap:.45rem}.resource-grid strong{font-size:.98rem;font-weight:700;line-height:1.4}.resource-grid small{color:#62645f;font-size:.83rem;line-height:1.5}.resource-grid b{flex:none;color:#2738d6;font-size:1.15rem;font-weight:500;line-height:1.3}.resource-grid a:hover strong{color:#2738d6;text-decoration:underline;text-underline-offset:.2em}.resource-grid a:focus-visible{outline:2px solid #2738d6;outline-offset:5px}.resource-grid a:nth-last-child(-n+2){border-bottom:1px solid #cbc6bc}.faq details{border-top:1px solid var(--color-border)}.faq details:last-child{border-bottom:1px solid var(--color-border)}.faq summary{padding:1rem 0;font-weight:780;cursor:pointer}.faq details p{margin-top:0}.sources{padding:1.3rem;background:#ebe7de}.sources ul{padding:0;list-style:none}.sources li{display:grid;gap:.25rem;padding:.8rem 0;border-bottom:1px solid #c8c1b5}.sources li:last-child{border:0}.sources a{color:#2738d6;font-weight:750}.sources span{color:var(--color-muted);font-size:.88rem}.next-step{display:flex;justify-content:space-between;align-items:end;gap:2rem;padding:2rem!important;color:white;background:#17181b}.next-step h2{margin:.35rem 0}.next-step p{color:#d4d3cf!important}.next-step>a{flex:none;padding:.8rem 1rem;background:#ff6848;color:#17181b;font-weight:800;text-decoration:none}.related{margin-top:6rem;padding-top:1.5rem;border-top:3px solid #17181b}.related>div{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:1rem}.related a{display:grid;gap:1rem;min-height:12rem;padding:1.2rem;border:1px solid var(--color-border);color:#17181b;text-decoration:none}.related a span,.related a b{color:#2738d6;font-size:.72rem}.related a strong{font-family:Georgia,'Times New Roman',serif;font-size:1.25rem;font-weight:500}.related a b{align-self:end}@media(max-width:700px){.resource-grid{grid-template-columns:1fr}.resource-grid a:nth-last-child(2){border-bottom:0}.resource-grid a{min-height:5rem}.next-step{align-items:stretch;flex-direction:column}.next-step>a{text-align:center}.related>div{grid-template-columns:1fr}.related a{min-height:8rem}}
</style>
