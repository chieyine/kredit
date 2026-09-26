<script lang="ts">
	import { onMount } from 'svelte';
	import type { Slide } from '$lib/deck/types';

	let { slides, title }: { slides: Slide[]; title: string } = $props();

	let index = $state(0);
	let printing = $state(false);
	// set once the controls respond, so tests and tooling can wait for it
	let ready = $state(false);
	const total = $derived(slides.length);
	// Bars share one scale per slide so a longer bar is always a larger figure.
	const scale = (slide: Slide) => Math.max(...(slide.bars ?? []).map((bar) => bar.after), 1) * 1.2;
	const pct = (value: number, slide: Slide) => `${(value / scale(slide)) * 100}%`;
	const naira = (value: number) => `₦${value.toLocaleString('en-NG', { maximumFractionDigits: 1 })}bn`;
	const times = (bar: { before: number; after: number }) => `${(bar.after / bar.before).toFixed(1)}×`;

	function go(n: number) {
		index = Math.max(0, Math.min(total - 1, n));
		if (typeof history !== 'undefined') history.replaceState(history.state, '', `#${index + 1}`);
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
		ready = true;
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

<div class="deck" class:printing data-ready={ready || undefined}>
	{#each slides as slide, i (i)}
		<!-- Every slide stays in the DOM so browser print gives one page each. -->
		<section class="slide" class:current={i === index} aria-hidden={i === index || printing ? undefined : 'true'}>
			<div class="slide-inner" data-kind={slide.kind}>
				<header class="slide-head">
					<span class="seal" aria-hidden="true">K</span>
					{#if slide.eyebrow}<p class="deck-eyebrow">{slide.eyebrow}</p>{/if}
				</header>

				{#if slide.kind === 'cover'}
					<div class="cover" class:with-record={slide.record}>
						<div class="cover-text">
							<!-- eslint-disable-next-line svelte/no-at-html-tags -- Slide titles come from the hardcoded decks in $lib/deck, never from user or network input. If decks ever become editable content this must be sanitised first. -->
							<h1 class="in">{@html slide.title}</h1>
							{#if slide.lede}<p class="deck-lede in">{slide.lede}</p>{/if}
							{#if slide.links}
								<div class="cover-links in">
									{#each slide.links as [label, href], linkIndex (linkIndex)}
										<a class="cover-link" class:first={linkIndex === 0} {href}
											>{label} <span aria-hidden="true">→</span></a
										>
									{/each}
								</div>
							{/if}
							{#if slide.meta}<p class="cover-meta in">{slide.meta}</p>{/if}
						</div>
						{#if slide.record}
							<!-- An example, labelled as one: it shows what a sale record holds, not anyone's real trade. -->
							<figure class="record in" aria-label="An example sale record">
								<figcaption><span>Example sale</span><span class="ref">KR-0412</span></figcaption>
								<dl>
									<div>
										<dt>Seller</dt>
										<dd>A distributor in Onitsha</dd>
									</div>
									<div>
										<dt>Buyer</dt>
										<dd>A provisions shop in Aba</dd>
									</div>
									<div>
										<dt>Goods</dt>
										<dd>40 cartons of noodles</dd>
									</div>
									<div>
										<dt>Total</dt>
										<dd class="num">₦480,000</dd>
									</div>
								</dl>
								<ol>
									<li><span>Terms accepted by both sides</span><time>12 Mar</time></li>
									<li><span>Bank permission given</span><time>12 Mar</time></li>
									<li><span>Delivered, 40 of 40 cartons</span><time>14 Mar</time></li>
									<li><span>Paid ₦160,000 · 1 of 3</span><time>28 Mar</time></li>
									<li class="next"><span>Next payment ₦160,000</span><time>11 Apr</time></li>
								</ol>
								<footer><span>Left to pay</span><strong class="num">₦320,000</strong></footer>
							</figure>
						{/if}
					</div>
				{:else}
					<div class="body">
						<div class="title-block" class:side={slide.kind === 'split'}>
							<!-- eslint-disable-next-line svelte/no-at-html-tags -- Slide titles come from the hardcoded decks in $lib/deck, never from user or network input. If decks ever become editable content this must be sanitised first. -->
							<h2 class="in">{@html slide.title}</h2>
							{#if slide.lede}<p class="deck-lede in">{slide.lede}</p>{/if}
							{#if slide.kind === 'split' && slide.note}<p class="slide-note in">{slide.note}</p>{/if}
						</div>

						{#if slide.kind === 'bars' && slide.bars}
							<div class="bars in">
								{#if slide.barLabels}<p class="legend">
										<span class="key before"></span>{slide.barLabels[0]}<span class="key after"></span>{slide
											.barLabels[1]}
									</p>{/if}
								{#each slide.bars as bar, barIndex (barIndex)}
									<div class="bar-row" style={`--d:${barIndex * 140}ms`}>
										<p class="bar-label">{bar.label}</p>
										<div class="bar-pair">
											<div class="bar before" style={`--w:${pct(bar.before, slide)}`}>
												<span>{naira(bar.before)}</span>
											</div>
											<div class="bar after" style={`--w:${pct(bar.after, slide)}`}>
												<span>{naira(bar.after)}</span>
											</div>
										</div>
										<p class="bar-change">{times(bar)}</p>
									</div>
								{/each}
							</div>
						{/if}

						{#if slide.kind === 'chain' && slide.tiers}
							<div class="chain in">
								<p class="flow down">Goods and credit go down the chain <span aria-hidden="true">→</span></p>
								<ol>
									{#each slide.tiers as [name, detail], tierIndex (tierIndex)}
										<li style={`--d:${tierIndex * 120}ms`}><strong>{name}</strong><span>{detail}</span></li>
									{/each}
								</ol>
								<p class="flow up"><span aria-hidden="true">←</span> What is owed comes back up, if it comes</p>
							</div>
						{/if}

						{#if slide.kind === 'compare' && slide.compare}
							<div class="compare in">
								{#each slide.compare as column, columnIndex (columnIndex)}
									<div class="column" class:ours={columnIndex === 1}>
										<p class="column-head">{column.heading}</p>
										<ul>
											{#each column.items as item, itemIndex (itemIndex)}<li>{item}</li>{/each}
										</ul>
									</div>
								{/each}
							</div>
						{/if}

						{#if slide.figures}
							<div class="figures in" class:compact={slide.kind === 'chain'} style={`--cols:${slide.figures.length}`}>
								{#each slide.figures as f, figureIndex (figureIndex)}
									<div class="fig">
										<strong class="fig-value" class:alert={f.tone === 'alert'}>{f.value}</strong>
										<p class="fig-label">{f.label}</p>
										{#if f.note}<p class="fig-note">{f.note}</p>{/if}
									</div>
								{/each}
							</div>
						{/if}

						{#if slide.kind === 'split' && slide.points}
							<ol class="points in">
								{#each slide.points as [head, detail], pointIndex (pointIndex)}
									<li><strong>{head}</strong><span>{detail}</span></li>
								{/each}
							</ol>
						{/if}

						{#if slide.kind === 'steps' && slide.points}
							<ol class="sequence in" style={`--cols:${slide.points.length}`}>
								{#each slide.points as [head, detail], stepIndex (stepIndex)}
									<li style={`--d:${stepIndex * 120}ms`}>
										<span class="step-no">{String(stepIndex + 1).padStart(2, '0')}</span>
										<strong>{head}</strong><span>{detail}</span>
									</li>
								{/each}
							</ol>
						{/if}

						{#if slide.quote}
							<blockquote class="quote in">
								<p>“{slide.quote.text}”</p>
								<cite>{slide.quote.who}</cite>
							</blockquote>
						{/if}

						{#if slide.note && slide.kind !== 'split'}<p class="slide-note in">{slide.note}</p>{/if}
					</div>
				{/if}

				<footer class="slide-foot">
					{#if slide.source}<p class="source">Source: {slide.source}</p>{:else}<span></span>{/if}
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
	 * sees is exactly what browser print gives back as a PDF page. Sizes are in
	 * container units (cqw) of that stage; on a phone the stage becomes a
	 * scrolling page and the sizes switch to rem (see the end of this block).
	 * Colours resolve against the site's own tokens, so the deck cannot drift
	 * from the brand.
	 */
	.deck {
		position: fixed;
		inset: 0;
		display: grid;
		place-items: center;
		background: var(--color-background-sunk, #080a0e);
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
		/* padding counts inside the stage, so a printed slide is exactly one page */
		box-sizing: border-box;
		/* the band below the stage is where the controls live, so they never sit
		   on top of the source line */
		position: relative;
		width: min(100%, calc((100svh - 8rem) * 16 / 9));
		aspect-ratio: 16 / 9;
		display: grid;
		grid-template-rows: auto minmax(0, 1fr) auto;
		gap: 1.6cqw;
		padding: 3.2cqw 3.6cqw 2.4cqw;
		border: 1px solid var(--color-border);
		background:
			radial-gradient(70% 90% at 100% 0%, rgb(45 56 92 / 0.42), transparent 60%),
			radial-gradient(45% 60% at 0% 100%, rgb(226 96 58 / 0.08), transparent 65%), var(--color-surface);
		box-shadow: 0 40px 90px rgb(0 0 0 / 0.45);
		container-type: inline-size;
		overflow: hidden;
	}
	/* a hairline grid, like ledger paper, kept faint enough to stay out of the way */
	.slide-inner::before {
		content: '';
		position: absolute;
		inset: 0;
		background-image: linear-gradient(to right, rgb(255 255 255 / 0.025) 1px, transparent 1px);
		background-size: calc(100% / 12) 100%;
		pointer-events: none;
	}
	.slide-inner > * {
		position: relative;
	}

	.slide-head {
		display: flex;
		align-items: center;
		gap: 1cqw;
	}
	.seal {
		display: grid;
		place-items: center;
		flex: none;
		width: 2cqw;
		height: 2cqw;
		min-width: 20px;
		min-height: 20px;
		background: var(--color-accent-ink);
		color: #fffcf7;
		font-family: var(--font-display);
		font-weight: 600;
		font-size: 1.1cqw;
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
		font-size: 1cqw;
		font-weight: 600;
		letter-spacing: 0.16em;
		text-transform: uppercase;
	}

	/* ---------- Type ---------- */
	:global(.deck em) {
		font-style: normal;
		color: var(--color-accent-ink);
	}
	h1,
	h2 {
		margin: 0;
		font-family: var(--font-display);
		font-weight: 450;
		text-wrap: balance;
	}
	h1 {
		font-variation-settings: 'opsz' 144;
		font-size: 5.6cqw;
		line-height: 0.98;
		letter-spacing: -0.04em;
	}
	h2 {
		font-variation-settings: 'opsz' 110;
		font-size: 3.9cqw;
		line-height: 1.04;
		letter-spacing: -0.032em;
		max-width: 24ch;
	}
	.deck-lede {
		margin: 1.4cqw 0 0;
		max-width: 60ch;
		color: var(--color-muted);
		font-size: 1.5cqw;
		line-height: 1.5;
		text-wrap: pretty;
	}
	.num,
	.fig-value,
	.bar span,
	.bar-change {
		font-variant-numeric: tabular-nums;
	}

	/* ---------- Cover ---------- */
	.cover {
		align-self: center;
		display: grid;
		gap: 4cqw;
		align-items: center;
	}
	.cover.with-record {
		grid-template-columns: minmax(0, 1.25fr) minmax(0, 1fr);
	}
	.cover:not(.with-record) h1 {
		font-size: 7cqw;
		max-width: 14ch;
	}
	.cover-meta {
		margin: 3cqw 0 0;
		color: var(--color-muted);
		font-size: 1.1cqw;
		letter-spacing: 0.04em;
	}
	.cover-links {
		display: flex;
		flex-wrap: wrap;
		gap: 1.2cqw;
		margin-top: 2.8cqw;
	}
	.cover-link {
		display: inline-flex;
		align-items: center;
		gap: 0.5em;
		min-height: 44px;
		padding: 0.7cqw 1.4cqw;
		border: 1px solid var(--color-border-strong);
		color: var(--color-foreground);
		font-size: 1.25cqw;
		font-weight: 550;
		text-decoration: none;
	}
	.cover-link.first {
		border-color: var(--color-accent-ink);
		background: var(--color-accent-ink);
		color: #0b0d12;
	}
	.cover-link:hover,
	.cover-link:focus-visible {
		border-color: var(--color-accent-ink);
		color: var(--color-accent-ink);
	}
	.cover-link.first:hover,
	.cover-link.first:focus-visible {
		background: #ef7049;
		color: #0b0d12;
	}

	/* The example sale: the same object the product and the home page show. */
	.record {
		margin: 0;
		border: 1px solid var(--color-border-strong);
		background: linear-gradient(176deg, #171b24, #101319 60%);
		box-shadow: 0 30px 60px rgb(0 0 0 / 0.4);
		font-size: 1.05cqw;
	}
	.record figcaption,
	.record footer {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1cqw 1.4cqw;
		color: var(--color-muted);
	}
	.record figcaption {
		border-bottom: 1px solid var(--color-border);
		font-weight: 600;
	}
	.record .ref {
		font-family: var(--font-mono);
		font-size: 0.95cqw;
	}
	.record dl {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.9cqw 1.4cqw;
		margin: 0;
		padding: 1.2cqw 1.4cqw;
	}
	.record dt {
		color: var(--color-muted);
		font-size: 0.9cqw;
	}
	.record dd {
		margin: 0.2cqw 0 0;
		font-weight: 600;
	}
	.record ol {
		margin: 0;
		padding: 0.4cqw 1.4cqw 0.8cqw 2.8cqw;
		list-style: none;
		border-top: 1px solid var(--color-border);
	}
	.record li {
		position: relative;
		display: flex;
		justify-content: space-between;
		gap: 1cqw;
		padding: 0.55cqw 0;
	}
	.record li::before {
		content: '';
		position: absolute;
		left: -1.25cqw;
		top: 1cqw;
		width: 0.5cqw;
		height: 0.5cqw;
		background: var(--color-accent-ink);
		transform: rotate(45deg);
	}
	.record li.next {
		color: var(--color-muted);
	}
	.record li.next::before {
		background: transparent;
		outline: 1px solid var(--color-border-strong);
	}
	.record time {
		color: var(--color-muted);
		font-size: 0.95cqw;
	}
	.record footer {
		border-top: 1px solid var(--color-border);
		background: rgb(0 0 0 / 0.25);
	}
	.record footer strong {
		color: var(--color-foreground);
		font-family: var(--font-display);
		font-size: 1.9cqw;
		font-weight: 500;
	}

	/* ---------- Body layouts ---------- */
	.body {
		align-self: center;
		display: grid;
		gap: 2.4cqw;
		min-height: 0;
	}
	[data-kind='split'] .body {
		grid-template-columns: minmax(0, 5fr) minmax(0, 7fr);
		gap: 4cqw;
		align-items: center;
	}
	.title-block.side h2 {
		font-size: 3.6cqw;
	}

	/* Figures: the numbers are the argument, so they get the display face. */
	.figures {
		display: grid;
		grid-template-columns: repeat(var(--cols, 3), minmax(0, 1fr));
		border-top: 1px solid var(--color-border-strong);
	}
	.fig {
		padding: 1.8cqw 2cqw 0 0;
	}
	.fig + .fig {
		padding-left: 2cqw;
		border-left: 1px solid var(--color-border);
	}
	.fig-value {
		display: block;
		font-family: var(--font-display);
		font-variation-settings: 'opsz' 144;
		font-size: 5.4cqw;
		font-weight: 420;
		letter-spacing: -0.04em;
		line-height: 1;
	}
	.fig-value.alert {
		color: var(--color-accent-ink);
	}
	.fig-label {
		margin: 1cqw 0 0;
		color: var(--color-foreground);
		font-size: 1.2cqw;
		font-weight: 600;
	}
	.fig-note {
		margin: 0.35cqw 0 0;
		color: var(--color-muted);
		font-size: 1.1cqw;
		line-height: 1.45;
	}
	.figures.compact .fig-value {
		font-size: 3.4cqw;
	}
	.figures.compact .fig {
		padding-top: 1.3cqw;
	}

	/* Bars: a year apart, on one scale per slide. */
	.bars {
		display: grid;
		gap: 1.8cqw;
	}
	.legend {
		display: flex;
		align-items: center;
		gap: 0.7cqw;
		margin: 0;
		color: var(--color-muted);
		font-size: 1.05cqw;
	}
	.key {
		display: inline-block;
		width: 1.4cqw;
		height: 0.7cqw;
	}
	.key.before {
		background: var(--color-border-strong);
	}
	.key.after {
		margin-left: 1.4cqw;
		background: var(--color-accent-ink);
	}
	.bar-row {
		display: grid;
		grid-template-columns: 16cqw minmax(0, 1fr) 8cqw;
		gap: 2cqw;
		align-items: center;
		padding-bottom: 1.8cqw;
		border-bottom: 1px solid var(--color-border);
	}
	.bar-label {
		margin: 0;
		font-size: 1.5cqw;
		font-weight: 600;
	}
	.bar-pair {
		display: grid;
		gap: 0.5cqw;
	}
	.bar {
		position: relative;
		height: 1.9cqw;
		width: var(--w);
		min-width: 0.4cqw;
		transform-origin: left;
	}
	.bar.before {
		background: var(--color-border-strong);
	}
	.bar.after {
		background: var(--color-accent-ink);
	}
	.bar span {
		position: absolute;
		left: calc(100% + 0.8cqw);
		top: 50%;
		transform: translateY(-50%);
		color: var(--color-muted);
		font-size: 1.1cqw;
		white-space: nowrap;
	}
	.bar.after span {
		color: var(--color-foreground);
		font-weight: 600;
	}
	.bar-change {
		margin: 0;
		color: var(--color-accent-ink);
		font-family: var(--font-display);
		font-size: 3.6cqw;
		letter-spacing: -0.03em;
		text-align: right;
	}

	/* The chain: four tiers joined by a rail, credit flowing down it. */
	.chain {
		display: grid;
		gap: 0.9cqw;
	}
	.chain ol {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		margin: 0;
		padding: 0;
		list-style: none;
		position: relative;
	}
	.chain ol::before {
		content: '';
		position: absolute;
		left: 0.3cqw;
		right: 12%;
		top: 0.35cqw;
		height: 1px;
		background: linear-gradient(to right, var(--color-accent-ink), var(--color-border-strong));
	}
	.chain li {
		position: relative;
		display: grid;
		gap: 0.4cqw;
		padding: 1.8cqw 2cqw 0 0;
	}
	.chain li::before {
		content: '';
		position: absolute;
		left: 0;
		top: 0;
		width: 0.7cqw;
		height: 0.7cqw;
		background: var(--color-surface);
		outline: 1px solid var(--color-accent-ink);
		transform: rotate(45deg);
	}
	.chain li:first-child::before {
		background: var(--color-accent-ink);
	}
	.chain strong {
		font-family: var(--font-display);
		font-size: 2cqw;
		font-weight: 500;
	}
	.chain li span {
		color: var(--color-muted);
		font-size: 1.15cqw;
		line-height: 1.45;
	}
	.flow {
		margin: 0;
		color: var(--color-muted);
		font-size: 0.95cqw;
		font-weight: 600;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}
	.flow.down {
		color: var(--color-accent-ink);
	}

	/* Compare: today on the left, on Kredit on the right. */
	.compare {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 2cqw;
	}
	.column {
		padding: 2cqw 2.2cqw;
		border: 1px solid var(--color-border);
		background: rgb(0 0 0 / 0.18);
	}
	.column.ours {
		border-color: var(--color-accent-ink);
		background: linear-gradient(170deg, rgb(226 96 58 / 0.1), transparent 55%), rgb(0 0 0 / 0.18);
	}
	.column-head {
		margin: 0 0 1.2cqw;
		color: var(--color-muted);
		font-size: 1cqw;
		font-weight: 700;
		letter-spacing: 0.14em;
		text-transform: uppercase;
	}
	.column.ours .column-head {
		color: var(--color-accent-ink);
	}
	.column ul {
		display: grid;
		gap: 1.2cqw;
		margin: 0;
		padding: 0;
		list-style: none;
	}
	.column li {
		position: relative;
		padding-left: 2cqw;
		font-size: 1.45cqw;
		line-height: 1.45;
	}
	.column li::before {
		content: '';
		position: absolute;
		left: 0;
		top: 0.72em;
		width: 0.9cqw;
		height: 1px;
		background: var(--color-border-strong);
	}
	.column:not(.ours) li {
		color: var(--color-muted);
	}
	.column.ours li::before {
		top: 0.5em;
		width: 0.55cqw;
		height: 0.55cqw;
		background: var(--color-accent-ink);
		transform: rotate(45deg);
	}

	/* Split: numbered points beside the headline. */
	.points {
		display: grid;
		margin: 0;
		padding: 0;
		list-style: none;
		counter-reset: p;
		border-top: 1px solid var(--color-border-strong);
	}
	.points li {
		display: grid;
		grid-template-columns: 3.2cqw 1fr;
		gap: 0 1cqw;
		padding: 1.5cqw 0;
		border-bottom: 1px solid var(--color-border);
		counter-increment: p;
	}
	.points li::before {
		content: counter(p, decimal-leading-zero);
		grid-row: span 2;
		color: var(--color-accent-ink);
		font-family: var(--font-display);
		font-size: 1.6cqw;
		line-height: 1.2;
	}
	.points strong {
		font-size: 1.65cqw;
		font-weight: 600;
		letter-spacing: -0.01em;
		line-height: 1.25;
	}
	.points span {
		margin-top: 0.4cqw;
		color: var(--color-muted);
		font-size: 1.3cqw;
		line-height: 1.5;
	}

	/* Steps: left to right along a rail, the way the product runs a sale. */
	.sequence {
		display: grid;
		grid-template-columns: repeat(var(--cols, 4), minmax(0, 1fr));
		gap: 2.4cqw;
		margin: 0;
		padding: 0;
		list-style: none;
		position: relative;
	}
	.sequence::before {
		content: '';
		position: absolute;
		left: 0;
		right: 0;
		top: 2.2cqw;
		height: 1px;
		background: var(--color-border-strong);
	}
	.sequence li {
		display: grid;
		align-content: start;
		gap: 0.6cqw;
	}
	.step-no {
		position: relative;
		display: inline-grid;
		place-items: center;
		width: 4.4cqw;
		height: 4.4cqw;
		margin-bottom: 1cqw;
		border: 1px solid var(--color-accent-ink);
		background: var(--color-surface);
		color: var(--color-accent-ink);
		font-family: var(--font-display);
		font-size: 1.6cqw;
	}
	.sequence strong {
		font-family: var(--font-display);
		font-size: 2.2cqw;
		font-weight: 500;
		letter-spacing: -0.02em;
	}
	.sequence li > span:last-child {
		color: var(--color-muted);
		font-size: 1.3cqw;
		line-height: 1.5;
	}

	/* Quote: the words are the slide. */
	.quote {
		margin: 0;
		padding-left: 2.4cqw;
		border-left: 2px solid var(--color-accent-ink);
	}
	.quote p {
		margin: 0;
		font-family: var(--font-display);
		font-size: 3.4cqw;
		font-weight: 400;
		line-height: 1.2;
		letter-spacing: -0.02em;
		max-width: 38ch;
	}
	.quote cite {
		display: block;
		margin-top: 1.2cqw;
		color: var(--color-muted);
		font-size: 1.15cqw;
		font-style: normal;
		letter-spacing: 0.04em;
	}
	[data-kind='quote'] h2 {
		font-size: 2.4cqw;
		color: var(--color-muted);
	}

	.slide-note {
		margin: 0;
		padding: 1.1cqw 1.4cqw;
		border-left: 2px solid var(--color-border-strong);
		background: rgb(255 255 255 / 0.03);
		color: var(--color-muted);
		font-size: 1.2cqw;
		line-height: 1.5;
		max-width: 80ch;
	}
	.title-block.side .slide-note {
		margin-top: 2cqw;
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
		max-width: 90ch;
		color: var(--color-muted);
		font-size: 0.9cqw;
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

	/* ---------- Motion ----------
	   The resting state is the finished slide; motion only runs from an earlier
	   state into it, and never when the reader asks for less motion. */
	@media (prefers-reduced-motion: no-preference) {
		.slide.current .in {
			animation: rise 620ms cubic-bezier(0.16, 0.7, 0.2, 1) both;
		}
		.slide.current .in:nth-child(2) {
			animation-delay: 80ms;
		}
		.slide.current .in:nth-child(3) {
			animation-delay: 160ms;
		}
		.slide.current .body > .in,
		.slide.current .record {
			animation-delay: 220ms;
		}
		.slide.current .bar {
			animation: grow 900ms cubic-bezier(0.16, 0.7, 0.2, 1) calc(300ms + var(--d, 0ms)) both;
		}
		.slide.current .chain li,
		.slide.current .sequence li {
			animation: rise 560ms cubic-bezier(0.16, 0.7, 0.2, 1) calc(260ms + var(--d, 0ms)) both;
		}
	}
	@keyframes rise {
		from {
			opacity: 0;
			transform: translateY(14px);
		}
	}
	@keyframes grow {
		from {
			transform: scaleX(0);
		}
	}

	/* ---------- Controls ---------- */
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

	/* ---------- Phones ----------
	   A 16:9 stage on a phone would make the words a few pixels high, so the
	   slide becomes a scrolling page with fixed, readable sizes. */
	@media (max-width: 720px) {
		.slide {
			padding: 0 0 4.6rem;
			place-items: stretch;
		}
		.slide-inner {
			aspect-ratio: auto;
			width: 100%;
			height: 100%;
			overflow-y: auto;
			padding: 3.4rem 1.25rem 1.25rem;
			gap: 1.25rem;
			border: 0;
			box-shadow: none;
			/* a column that scrolls: nothing may shrink under its own content */
			display: flex;
			flex-direction: column;
		}
		.body {
			min-height: auto;
		}
		.slide-foot {
			margin-top: auto;
		}
		.slide-inner::before {
			display: none;
		}
		.seal {
			font-size: 0.75rem;
		}
		.deck-eyebrow,
		.flow,
		.column-head,
		.legend,
		.pager {
			font-size: 0.7rem;
		}
		h1,
		.cover:not(.with-record) h1 {
			font-size: 2.3rem;
		}
		h2,
		.title-block.side h2 {
			font-size: 1.75rem;
			max-width: none;
		}
		[data-kind='quote'] h2 {
			font-size: 1.2rem;
		}
		.deck-lede {
			margin-top: 0.75rem;
			font-size: 1rem;
		}
		.cover.with-record,
		[data-kind='split'] .body,
		.compare,
		.figures,
		.chain ol,
		.sequence {
			grid-template-columns: 1fr;
		}
		.cover,
		.body,
		[data-kind='split'] .body {
			gap: 1.5rem;
			align-self: start;
		}
		.cover-meta,
		.source,
		.fig-note,
		.record,
		.record dt,
		.record time,
		.record .ref,
		.chain li span {
			font-size: 0.85rem;
		}
		.cover-link {
			font-size: 0.95rem;
			padding: 0.5rem 0.9rem;
		}
		.record figcaption,
		.record footer,
		.record dl {
			padding: 0.75rem 1rem;
		}
		.record ol {
			padding: 0.4rem 1rem 0.6rem 2rem;
		}
		.record li::before {
			left: -1rem;
			top: 0.8rem;
			width: 0.4rem;
			height: 0.4rem;
		}
		.record footer strong {
			font-size: 1.3rem;
		}
		.fig {
			padding: 1rem 0 0;
		}
		.fig + .fig {
			padding-left: 0;
			border-left: 0;
			border-top: 1px solid var(--color-border);
			margin-top: 1rem;
		}
		.fig-value,
		.figures.compact .fig-value {
			font-size: 2.6rem;
		}
		.fig-label {
			margin-top: 0.4rem;
			font-size: 0.9rem;
		}
		.bar-row {
			grid-template-columns: 1fr auto;
			gap: 0.6rem 1rem;
		}
		.bar-pair {
			grid-column: 1 / -1;
			grid-row: 2;
			gap: 0.35rem;
			padding-right: 4.5rem;
		}
		.bar {
			height: 1rem;
		}
		.bar-label {
			font-size: 1rem;
		}
		.bar span {
			font-size: 0.75rem;
			left: calc(100% + 0.4rem);
		}
		.bar-change {
			font-size: 1.6rem;
		}
		.key {
			width: 0.9rem;
			height: 0.45rem;
		}
		.key.after {
			margin-left: 0.9rem;
		}
		.chain ol::before {
			top: 0;
			bottom: 0;
			left: 0.3rem;
			right: auto;
			width: 1px;
			height: auto;
		}
		.chain li {
			padding: 0 0 1rem 1.6rem;
		}
		.chain li::before {
			width: 0.6rem;
			height: 0.6rem;
		}
		.chain strong {
			font-size: 1.2rem;
		}
		.column {
			padding: 1rem 1.1rem;
		}
		.column ul {
			gap: 0.8rem;
		}
		.column li {
			padding-left: 1.2rem;
			font-size: 0.95rem;
		}
		.column li::before {
			width: 0.6rem;
		}
		.column.ours li::before {
			width: 0.4rem;
			height: 0.4rem;
		}
		.points li {
			grid-template-columns: 2rem 1fr;
			padding: 0.9rem 0;
		}
		.points li::before {
			font-size: 1rem;
		}
		.points strong {
			font-size: 1.02rem;
		}
		.points span,
		.sequence li > span:last-child {
			font-size: 0.92rem;
		}
		.sequence {
			gap: 1.2rem;
		}
		.sequence::before {
			top: 0;
			bottom: 0;
			left: 1.25rem;
			right: auto;
			width: 1px;
			height: auto;
		}
		.sequence li {
			grid-template-columns: 2.5rem 1fr;
			gap: 0.2rem 1rem;
		}
		.step-no {
			grid-row: span 2;
			width: 2.5rem;
			height: 2.5rem;
			margin: 0;
			font-size: 0.95rem;
		}
		.sequence strong {
			font-size: 1.15rem;
		}
		.quote {
			padding-left: 1rem;
		}
		.quote p {
			font-size: 1.35rem;
		}
		.quote cite {
			font-size: 0.85rem;
			margin-top: 0.7rem;
		}
		.slide-note {
			padding: 0.75rem 0.9rem;
			font-size: 0.9rem;
		}
		.slide-foot {
			flex-direction: column;
			align-items: flex-start;
			gap: 0.5rem;
		}
		.deck-progress {
			width: 6rem;
		}
	}

	/* Print is the export path: every slide becomes one landscape page. */
	@page {
		size: 1600px 900px;
		margin: 0;
	}
	@media print {
		.deck {
			position: static;
			display: block;
			background: #0b0d12;
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
		.slide:last-of-type {
			break-after: auto;
			page-break-after: auto;
		}
		.slide-inner {
			width: 100%;
			height: 100vh;
			border: 0;
			box-shadow: none;
			aspect-ratio: auto;
			-webkit-print-color-adjust: exact;
			print-color-adjust: exact;
		}
		.slide .in,
		.slide .bar,
		.slide li {
			animation: none !important;
		}
	}
</style>
