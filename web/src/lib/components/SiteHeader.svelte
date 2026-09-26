<script lang="ts">
	import { page } from '$app/state';

	const links = [
		['How it works', '/how-it-works'],
		['Selling to individuals', '/consumers'],
		['Pricing', '/pricing'],
		['Questions', '/faq'],
		['Contact', '/contact']
	];
	let menu: HTMLDetailsElement;
	let scrolled = $state(false);
	function closeMobileMenu() {
		if (menu) menu.open = false;
	}
</script>

{#snippet navigation()}
	<div class="site-menu">
		<div class="nav-links">
			{#each links as [label, href] (href)}
				<a
					class:active={page.url.pathname === href}
					aria-current={page.url.pathname === href ? 'page' : undefined}
					{href}
					onclick={closeMobileMenu}>{label}</a
				>
			{/each}
		</div>
		<div class="nav-actions">
			<a href="/start" onclick={closeMobileMenu}>Sign in</a>
			<a class="header-cta" href="/demo" onclick={closeMobileMenu}>Try the demo <span aria-hidden="true">→</span></a>
		</div>
	</div>
{/snippet}

<svelte:window onscroll={() => (scrolled = window.scrollY > 8)} />

<header class="site-header" class:is-scrolled={scrolled}>
	<nav class="shell" aria-label="Main navigation">
		<a class="wordmark" href="/" aria-label="Kredit home" onclick={closeMobileMenu}><span>K</span><b>Kredit</b></a>
		<div class="desktop-site-menu">{@render navigation()}</div>
		<details class="site-menu-disclosure" bind:this={menu}>
			<summary><span aria-hidden="true">☰</span>Menu</summary>
			{@render navigation()}
		</details>
	</nav>
</header>

<style>
	.desktop-site-menu {
		flex: 1;
	}
	.site-menu-disclosure {
		display: none;
	}
	@media (max-width: 720px) {
		.desktop-site-menu {
			display: none;
		}
		.site-menu-disclosure {
			display: block;
		}
	}
</style>
