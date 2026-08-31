<script lang="ts">
	import { jsonLd, SITE_URL } from '$lib/seo';
	let { data } = $props();
	const hubURL = `${SITE_URL}/blog/topic/${data.details.slug}`;
	const listSchema = {
		'@context': 'https://schema.org',
		'@type': 'CollectionPage',
		name: data.details.title,
		description: data.details.description,
		url: hubURL,
		inLanguage: 'en-NG',
		mainEntity: {
			'@type': 'ItemList',
			numberOfItems: data.articles.length,
			itemListElement: data.articles.map((article: any, index: number) => ({ '@type': 'ListItem', position: index + 1, name: article.title, url: `${SITE_URL}/blog/${article.slug}` }))
		}
	};
	const breadcrumbSchema = { '@context': 'https://schema.org', '@type': 'BreadcrumbList', itemListElement: [
		{ '@type': 'ListItem', position: 1, name: 'Home', item: `${SITE_URL}/` },
		{ '@type': 'ListItem', position: 2, name: 'Helpful guides', item: `${SITE_URL}/blog` },
		{ '@type': 'ListItem', position: 3, name: data.category, item: hubURL }
	]};
</script>

<svelte:head>
	{@html `<script type="application/ld+json">${jsonLd(listSchema)}<\/script>`}
	{@html `<script type="application/ld+json">${jsonLd(breadcrumbSchema)}<\/script>`}
</svelte:head>

<nav class="crumbs" aria-label="Breadcrumb"><a href="/">Home</a><span>/</span><a href="/blog">Helpful guides</a><span>/</span><span>{data.category}</span></nav>
<main class="topic-hub">
	<header><p class="eyebrow">{data.articles.length} simple guides</p><h1>{data.details.title}</h1><p>{data.details.description}</p></header>
	<section aria-labelledby="topic-list-title"><div class="list-head"><h2 id="topic-list-title">Start with the question you need to answer.</h2><span>{data.category}</span></div>
		<div class="topic-list">{#each data.articles as article, index}<a href={`/blog/${article.slug}`}><b>{String(index + 1).padStart(2, '0')}</b><div><h3>{article.title}</h3><p>{article.description}</p></div><span>{article.readingMinutes} min <i aria-hidden="true">→</i></span></a>{/each}</div>
	</section>
	<aside><div><strong>Not sure where to begin?</strong><p>See every Kredit guide or check the meaning of a word.</p></div><a href="/blog">All helpful guides</a><a href="/glossary">Meaning of words</a></aside>
</main>

<style>
	.crumbs{display:flex;flex-wrap:wrap;gap:.55rem;margin-bottom:2rem;color:var(--color-muted);font-size:.78rem}.crumbs a{color:inherit}.topic-hub>header{display:grid;grid-template-columns:1.25fr .75fr;gap:3rem;align-items:end;padding:3rem 0 5rem;border-bottom:3px solid #17181b}.topic-hub h1{max-width:12ch;margin:.6rem 0 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3rem,7vw,6rem);font-weight:500;line-height:.92;letter-spacing:-.05em}.topic-hub>header>p:last-child{max-width:35rem;margin:0;color:#5f625e;font-size:1.08rem;line-height:1.7}.topic-hub section{margin-top:4rem}.list-head{display:flex;justify-content:space-between;gap:2rem;align-items:end}.list-head h2{max-width:18ch;font-family:Georgia,'Times New Roman',serif;font-size:clamp(2rem,4vw,3.2rem);font-weight:500;line-height:1}.list-head span{color:#2738d6;font-size:.72rem;font-weight:800;text-transform:uppercase}.topic-list{border-top:1px solid #aaa59a}.topic-list a{display:grid;grid-template-columns:3rem 1fr auto;gap:2rem;align-items:start;padding:1.6rem 0;border-bottom:1px solid #c9c3b8;color:#17181b;text-decoration:none}.topic-list>a>b{color:#2738d6;font-size:.72rem}.topic-list h3{margin:0;font-family:Georgia,'Times New Roman',serif;font-size:1.55rem;font-weight:500}.topic-list p{max-width:48rem;margin:.6rem 0 0;color:#62645f;line-height:1.6}.topic-list>a>span{color:#686a66;font-size:.72rem;white-space:nowrap}.topic-list i{margin-left:.7rem;color:#ff6848;font-style:normal}.topic-list a:hover h3{color:#2738d6}.topic-hub>aside{display:flex;align-items:center;gap:1rem;margin-top:4rem;padding:1.4rem;background:#17181b;color:white}.topic-hub>aside div{margin-right:auto}.topic-hub>aside p{margin:.3rem 0 0;color:#b9b8b3}.topic-hub>aside a{padding:.7rem .9rem;background:#2738d6;color:white;text-decoration:none;font-weight:700}@media(max-width:720px){.topic-hub>header{grid-template-columns:1fr;gap:1.5rem}.list-head{align-items:start;flex-direction:column}.topic-list a{grid-template-columns:2rem 1fr;gap:1rem}.topic-list>a>span{grid-column:2}.topic-hub>aside{align-items:stretch;flex-direction:column}.topic-hub>aside div{margin:0}}
</style>
