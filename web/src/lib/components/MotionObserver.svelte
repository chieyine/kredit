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
		'.site-footer .footer-grid > div'
	].join(',');

	function prepare() {
		if (!observer || typeof document === 'undefined') return;
		const items = Array.from(document.querySelectorAll<HTMLElement>(selector));
		items.forEach((item, index) => {
			if (item.dataset.motionReveal !== undefined) return;
			item.dataset.motionReveal = '';
			item.style.setProperty('--motion-order', String(index % 4));
			const rect = item.getBoundingClientRect();
			if (rect.top < window.innerHeight * 0.88) item.classList.add('is-revealed');
			else observer?.observe(item);
		});
	}

	function schedulePrepare() {
		cancelAnimationFrame(frame);
		frame = requestAnimationFrame(prepare);
	}

	afterNavigate(schedulePrepare);

	onMount(() => {
		if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
		document.documentElement.classList.add('motion-ready');
		observer = new IntersectionObserver(
			(entries) => {
				for (const entry of entries) {
					if (!entry.isIntersecting) continue;
					(entry.target as HTMLElement).classList.add('is-revealed');
					observer?.unobserve(entry.target);
				}
			},
			{ threshold: 0.12, rootMargin: '0px 0px -7% 0px' }
		);
		schedulePrepare();
		return () => {
			cancelAnimationFrame(frame);
			observer?.disconnect();
			document.documentElement.classList.remove('motion-ready');
		};
	});
</script>
