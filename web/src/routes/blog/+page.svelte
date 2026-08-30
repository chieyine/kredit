<script lang="ts">
	const posts = [
		{ href: '/blog/sell-on-credit-safely', title: 'How to sell on credit and protect your money', excerpt: 'Five things to agree before goods leave your shop.', date: '10 August 2026', minutes: 6, issue: '01' },
		{ href: '/blog/why-credit-agreements-fail', title: 'Why credit sales turn into arguments', excerpt: 'Most problems start because the goods, date or amount were not clearly recorded.', date: '4 August 2026', minutes: 5, issue: '02' },
		{ href: '/blog/collections-last-resort', title: 'What to do when a customer pays late', excerpt: 'Remind them, allow extra time, record any transfer, then debit only as a last step.', date: '28 July 2026', minutes: 5, issue: '03' },
		{ href: '/blog/trade-credit-vs-loan', title: 'Goods on credit or a loan app?', excerpt: 'A simple look at two ways your customer can buy stock now and pay later.', date: '20 July 2026', minutes: 4, issue: '04' }
	];
	const featured = posts[0];
</script>


<header class="playbook-head">
	<div><p class="eyebrow">Helpful guides</p><h1>Simple advice for selling on credit.</h1></div>
	<p>Short, useful guides for sellers and the customers who buy from them. No big finance words.</p>
</header>

<section class="featured">
	<div class="feature-mark" aria-hidden="true"><span>GUIDE</span><strong>01</strong><i>T</i></div>
	<a href={featured.href}>
		<div class="meta"><span>{featured.date}</span><span>{featured.minutes} minute read</span></div>
		<h2>{featured.title}</h2>
		<p>{featured.excerpt}</p>
		<span class="read">Read guide <b>↗</b></span>
	</a>
</section>

<section class="archive" aria-labelledby="archive-title">
	<div class="archive-head"><p id="archive-title">More helpful guides</p><span>Guide / Reading time</span></div>
	{#each posts.slice(1) as post}
		<a class="post-row" href={post.href}>
			<span class="issue">{post.issue}</span>
			<div><h2>{post.title}</h2><p>{post.excerpt}</p></div>
			<span class="post-meta">{post.minutes} min<br />{post.date}</span>
			<i aria-hidden="true">↗</i>
		</a>
	{/each}
</section>

<aside class="playbook-end"><p>Seen a word you do not understand?</p><a href="/glossary">See simple meanings <span>↗</span></a></aside>

<style>
	.playbook-head { display: grid; grid-template-columns: 1.4fr .6fr; gap: clamp(4rem, 10vw, 10rem); align-items: end; }
	.playbook-head h1 { max-width: 11ch; margin: 1.2rem 0 0; }
	.playbook-head > p { margin: 0; padding-top: 1.2rem; border-top: 1px solid #17181b; color: #686a66; font-size: 1.05rem; line-height: 1.7; }
	.featured { display: grid; grid-template-columns: .75fr 1.25fr; min-height: 33rem; margin-top: clamp(4rem, 8vw, 7rem); color: white; background: #2738d6; }.feature-mark { position: relative; display: flex; justify-content: space-between; padding: 1.5rem; overflow: hidden; border-right: 1px solid rgb(255 255 255 / .3); }.feature-mark span { font-size: .65rem; font-weight: 800; letter-spacing: .14em; }.feature-mark strong { font-family: Georgia, 'Times New Roman', serif; font-size: 1.4rem; font-weight: 500; }.feature-mark i { position: absolute; left: 50%; bottom: -4rem; transform: translateX(-50%); color: #ff6848; font-family: Georgia, 'Times New Roman', serif; font-size: 24rem; font-style: normal; line-height: .8; }
	.featured > a { display: flex; flex-direction: column; padding: clamp(2rem, 5vw, 4rem); color: white; text-decoration: none; }.meta { display: flex; justify-content: space-between; gap: 1rem; color: #c7cbff; font-size: .7rem; }.featured h2 { max-width: 13ch; margin: auto 0 1.2rem; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(2.7rem, 5vw, 5rem); font-weight: 500; line-height: .93; letter-spacing: -.05em; }.featured p { max-width: 36rem; margin: 0 0 2rem; color: #d8daff; line-height: 1.65; }.read { display: flex; justify-content: space-between; padding-top: 1rem; border-top: 1px solid rgb(255 255 255 / .35); font-size: .78rem; font-weight: 760; }.read b { color: #ffb7a6; font-size: 1.1rem; }
	.archive { margin-top: 7rem; }.archive-head { display: flex; justify-content: space-between; gap: 2rem; padding-bottom: 1rem; border-bottom: 3px solid #17181b; font-size: .68rem; font-weight: 780; letter-spacing: .1em; text-transform: uppercase; }.archive-head p { margin: 0; }.archive-head span { color: #62645f; }.post-row { display: grid; grid-template-columns: 3rem 1fr 8rem 2rem; gap: clamp(1rem, 4vw, 4rem); align-items: start; padding: 2.4rem 0; border-bottom: 1px solid #cec9bf; color: #17181b; text-decoration: none; }.issue { color: #2738d6; font-size: .68rem; font-weight: 800; }.post-row h2 { max-width: 25ch; margin: 0 0 .7rem; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(1.6rem, 3vw, 2.5rem); font-weight: 500; line-height: 1.05; letter-spacing: -.035em; }.post-row p { max-width: 45rem; margin: 0; color: #686a66; line-height: 1.65; }.post-meta { color: #62645f; font-size: .68rem; line-height: 1.6; }.post-row i { color: #2738d6; font-size: 1.1rem; font-style: normal; }.post-row:hover h2 { color: #2738d6; }
	.playbook-end { display: flex; justify-content: space-between; gap: 2rem; align-items: center; margin-top: 5rem; padding-top: 1.3rem; border-top: 3px solid #17181b; }.playbook-end p { margin: 0; font-family: Georgia, 'Times New Roman', serif; font-size: 1.3rem; }.playbook-end a { display: inline-flex; gap: 2rem; color: #2738d6; font-weight: 760; text-decoration: none; }
	@media (max-width: 780px) { .playbook-head { grid-template-columns: 1fr; gap: 2.5rem; }.featured { grid-template-columns: 1fr; }.feature-mark { min-height: 13rem; border-right: 0; border-bottom: 1px solid rgb(255 255 255 / .3); }.feature-mark i { bottom: -8rem; font-size: 20rem; }.featured > a { min-height: 25rem; }.post-row { grid-template-columns: 2rem 1fr 2rem; }.post-meta { grid-column: 2; grid-row: 2; }.post-row i { grid-column: 3; grid-row: 1; } }
	@media (max-width: 520px) { .featured { margin-inline: -1.25rem; }.archive { margin-top: 5rem; }.archive-head span { display: none; }.post-row { gap: .8rem; }.post-row p { display: none; }.playbook-end { align-items: flex-start; flex-direction: column; } }
</style>
