<script lang="ts">
	import { afterNavigate } from '$app/navigation';
	import { onMount } from 'svelte';

	let observer: IntersectionObserver | undefined;
	let frame = 0;

	const selector = [
		'.motion-scope > main > section',
		'.motion-scope main > header',
		'.motion-scope main > .toolbar',
		'.motion-scope main > .card',
		'.motion-scope main > .tips',
		'.motion-scope main > .records > article',
		'.motion-scope .steps > li',
		'.motion-scope .faq > div',
		'.motion-scope .card-grid > article',
		'.motion-scope .price-grid > article',
		'.motion-scope .process-grid > article',
		'.motion-scope .capability-list > article',
		'.motion-scope .control-list > article',
		'.motion-scope .journey > article',
		// Any element can opt in; its own styles decide what "revealed" means.
		'.motion-scope [data-reveal]'
	].join(',');

	function prepare() {
		if (!observer || typeof document === 'undefined') return;
		const items = Array.from(document.querySelectorAll<HTMLElement>(selector));
		items.forEach((item, index) => {
			if (item.dataset.motionReveal !== undefined) return;
			// The first block of a page is the first screen. It has its own entrance
			// and must never wait on a scroll to become visible.
			if (item.parentElement?.tagName === 'MAIN' && item === item.parentElement.firstElementChild) return;
			item.dataset.motionReveal = '';
			item.style.setProperty('--motion-order', String(index % 4));
			const rect = item.getBoundingClientRect();
			// Already on screen: reveal on the next frame, so the first state is
			// painted and the transition actually runs.
			if (rect.top < window.innerHeight * 0.88)
				requestAnimationFrame(() => requestAnimationFrame(() => item.classList.add('is-revealed')));
			else observer?.observe(item);
		});
	}

	function schedulePrepare() {
		cancelAnimationFrame(frame);
		frame = requestAnimationFrame(prepare);
	}

	afterNavigate(schedulePrepare);

	onMount(() => {
		if (!('IntersectionObserver' in window) || window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
		document.documentElement.classList.add('motion-ready');
		observer = new IntersectionObserver(
			(entries) => {
				for (const entry of entries) {
					if (!entry.isIntersecting) continue;
					(entry.target as HTMLElement).classList.add('is-revealed');
					observer?.unobserve(entry.target);
				}
			},
			{ threshold: 0, rootMargin: '0px 0px -7% 0px' }
		);
		schedulePrepare();
		return () => {
			cancelAnimationFrame(frame);
			observer?.disconnect();
			document.documentElement.classList.remove('motion-ready');
		};
	});
</script>
