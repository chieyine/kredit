<script lang="ts">
	import { onMount } from 'svelte';

	let stage = $state(0);
	let amount = $state(500_000);
	let interactive = $state(false);
	const amounts = [250_000, 500_000, 1_200_000];
	const stages = [
		{ role: 'Seller', title: 'Write down the sale', action: 'Send to my customer' },
		{ role: 'Customer', title: 'Check every detail', action: 'Accept this sale' },
		{ role: 'Seller', title: 'Release the goods', action: 'The goods have left' },
		{ role: 'Customer', title: 'Confirm the goods', action: 'I received the goods' },
		{ role: 'Seller', title: 'Record a payment', action: 'Record sample payment' },
		{ role: 'Both sides', title: 'One clear record', action: '' }
	] as const;
	const payment = $derived(Math.round(amount / 3));
	const balance = $derived(amount - payment);
	const current = $derived(stages[stage]);
	const progress = $derived(((stage + 1) / stages.length) * 100);
	function money(value: number) { return new Intl.NumberFormat('en-NG', { style: 'currency', currency: 'NGN', maximumFractionDigits: 0 }).format(value); }
	function next() { if (stage < stages.length - 1) stage += 1; }
	function restart() { stage = 0; }
	onMount(() => { interactive = true; });
</script>

<svelte:head><meta name="theme-color" content="#17181b" /></svelte:head>

<main class="demo-page">
	<section class="demo-intro shell" aria-labelledby="demo-title">
		<div><p class="eyebrow"><span></span> Try Kredit yourself</p><h1 id="demo-title">See a credit sale<br /><em>from start to payment.</em></h1></div>
		<div class="intro-copy"><p>Take both sides of a sample sale. This guided example shows the steps for the seller and customer.</p><div class="demo-promise"><b>No sign-in</b><b>No real money</b><b>About 60 seconds</b></div></div>
	</section>

	<section class="experience" aria-label="Interactive Kredit demonstration">
		<div class="shell experience-shell">
			<aside class="journey" aria-label="Demo progress">
				<div class="journey-top"><span>Sample sale</span><strong>{stage + 1} of {stages.length}</strong></div>
				<div class="progress" aria-hidden="true"><span style={`width:${progress}%`}></span></div>
				<ol>{#each stages as item, index}<li class:done={index < stage} class:active={index === stage}><span>{index < stage ? '✓' : index + 1}</span><div><small>{item.role}</small><strong>{item.title}</strong></div></li>{/each}</ol>
				<p class="sample-note">This is only a sample. Nothing is saved and no message is sent.</p>
			</aside>

			<div class="demo-stage">
				<header><div><p>You are now the</p><strong>{current.role}</strong></div><span class="live-mark"><i></i> Live sample</span></header>
				<div class="stage-copy">
					<p class="stage-number">STEP {String(stage + 1).padStart(2, '0')}</p><h2>{current.title}</h2>
					{#if stage === 0}<p>Start with the goods, money and payment day. Your customer will see the same details.</p>
					{:else if stage === 1}<p>Nothing is hidden. The customer checks the goods, amount and date before saying yes.</p>
					{:else if stage === 2}<p>The seller records when the goods leave and keeps the delivery proof with the sale.</p>
					{:else if stage === 3}<p>The customer confirms what arrived. A problem can be reported before payment continues.</p>
					{:else if stage === 4}<p>When money arrives, the seller records it once. Kredit changes the balance for both sides.</p>
					{:else}<p>The seller and customer now have the same answer about the goods, payment and money left.</p>{/if}
				</div>

				<div class="product-frame" class:complete={stage === 5}>
					<div class="product-bar"><a href="/" aria-label="Kredit home"><span>K</span>Kredit</a><b>{current.role} view</b></div>
					{#if stage === 0}<div class="amount-choice"><p>Choose a sample amount</p><div>{#each amounts as option}<button disabled={!interactive} class:chosen={amount === option} onclick={() => amount = option}>{money(option)}</button>{/each}</div></div>{/if}
					<div class="deal-head"><div><small>Customer</small><h3>Adebayo Stores</h3></div><span class:accepted={stage >= 2}>{stage === 0 ? 'Not sent' : stage === 1 ? 'Waiting for you' : 'Accepted'}</span></div>
					{#if stage === 1}<div class="plain-notice"><b>Please check before you accept</b><span>The seller cannot quietly change these details after you say yes.</span></div>{/if}
					<div class="deal-facts"><div><small>Goods</small><strong>40 cartons of cooking oil</strong></div><div><small>Total to pay</small><strong>{money(amount)}</strong></div><div><small>Pay before</small><strong>18 September 2026</strong></div><div><small>Extra time</small><strong>3 days</strong></div></div>
					{#if stage >= 2}<div class="record-line"><span>✓</span><div><b>Sale accepted</b><small>Adebayo Stores · Today, 10:42</small></div></div>{/if}
					{#if stage >= 3}<div class="record-line"><span>✓</span><div><b>Goods released</b><small>Delivery note saved by Kora Wholesale</small></div></div>{/if}
					{#if stage >= 4}<div class="record-line"><span>✓</span><div><b>Goods received</b><small>Customer confirmed · No problem reported</small></div></div>{/if}
					{#if stage >= 4}<div class="balance-card"><div><small>{stage === 5 ? 'Money left to pay' : 'Current balance'}</small><strong>{stage === 5 ? money(balance) : money(amount)}</strong></div><span>{stage === 5 ? `Payment of ${money(payment)} recorded` : `Sample payment: ${money(payment)}`}</span></div>{/if}
					{#if stage < 5}<button class="next-action" disabled={!interactive} onclick={next}>{current.action}<span aria-hidden="true">→</span></button>
					{:else}<div class="finish-panel" aria-live="polite"><span class="finish-check">✓</span><div><small>SAMPLE COMPLETE</small><h3>Everyone sees {money(balance)} left.</h3><p>One sale. One balance. Every important step kept together.</p></div></div>{/if}
				</div>
				<div class="stage-footer">{#if stage > 0}<button class="restart" onclick={restart}>Start again</button>{/if}{#if stage === 5}<a class="start-real" href="/app">Add my first real sale <span>↗</span></a>{:else}<span>Your choices stay on this device.</span>{/if}</div>
			</div>
		</div>
	</section>

	<section class="after-demo shell">
		<div><p class="eyebrow"><span></span> What just happened?</p><h2>The details worth keeping together.</h2></div>
		<div class="answers"><p><span>01</span><b>What goods?</b>40 cartons of cooking oil.</p><p><span>02</span><b>How much?</b>{money(amount)}.</p><p><span>03</span><b>Who agreed?</b>The customer accepted.</p><p><span>04</span><b>Did goods arrive?</b>Both sides confirmed.</p><p><span>05</span><b>What was paid?</b>{money(payment)}.</p><p><span>06</span><b>What is left?</b>{money(balance)}.</p></div>
		<a class="final-cta" href="/app">Start with one customer <span aria-hidden="true">↗</span></a>
	</section>
</main>

<style>
	:global(body){background:#f5f2ea}.demo-page{overflow:hidden;color:#17181b;background:#f5f2ea}.demo-intro{display:grid;grid-template-columns:1.2fr .7fr;gap:clamp(3rem,8vw,8rem);align-items:end;padding-top:clamp(4rem,9vw,8rem);padding-bottom:clamp(3rem,6vw,5rem)}.eyebrow{display:flex;align-items:center;gap:.65rem;margin:0 0 1.4rem;color:#5d625e;font-size:.7rem;font-weight:850;letter-spacing:.13em;text-transform:uppercase}.eyebrow span{width:1.8rem;height:2px;background:#e85f3d}.demo-intro h1{max-width:12ch;margin:0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(3.5rem,7vw,6.8rem);font-weight:500;line-height:.9;letter-spacing:-.06em}.demo-intro h1 em{color:#e85f3d;font-weight:500}.intro-copy>p{margin:0;color:#5e6661;font-size:1.1rem;line-height:1.7}.demo-promise{display:flex;flex-wrap:wrap;gap:.5rem;margin-top:1.4rem}.demo-promise b{padding:.4rem .6rem;border:1px solid #c7c1b6;font-size:.68rem}
	.experience{padding:clamp(1rem,3vw,2rem) 0 clamp(5rem,9vw,9rem);background:#17181b}.experience-shell{display:grid;grid-template-columns:minmax(16rem,.55fr) minmax(0,1.45fr);gap:clamp(2rem,5vw,5rem);padding-top:clamp(2rem,5vw,4.5rem)}.journey{color:#f5f2ea}.journey-top{display:flex;align-items:center;justify-content:space-between;color:#aaa9a5;font-size:.7rem;text-transform:uppercase;letter-spacing:.1em}.journey-top strong{color:#fff}.progress{height:3px;margin:1rem 0 2rem;background:#404147}.progress span{display:block;height:100%;background:#ff6848;transition:width .35s ease}.journey ol{display:grid;margin:0;padding:0;list-style:none}.journey li{display:grid;grid-template-columns:2rem 1fr;gap:.8rem;min-height:4.6rem;color:#737570}.journey li>span{display:grid;place-items:center;width:1.6rem;height:1.6rem;border:1px solid #55565b;border-radius:50%;font-size:.68rem}.journey li div{display:grid;align-content:start;gap:.22rem;padding-bottom:1rem;border-bottom:1px solid #34353a}.journey li small{font-size:.6rem;font-weight:800;letter-spacing:.1em;text-transform:uppercase}.journey li strong{font-family:Georgia,serif;font-size:.94rem;font-weight:500}.journey li.done{color:#9b9c98}.journey li.done>span{border-color:#2738d6;background:#2738d6;color:#fff}.journey li.active{color:#fff}.journey li.active>span{border-color:#ff6848;background:#ff6848;color:#17181b;font-weight:900}.sample-note{max-width:22rem;margin:1.5rem 0 0;color:#8f918d;font-size:.7rem;line-height:1.6}
	.demo-stage{min-width:0}.demo-stage>header{display:flex;align-items:center;justify-content:space-between;color:#fff}.demo-stage>header div{display:flex;align-items:baseline;gap:.5rem}.demo-stage>header p{margin:0;color:#8f918d;font-size:.72rem}.demo-stage>header strong{font-family:Georgia,serif;font-size:1.2rem}.live-mark{display:flex;align-items:center;gap:.45rem;color:#bbbcb8;font-size:.65rem;font-weight:750;letter-spacing:.08em;text-transform:uppercase}.live-mark i{width:.5rem;height:.5rem;border-radius:50%;background:#ff6848;box-shadow:0 0 0 .3rem rgba(255,104,72,.12)}.stage-copy{display:grid;grid-template-columns:.3fr 1fr 1.2fr;gap:1.5rem;align-items:start;margin:2.2rem 0;color:#fff}.stage-number{margin:.4rem 0;color:#ff8b70;font-size:.65rem;font-weight:850;letter-spacing:.12em}.stage-copy h2{margin:0;font-family:Georgia,serif;font-size:clamp(2rem,4vw,3.5rem);font-weight:500;line-height:.95}.stage-copy>p:last-child{max-width:28rem;margin:.2rem 0 0;color:#acada9;line-height:1.65}
	.product-frame{position:relative;min-height:32rem;padding:1.2rem;background:#faf8f2;box-shadow:15px 15px 0 #2738d6;animation:frame-in .35s cubic-bezier(.2,.8,.2,1)}.product-frame.complete{box-shadow:15px 15px 0 #e85f3d}.product-bar{display:flex;align-items:center;justify-content:space-between;padding-bottom:1rem;border-bottom:1px solid #d5d0c7}.product-bar a{display:flex;align-items:center;gap:.55rem;color:#17181b;font-family:Georgia,serif;font-weight:700;text-decoration:none}.product-bar a span{display:grid;place-items:center;width:1.8rem;height:1.8rem;background:#2738d6;color:#fff}.product-bar>b{color:#666862;font-size:.65rem;letter-spacing:.1em;text-transform:uppercase}.amount-choice{display:flex;align-items:center;justify-content:space-between;gap:1rem;padding:1rem 0;border-bottom:1px solid #d5d0c7}.amount-choice p{margin:0;font-size:.74rem;font-weight:800}.amount-choice div{display:flex;flex-wrap:wrap;gap:.35rem}.amount-choice button{min-height:2.25rem;padding:.35rem .55rem;border:1px solid #bcb7ad;background:transparent;color:#555955;font:inherit;font-size:.68rem;font-weight:750;cursor:pointer}.amount-choice button:disabled{opacity:1;color:#555955;cursor:default}.amount-choice button.chosen,.amount-choice button.chosen:disabled{border-color:#2738d6;background:#2738d6;color:#fff;opacity:1}.deal-head{display:flex;align-items:start;justify-content:space-between;gap:1rem;padding:1.25rem 0}.deal-head small,.deal-facts small,.balance-card small{display:block;color:#555955;font-size:.62rem;font-weight:800;letter-spacing:.08em;text-transform:uppercase}.deal-head h3{margin:.28rem 0 0;font-family:Georgia,serif;font-size:1.35rem}.deal-head>span{padding:.38rem .55rem;background:#e6e2d9;color:#5f625e;font-size:.65rem;font-weight:800}.deal-head>span.accepted{background:#e9ebff;color:#2738d6}.plain-notice{display:grid;gap:.2rem;margin-bottom:1rem;padding:.8rem 1rem;border-left:4px solid #e85f3d;background:#fff0ea}.plain-notice b{font-size:.76rem}.plain-notice span{color:#686b67;font-size:.68rem}.deal-facts{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));border-top:1px solid #c9c4bb;border-left:1px solid #c9c4bb}.deal-facts>div{display:grid;gap:.35rem;padding:.8rem;border-right:1px solid #c9c4bb;border-bottom:1px solid #c9c4bb}.deal-facts strong{font-size:.8rem}.record-line{display:grid;grid-template-columns:1.8rem 1fr;gap:.65rem;align-items:start;padding:.75rem 0;border-bottom:1px solid #ddd8cf}.record-line>span{display:grid;place-items:center;width:1.4rem;height:1.4rem;border-radius:50%;background:#2738d6;color:#fff;font-size:.66rem}.record-line div{display:grid;gap:.15rem}.record-line b{font-size:.75rem}.record-line small{color:#6c6e6a;font-size:.64rem}.balance-card{display:flex;align-items:end;justify-content:space-between;gap:1rem;margin-top:1rem;padding:1rem;color:#fff;background:#17181b}.balance-card strong{display:block;margin-top:.25rem;font-family:Georgia,serif;font-size:1.7rem;font-weight:500}.balance-card>span{color:#c7c8c3;font-size:.66rem}.next-action{display:flex;align-items:center;justify-content:space-between;width:100%;min-height:3.4rem;margin-top:1rem;padding:.7rem 1rem;border:0;background:#2738d6;color:#fff;font:inherit;font-size:.8rem;font-weight:850;cursor:pointer}.next-action:disabled{opacity:1;color:#fff;cursor:default}.finish-panel{display:grid;grid-template-columns:4rem 1fr;gap:1rem;align-items:center;margin-top:1.2rem;padding:1.2rem;background:#17181b;color:#fff}.finish-check{display:grid;place-items:center;width:3.5rem;height:3.5rem;border-radius:50%;background:#e85f3d;color:#17181b;font-size:1.7rem;font-weight:900;animation:check-in .45s cubic-bezier(.2,.9,.3,1.2)}.finish-panel small{color:#ff9b83;font-size:.6rem;font-weight:850;letter-spacing:.12em}.finish-panel h3{margin:.3rem 0;font-family:Georgia,serif;font-size:1.5rem;font-weight:500}.finish-panel p{margin:0;color:#b7b8b4;font-size:.72rem}.stage-footer{display:flex;align-items:center;justify-content:space-between;gap:1rem;min-height:4rem;padding-top:1rem;color:#8f918d;font-size:.68rem}.stage-footer button{border:0;border-bottom:1px solid #8f918d;background:transparent;color:#d0d1cc;font:inherit;cursor:pointer}.start-real{display:inline-flex;align-items:center;gap:1.5rem;min-height:2.8rem;padding:.4rem .8rem;background:#e85f3d;color:#17181b;font-weight:850;text-decoration:none}
	.after-demo{padding-top:clamp(6rem,10vw,10rem);padding-bottom:clamp(6rem,10vw,10rem)}.after-demo>div:first-child{display:grid;grid-template-columns:.5fr 1.3fr;gap:3rem;align-items:start}.after-demo h2{max-width:16ch;margin:0;font-family:Georgia,serif;font-size:clamp(2.8rem,6vw,5.6rem);font-weight:500;line-height:.95;letter-spacing:-.05em}.answers{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));margin-top:4rem;border-top:1px solid #c8c3b9;border-left:1px solid #c8c3b9}.answers p{display:grid;gap:.35rem;min-height:8rem;margin:0;padding:1rem;border-right:1px solid #c8c3b9;border-bottom:1px solid #c8c3b9;color:#5f645f}.answers span{color:#9f351f;font-size:.65rem;font-weight:850}.answers b{color:#17181b;font-family:Georgia,serif;font-size:1.1rem}.final-cta{display:inline-flex;align-items:center;gap:2rem;min-height:3.5rem;margin-top:2.5rem;padding:0 1.2rem;background:#2738d6;color:#fff;font-weight:850;text-decoration:none;box-shadow:8px 8px 0 #17181b}@keyframes frame-in{from{transform:translateY(8px);opacity:.75}to{transform:none;opacity:1}}@keyframes check-in{from{transform:scale(.4) rotate(-18deg);opacity:0}to{transform:none;opacity:1}}
	@media(max-width:850px){.demo-intro{grid-template-columns:1fr;gap:2rem}.experience-shell{grid-template-columns:1fr}.journey ol{grid-template-columns:repeat(6,minmax(0,1fr));gap:.25rem}.journey li{display:block;min-height:auto}.journey li>span{margin-bottom:.45rem}.journey li div{display:none}.stage-copy{grid-template-columns:auto 1fr}.stage-copy>p:last-child{grid-column:2}.after-demo>div:first-child{grid-template-columns:1fr}.answers{grid-template-columns:repeat(2,minmax(0,1fr))}}@media(max-width:600px){.demo-intro{padding-top:4rem}.demo-intro h1{font-size:clamp(3rem,16vw,4.4rem)}.experience{padding-top:.5rem}.experience-shell{padding-inline:1rem}.demo-stage>header{align-items:flex-start}.demo-stage>header div{align-items:flex-start;flex-direction:column;gap:.15rem}.stage-copy{grid-template-columns:1fr;gap:.6rem;margin:1.5rem 0}.stage-copy>p:last-child{grid-column:auto}.product-frame{min-height:31rem;padding:.9rem;box-shadow:8px 8px 0 #2738d6}.product-frame.complete{box-shadow:8px 8px 0 #e85f3d}.amount-choice{align-items:flex-start;flex-direction:column}.deal-facts{grid-template-columns:1fr}.balance-card{align-items:flex-start;flex-direction:column}.finish-panel{grid-template-columns:1fr}.stage-footer{align-items:flex-start;flex-direction:column;padding-top:1.3rem}.answers{grid-template-columns:1fr}.answers p{min-height:auto}.after-demo h2{font-size:clamp(2.7rem,14vw,4rem)}}@media(prefers-reduced-motion:reduce){.product-frame,.finish-check,.progress span{animation:none;transition:none}}
</style>
