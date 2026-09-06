<script lang="ts">
	const controls = [
		['You read everything first', 'The seller, the goods, the money and the payment day are all in front of you before you say yes to anything.'],
		['You control the bank debit', 'You can see the most Kredit may ever take from your account, and you can stop that permission whenever you like.'],
		['You say what actually arrived', 'Tell us when the goods reach you, or report a problem while everybody still remembers what happened.'],
		['A problem goes on record', 'Say which goods or how much money is affected, and add a photo or a paper if you have one.'],
		['Every payment you made is there', 'Transfers, cash and bank debits all sit in one history you can open any time.'],
		['Nobody is scoring you', 'Your account only shows what actually happened in your own sales. It is not a public credit rating.']
	];
	const sequence = ['Read the sale', 'Say yes to the terms', 'Receive the goods', 'Confirm they arrived, or report a problem'];
</script>

<main class="buyer-page">
	<section class="buyer-hero shell">
		<div><p class="eyebrow">For customers</p><h1>No seller can <em>surprise you.</em></h1></div>
		<div class="hero-side"><p>Open the sale you agreed to. See what is still left, and everything you have already paid.</p><div class="actions"><a class="primary-button" href="/buyer">Open my account ↗</a><a href="/how-it-works">See how Kredit works</a></div></div>
	</section>

	<section class="sequence-section">
		<div class="shell"><div class="sequence-head"><p class="eyebrow inverse">Nothing hidden</p><h2>Nobody is rushing you.</h2></div><ol>{#each sequence as step, index}<li><span>0{index + 1}</span><strong>{step}</strong></li>{/each}</ol></div>
	</section>

	<section class="controls-section shell">
		<div class="section-head"><p class="eyebrow">You stay in control</p><h2>You see everything before you agree to anything.</h2></div>
		<div class="control-list">{#each controls as [title, body], index}<article><span>0{index + 1}</span><div><h3>{title}</h3><p>{body}</p></div></article>{/each}</div>
	</section>

	<section class="privacy-section">
		<div class="shell privacy-layout"><div><p class="eyebrow inverse">Private by design</p><h2>Your money details stay private.</h2></div><div><p>If somebody forwards one of your receipts, it does not carry your name or your bank details.</p><p>You, the seller staff who are allowed, and the Kredit staff helping you can only open what that particular sale, payment or support case needs. The privacy notice spells it out.</p><a href="/security">See how Kredit keeps your account safe <span>↗</span></a></div></div>
	</section>

	<section class="buyer-cta shell"><span>DID A SELLER SEND YOU A LINK?</span><h2>Open the private link they sent you.</h2><p>The link stops working after a while. That is for your own safety. If it will not open, just ask the seller to send you a fresh one. And never give your sign-in code to anybody.</p><a href="/buyer">Open my account <b>↗</b></a></section>
</main>

<style>
	.buyer-page { color: #17181b; background: #f5f2ea; }.buyer-hero { display: grid; grid-template-columns: 1.3fr .7fr; gap: clamp(4rem, 10vw, 10rem); align-items: end; padding-top: clamp(5rem, 10vw, 9rem); padding-bottom: clamp(6rem, 10vw, 9rem); }.buyer-hero h1 { margin: 1rem 0 0; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(4rem, 8vw, 7.5rem); font-weight: 500; line-height: .88; letter-spacing: -.065em; }.buyer-hero h1 em { color: #b83d26; font-weight: 500; }.hero-side { padding-top: 1.2rem; border-top: 1px solid #17181b; }.hero-side > p { margin: 0; color: #5f615e; font-size: 1.1rem; line-height: 1.7; }.actions { display: flex; align-items: center; flex-wrap: wrap; gap: 1.4rem; margin-top: 2rem; }.actions > a:not(.primary-button) { color: #17181b; font-weight: 760; text-decoration: underline; text-underline-offset: .3rem; }
	.sequence-section { padding: 7rem 0; color: #f5f2ea; background: #2738d6; }.inverse { color: #c9cdff; }.sequence-head { display: grid; grid-template-columns: .55fr 1.45fr; gap: 3rem; }.sequence-head h2 { max-width: 13ch; margin: 0; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(3.2rem, 6vw, 5.8rem); font-weight: 500; line-height: .93; letter-spacing: -.05em; }.sequence-section ol { display: grid; grid-template-columns: repeat(4, 1fr); margin: 5rem 0 0; padding: 0; border-top: 1px solid rgb(255 255 255 / .35); border-left: 1px solid rgb(255 255 255 / .35); list-style: none; }.sequence-section li { min-height: 10rem; padding: 1.3rem; border-right: 1px solid rgb(255 255 255 / .35); border-bottom: 1px solid rgb(255 255 255 / .35); }.sequence-section li span { display: block; margin-bottom: 4rem; color: #c9cdff; font-size: .68rem; }.sequence-section li strong { font-family: Georgia, 'Times New Roman', serif; font-size: 1.2rem; font-weight: 500; }
	.controls-section { padding-top: 9rem; padding-bottom: 9rem; }.section-head { display: grid; grid-template-columns: .55fr 1.45fr; gap: 3rem; }.section-head h2 { max-width: 12ch; margin: 0; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(3.2rem, 6vw, 6rem); font-weight: 500; line-height: .92; letter-spacing: -.055em; }.control-list { display: grid; grid-template-columns: 1fr 1fr; margin-top: 5rem; border-top: 1px solid #cec9bf; border-left: 1px solid #cec9bf; }.control-list article { display: grid; grid-template-columns: 2.5rem 1fr; gap: 1rem; padding: 2rem; border-right: 1px solid #cec9bf; border-bottom: 1px solid #cec9bf; }.control-list article > span { color: #b83d26; font-size: .68rem; font-weight: 800; }.control-list h3 { margin: 0 0 .7rem; font-family: Georgia, 'Times New Roman', serif; font-size: 1.4rem; font-weight: 500; }.control-list p { margin: 0; color: #686a66; line-height: 1.65; }
	.privacy-section { padding: 8rem 0; color: #f5f2ea; background: #17181b; }.privacy-layout { display: grid; grid-template-columns: 1fr 1fr; gap: clamp(4rem, 10vw, 10rem); }.privacy-layout h2 { max-width: 12ch; margin: 1rem 0 0; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(3.2rem, 6vw, 5.8rem); font-weight: 500; line-height: .93; letter-spacing: -.05em; }.privacy-layout > div:last-child { padding-top: 1.2rem; border-top: 1px solid #696a6e; }.privacy-layout p { margin: 0 0 1.2rem; color: #aaa9a4; font-size: 1.02rem; line-height: 1.75; }.privacy-layout a { display: inline-flex; gap: 2rem; margin-top: 1rem; color: #b4baff; font-weight: 760; text-decoration: none; }
	.buyer-cta { padding-top: 8rem; padding-bottom: 9rem; text-align: center; }.buyer-cta > span { color: #2738d6; font-size: .68rem; font-weight: 800; letter-spacing: .13em; }.buyer-cta h2 { max-width: 13ch; margin: 1.4rem auto 1rem; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(3.2rem, 6vw, 6rem); font-weight: 500; line-height: .93; letter-spacing: -.055em; }.buyer-cta p { max-width: 35rem; margin: 0 auto 2rem; color: #686a66; line-height: 1.7; }.buyer-cta a { display: inline-flex; gap: 3rem; padding: 1rem 1.2rem; color: white; background: #2738d6; font-weight: 760; text-decoration: none; }.buyer-cta b { color: #ffb8a7; }
	@media (max-width: 800px) { .buyer-hero, .sequence-head, .section-head, .privacy-layout { grid-template-columns: 1fr; gap: 2.5rem; }.hero-side { max-width: 32rem; }.sequence-section ol, .control-list { grid-template-columns: 1fr 1fr; } }
	@media (max-width: 540px) { .buyer-hero { padding-top: 4rem; }.sequence-section, .controls-section, .privacy-section { padding-top: 6rem; padding-bottom: 6rem; }.sequence-section ol, .control-list { grid-template-columns: 1fr; }.sequence-section li { min-height: auto; }.sequence-section li span { margin-bottom: 2rem; }.control-list article { padding: 1.5rem 1rem; }.buyer-cta { padding-top: 6rem; padding-bottom: 7rem; } }
</style>
