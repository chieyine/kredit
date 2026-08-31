<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import SystemBanner from '$lib/components/SystemBanner.svelte';
	import SiteHeader from '$lib/components/SiteHeader.svelte';
	import SiteFooter from '$lib/components/SiteFooter.svelte';
	import MotionObserver from '$lib/components/MotionObserver.svelte';
	import { jsonLd, nonIndexablePaths, seoForPath, SITE_URL } from '$lib/seo';
	import { page } from '$app/state';
	import { applySavedDisplayChoice } from '$lib/product-tools';

	let { children } = $props();
	let offline = $state(false);
	let privateShell = $derived(/^\/(app|buyer|admin|c|pay|receipt|secure|recover|buyer-invitations)(\/|$)/.test(page.url.pathname));
	let publicChrome = $derived(page.url.pathname === '/app' || !privateShell);

	const normalizedPath = $derived(page.url.pathname.length > 1 ? page.url.pathname.replace(/\/$/, '') : page.url.pathname);
	const canonical = $derived(SITE_URL + normalizedPath);
	const routeArticle = $derived((page.data as any)?.article);
	const routeSEO = $derived((page.data as any)?.seo);
	const legalActive = $derived(Boolean((page.data as any)?.legal?.active));
	const seo = $derived(routeArticle ? { title: routeArticle.title, description: routeArticle.description, type: 'article' as const, published: routeArticle.published, modified: routeArticle.modified, wordCount: routeArticle.wordCount, category: routeArticle.category } : routeSEO ?? seoForPath(normalizedPath));
	const indexable = $derived(!privateShell && (!nonIndexablePaths.has(normalizedPath) || legalActive) && page.status < 400);
	const organizationSchema = {
		'@context': 'https://schema.org',
		'@type': 'Organization',
		name: 'Kredit',
		url: SITE_URL,
		logo: `${SITE_URL}/icon-512.png`,
		description:
			'Kredit helps Nigerian businesses sell goods on credit, track payments and collect money.',
		areaServed: 'NG',
	};
	const websiteSchema = {
		'@context': 'https://schema.org',
		'@type': 'WebSite',
		name: 'Kredit',
		url: SITE_URL,
		inLanguage: 'en-NG'
	};
	const pageSchema = $derived({
		'@context': 'https://schema.org',
		'@type': seo.type === 'article' ? 'Article' : 'WebPage',
		name: seo.title,
		headline: seo.type === 'article' ? seo.title : undefined,
		description: seo.description,
		url: canonical,
		inLanguage: 'en-NG',
		isPartOf: { '@type': 'WebSite', name: 'Kredit', url: SITE_URL },
		publisher: seo.type === 'article' ? { '@type': 'Organization', name: 'Kredit', logo: { '@type': 'ImageObject', url: `${SITE_URL}/icon-512.png` } } : undefined,
		image: seo.type === 'article' ? `${SITE_URL}/og.png` : undefined,
		mainEntityOfPage: seo.type === 'article' ? { '@type': 'WebPage', '@id': canonical } : undefined,
		datePublished: seo.published,
		dateModified: seo.modified,
		wordCount: seo.wordCount,
		articleSection: seo.category,
		author: seo.type === 'article' ? { '@type': 'Organization', name: 'Kredit Editorial Team' } : undefined
	});

	onMount(() => {
		applySavedDisplayChoice();
		offline = !navigator.onLine;
		const online = () => (offline = false);
		const disconnected = () => (offline = true);
		window.addEventListener('online', online);
		window.addEventListener('offline', disconnected);
		if ('serviceWorker' in navigator) {
			navigator.serviceWorker.register('/service-worker.js').catch(() => {
				// Offline support is progressive enhancement; financial actions remain online-only.
			});
		}
		return () => {
			window.removeEventListener('online', online);
			window.removeEventListener('offline', disconnected);
		};
	});
</script>

<svelte:head>
	<title>{seo.title}</title>
	<meta
		name="description"
		content={seo.description}
	/>
	<link rel="canonical" href={canonical} />
	<link rel="alternate" hreflang="en-NG" href={canonical} />
	<link rel="alternate" hreflang="x-default" href={canonical} />
	<link rel="alternate" type="application/rss+xml" title="Kredit helpful guides" href="https://kredit.com.ng/blog/rss.xml" />
	<meta property="og:site_name" content="Kredit" />
	<meta property="og:type" content={seo.type ?? 'website'} />
	<meta property="og:url" content={canonical} />
	<meta property="og:title" content={seo.title} />
	<meta property="og:description" content={seo.description} />
	<meta property="og:image" content={`${SITE_URL}/og.png`} />
	<meta property="og:image:type" content="image/png" />
	<meta property="og:image:alt" content="Kredit — give goods on credit and get paid with confidence" />
	<meta property="og:image:width" content="1200" />
	<meta property="og:image:height" content="630" />
	<meta property="og:locale" content="en_NG" />
	{#if seo.published}<meta property="article:published_time" content={`${seo.published}T08:00:00+01:00`} />{/if}
	{#if seo.modified}<meta property="article:modified_time" content={`${seo.modified}T08:00:00+01:00`} />{/if}
	{#if seo.category}<meta property="article:section" content={seo.category} />{/if}
	<meta name="twitter:card" content="summary_large_image" />
	<meta name="twitter:title" content={seo.title} />
	<meta name="twitter:description" content={seo.description} />
	<meta name="twitter:image" content={`${SITE_URL}/og.png`} />
	<meta name="twitter:image:alt" content="Kredit — give goods on credit and get paid with confidence" />
	{#if !privateShell}
		{@html `<script type="application/ld+json">${jsonLd(organizationSchema)}<\/script>`}
		{@html `<script type="application/ld+json">${jsonLd(websiteSchema)}<\/script>`}
		{@html `<script type="application/ld+json">${jsonLd(pageSchema)}<\/script>`}
	{/if}
	<meta name="robots" content={indexable ? 'index,follow,max-image-preview:large,max-snippet:-1' : 'noindex,nofollow'} />
</svelte:head>

<a class="skip-link" href="#main-content">Skip to content</a>
<MotionObserver />
{#if offline}<SystemBanner tone="warning" message="You are offline. Financial actions are not submitted or queued until you reconnect." />{/if}
{#if publicChrome}<SiteHeader />{/if}
{#if privateShell}
	<div id="main-content" tabindex="-1">{@render children()}</div>
{:else}
	<div id="main-content" class="motion-scope public-route" tabindex="-1">
		{#key page.url.pathname}{@render children()}{/key}
	</div>
{/if}
{#if publicChrome}<SiteFooter />{/if}
