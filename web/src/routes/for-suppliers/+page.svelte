<script lang="ts">
	const capabilities = [
		['You know exactly who took your goods', 'Send the sale to the right person. They confirm who they are before anything leaves your shop.'],
		['Keep the agreed history', 'Accepted terms stay on record. Any approved correction or payment-date change has its own history.'],
		['You keep the proof', 'Save the delivery note, and let your customer confirm with their own hand that the goods reached them.'],
		['You always know what is left', 'Every payment brings the balance down. Open your phone and see what each person still owes you.'],
		['Keep the facts together', 'When something goes wrong, the messages, photos and papers are already sitting with that sale.'],
		['Your staff only see what they should', 'Decide who can add sales, who can confirm money, who can only look and who runs the whole account.']
	];
	const journey = ['Add your business', 'Add your customer', 'Add the sale', 'Confirm the goods arrived', 'Follow reminders and payments', 'Close the sale when it is fully paid'];
</script>

<main class="supplier-page">
	<section class="supplier-hero shell">
		<div><p class="eyebrow">For sellers</p><h1>Sell on credit <em>without losing sleep.</em></h1></div>
		<div class="hero-side"><p>You know who owes you, how much is left and which day it is due. The sale, the delivery proof and every payment sit in one place.</p><div class="actions"><a class="primary-button" href="/app">Add your first sale ↗</a><a href="/demo">See it work first</a></div><small>You still choose who takes your goods. Kredit only writes it down and helps you follow the money. We do not lend anybody anything.</small></div>
	</section>

	<section class="operating shell">
		<div class="section-head"><p class="eyebrow">Built for real trade</p><h2>Stop chasing people for your own money.</h2></div>
		<div class="capability-list">{#each capabilities as [title, body], index}<article><span>0{index + 1}</span><div><h3>{title}</h3><p>{body}</p></div></article>{/each}</div>
	</section>

	<section class="workflow-section">
		<div class="shell workflow-layout">
			<div><p class="eyebrow inverse">One clear way to work</p><h2>Six steps, and the sale is closed.</h2><p>Write it down before the goods leave. Keep the delivery proof. Then follow the money until the balance reaches zero.</p></div>
			<ol>{#each journey as step, index}<li><span>0{index + 1}</span><strong>{step}</strong></li>{/each}</ol>
		</div>
	</section>

	<section class="facts shell">
		<div><p class="eyebrow">Your customer. Your decision.</p><h2>No secret credit score.</h2></div>
		<div><p>You see the person, how they have paid you before and what they still owe you.</p><p>Kredit does not decide who deserves your goods. You do. Other sellers cannot open your private sale records. Limited trade-history sharing requires the applicable customer consent and access checks.</p></div>
	</section>

	<section class="supplier-cta shell"><span>START WITH ONE CUSTOMER</span><h2>Write it down before the goods leave your shop.</h2><a href="/app">Add your first sale <b>↗</b></a></section>
</main>

<style>
	.supplier-page { color: #17181b; background: #f5f2ea; }.supplier-hero { display: grid; grid-template-columns: 1.3fr .7fr; gap: clamp(4rem, 10vw, 10rem); align-items: end; padding-top: clamp(5rem, 10vw, 9rem); padding-bottom: clamp(6rem, 10vw, 9rem); }.supplier-hero h1 { margin: 1rem 0 0; font-family: var(--font-serif); font-size: clamp(4rem, 8vw, 7.5rem); font-weight: 500; line-height: .88; letter-spacing: -.065em; }.supplier-hero h1 em { color: #2738d6; font-weight: 500; }.hero-side { padding-top: 1.2rem; border-top: 1px solid #17181b; }.hero-side > p { margin: 0; color: #5f615e; font-size: 1.1rem; line-height: 1.7; }.actions { display: flex; align-items: center; flex-wrap: wrap; gap: 1.4rem; margin: 2rem 0; }.actions > a:not(.primary-button) { color: #17181b; font-weight: 760; text-decoration: underline; text-underline-offset: .3rem; }.hero-side small { display: block; color: #62645f; line-height: 1.55; }
	.operating { padding-top: 8rem; padding-bottom: 9rem; border-top: 1px solid #cec9bf; }.section-head { display: grid; grid-template-columns: .55fr 1.45fr; gap: 3rem; }.section-head h2, .facts h2 { max-width: 12ch; margin: 0; font-family: var(--font-serif); font-size: clamp(3.2rem, 6vw, 6rem); font-weight: 500; line-height: .92; letter-spacing: -.055em; }.capability-list { display: grid; grid-template-columns: 1fr 1fr; margin-top: 5rem; border-top: 1px solid #cec9bf; border-left: 1px solid #cec9bf; }.capability-list article { display: grid; grid-template-columns: 2.5rem 1fr; gap: 1rem; padding: 2rem; border-right: 1px solid #cec9bf; border-bottom: 1px solid #cec9bf; }.capability-list article > span { color: #2738d6; font-size: .68rem; font-weight: 800; }.capability-list h3 { margin: 0 0 .7rem; font-family: var(--font-serif); font-size: 1.4rem; font-weight: 500; }.capability-list p { margin: 0; color: #686a66; line-height: 1.65; }
	.workflow-section { padding: 8rem 0; color: #f5f2ea; background: #17181b; }.workflow-layout { display: grid; grid-template-columns: .8fr 1.2fr; gap: clamp(4rem, 10vw, 10rem); }.inverse { color: #aab0ff; }.workflow-layout h2 { max-width: 10ch; margin: 1.2rem 0 1.5rem; font-family: var(--font-serif); font-size: clamp(3.2rem, 6vw, 5.7rem); font-weight: 500; line-height: .92; letter-spacing: -.05em; }.workflow-layout > div > p:last-child { max-width: 30rem; color: #a6a6a1; line-height: 1.7; }.workflow-layout ol { margin: 0; padding: 0; border-top: 1px solid #48494d; list-style: none; }.workflow-layout li { display: grid; grid-template-columns: 3rem 1fr; gap: 1rem; padding: 1.25rem 0; border-bottom: 1px solid #48494d; }.workflow-layout li span { color: #939cff; font-size: .68rem; }.workflow-layout li strong { font-family: var(--font-serif); font-size: 1.15rem; font-weight: 500; }
	.facts { display: grid; grid-template-columns: 1fr 1fr; gap: clamp(4rem, 10vw, 10rem); padding-top: 9rem; padding-bottom: 9rem; }.facts > div:last-child { padding-top: 1.2rem; border-top: 1px solid #17181b; }.facts p { margin: 0 0 1.2rem; color: #686a66; font-size: 1.05rem; line-height: 1.75; }
	.supplier-cta { padding-top: 7rem; padding-bottom: 8rem; border-top: 1px solid #cec9bf; text-align: center; }.supplier-cta > span { color: #2738d6; font-size: .68rem; font-weight: 800; letter-spacing: .13em; }.supplier-cta h2 { max-width: 13ch; margin: 1.5rem auto 2rem; font-family: var(--font-serif); font-size: clamp(3.2rem, 6vw, 6rem); font-weight: 500; line-height: .93; letter-spacing: -.055em; }.supplier-cta a { display: inline-flex; gap: 3rem; padding: 1rem 1.2rem; color: white; background: #2738d6; font-weight: 760; text-decoration: none; }.supplier-cta b { color: #ffb8a7; }
	@media (max-width: 800px) { .supplier-hero, .section-head, .workflow-layout, .facts { grid-template-columns: 1fr; gap: 2.5rem; }.hero-side { max-width: 32rem; }.capability-list { grid-template-columns: 1fr; } }
	@media (max-width: 520px) { .supplier-hero { padding-top: 4rem; }.operating, .workflow-section, .facts { padding-top: 6rem; padding-bottom: 6rem; }.capability-list article { padding: 1.5rem 1rem; }.supplier-cta { padding-top: 5rem; padding-bottom: 6rem; } }
</style>
