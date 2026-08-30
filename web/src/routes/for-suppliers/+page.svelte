<script lang="ts">
	const capabilities = [
		['Know your customer', 'Send the sale to the right person. Let them confirm their details.'],
		['Write the sale down', 'Add the goods, amount and payment day. Nobody can change it after both sides agree.'],
		['Show that goods arrived', 'Save the delivery. Let your customer say they got the goods.'],
		['See the money left', 'Every payment brings down the amount your customer owes.'],
		['Settle problems clearly', 'Keep the message, photo or paper that shows what went wrong.'],
		['Control your staff', 'Choose who can add sales, confirm money and make changes.']
	];
	const journey = ['Add your business details', 'Invite a trusted customer', 'Write down the deal', 'Confirm delivery', 'Track payment and send reminders', 'Close the deal when fully paid'];
</script>


<main class="supplier-page">
	<section class="supplier-hero shell">
		<div><p class="eyebrow">For sellers</p><h1>Give goods now.<br /><em>Collect your money later.</em></h1></div>
		<div class="hero-side"><p>Know who took the goods, how much they owe and when they should pay.</p><div class="actions"><a class="primary-button" href="/app">Add a sale ↗</a><a href="/pricing">See the price</a></div><small>You give the goods. You choose the customer. Kredit does not give loans.</small></div>
	</section>

	<section class="operating shell">
		<div class="section-head"><p class="eyebrow">Made for sellers</p><h2>Know your money.<br />Chase less.</h2></div>
		<div class="capability-list">{#each capabilities as [title, body], index}<article><span>0{index + 1}</span><div><h3>{title}</h3><p>{body}</p></div></article>{/each}</div>
	</section>

	<section class="workflow-section">
		<div class="shell workflow-layout">
			<div><p class="eyebrow inverse">A better way to sell</p><h2>Do one thing at a time.</h2><p>Add the customer. Add the sale. Give the goods. See the payment.</p></div>
			<ol>{#each journey as step, index}<li><span>0{index + 1}</span><strong>{step}</strong></li>{/each}</ol>
		</div>
	</section>

	<section class="facts shell">
		<div><p class="eyebrow">You choose who you trust</p><h2>No secret score.</h2></div>
		<div><p>See the customer, the money they owe you and how they paid you before.</p><p>Kredit does not choose your customer. Another seller cannot see your private notes.</p></div>
	</section>

	<section class="supplier-cta shell"><span>START WITH ONE CUSTOMER</span><h2>Add the sale before goods leave.</h2><a href="/app">Add your first sale <b>↗</b></a></section>
</main>

<style>
	.supplier-page { color: #17181b; background: #f5f2ea; }.supplier-hero { display: grid; grid-template-columns: 1.3fr .7fr; gap: clamp(4rem, 10vw, 10rem); align-items: end; padding-top: clamp(5rem, 10vw, 9rem); padding-bottom: clamp(6rem, 10vw, 9rem); }.supplier-hero h1 { margin: 1rem 0 0; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(4rem, 8vw, 7.5rem); font-weight: 500; line-height: .88; letter-spacing: -.065em; }.supplier-hero h1 em { color: #2738d6; font-weight: 500; }.hero-side { padding-top: 1.2rem; border-top: 1px solid #17181b; }.hero-side > p { margin: 0; color: #5f615e; font-size: 1.1rem; line-height: 1.7; }.actions { display: flex; align-items: center; flex-wrap: wrap; gap: 1.4rem; margin: 2rem 0; }.actions > a:not(.primary-button) { color: #17181b; font-weight: 760; text-decoration: underline; text-underline-offset: .3rem; }.hero-side small { display: block; color: #62645f; line-height: 1.55; }
	.operating { padding-top: 8rem; padding-bottom: 9rem; border-top: 1px solid #cec9bf; }.section-head { display: grid; grid-template-columns: .55fr 1.45fr; gap: 3rem; }.section-head h2, .facts h2 { max-width: 12ch; margin: 0; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(3.2rem, 6vw, 6rem); font-weight: 500; line-height: .92; letter-spacing: -.055em; }.capability-list { display: grid; grid-template-columns: 1fr 1fr; margin-top: 5rem; border-top: 1px solid #cec9bf; border-left: 1px solid #cec9bf; }.capability-list article { display: grid; grid-template-columns: 2.5rem 1fr; gap: 1rem; padding: 2rem; border-right: 1px solid #cec9bf; border-bottom: 1px solid #cec9bf; }.capability-list article > span { color: #2738d6; font-size: .68rem; font-weight: 800; }.capability-list h3 { margin: 0 0 .7rem; font-family: Georgia, 'Times New Roman', serif; font-size: 1.4rem; font-weight: 500; }.capability-list p { margin: 0; color: #686a66; line-height: 1.65; }
	.workflow-section { padding: 8rem 0; color: #f5f2ea; background: #17181b; }.workflow-layout { display: grid; grid-template-columns: .8fr 1.2fr; gap: clamp(4rem, 10vw, 10rem); }.inverse { color: #aab0ff; }.workflow-layout h2 { max-width: 10ch; margin: 1.2rem 0 1.5rem; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(3.2rem, 6vw, 5.7rem); font-weight: 500; line-height: .92; letter-spacing: -.05em; }.workflow-layout > div > p:last-child { max-width: 30rem; color: #a6a6a1; line-height: 1.7; }.workflow-layout ol { margin: 0; padding: 0; border-top: 1px solid #48494d; list-style: none; }.workflow-layout li { display: grid; grid-template-columns: 3rem 1fr; gap: 1rem; padding: 1.25rem 0; border-bottom: 1px solid #48494d; }.workflow-layout li span { color: #939cff; font-size: .68rem; }.workflow-layout li strong { font-family: Georgia, 'Times New Roman', serif; font-size: 1.15rem; font-weight: 500; }
	.facts { display: grid; grid-template-columns: 1fr 1fr; gap: clamp(4rem, 10vw, 10rem); padding-top: 9rem; padding-bottom: 9rem; }.facts > div:last-child { padding-top: 1.2rem; border-top: 1px solid #17181b; }.facts p { margin: 0 0 1.2rem; color: #686a66; font-size: 1.05rem; line-height: 1.75; }
	.supplier-cta { padding-top: 7rem; padding-bottom: 8rem; border-top: 1px solid #cec9bf; text-align: center; }.supplier-cta > span { color: #2738d6; font-size: .68rem; font-weight: 800; letter-spacing: .13em; }.supplier-cta h2 { max-width: 13ch; margin: 1.5rem auto 2rem; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(3.2rem, 6vw, 6rem); font-weight: 500; line-height: .93; letter-spacing: -.055em; }.supplier-cta a { display: inline-flex; gap: 3rem; padding: 1rem 1.2rem; color: white; background: #2738d6; font-weight: 760; text-decoration: none; }.supplier-cta b { color: #ffb8a7; }
	@media (max-width: 800px) { .supplier-hero, .section-head, .workflow-layout, .facts { grid-template-columns: 1fr; gap: 2.5rem; }.hero-side { max-width: 32rem; }.capability-list { grid-template-columns: 1fr; } }
	@media (max-width: 520px) { .supplier-hero { padding-top: 4rem; }.operating, .workflow-section, .facts { padding-top: 6rem; padding-bottom: 6rem; }.capability-list article { padding: 1.5rem 1rem; }.supplier-cta { padding-top: 5rem; padding-bottom: 6rem; } }
</style>
