<script lang="ts">
	const steps = [
		{ number: '01', title: 'Add the sale.', body: 'Write the goods, amount and day the customer will pay.', facts: [['Money to pay', '₦1,200,000'], ['Pay in', '30 days'], ['Extra days', '3 days']] },
		{ number: '02', title: 'Your customer says yes.', body: 'Send a private link. Your customer checks the sale and accepts it.', facts: [['Customer', 'Confirmed'], ['Sale', 'Accepted'], ['Debit', 'Allowed']] },
		{ number: '03', title: 'Confirm the goods.', body: 'You mark the goods as sent. Your customer says they received them.', facts: [['You sent it', 'Saved'], ['They got it', 'Confirmed'], ['Proof', 'Saved']] },
		{ number: '04', title: 'See the money.', body: 'See reminders, transfers, problems and payments in one place.', facts: [['Reminders', 'WhatsApp'], ['Problems', 'Can be reported'], ['Debit', 'Only if late']] }
	];

	const events = [
		['10 Sep · 09:14', 'Deal created', 'Payment due in 30 days'],
		['10 Sep · 10:42', 'Customer accepted', 'Customer details confirmed'],
		['12 Sep · 14:06', 'Goods received', 'Delivery proof saved'],
		['12 Sep · 14:07', 'Payment tracking started', '₦1,200,000 left to pay']
	];
</script>


<main class="how-page">
	<section class="intro shell">
		<div><p class="eyebrow">How it works</p><h1>Four easy steps.</h1></div>
		<div class="intro-note"><span>It is simple</span><p>Add the sale. Your customer says yes. Confirm the goods. See the payments.</p></div>
	</section>

	<section class="journey shell" aria-label="The Kredit agreement journey">
		{#each steps as step}
			<article>
				<span class="step-number">{step.number}</span>
				<div class="step-copy"><h2>{step.title}</h2><p>{step.body}</p></div>
				<dl>{#each step.facts as [label, value]}<div><dt>{label}</dt><dd>{value}</dd></div>{/each}</dl>
			</article>
		{/each}
	</section>

	<section class="record-section">
		<div class="shell record-layout">
			<div class="record-copy"><p class="eyebrow inverse">Everything in one place</p><h2>No lost paper.<br />No long search.</h2><p>You and your customer see the same amount, payment day and payment record.</p></div>
			<div class="record" aria-label="Example agreement record">
				<div class="record-head"><span>TC–2048</span><strong>Adebayo Stores</strong><span>ACTIVE</span></div>
				<ol>{#each events as [time, title, detail]}<li><time>{time}</time><div><strong>{title}</strong><span>{detail}</span></div><i>✓</i></li>{/each}</ol>
				<div class="record-total"><span>Amount left to pay</span><strong>₦1,200,000</strong></div>
			</div>
		</div>
	</section>

	<section class="exceptions shell">
		<div class="exceptions-title"><p class="eyebrow">If there is a problem</p><h2>Kredit shows what to do.</h2></div>
		<div class="exception-list">
			<article><span>01</span><div><h3>The goods have a problem</h3><p>Your customer shows the problem. Kredit will not debit that amount until both of you settle it.</p></div></article>
			<article><span>02</span><div><h3>Your customer pays you directly</h3><p>They tell Kredit. You confirm the payment. The money left to pay goes down.</p></div></article>
			<article><span>03</span><div><h3>Your customer pays late</h3><p>Kredit sends reminders. After the extra days end, Kredit can debit only the money still owed.</p></div></article>
		</div>
		<div class="next"><p>Want to know the cost?</p><a href="/pricing">See simple pricing <span>↗</span></a></div>
	</section>
</main>

<style>
	.how-page { color: #17181b; background: #f5f2ea; }
	.intro { display: grid; grid-template-columns: 1.5fr .55fr; gap: clamp(4rem, 10vw, 10rem); align-items: end; padding-top: clamp(5rem, 10vw, 9rem); padding-bottom: clamp(5rem, 9vw, 8rem); }
	h1 { max-width: 11ch; margin: 1.2rem 0 0; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(4rem, 8vw, 7.5rem); font-weight: 500; line-height: .88; letter-spacing: -.065em; }
	.intro-note { padding-top: 1rem; border-top: 1px solid #17181b; }.intro-note span { color: #2738d6; font-size: .68rem; font-weight: 780; letter-spacing: .11em; text-transform: uppercase; }.intro-note p { margin: 1.2rem 0 0; color: #60625f; line-height: 1.65; }
	.journey { padding-bottom: 9rem; }.journey article { display: grid; grid-template-columns: 4rem 1.15fr .85fr; gap: clamp(1.5rem, 5vw, 5rem); padding: 3.3rem 0; border-top: 1px solid #cec9bf; }.journey article:last-child { border-bottom: 1px solid #cec9bf; }.step-number { color: #2738d6; font-size: .74rem; font-weight: 800; letter-spacing: .1em; }.step-copy h2 { max-width: 16ch; margin: 0 0 1rem; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(2rem, 4vw, 3.5rem); font-weight: 500; line-height: 1; letter-spacing: -.045em; }.step-copy p { max-width: 35rem; margin: 0; color: #686a66; line-height: 1.7; }.journey dl { margin: 0; border-top: 1px solid #cec9bf; }.journey dl div { display: flex; justify-content: space-between; gap: 2rem; padding: .75rem 0; border-bottom: 1px solid #cec9bf; }.journey dt { color: #62645f; font-size: .75rem; }.journey dd { margin: 0; font-size: .78rem; font-weight: 760; text-align: right; }
	.record-section { padding: clamp(6rem, 10vw, 10rem) 0; color: #f5f2ea; background: #17181b; }.record-layout { display: grid; grid-template-columns: .72fr 1.28fr; gap: clamp(4rem, 9vw, 9rem); align-items: center; }.inverse { color: #9aa2ff; }.record-copy h2 { max-width: 12ch; margin: 1.2rem 0 1.5rem; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(3.2rem, 5vw, 5rem); font-weight: 500; line-height: .95; letter-spacing: -.05em; }.record-copy > p:last-child { max-width: 31rem; color: #aaa9a4; line-height: 1.7; }
	.record { border: 1px solid #4b4c50; background: #202126; }.record-head { display: grid; grid-template-columns: 1fr 2fr 1fr; gap: 1rem; padding: 1rem 1.2rem; border-bottom: 1px solid #4b4c50; font-size: .7rem; }.record-head strong { color: white; }.record-head span:last-child { color: #c9cdff; text-align: right; }.record ol { margin: 0; padding: 0 1.2rem; list-style: none; }.record li { display: grid; grid-template-columns: 7.5rem 1fr auto; gap: 1.5rem; align-items: center; padding: 1.25rem 0; border-bottom: 1px solid #3c3d41; }.record time { color: #a5a6a2; font-size: .68rem; }.record li div { display: grid; gap: .25rem; }.record li strong { font-size: .86rem; }.record li span { color: #b0b1ad; font-size: .72rem; }.record li i { display: grid; place-items: center; width: 1.5rem; height: 1.5rem; border: 1px solid #7e88ff; color: #c9cdff; font-size: .65rem; font-style: normal; }.record-total { display: flex; justify-content: space-between; gap: 2rem; padding: 1.5rem 1.2rem; background: #2738d6; }.record-total span { font-size: .75rem; }.record-total strong { font-family: Georgia, 'Times New Roman', serif; font-size: 1.4rem; font-weight: 500; }
	.exceptions { padding-top: clamp(6rem, 10vw, 10rem); padding-bottom: clamp(6rem, 10vw, 10rem); }.exceptions-title { display: grid; grid-template-columns: 1fr 2fr; gap: 3rem; align-items: start; }.exceptions-title h2 { max-width: 11ch; margin: 0; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(3.2rem, 6vw, 6rem); font-weight: 500; line-height: .92; letter-spacing: -.055em; }.exception-list { margin-top: 5rem; border-top: 1px solid #cec9bf; }.exception-list article { display: grid; grid-template-columns: 4rem 1fr; gap: 2rem; padding: 2.2rem 0; border-bottom: 1px solid #cec9bf; }.exception-list article > span { color: #2738d6; font-size: .7rem; font-weight: 800; }.exception-list h3 { margin: 0 0 .7rem; font-family: Georgia, 'Times New Roman', serif; font-size: 1.55rem; font-weight: 500; }.exception-list p { max-width: 48rem; margin: 0; color: #686a66; line-height: 1.7; }.next { display: flex; justify-content: space-between; align-items: center; gap: 2rem; margin-top: 4rem; padding-top: 1.2rem; border-top: 3px solid #17181b; }.next p { margin: 0; }.next a { display: inline-flex; gap: 2rem; color: #2738d6; font-weight: 760; text-decoration: none; }
	@media (max-width: 800px) { .intro, .record-layout, .exceptions-title { grid-template-columns: 1fr; gap: 2.5rem; }.intro-note { max-width: 28rem; }.journey article { grid-template-columns: 2.5rem 1fr; }.journey dl { grid-column: 2; }.record-copy { max-width: 36rem; } }
	@media (max-width: 560px) { .intro { padding-top: 4rem; }.journey article { grid-template-columns: 1fr; gap: 1rem; padding: 2.3rem 0; }.journey dl { grid-column: 1; margin-top: 1rem; }.record li { grid-template-columns: 1fr auto; gap: .5rem 1rem; }.record time { grid-column: 1 / -1; }.exceptions-title { gap: 1rem; }.exception-list article { grid-template-columns: 2rem 1fr; gap: 1rem; }.next { align-items: flex-start; flex-direction: column; } }
</style>
