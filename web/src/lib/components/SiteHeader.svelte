<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';

	const links = [
		['How it works', '/how-it-works'],
		['For sellers', '/for-suppliers'],
		['For customers', '/for-buyers'],
		['Price', '/pricing']
	];

	let menuOpen = $state(false);

	onMount(() => {
		const query = window.matchMedia('(min-width: 721px)');
		const sync = () => (menuOpen = query.matches);
		sync();
		query.addEventListener('change', sync);
		return () => query.removeEventListener('change', sync);
	});

	function closeMobileMenu() {
		if (typeof window !== 'undefined' && window.matchMedia('(max-width: 720px)').matches) {
			menuOpen = false;
		}
	}
</script>

<header class="site-header">
	<nav class="shell" aria-label="Main navigation">
		<a class="wordmark" href="/" aria-label="Kredit home" onclick={closeMobileMenu}>
			<span>K</span><b>Kredit</b>
		</a>
		<details class="site-menu-disclosure" bind:open={menuOpen}>
			<summary><span aria-hidden="true">☰</span>Menu</summary>
			<div id="public-navigation" class="site-menu">
				<div class="nav-links">
					{#each links as [label, href] (href)}
						<a
							class:active={page.url.pathname === href}
							aria-current={page.url.pathname === href ? 'page' : undefined}
							{href}
							onclick={closeMobileMenu}
						>
							{label}
						</a>
					{/each}
				</div>
				<div class="nav-actions">
					<a href="/security" onclick={closeMobileMenu}>Is it safe?</a>
					<a class="header-cta" href="/app" onclick={closeMobileMenu}>
						Start now <span aria-hidden="true">↗</span>
					</a>
				</div>
			</div>
		</details>
	</nav>
</header>

