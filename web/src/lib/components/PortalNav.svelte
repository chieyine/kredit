<script lang="ts">
	import { page } from '$app/state';

	type PortalLink = [string, string];
	type MobileLink = [string, string, string];

	let {
		label,
		homeHref,
		links,
		mobilePrimary = [],
		mobileMore = [],
		dark = false,
		onsearch,
		onsignout,
		searchReady = true
	}: {
		label: string;
		homeHref: string;
		links: PortalLink[];
		mobilePrimary?: MobileLink[];
		mobileMore?: PortalLink[];
		dark?: boolean;
		onsearch?: () => void;
		onsignout?: () => void;
		searchReady?: boolean;
	} = $props();

	let open = $state(false);
	let moreOpen = $state(false);
	const menuID = `portal-nav-${label.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`;
	const moreID = `${menuID}-more`;
	const hasMobileBar = $derived(mobilePrimary.length > 0);

	function current(href: string) {
		if (href === homeHref) return page.url.pathname === href;
		return page.url.pathname === href || page.url.pathname.startsWith(`${href}/`);
	}

	function closeMenus() {
		open = false;
		moreOpen = false;
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') closeMenus();
	}

	$effect(() => {
		if (!moreOpen || typeof document === 'undefined') return;
		const oldOverflow = document.body.style.overflow;
		document.body.style.overflow = 'hidden';
		return () => {
			document.body.style.overflow = oldOverflow;
		};
	});
</script>

<svelte:window onkeydown={handleKeydown} />

<nav class:dark class:has-mobile-bar={hasMobileBar} aria-label={label}>
	<a class="brand" href={homeHref} aria-label="Kredit portal home"><span aria-hidden="true">K</span>Kredit</a>
	<span class="account-label">{label.replace(' account', '')}</span>
	<button class="menu-toggle" class:hidden-mobile-toggle={hasMobileBar} type="button" aria-controls={menuID} aria-expanded={open} onclick={() => (open = !open)}>
		<span aria-hidden="true">{open ? '×' : '☰'}</span>{open ? 'Close' : 'Menu'}
	</button>
	<div id={menuID} class="portal-menu" class:open>
		{#if hasMobileBar}
			<div class="desktop-primary">
				{#each mobilePrimary as [linkLabel, href]}
					<a {href} aria-current={current(href) ? 'page' : undefined} onclick={closeMenus}>{linkLabel}</a>
				{/each}
				<button type="button" class:active={moreOpen || mobileMore.some(([, href]) => current(href))} aria-controls={moreID} aria-expanded={moreOpen} onclick={() => (moreOpen = true)}>More <span aria-hidden="true">↓</span></button>
			</div>
		{:else}
			<div class="portal-links">
				{#each links as [linkLabel, href]}
					<a {href} aria-current={current(href) ? 'page' : undefined} onclick={closeMenus}>{linkLabel}</a>
				{/each}
			</div>
		{/if}
		<div class="portal-actions">
			{#if onsearch}<button class="search palette-trigger" data-ready={searchReady} disabled={!searchReady} type="button" onclick={onsearch}>Search <kbd>⌘K</kbd></button>{/if}
			{#if onsignout}<button class="sign-out" type="button" onclick={onsignout}>Sign out</button>{/if}
		</div>
	</div>
</nav>

{#if hasMobileBar}
	<div class="mobile-nav" aria-label={`${label} main pages`}>
		{#each mobilePrimary as [linkLabel, href, icon]}
			<a {href} aria-current={current(href) ? 'page' : undefined} onclick={closeMenus}>
				<span class="mobile-icon" data-icon={icon} aria-hidden="true"></span>
				<span>{linkLabel}</span>
			</a>
		{/each}
		<button type="button" class:active={moreOpen || mobileMore.some(([, href]) => current(href))} aria-controls={moreID} aria-expanded={moreOpen} onclick={() => (moreOpen = true)}>
			<span class="mobile-icon" data-icon="more" aria-hidden="true"></span>
			<span>More</span>
		</button>
	</div>

	{#if moreOpen}
		<button class="sheet-backdrop" type="button" aria-label="Close more pages" onclick={() => (moreOpen = false)}></button>
		<div class="more-sheet" id={moreID} role="dialog" aria-modal="true" aria-label="More pages">
			<header>
				<div><span>More</span><strong>{label}</strong></div>
				<button type="button" aria-label="Close more pages" onclick={() => (moreOpen = false)}>×</button>
			</header>
			<div class="sheet-content">
				<div class="more-links" role="navigation" aria-label="More account pages" data-sveltekit-preload-data="tap">
					{#each mobileMore as [linkLabel, href]}
						<a {href} aria-current={current(href) ? 'page' : undefined} onclick={closeMenus}>
							<span>{linkLabel}</span><span aria-hidden="true">→</span>
						</a>
					{/each}
				</div>
				<div class="sheet-actions">
					{#if onsignout}<button class="mobile-sign-out" type="button" onclick={onsignout}>Sign out</button>{/if}
				</div>
			</div>
		</div>
	{/if}
{/if}

<style>
	nav { display:flex;align-items:center;gap:1.2rem;padding:.75rem max(1rem,calc((100vw - 76rem)/2));border-bottom:1px solid #34363c;background:#17181b;position:sticky;top:0;z-index:40 }
	.brand{display:inline-flex;align-items:center;gap:.65rem;color:#fff;font-family:Georgia,'Times New Roman',serif;font-weight:650;font-size:1.15rem;text-decoration:none;white-space:nowrap}.brand span{display:grid;place-items:center;width:2rem;height:2rem;background:#2738d6;color:#fff}
	.account-label{display:none;color:#9c9da2;font-size:.68rem;font-weight:800;letter-spacing:.12em;text-transform:uppercase}
	.portal-menu{display:flex;align-items:center;gap:.75rem;min-width:0;flex:1}.portal-links{display:flex;align-items:center;gap:.1rem;min-width:0;overflow-x:auto;scrollbar-width:none}.portal-links::-webkit-scrollbar{display:none}.portal-links a{display:inline-flex;align-items:center;min-height:2.75rem;padding:.15rem .68rem;color:#aaa9a5;font-size:.84rem;font-weight:680;text-decoration:none;white-space:nowrap}.portal-links a:hover{color:#fff}.portal-links a[aria-current='page']{color:#fff;background:#2738d6}
	.portal-actions{display:flex;align-items:center;gap:.35rem;margin-left:auto}.portal-actions button,.menu-toggle{border:0;background:transparent;color:#c7c6c1;font:inherit;font-size:.82rem;font-weight:680;cursor:pointer;white-space:nowrap}.portal-actions .search{display:inline-flex;align-items:center;gap:.4rem;padding:.45rem .7rem;border:1px solid #4b4c51}.search kbd{font-family:inherit;font-size:.7rem;color:#b8b9bc}.menu-toggle{display:none}.menu-toggle span{font-size:1.2rem;color:#ff6848}
	nav.dark{background:#17181b;border-color:#34363c}.dark .brand,.dark .portal-links a,.dark .portal-actions button{color:#e7e5df}.dark .portal-links a:hover,.dark .portal-links a[aria-current='page']{color:#fff;background:#2738d6}
	.desktop-primary{display:flex;align-items:center;gap:.1rem;min-width:0}.desktop-primary a,.desktop-primary button{display:inline-flex;align-items:center;gap:.45rem;min-height:2.75rem;padding:.15rem .78rem;border:0;background:transparent;color:#aaa9a5;font:inherit;font-size:.84rem;font-weight:680;text-decoration:none;white-space:nowrap;cursor:pointer}.desktop-primary a:hover,.desktop-primary button:hover{color:#fff}.desktop-primary a[aria-current='page'],.desktop-primary button.active{color:#fff;background:#2738d6}.desktop-primary button span{color:#ff8b70}
	.mobile-nav{display:none}
	.sheet-backdrop{position:fixed;z-index:60;inset:0;display:block;width:100%;height:100%;border:0;background:rgba(23,24,27,.46);cursor:pointer}.more-sheet{position:fixed;z-index:61;top:3.65rem;right:0;bottom:0;display:block;width:min(26rem,calc(100vw - 2rem));overflow:hidden;border-left:1px solid #cec9bf;border-top:4px solid #2738d6;background:#faf8f2;box-shadow:-24px 0 70px rgba(23,24,27,.2);animation:drawer-in .22s cubic-bezier(.22,.8,.28,1)}
	.more-sheet header{display:flex;align-items:center;justify-content:space-between;padding:1.05rem 1.25rem;border-bottom:1px solid #d7d2c8}.more-sheet header div{display:flex;flex-direction:column;gap:.14rem}.more-sheet header span{color:#2738d6;font-size:.68rem;font-weight:850;letter-spacing:.14em;text-transform:uppercase}.more-sheet header strong{font-family:Georgia,'Times New Roman',serif;font-size:1.45rem}.more-sheet header button{display:grid;place-items:center;width:2.75rem;height:2.75rem;border:1px solid #b8b4ab;background:transparent;color:#17181b;font-size:1.6rem;cursor:pointer}.sheet-content{max-height:calc(100vh - 8.8rem);overflow-y:auto;padding:.35rem 1.25rem 1.4rem}.more-links{display:grid}.more-links a{display:flex;align-items:center;justify-content:space-between;gap:1rem;min-height:3.6rem;padding:.35rem .1rem;border-bottom:1px solid #d7d2c8;color:#17181b;font-size:.92rem;font-weight:740;text-decoration:none}.more-links a span:last-child{color:#ff6848;font-size:1rem}.more-links a[aria-current='page']{color:#2738d6}.sheet-actions{display:flex;padding-top:1rem}.sheet-actions button{width:100%;min-height:3.2rem;padding:.65rem;border:1px solid #17181b;background:transparent;color:#17181b;font:inherit;font-size:.82rem;font-weight:780;cursor:pointer}.sheet-actions .mobile-sign-out{border-color:#17181b;background:#17181b;color:#fff}

	@media(max-width:760px){
		nav{position:sticky;flex-wrap:wrap;padding:.72rem 1rem}.account-label{display:block;margin-left:auto}.has-mobile-bar .portal-menu{display:none}.hidden-mobile-toggle{display:none!important}
		.menu-toggle{display:inline-flex;min-height:2.65rem;align-items:center;gap:.5rem;margin-left:auto;padding:0 .85rem;border:1px solid #55565b}.portal-menu{display:none;width:100%;align-items:stretch;flex-direction:column;padding:1rem 0 .35rem}.portal-menu.open{display:flex}.portal-links{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.35rem;overflow:visible}.portal-links a{min-height:3rem;padding:.4rem .75rem;white-space:normal;border:1px solid #34363c}.portal-actions{width:100%;margin:0;padding-top:.75rem;border-top:1px solid #34363c}.portal-actions .sign-out{margin-left:auto}
		.mobile-nav{position:fixed;z-index:55;left:0;right:0;bottom:0;display:grid;grid-template-columns:repeat(5,minmax(0,1fr));min-height:4.7rem;padding:0 .25rem max(.35rem,env(safe-area-inset-bottom));border-top:1px solid #d7d2c8;background:rgba(250,248,242,.98);box-shadow:0 -8px 28px rgba(23,24,27,.09);backdrop-filter:blur(14px)}
		.mobile-nav a,.mobile-nav button{position:relative;display:flex;min-width:0;min-height:4.35rem;flex-direction:column;align-items:center;justify-content:center;gap:.32rem;padding:.48rem .15rem .25rem;border:0;background:transparent;color:#77746e;font:inherit;font-size:.66rem;font-weight:760;line-height:1.1;text-align:center;text-decoration:none;cursor:pointer;transition:color .16s ease,background-color .18s ease}
		.mobile-nav a::after,.mobile-nav button::after{content:'';position:absolute;top:0;left:22%;right:22%;height:3px;background:transparent}
		.mobile-nav a[aria-current='page'],.mobile-nav button.active{background:linear-gradient(180deg,rgba(39,56,214,.08),transparent 70%);color:#2738d6}.mobile-nav a[aria-current='page']::after,.mobile-nav button.active::after{background:#2738d6}
		.mobile-icon{position:relative;display:block;width:1.45rem;height:1.45rem;color:currentColor;transition:transform .18s cubic-bezier(.2,.9,.3,1)}
		.mobile-nav a[aria-current='page'] .mobile-icon,.mobile-nav button.active .mobile-icon{transform:translateY(-1px) scale(1.06)}
		.mobile-icon::before,.mobile-icon::after{content:'';position:absolute;box-sizing:border-box}
		.mobile-icon[data-icon='home']::before{inset:3px 3px 2px;border:2px solid currentColor;border-top:0}
		.mobile-icon[data-icon='home']::after{width:13px;height:13px;left:5px;top:0;border-left:2px solid currentColor;border-top:2px solid currentColor;transform:rotate(45deg)}
		.mobile-icon[data-icon='add']::before{width:18px;height:2px;left:3px;top:11px;background:currentColor}
		.mobile-icon[data-icon='add']::after{width:2px;height:18px;left:11px;top:3px;background:currentColor}
		.mobile-icon[data-icon='customers']::before{width:9px;height:9px;left:7px;top:1px;border:2px solid currentColor;border-radius:50%}
		.mobile-icon[data-icon='customers']::after{width:18px;height:10px;left:3px;bottom:1px;border:2px solid currentColor;border-radius:10px 10px 2px 2px}
		.mobile-icon[data-icon='payments']::before,.mobile-icon[data-icon='owe']::before{inset:3px 1px;border:2px solid currentColor;border-radius:2px}
		.mobile-icon[data-icon='payments']::after,.mobile-icon[data-icon='owe']::after{width:7px;height:2px;right:4px;top:11px;background:currentColor;box-shadow:-10px -5px 0 -0.3px currentColor}
		.mobile-icon[data-icon='sales']::before{inset:1px 3px;border:2px solid currentColor;border-radius:2px}
		.mobile-icon[data-icon='sales']::after{width:9px;height:5px;left:7px;top:8px;border-left:2px solid currentColor;border-bottom:2px solid currentColor;transform:rotate(-45deg)}
		.mobile-icon[data-icon='limits']::before{width:3px;height:10px;left:3px;bottom:2px;background:currentColor;box-shadow:7px -5px 0 currentColor,14px -10px 0 currentColor}
		.mobile-icon[data-icon='limits']::after{left:1px;right:1px;bottom:0;height:2px;background:currentColor}
		.mobile-icon[data-icon='more']::before{width:4px;height:4px;left:2px;top:10px;border-radius:50%;background:currentColor;box-shadow:8px 0 0 currentColor,16px 0 0 currentColor}
		.sheet-backdrop{z-index:70;background:rgba(23,24,27,.58)}.more-sheet{z-index:71;top:auto;left:0;right:0;bottom:0;width:auto;max-height:min(82vh,46rem);border-left:0;border-top:4px solid #2738d6;box-shadow:0 -24px 70px rgba(23,24,27,.28);animation:sheet-in .2s ease-out}
		.more-sheet header{display:flex;align-items:center;justify-content:space-between;padding:1.05rem 1.15rem;border-bottom:1px solid #d7d2c8}.more-sheet header div{display:flex;flex-direction:column;gap:.14rem}.more-sheet header span{color:#2738d6;font-size:.68rem;font-weight:850;letter-spacing:.14em;text-transform:uppercase}.more-sheet header strong{font-family:Georgia,'Times New Roman',serif;font-size:1.35rem}.more-sheet header button{display:grid;place-items:center;width:2.75rem;height:2.75rem;border:1px solid #b8b4ab;background:transparent;color:#17181b;font-size:1.6rem;cursor:pointer}
		.sheet-content{max-height:calc(min(72vh,38rem) - 5rem);overflow-y:auto;padding:.3rem 1.15rem max(1.25rem,env(safe-area-inset-bottom))}.more-links a{min-height:3.45rem;font-size:.9rem}
		.sheet-actions{display:flex;gap:.5rem;padding-top:1rem}.sheet-actions button{flex:1;min-height:3.2rem;padding:.65rem;border:1px solid #17181b;background:transparent;color:#17181b;font:inherit;font-size:.82rem;font-weight:780;cursor:pointer}.sheet-actions .mobile-sign-out{border-color:#17181b;background:#17181b;color:#fff}
	}
	@media(max-width:360px){.account-label{display:none}}
	@keyframes sheet-in{from{transform:translateY(100%)}to{transform:translateY(0)}}
	@keyframes drawer-in{from{transform:translateX(100%)}to{transform:translateX(0)}}
	@media(prefers-reduced-motion:reduce){.more-sheet{animation:none}}
</style>
