<script lang="ts">
	import { page } from '$app/state';
	import { tick } from 'svelte';

	type PortalLink = [string, string];
	type MobileLink = [string, string, string];
	type MoreLink = [string, string, string?];

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
		mobileMore?: MoreLink[];
		dark?: boolean;
		onsearch?: () => void;
		onsignout?: () => void | Promise<void>;
		searchReady?: boolean;
	} = $props();

	let open = $state(false);
	let signingOut = $state(false),
		signOutError = $state('');
	let dialog = $state<HTMLDialogElement | null>(null);
	async function leaveAccount() {
		if (!onsignout || signingOut) return;
		signingOut = true;
		signOutError = '';
		try {
			await onsignout();
		} catch {
			signOutError =
				'Sign-out was not confirmed. Your account may still be open. Try again before leaving this device.';
		} finally {
			signingOut = false;
		}
	}
	let moreOpen = $state(false);
	let moreTrigger: HTMLButtonElement | null = null;
	let closeButton = $state<HTMLButtonElement | null>(null);
	let menuID = $derived(`portal-nav-${label.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`);
	let moreID = $derived(`${menuID}-more`);
	const hasMobileBar = $derived(mobilePrimary.length > 0);

	function current(href: string) {
		const path = href.split('?')[0];
		if (href === homeHref) return page.url.pathname === path;
		return page.url.pathname === path || page.url.pathname.startsWith(`${path}/`);
	}

	function menuGroups() {
		const groups: { label: string; links: MoreLink[] }[] = [];
		for (const link of mobileMore) {
			const groupLabel = link[2] ?? 'Other pages';
			let group = groups.find((item) => item.label === groupLabel);
			if (!group) {
				group = { label: groupLabel, links: [] };
				groups.push(group);
			}
			group.links.push(link);
		}
		return groups;
	}

	function closeMenus(restoreFocus = false) {
		open = false;
		dialog?.close();
		moreOpen = false;
		if (restoreFocus) void tick().then(() => moreTrigger?.focus());
	}

	function openMore(event: MouseEvent) {
		moreTrigger = event.currentTarget as HTMLButtonElement;
		moreOpen = true;
		void tick().then(() => {
			dialog?.showModal();
			closeButton?.focus();
		});
	}

	function handleKeydown(event: KeyboardEvent) {
		if (!moreOpen || !dialog?.open) return;
		if (event.key === 'Escape') {
			event.preventDefault();
			closeMenus(true);
			return;
		}
		if (event.key !== 'Tab') return;
		const controls = Array.from(
			dialog.querySelectorAll<HTMLElement>(
				'a[href],button:not(:disabled),input:not(:disabled),select:not(:disabled),textarea:not(:disabled),[tabindex]:not([tabindex="-1"])'
			)
		).filter((node) => node.tabIndex >= 0 && node.getClientRects().length > 0);
		const first = controls[0],
			last = controls.at(-1);
		if (!first || !last) {
			event.preventDefault();
			dialog.focus();
			return;
		}
		if (event.shiftKey && (document.activeElement === first || !dialog.contains(document.activeElement))) {
			event.preventDefault();
			last.focus();
		} else if (!event.shiftKey && (document.activeElement === last || !dialog.contains(document.activeElement))) {
			event.preventDefault();
			first.focus();
		}
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
	<button
		class="menu-toggle"
		class:hidden-mobile-toggle={hasMobileBar}
		type="button"
		aria-controls={menuID}
		aria-expanded={open}
		onclick={() => (open = !open)}
	>
		<span aria-hidden="true">{open ? '×' : '☰'}</span>{open ? 'Close' : 'Menu'}
	</button>
	<div id={menuID} class="portal-menu" class:open>
		{#if hasMobileBar}
			<div class="desktop-primary">
				{#each mobilePrimary as [linkLabel, href, icon], i (i)}
					<a
						{href}
						class:give={icon === 'add'}
						aria-current={current(href) ? 'page' : undefined}
						onclick={() => closeMenus()}>{linkLabel}</a
					>
				{/each}
			</div>
		{:else}
			<div class="portal-links">
				{#each links as [linkLabel, href], i (i)}
					<a {href} aria-current={current(href) ? 'page' : undefined} onclick={() => closeMenus()}>{linkLabel}</a>
				{/each}
			</div>
		{/if}
		<div class="portal-actions">
			{#if hasMobileBar}<button
					class="desktop-header-menu"
					type="button"
					class:active={moreOpen || mobileMore.some(([, href]) => current(href))}
					aria-controls={moreID}
					aria-expanded={moreOpen}
					onclick={openMore}><span aria-hidden="true">☰</span> Menu</button
				>{/if}
			{#if onsearch}<button
					class="search palette-trigger"
					data-ready={searchReady}
					disabled={!searchReady}
					type="button"
					onclick={onsearch}>Search <kbd>⌘K</kbd></button
				>{/if}
			{#if onsignout}<button class="sign-out" type="button" disabled={signingOut} onclick={leaveAccount}
					>{signingOut ? 'Signing out…' : 'Sign out'}</button
				>{/if}
		</div>
	</div>
	{#if hasMobileBar}
		<button class="mobile-header-menu" type="button" aria-controls={moreID} aria-expanded={moreOpen} onclick={openMore}>
			<span aria-hidden="true">☰</span>Menu
		</button>
	{/if}
</nav>

{#if hasMobileBar}
	<div class="mobile-nav" style:--nav-count={mobilePrimary.length} role="navigation" aria-label={`${label} main pages`}>
		{#each mobilePrimary as [linkLabel, href, icon], i (i)}
			<a
				{href}
				class:give={icon === 'add'}
				aria-current={current(href) ? 'page' : undefined}
				onclick={() => closeMenus()}
			>
				<span class="mobile-icon" data-icon={icon} aria-hidden="true"></span>
				<span>{linkLabel}</span>
			</a>
		{/each}
	</div>

	{#if moreOpen}
		<dialog
			bind:this={dialog}
			class="more-sheet"
			id={moreID}
			aria-label={`${label} menu`}
			oncancel={(event) => {
				event.preventDefault();
				closeMenus(true);
			}}
		>
			<header>
				<div><span>All pages</span><strong id={`${moreID}-title`}>{label}</strong></div>
				<button bind:this={closeButton} type="button" aria-label="Close account menu" onclick={() => closeMenus(true)}
					>×</button
				>
			</header>
			<div class="sheet-content">
				<div class="more-links" role="navigation" aria-label="Account menu pages" data-sveltekit-preload-data="tap">
					{#each menuGroups() as group (group.label)}
						<section class="menu-group">
							<h2>{group.label}</h2>
							<div>
								{#each group.links as [linkLabel, href], linkIndex (linkIndex)}<a
										{href}
										aria-current={current(href) ? 'page' : undefined}
										onclick={() => closeMenus()}><span>{linkLabel}</span><span aria-hidden="true">→</span></a
									>{/each}
							</div>
						</section>
					{/each}
				</div>
				{#if signOutError}<p class="signout-error" role="alert">{signOutError}</p>{/if}
				<div class="sheet-actions">
					{#if onsearch}<button
							type="button"
							onclick={() => {
								closeMenus();
								onsearch?.();
							}}>Search this account</button
						>{/if}
					{#if onsignout}<button class="mobile-sign-out" type="button" disabled={signingOut} onclick={leaveAccount}
							>{signingOut ? 'Signing out…' : 'Sign out'}</button
						>{/if}
				</div>
			</div>
		</dialog>
	{/if}
{/if}
{#if signOutError && !moreOpen}<div class="signout-error" role="alert">
		<p>{signOutError}</p>
		<button disabled={signingOut} onclick={leaveAccount}>Try signing out again</button>
	</div>{/if}

<style>
	.signout-error {
		position: relative;
		z-index: 58;
		margin: 0;
		padding: 1rem;
		background: var(--color-background);
		color: var(--color-warning);
		border-bottom: 1px solid var(--color-warning);
		line-height: 1.6;
	}
	.signout-error p {
		margin: 0 0 0.5rem;
	}
	.signout-error button {
		padding: 0.6rem 1rem;
		border: 1px solid currentColor;
		background: transparent;
		color: inherit;
	}
	.more-sheet:not([open]) {
		display: none;
	}
	.more-sheet {
		margin: 0 0 0 auto;
		max-width: none;
		max-height: none;
		padding: 0;
		color: var(--color-foreground);
	}
	.more-sheet::backdrop {
		background: rgb(23 24 27 / 0.48);
	}

	/* The workspace bar is a dark surface, so inside it the tokens have to mean
	   what they mean on the ink ground. Without this, "muted" resolves to a dark
	   grey and every inactive link disappears into the bar. Same device the
	   public site uses in section 2 of app.css. */
	nav {
		display: flex;
		align-items: center;
		gap: 1.2rem;
		padding: 0.75rem max(1rem, calc((100vw - 76rem) / 2));
		border-bottom: 1px solid rgb(255 255 255 / 0.07);
		background: #0b0d12;
		position: sticky;
		top: 0;
		z-index: 40;
		--color-foreground: #f4f1ea;
		--color-muted: #a9a69e;
		--color-on-primary: #f4f1ea;
		--color-primary: #e2603a;
		--color-border: #272c38;
		--color-border-strong: #39404f;
	}
	.brand {
		display: inline-flex;
		align-items: center;
		gap: 0.65rem;
		color: var(--color-on-primary);
		font-family: var(--font-serif);
		font-weight: 650;
		font-size: 1.15rem;
		text-decoration: none;
		white-space: nowrap;
	}
	.brand span {
		display: grid;
		place-items: center;
		width: 2rem;
		height: 2rem;
		background: var(--color-primary);
		color: var(--color-on-primary);
	}
	.account-label {
		display: none;
		color: var(--color-muted);
		font-size: 0.68rem;
		font-weight: 800;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}
	.portal-menu {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		min-width: 0;
		flex: 1;
	}
	.portal-links {
		display: flex;
		align-items: center;
		gap: 0.1rem;
		min-width: 0;
		overflow-x: auto;
		scrollbar-width: none;
	}
	.portal-links::-webkit-scrollbar {
		display: none;
	}
	.portal-links a {
		position: relative;
		display: inline-flex;
		align-items: center;
		min-height: 2.75rem;
		padding: 0.15rem 0.68rem;
		color: var(--color-muted);
		font-size: 0.84rem;
		font-weight: 680;
		text-decoration: none;
		white-space: nowrap;
	}
	.portal-links a:hover {
		color: var(--color-on-primary);
	}
	.portal-links a[aria-current='page'] {
		color: var(--color-on-primary);
		background: transparent;
	}
	.portal-links a[aria-current='page']::after {
		content: '';
		position: absolute;
		left: 0.68rem;
		right: 0.68rem;
		bottom: 0.3rem;
		height: 1px;
		background: var(--color-primary);
	}
	.portal-actions {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		margin-left: auto;
	}
	.portal-actions button,
	.menu-toggle,
	.mobile-header-menu {
		border: 0;
		background: transparent;
		color: var(--color-muted);
		font: inherit;
		font-size: 0.82rem;
		font-weight: 680;
		cursor: pointer;
		white-space: nowrap;
	}
	.portal-actions .search,
	.portal-actions .desktop-header-menu {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.45rem 0.7rem;
		border: 1px solid var(--color-border-strong);
	}
	.portal-actions .desktop-header-menu.active {
		border-color: var(--color-primary);
		color: var(--color-on-primary);
	}
	.desktop-header-menu span {
		color: var(--color-overdue);
		font-size: 1rem;
	}
	.search kbd {
		font-family: inherit;
		font-size: 0.7rem;
		color: var(--color-muted);
	}
	.menu-toggle,
	.mobile-header-menu {
		display: none;
	}
	.menu-toggle span,
	.mobile-header-menu span {
		font-size: 1.2rem;
		color: var(--color-accent);
	}
	nav.dark {
		background: #0b0d12;
		border-color: rgb(255 255 255 / 0.07);
	}
	.dark .brand,
	.dark .portal-links a,
	.dark .portal-actions button {
		color: var(--color-on-primary);
	}
	.dark .portal-links a:hover,
	.dark .portal-links a[aria-current='page'] {
		color: var(--color-on-primary);
		background: transparent;
	}
	.portal-links a[aria-current='page']::after {
		content: '';
		position: absolute;
		left: 0.68rem;
		right: 0.68rem;
		bottom: 0.3rem;
		height: 1px;
		background: var(--color-primary);
	}
	.desktop-primary {
		display: flex;
		align-items: center;
		gap: 0.1rem;
		min-width: 0;
	}
	.desktop-primary a {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		min-height: 2.75rem;
		padding: 0.15rem 0.78rem;
		border: 0;
		background: transparent;
		color: var(--color-muted);
		font: inherit;
		font-size: 0.84rem;
		font-weight: 680;
		text-decoration: none;
		white-space: nowrap;
		cursor: pointer;
	}
	.desktop-primary a:hover {
		color: var(--color-on-primary);
	}
	.desktop-primary a[aria-current='page'] {
		position: relative;
		color: var(--color-on-primary);
		background: transparent;
	}
	/* Giving credit is the one action; it reads as a button, not a tab. */
	.desktop-primary a.give {
		margin-left: 0.5rem;
		padding-inline: 1rem;
		border-radius: 999px;
		background: var(--color-accent);
		color: #fff;
	}
	.desktop-primary a.give:hover,
	.desktop-primary a.give[aria-current='page'] {
		color: #fff;
		filter: brightness(1.08);
	}
	.desktop-primary a.give[aria-current='page']::after {
		display: none;
	}
	.desktop-primary a[aria-current='page']::after {
		content: '';
		position: absolute;
		left: 0.68rem;
		right: 0.68rem;
		bottom: 0.3rem;
		height: 1px;
		background: var(--color-primary);
	}
	.mobile-nav {
		display: none;
	}
	.more-sheet {
		position: fixed;
		z-index: 61;
		top: 3.65rem;
		right: 0;
		bottom: 0;
		display: block;
		width: min(26rem, calc(100vw - 2rem));
		overflow: hidden;
		border-left: 1px solid var(--color-border);
		border-top: 4px solid var(--color-primary);
		background: var(--color-surface);
		box-shadow: -24px 0 70px rgba(23, 24, 27, 0.2);
		animation: drawer-in 0.22s cubic-bezier(0.22, 0.8, 0.28, 1);
	}
	.more-sheet header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1.05rem 1.25rem;
		border-bottom: 1px solid var(--color-border);
	}
	.more-sheet header div {
		display: flex;
		flex-direction: column;
		gap: 0.14rem;
	}
	.more-sheet header span {
		color: var(--color-primary);
		font-size: 0.68rem;
		font-weight: 850;
		letter-spacing: 0.14em;
		text-transform: uppercase;
	}
	.more-sheet header strong {
		font-family: var(--font-serif);
		font-size: 1.45rem;
	}
	.more-sheet header button {
		display: grid;
		place-items: center;
		width: 2.75rem;
		height: 2.75rem;
		border: 1px solid var(--color-border);
		background: transparent;
		color: var(--color-primary);
		font-size: 1.6rem;
		cursor: pointer;
	}
	.sheet-content {
		max-height: calc(100vh - 8.8rem);
		overflow-y: auto;
		padding: 0.35rem 1.25rem 1.4rem;
	}
	.more-links {
		display: grid;
		gap: 0.35rem;
	}
	.menu-group {
		margin-top: 1rem;
	}
	.menu-group h2 {
		margin: 0;
		padding: 0.55rem 0.7rem;
		background: var(--color-surface-muted);
		color: var(--color-foreground);
		font-size: 0.78rem;
		font-weight: 850;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	.menu-group > div {
		display: grid;
	}
	.more-links a {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		min-height: 3.6rem;
		padding: 0.35rem 0.7rem;
		border-bottom: 1px solid var(--color-border);
		color: var(--color-primary);
		font-size: 0.92rem;
		font-weight: 740;
		text-decoration: none;
	}
	.more-links a span:last-child {
		color: var(--color-accent);
		font-size: 1rem;
	}
	.more-links a[aria-current='page'] {
		color: var(--color-primary);
		background: var(--color-background);
	}
	.sheet-actions {
		display: flex;
		gap: 0.5rem;
		padding-top: 1rem;
	}
	.sheet-actions button {
		width: 100%;
		min-height: 3.2rem;
		padding: 0.65rem;
		border: 1px solid var(--color-primary);
		background: transparent;
		color: var(--color-primary);
		font: inherit;
		font-size: 0.82rem;
		font-weight: 780;
		cursor: pointer;
	}
	.sheet-actions .mobile-sign-out {
		border-color: var(--color-primary);
		background: var(--color-primary);
		color: var(--color-on-primary);
	}

	/* Between a phone and a full desktop the bar holds five links, the menu and
	   search. Sign out is also in the menu sheet, and the shortcut hint is for
	   keyboards, so both leave the bar here rather than crowd the links. */
	@media (min-width: 761px) and (max-width: 1100px) {
		nav {
			gap: 0.7rem;
		}
		.portal-actions .sign-out,
		.search kbd {
			display: none;
		}
		.portal-links a {
			padding-inline: 0.5rem;
		}
	}

	@media (max-width: 760px) {
		nav {
			position: sticky;
			flex-wrap: wrap;
			padding: 0.72rem 1rem;
		}
		.account-label {
			display: block;
			margin-left: auto;
		}
		.has-mobile-bar .portal-menu {
			display: none;
		}
		.hidden-mobile-toggle {
			display: none !important;
		}
		.mobile-header-menu {
			display: inline-flex;
			min-height: 2.65rem;
			align-items: center;
			gap: 0.5rem;
			padding: 0 0.8rem;
			border: 1px solid var(--color-border-strong);
		}
		.menu-toggle {
			display: inline-flex;
			min-height: 2.65rem;
			align-items: center;
			gap: 0.5rem;
			margin-left: auto;
			padding: 0 0.85rem;
			border: 1px solid var(--color-border-strong);
		}
		.portal-menu {
			display: none;
			width: 100%;
			align-items: stretch;
			flex-direction: column;
			padding: 1rem 0 0.35rem;
		}
		.portal-menu.open {
			display: flex;
		}
		.portal-links {
			display: grid;
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: 0.35rem;
			overflow: visible;
		}
		.portal-links a {
			min-height: 3rem;
			padding: 0.4rem 0.75rem;
			white-space: normal;
			border: 1px solid var(--color-border-strong);
		}
		.portal-actions {
			width: 100%;
			margin: 0;
			padding-top: 0.75rem;
			border-top: 1px solid var(--color-border-strong);
		}
		.portal-actions .sign-out {
			margin-left: auto;
		}
		.mobile-nav {
			position: fixed;
			z-index: 55;
			left: 0;
			right: 0;
			bottom: 0;
			display: grid;
			grid-template-columns: repeat(var(--nav-count, 4), minmax(0, 1fr));
			min-height: 4.7rem;
			padding: 0 0.25rem max(0.35rem, env(safe-area-inset-bottom));
			border-top: 1px solid var(--color-border);
			background: rgba(250, 248, 242, 0.98);
			box-shadow: 0 -8px 28px rgba(23, 24, 27, 0.09);
			backdrop-filter: none;
		}
		.mobile-nav a {
			position: relative;
			display: flex;
			min-width: 0;
			min-height: 4.35rem;
			flex-direction: column;
			align-items: center;
			justify-content: center;
			gap: 0.32rem;
			padding: 0.48rem 0.15rem 0.25rem;
			border: 0;
			background: transparent;
			color: var(--color-foreground);
			font: inherit;
			font-size: 0.78rem;
			font-weight: 760;
			line-height: 1.1;
			text-align: center;
			text-decoration: none;
			cursor: pointer;
			transition:
				color 0.16s ease,
				background-color 0.18s ease;
		}
		.mobile-nav a::after {
			content: '';
			position: absolute;
			top: 0;
			left: 22%;
			right: 22%;
			height: 3px;
			background: transparent;
		}
		.mobile-nav a[aria-current='page'] {
			background: var(--color-background);
			color: var(--color-primary);
		}
		.mobile-nav a[aria-current='page']::after {
			background: var(--color-primary);
		}
		.mobile-icon {
			position: relative;
			display: block;
			width: 1.45rem;
			height: 1.45rem;
			color: currentColor;
			transition: transform 0.18s cubic-bezier(0.2, 0.9, 0.3, 1);
		}
		.mobile-nav a.give {
			color: var(--color-accent);
		}
		.mobile-nav a.give .mobile-icon {
			border-radius: 999px;
			background: var(--color-accent);
			color: #fff;
		}
		.mobile-nav a[aria-current='page'] .mobile-icon {
			transform: translateY(-1px) scale(1.06);
		}
		.mobile-icon::before,
		.mobile-icon::after {
			content: '';
			position: absolute;
			box-sizing: border-box;
		}
		.mobile-icon[data-icon='home']::before {
			inset: 3px 3px 2px;
			border: 2px solid currentColor;
			border-top: 0;
		}
		.mobile-icon[data-icon='home']::after {
			width: 13px;
			height: 13px;
			left: 5px;
			top: 0;
			border-left: 2px solid currentColor;
			border-top: 2px solid currentColor;
			transform: rotate(45deg);
		}
		.mobile-icon[data-icon='add']::before {
			width: 18px;
			height: 2px;
			left: 3px;
			top: 11px;
			background: currentColor;
		}
		.mobile-icon[data-icon='add']::after {
			width: 2px;
			height: 18px;
			left: 11px;
			top: 3px;
			background: currentColor;
		}
		.mobile-icon[data-icon='customers']::before {
			width: 9px;
			height: 9px;
			left: 7px;
			top: 1px;
			border: 2px solid currentColor;
			border-radius: 50%;
		}
		.mobile-icon[data-icon='customers']::after {
			width: 18px;
			height: 10px;
			left: 3px;
			bottom: 1px;
			border: 2px solid currentColor;
			border-radius: 10px 10px 2px 2px;
		}
		.mobile-icon[data-icon='payments']::before,
		.mobile-icon[data-icon='owe']::before {
			inset: 3px 1px;
			border: 2px solid currentColor;
			border-radius: 2px;
		}
		.mobile-icon[data-icon='payments']::after,
		.mobile-icon[data-icon='owe']::after {
			width: 7px;
			height: 2px;
			right: 4px;
			top: 11px;
			background: currentColor;
			box-shadow: -10px -5px 0 -0.3px currentColor;
		}
		.mobile-icon[data-icon='sales']::before {
			inset: 1px 3px;
			border: 2px solid currentColor;
			border-radius: 2px;
		}
		.mobile-icon[data-icon='sales']::after {
			width: 9px;
			height: 5px;
			left: 7px;
			top: 8px;
			border-left: 2px solid currentColor;
			border-bottom: 2px solid currentColor;
			transform: rotate(-45deg);
		}
		.mobile-icon[data-icon='limits']::before {
			width: 3px;
			height: 10px;
			left: 3px;
			bottom: 2px;
			background: currentColor;
			box-shadow:
				7px -5px 0 currentColor,
				14px -10px 0 currentColor;
		}
		.mobile-icon[data-icon='limits']::after {
			left: 1px;
			right: 1px;
			bottom: 0;
			height: 2px;
			background: currentColor;
		}
		.more-sheet {
			z-index: 71;
			top: auto;
			left: 0;
			right: 0;
			bottom: 0;
			width: auto;
			margin: 0;
			max-height: min(82vh, 46rem);
			border-left: 0;
			border-top: 4px solid var(--color-primary);
			box-shadow: 0 -24px 70px rgba(23, 24, 27, 0.28);
			animation: sheet-in 0.2s ease-out;
		}
		.more-sheet header {
			display: flex;
			align-items: center;
			justify-content: space-between;
			padding: 1.05rem 1.15rem;
			border-bottom: 1px solid var(--color-border);
		}
		.more-sheet header div {
			display: flex;
			flex-direction: column;
			gap: 0.14rem;
		}
		.more-sheet header span {
			color: var(--color-primary);
			font-size: 0.68rem;
			font-weight: 850;
			letter-spacing: 0.14em;
			text-transform: uppercase;
		}
		.more-sheet header strong {
			font-family: var(--font-serif);
			font-size: 1.35rem;
		}
		.more-sheet header button {
			display: grid;
			place-items: center;
			width: 2.75rem;
			height: 2.75rem;
			border: 1px solid var(--color-border);
			background: transparent;
			color: var(--color-primary);
			font-size: 1.6rem;
			cursor: pointer;
		}
		.sheet-content {
			max-height: calc(min(72vh, 38rem) - 5rem);
			overflow-y: auto;
			padding: 0.3rem 1.15rem max(1.25rem, env(safe-area-inset-bottom));
		}
		.more-links a {
			min-height: 3.45rem;
			font-size: 0.9rem;
		}
		.sheet-actions {
			display: flex;
			gap: 0.5rem;
			padding-top: 1rem;
		}
		.sheet-actions button {
			flex: 1;
			min-height: 3.2rem;
			padding: 0.65rem;
			border: 1px solid var(--color-primary);
			background: transparent;
			color: var(--color-primary);
			font: inherit;
			font-size: 0.82rem;
			font-weight: 780;
			cursor: pointer;
		}
		.sheet-actions .mobile-sign-out {
			border-color: var(--color-primary);
			background: var(--color-primary);
			color: var(--color-on-primary);
		}
	}
	@media (max-width: 390px) {
		.account-label {
			display: none;
		}
		.mobile-header-menu {
			margin-left: auto;
		}
	}
	@keyframes sheet-in {
		from {
			transform: translateY(100%);
		}
		to {
			transform: translateY(0);
		}
	}
	@keyframes drawer-in {
		from {
			transform: translateX(100%);
		}
		to {
			transform: translateX(0);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.more-sheet {
			animation: none;
		}
	}
</style>
