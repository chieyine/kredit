<script lang="ts">
	import { page } from '$app/state';

	const links = [
		['How it works', '/how-it-works'],
		['For sellers', '/for-suppliers'],
		['For customers', '/for-buyers'],
		['Price', '/pricing']
	];

	function closeMobileMenu(event: MouseEvent) {
		if (typeof window !== 'undefined' && window.matchMedia('(max-width: 720px)').matches) {
			(event.currentTarget as HTMLElement).closest('details')?.removeAttribute('open');
		}
	}
</script>

<header class="site-header">
	<nav class="shell" aria-label="Main navigation">
		<a class="wordmark" href="/" aria-label="Kredit home" onclick={closeMobileMenu}>
			<span>K</span><b>Kredit</b>
		</a>
		<details class="site-menu-disclosure">
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
					<a href="/app" onclick={closeMobileMenu}>Sign in</a>
					<a class="header-cta" href="/demo" onclick={closeMobileMenu}>
						Try the demo <span aria-hidden="true">↗</span>
					</a>
				</div>
			</div>
		</details>
	</nav>
</header>
