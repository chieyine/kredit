<script lang="ts">
	import { onMount } from 'svelte';
	import type { Slide } from '$lib/deck/types';

	let { slides, title }: { slides: Slide[]; title: string } = $props();

	let index = $state(0);
	let printing = $state(false);
	const total = $derived(slides.length);

	function go(n: number) {
		index = Math.max(0, Math.min(total - 1, n));
		if (typeof history !== 'undefined') history.replaceState(null, '', `#${index + 1}`);
	}
	function key(event: KeyboardEvent) {
		if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey) return;
		// Preserve native keyboard activation of links, buttons and editing controls.
		const target = event.target;
		if (target instanceof Element && target.closest('a, button, input, textarea, select, [contenteditable]')) return;
		const k = event.key;
		if (k === 'ArrowRight' || k === 'PageDown' || k === ' ') {
			event.preventDefault();
			go(index + 1);
		} else if (k === 'ArrowLeft' || k === 'PageUp') {
			event.preventDefault();
			go(index - 1);
		} else if (k === 'Home') {
			event.preventDefault();
			go(0);
		} else if (k === 'End') {
			event.preventDefault();
			go(total - 1);
		}
	}
	onMount(() => {
		const fromHash = Number((location.hash || '').replace('#', ''));
		if (Number.isInteger(fromHash) && fromHash >= 1 && fromHash <= total) index = fromHash - 1;
		// someone deep-linking to #7 while the deck is already open should land there
		const hash = () => {
			const n = Number((location.hash || '').replace('#', ''));
			if (Number.isInteger(n) && n >= 1 && n <= total) index = n - 1;
		};
		window.addEventListener('hashchange', hash);
		const before = () => (printing = true);
		const after = () => (printing = false);
		window.addEventListener('beforeprint', before);
		window.addEventListener('afterprint', after);
		return () => {
			window.removeEventListener('hashchange', hash);
			window.removeEventListener('beforeprint', before);
			window.removeEventListener('afterprint', after);
		};
	});
</script>

<svelte:window onkeydown={key} />
<!-- Route-level discovery policy is emitted once by the shared layout. -->
<svelte:head><title>{title}</title></svelte:head>

<div class="deck" class:printing>
	{#each slides as slide, i}
		<!-- Every slide stays in the DOM so browser print gives one page each. -->
		<section class="slide" class:current={i === index} aria-hidden={i === index || printing ? undefined : 'true'}>
			<div class="slide-inner" data-kind={slide.kind}>
				<header class="slide-head">
					<span class="seal" aria-hidden="true">K</span>
					{#if slide.eyebrow}<p class="deck-eyebrow">{slide.eyebrow}</p>{/if}
				</header>

				{#if slide.kind === 'cover'}
					<div class="cover">
						<!-- eslint-disable-next-line svelte/no-at-html-tags -- Slide titles come from the hardcoded decks in $lib/deck, never from user or network input. If decks ever become editable content this must be sanitised first. -->
						<h1>{@html slide.title}</h1>
						{#if slide.lede}<p class="lede">{slide.lede}</p>{/if}
						{#if slide.links}
							<div class="cover-links">
								{#each slide.links as [label, href]}
									<a class="cover-link" {href}>{label} <span aria-hidden="true">→</span></a>
								{/each}
							</div>
						{/if}
						{#if slide.meta}<p class="cover-meta">{slide.meta}</p>{/if}
					</div>
				{:else}
					<div class="body">
						<!-- eslint-disable-next-line svelte/no-at-html-tags -- Slide titles come from the hardcoded decks in $lib/deck, never from user or network input. If decks ever become editable content this must be sanitised first. -->
						<h2>{@html slide.title}</h2>
						{#if slide.lede}<p class="lede">{slide.lede}</p>{/if}

						{#if slide.figures}
							<div class="figures" style={`--cols:${slide.figures.length}`}>
								{#each slide.figures as f}
									<div class="fig">
										<p class="fig-label">{f.label}</p>
										<strong class="fig-value" class:alert={f.tone === 'alert'}>{f.value}</strong>
										{#if f.note}<p class="fig-note">{f.note}</p>{/if}
									</div>
								{/each}
							</div>
						{/if}

						{#if slide.rows}
							<div class="rows">
								<div class="row row-head">
									{#each slide.rows.columns as c, ci}<span class:num={ci > 0}>{c}</span>{/each}
								</div>
								{#each slide.rows.data as r}
									<div class="row">
										{#each r as cell, ci}<span class:num={ci > 0}>{cell}</span>{/each}
									</div>
								{/each}
							</div>
						{/if}

						{#if slide.points}
							<ol class="points">
								{#each slide.points as [head, detail]}
									<li><strong>{head}</strong><span>{detail}</span></li>
								{/each}
							</ol>
						{/if}

						{#if slide.quote}
							<blockquote class="quote">
								<p>{slide.quote.text}</p>
								<cite>{slide.quote.who}</cite>
							</blockquote>
						{/if}

						{#if slide.note}<p class="slide-note">{slide.note}</p>{/if}
					</div>
				{/if}

				<footer class="slide-foot">
					{#if slide.source}<p class="source">{slide.source}</p>{:else}<span></span>{/if}
					<p class="pager"><span>{String(i + 1).padStart(2, '0')}</span> / {String(total).padStart(2, '0')}</p>
				</footer>
			</div>
		</section>
	{/each}

	<a class="deck-exit" href="/">Leave the deck <span aria-hidden="true">→</span></a>

	<nav class="deck-nav" aria-label="Slides">
		<button type="button" onclick={() => go(index - 1)} disabled={index === 0} aria-label="Previous slide">←</button>
		<span class="deck-progress" aria-hidden="true"><i style={`width:${((index + 1) / total) * 100}%`}></i></span>
		<button type="button" onclick={() => go(index + 1)} disabled={index === total - 1} aria-label="Next slide">→</button
		>
	</nav>
</div>

<style>
	/*
   * A slide is a fixed 16:9 stage that scales to the viewport, so what the room
   * sees is exactly what the browser print gives back as a PDF page. Everything
   * resolves against the site's own tokens, so the deck cannot drift from the
   * brand.
   */
	.deck {
		position: fixed;
		inset: 0;
		display: grid;
		place-items: center;
		background: var(--color-background);
		overflow: hidden;
	}

	.slide {
		position: absolute;
		inset: 0;
		display: none;
		place-items: center;
		padding: clamp(1rem, 3vw, 2.5rem) clamp(1rem, 3vw, 2.5rem) 4.6rem;
	}
	.slide.current {
		display: grid;
	}

	.slide-inner {
		/* the reserved band below is where the controls live, so they never sit on
       top of the source line */
		width: min(100%, calc((100svh - 8rem) * 16 / 9));
		aspect-ratio: 16 / 9;
		display: grid;
		grid-template-rows: auto minmax(0, 1fr) auto;
		gap: clamp(0.75rem, 1.6vw, 1.6rem);
		padding: clamp(1.4rem, 3.4vw, 3.4rem);
		border: 1px solid var(--color-border);
		background:
			radial-gradient(60% 90% at 88% -10%, rgb(45 56 92 / 0.5), transparent 62%),
			radial-gradient(50% 70% at 2% 8%, rgb(226 96 58 / 0.07), transparent 60%), var(--color-surface);
		container-type: inline-size;
		overflow: hidden;
	}

	.slide-head {
		display: flex;
		align-items: center;
		gap: 0.9rem;
	}
	.seal {
		display: grid;
		place-items: center;
		flex: none;
		width: 1.9cqw;
		height: 1.9cqw;
		min-width: 20px;
		min-height: 20px;
		background: var(--color-accent-ink);
		color: #fffcf7;
		font-family: var(--font-display);
		font-weight: 600;
		font-size: 1.05cqw;
		line-height: 1;
		clip-path: polygon(
			3% 5%,
			33% 1%,
			68% 4%,
			97% 0%,
			100% 32%,
			96% 66%,
			99% 96%,
			70% 99%,
			34% 96%,
			4% 100%,
			1% 67%,
			5% 34%
		);
	}
	.deck-eyebrow {
		margin: 0;
		color: var(--color-muted);
		font-size: 1.05cqw;
		font-weight: 600;
		letter-spacing: 0.16em;
		text-transform: uppercase;
	}

	.cover {
		align-self: center;
	}
	.cover h1 {
		margin: 0 0 2.4cqw;
		font-family: var(--font-display);
		font-variation-settings: 'opsz' 144;
		font-size: 7.4cqw;
		font-weight: 450;
		line-height: 0.95;
		letter-spacing: -0.045em;
		max-width: 16ch;
	}
	.cover-meta {
		margin: 3cqw 0 0;
		color: var(--color-muted);
		font-size: 1.3cqw;
		letter-spacing: 0.04em;
	}

	.body {
		align-self: center;
		min-height: 0;
		max-height: 100%;
	}
	.body h2 {
		margin: 0 0 1.35cqw;
		font-family: var(--font-display);
		font-variation-settings: 'opsz' 96;
		font-size: 4.6cqw;
		font-weight: 450;
		line-height: 1.02;
		letter-spacing: -0.035em;
		max-width: 20ch;
	}
	:global(.deck em) {
		font-style: normal;
		color: var(--color-accent-ink);
	}
	.lede {
		margin: 0;
		max-width: 58ch;
		color: var(--color-muted);
		font-size: 1.55cqw;
		line-height: 1.55;
	}

	/* Figures: the numbers are the argument, so they get the display face. */
	.figures {
		display: grid;
		grid-template-columns: repeat(var(--cols, 3), minmax(0, 1fr));
		gap: 0;
		margin-top: 2.6cqw;
		border-top: 1px solid var(--color-border);
	}
	.fig {
		padding: 1.8cqw 1.8cqw 0 0;
		border-right: 1px solid var(--color-border);
	}
	.fig:last-child {
		border-right: 0;
	}
	.fig + .fig {
		padding-left: 1.8cqw;
	}
	.fig-label {
		margin: 0;
		color: var(--color-muted);
		font-size: 0.95cqw;
		font-weight: 600;
		letter-spacing: 0.14em;
		text-transform: uppercase;
	}
	.fig-value {
		display: block;
		margin: 0.8cqw 0 0.5cqw;
		font-family: var(--font-display);
		font-variation-settings: 'opsz' 96;
		font-size: 4.3cqw;
		font-weight: 450;
		letter-spacing: -0.04em;
		line-height: 1;
		font-variant-numeric: tabular-nums;
	}
	.fig-value.alert {
		color: var(--color-accent-ink);
	}
	.fig-note {
		margin: 0;
		color: var(--color-muted);
		font-size: 1.05cqw;
		line-height: 1.45;
	}

	/* A ledger table, the same object the product shows. */
	.rows {
		margin-top: 2.2cqw;
		font-variant-numeric: tabular-nums;
	}
	.row {
		display: grid;
		grid-template-columns: 2fr repeat(3, minmax(0, 1fr));
		gap: 1.4cqw;
		padding: 1.15cqw 0;
		border-bottom: 1px solid var(--color-border);
		font-size: 1.35cqw;
	}
	.row span.num {
		text-align: right;
	}
	.row-head span {
		color: var(--color-muted);
		font-size: 0.95cqw;
		font-weight: 600;
		letter-spacing: 0.13em;
		text-transform: uppercase;
	}
	.row:not(.row-head) span:first-child {
		font-weight: 600;
	}

	.points {
		display: grid;
		gap: 0;
		margin: 2.2cqw 0 0;
		padding: 0;
		list-style: none;
		counter-reset: p;
		border-top: 1px solid var(--color-border);
	}
	.points li {
		display: grid;
		grid-template-columns: 3.4cqw 1fr;
		gap: 1.2cqw;
		align-items: baseline;
		padding: 1.15cqw 0;
		border-bottom: 1px solid var(--color-border);
		counter-increment: p;
	}
	.points li::before {
		content: counter(p, decimal-leading-zero);
		font-family: var(--font-display);
		color: var(--color-accent-ink);
		font-size: 1.5cqw;
	}
	.points strong {
		grid-column: 2;
		display: block;
		font-size: 1.6cqw;
		font-weight: 600;
		letter-spacing: -0.01em;
	}
	/* the detail belongs under the heading, not back in the marker column */
	.points span {
		grid-column: 2;
		display: block;
		margin-top: 0.35cqw;
		color: var(--color-muted);
		font-size: 1.3cqw;
		line-height: 1.5;
		max-width: 72ch;
	}

	.quote {
		margin: 2.4cqw 0 0;
		padding-left: 2cqw;
		border-left: 2px solid var(--color-accent-ink);
	}
	.quote p {
		margin: 0;
		font-family: var(--font-display);
		font-size: 2.5cqw;
		font-weight: 450;
		line-height: 1.22;
		letter-spacing: -0.025em;
		max-width: 26ch;
	}
	.quote cite {
		display: block;
		margin-top: 1.2cqw;
		color: var(--color-muted);
		font-size: 1.15cqw;
		font-style: normal;
		letter-spacing: 0.04em;
	}

	.slide-note {
		margin: 1.6cqw 0 0;
		padding: 1.05cqw 1.3cqw;
		border-left: 2px solid var(--color-border-strong);
		background: rgb(255 255 255 / 0.03);
		color: var(--color-muted);
		font-size: 1.2cqw;
		line-height: 1.5;
		max-width: 74ch;
	}

	.slide-foot {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: 2cqw;
		padding-top: 1cqw;
		border-top: 1px solid var(--color-border);
	}
	.source {
		margin: 0;
		max-width: 78ch;
		color: var(--color-muted);
		font-size: 0.95cqw;
		line-height: 1.45;
	}
	.pager {
		margin: 0;
		color: var(--color-muted);
		font-size: 1cqw;
		font-variant-numeric: tabular-nums;
		letter-spacing: 0.1em;
	}
	.pager span {
		color: var(--color-foreground);
	}

	.cover-links {
		display: flex;
		flex-wrap: wrap;
		gap: 1.6cqw 2.4cqw;
		margin-top: 3cqw;
	}
	.cover-link {
		display: inline-flex;
		align-items: center;
		gap: 0.5em;
		min-height: 44px;
		color: var(--color-foreground);
		font-size: 1.4cqw;
		font-weight: 500;
		text-decoration: none;
		border-bottom: 1px solid var(--color-border-strong);
		padding-bottom: 0.2em;
	}
	.cover-link:hover,
	.cover-link:focus-visible {
		color: var(--color-accent-ink);
		border-bottom-color: var(--color-accent-ink);
	}

	/* Presenting should never trap anyone on the last slide. */
	.deck-exit {
		position: fixed;
		top: 1.1rem;
		right: 1.25rem;
		z-index: 5;
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		min-height: 2.2rem;
		padding: 0 0.6rem;
		color: var(--color-muted);
		font-size: 0.78rem;
		font-weight: 500;
		letter-spacing: 0.02em;
		text-decoration: none;
	}
	.deck-exit:hover,
	.deck-exit:focus-visible {
		color: var(--color-accent-ink);
	}

	.deck-nav {
		position: fixed;
		left: 50%;
		bottom: 1.1rem;
		transform: translateX(-50%);
		display: flex;
		align-items: center;
		gap: 1rem;
		z-index: 5;
	}
	.deck-nav button {
		display: grid;
		place-items: center;
		width: 2.6rem;
		height: 2.6rem;
		min-height: 0;
		border: 1px solid var(--color-border-strong);
		background: rgb(11 13 18 / 0.7);
		color: var(--color-foreground);
		font-size: 1rem;
		cursor: pointer;
		backdrop-filter: blur(8px);
	}
	.deck-nav button:hover:not(:disabled) {
		border-color: var(--color-accent-ink);
		color: var(--color-accent-ink);
	}
	.deck-nav button:disabled {
		opacity: 0.35;
		cursor: not-allowed;
		background: rgb(11 13 18 / 0.7);
		color: var(--color-muted);
	}
	.deck-progress {
		display: block;
		width: 11rem;
		height: 2px;
		background: var(--color-border);
	}
	.deck-progress i {
		display: block;
		height: 100%;
		background: var(--color-accent-ink);
		transition: width 320ms cubic-bezier(0.16, 0.7, 0.2, 1);
	}

	@media (max-width: 720px) {
		.slide-inner {
			aspect-ratio: auto;
			width: 100%;
			height: 100%;
		}
		.deck-progress {
			width: 6rem;
		}
	}

	/* Print is the export path: every slide becomes one landscape page. */
	@media print {
		.deck {
			position: static;
			display: block;
			background: #fff;
			overflow: visible;
		}
		.deck-nav,
		.deck-exit {
			display: none;
		}
		.slide {
			position: static;
			display: grid !important;
			inset: auto;
			padding: 0;
			break-after: page;
			page-break-after: always;
		}
		.slide:last-child {
			break-after: auto;
			page-break-after: auto;
		}
		.slide-inner {
			width: 100%;
			height: 100vh;
			border: 0;
			aspect-ratio: auto;
			-webkit-print-color-adjust: exact;
			print-color-adjust: exact;
		}
	}
</style>
