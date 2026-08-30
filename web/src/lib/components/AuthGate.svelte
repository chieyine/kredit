<script lang="ts">
	import { onMount, type Snippet } from 'svelte';

	let { children, area = 'account' }: { children: Snippet; area?: string } = $props();
	let gateStatus = $state<'checking' | 'ready' | 'error'>('checking');
	let message = $state('');

	function signInURL() {
		if (typeof window === 'undefined') return '/app';
		const next = `${window.location.pathname}${window.location.search}`;
		return `/app?next=${encodeURIComponent(next)}`;
	}

	async function verify() {
		gateStatus = 'checking';
		message = '';
		try {
			const response = await fetch('/api/v1/me', { credentials: 'include', cache: 'no-store' });
			if (response.status === 401) {
				window.location.replace(signInURL());
				return;
			}
			if (!response.ok) throw new Error('We could not check your account.');
			gateStatus = 'ready';
		} catch (cause) {
			message = cause instanceof Error ? cause.message : 'We could not check your account.';
			gateStatus = 'error';
		}
	}

	onMount(verify);
</script>

{#if gateStatus === 'ready'}
	{@render children()}
{:else}
	<main class="account-gate" aria-live="polite">
		<a class="gate-brand" href="/" aria-label="Kredit home"><span aria-hidden="true">K</span>Kredit</a>
		<section>
			<p class="eyebrow">Private {area}</p>
			{#if gateStatus === 'checking'}
				<div class="gate-mark" aria-hidden="true"></div>
				<h1>Checking your account…</h1>
				<p>Please wait. We will never show private business information before your account is checked.</p>
			{:else}
				<h1>We cannot open your account.</h1>
				<p>{message} Check your connection and try again.</p>
				<div class="gate-actions"><button type="button" onclick={verify}>Try again</button><a href="/">Go to the home page</a></div>
			{/if}
		</section>
	</main>
{/if}

<style>
	.account-gate{box-sizing:border-box;min-height:100vh;padding:clamp(1.25rem,4vw,3rem);color:#fff;background:#17181b}.account-gate .eyebrow{color:#ff9b84}.gate-brand{display:inline-flex;align-items:center;gap:.65rem;color:#fff;font-family:Georgia,'Times New Roman',serif;font-size:1.15rem;font-weight:650;text-decoration:none}.gate-brand span{display:grid;place-items:center;width:2rem;height:2rem;background:#2738d6}.account-gate section{display:grid;align-content:center;max-width:48rem;min-height:calc(100vh - 9rem)}.account-gate h1{max-width:13ch;margin:.6rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3rem,8vw,6.5rem);font-weight:500;line-height:.9;letter-spacing:-.055em}.account-gate p:not(.eyebrow){max-width:36rem;color:#c7c6c1;font-size:1.05rem;line-height:1.65}.gate-mark{width:3.5rem;height:.45rem;margin-bottom:1rem;background:#ff5b3a;transform-origin:left;animation:checking 1.1s ease-in-out infinite}.gate-actions{display:flex;flex-wrap:wrap;gap:.75rem;margin-top:1.5rem}.gate-actions button,.gate-actions a{display:inline-flex;align-items:center;justify-content:center;min-height:2.8rem;padding:0 1rem;border:1px solid #fff;border-radius:0;background:#fff;color:#17181b;font:inherit;font-weight:750;text-decoration:none}.gate-actions a{background:transparent;color:#fff}@keyframes checking{0%,100%{transform:scaleX(.25)}50%{transform:scaleX(1)}}@media(prefers-reduced-motion:reduce){.gate-mark{animation:none}}
</style>
