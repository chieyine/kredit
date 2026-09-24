<script lang="ts">
	import { onMount } from 'svelte';
	import { read } from '$lib/consumer';
	import { record } from '$lib/api/reliable';
	let code = $state(''),
		name = $state(''),
		error = $state('');
	onMount(() => {
		code = new URLSearchParams(location.search).get('ref') ?? '';
		void read('/api/v1/dsa/code/' + encodeURIComponent(code))
			.then((v) => {
				name = String(record(v).name);
			})
			.catch(() => (error = 'This referral link is unavailable. Ask the agent to check their code.'));
	});
</script>

<svelte:head><title>Join Kredit through a field agent</title></svelte:head>
<main class="shell referral-welcome">
	<div class="welcome-copy">
		<p class="eyebrow">You were introduced to Kredit</p>
		<h1>Keep a proper record of the goods you give on credit.</h1>
		<p class="lede">See every credit sale, the day each payment is due, and what each customer still owes you.</p>
		<ol class="welcome-steps">
			<li><span>01</span>Sign in with your email or WhatsApp.</li>
			<li><span>02</span>Add your business and confirm who introduced you.</li>
			<li><span>03</span>Finish the setup and start writing your sales down.</li>
		</ol>
	</div>
	<section class="welcome-card">
		<p class="eyebrow">Your invitation</p>
		{#if error}<h2>This link is unavailable.</h2>
			<p role="alert">{error}</p>
			<a class="secondary" href="/signin">Continue to Kredit</a>{:else if name}<h2>{name} invited you.</h2>
			<p>They may earn a reward once your business joins and starts using Kredit.</p>
			<a
				class="primary"
				href={`/signin?next=${encodeURIComponent('/workspace/referral?code=' + encodeURIComponent(code))}`}
				>Start or sign in <span aria-hidden="true">→</span></a
			>
			<p class="welcome-safety">
				Sign in yourself. Never give your code to an agent, or to anybody else who asks for it.
			</p>{:else}<h2>Opening your invitation…</h2>
			<p role="status">Checking the referral details.</p>{/if}
	</section>
</main>
