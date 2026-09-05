<script lang="ts">
	import { articleCategoryDetails, articles, articleCategories } from '$lib/blog/articles';
	import { jsonLd } from '$lib/seo';
	let query=$state(''),category=$state('All guides');
	const posts = articles.map((article,index)=>({href:`/blog/${article.slug}`,title:article.title,excerpt:article.description,date:new Date(`${article.modified}T12:00:00`).toLocaleDateString('en-NG',{day:'numeric',month:'long',year:'numeric'}),minutes:article.readingMinutes,issue:String(index+1).padStart(3,'0'),category:article.category}));
	const featured = posts[0];
	const topics = articleCategories.map(category => ({ category, ...articleCategoryDetails[category], count: articles.filter(article => article.category === category).length }));
	const visiblePosts=$derived(posts.filter(post=>(category==='All guides'||post.category===category)&&`${post.title} ${post.excerpt} ${post.category}`.toLowerCase().includes(query.trim().toLowerCase())));
	const listSchema={'@context':'https://schema.org','@type':'ItemList',name:'Kredit guides for Nigerian businesses',numberOfItems:articles.length,itemListElement:articles.map((article,index)=>({'@type':'ListItem',position:index+1,name:article.title,url:`https://kredit.com.ng/blog/${article.slug}`}))};
</script>

<svelte:head>{@html `<script type="application/ld+json">${jsonLd(listSchema)}<\/script>`}</svelte:head>


<header class="playbook-head">
	<div><p class="eyebrow">Helpful guides</p><h1>Simple advice for selling on credit.</h1></div>
	<p>Short, useful guides for sellers and the customers who buy from them. From setting a credit limit to checking the last payment.</p>
</header>

<section class="featured">
	<div class="feature-mark" aria-hidden="true"><span>GUIDE</span><strong>01</strong><i>K</i></div>
	<a href={featured.href}>
		<div class="meta"><span>{featured.date}</span><span>{featured.minutes} minute read</span></div>
		<h2>{featured.title}</h2>
		<p>{featured.excerpt}</p>
		<span class="read">Read guide <b>↗</b></span>
	</a>
</section>

<nav class="topic-nav" aria-label="Browse guide categories"><p class="eyebrow">Browse one topic</p><div>{#each topics as topic}<a href={`/blog/topic/${topic.slug}`}><span>{topic.count} {topic.count === 1 ? 'guide' : 'guides'}</span><strong>{topic.category}</strong><i aria-hidden="true">→</i></a>{/each}</div></nav>

<section class="library" aria-labelledby="library-title"><div><p class="eyebrow">{articles.length} guides</p><h2 id="library-title">Find the answer you need.</h2></div><div class="filters"><label>Search guides<input type="search" value={query} oninput={(event)=>query=event.currentTarget.value} placeholder="For example: late payment" /></label><label>Topic<select bind:value={category}><option value="All guides">All guides</option>{#each articleCategories as item}<option value={item}>{item}</option>{/each}</select></label></div></section>
<section class="archive" aria-labelledby="archive-title">
	<div class="archive-head"><p id="archive-title">{visiblePosts.length} helpful {visiblePosts.length === 1 ? 'guide' : 'guides'}</p><span>Guide / Reading time</span></div>
	{#each visiblePosts as post}
		<a class="post-row" href={post.href}>
			<span class="issue">{post.issue}</span>
			<div><small>{post.category}</small><h2>{post.title}</h2><p>{post.excerpt}</p></div>
			<span class="post-meta">{post.minutes} min<br />Updated {post.date}</span>
			<i aria-hidden="true">↗</i>
		</a>
	{:else}<div class="no-results"><h2>No guide matches that search.</h2><p>Try a shorter word such as “payment”, “customer” or “credit”.</p><button onclick={()=>{query='';category='All guides'}}>Show every guide</button></div>
	{/each}
</section>

<aside class="playbook-end"><p>Seen a word you do not understand?</p><a href="/glossary">See simple meanings <span>↗</span></a></aside>

<style>
	.playbook-head { display: grid; grid-template-columns: 1.4fr .6fr; gap: clamp(4rem, 10vw, 10rem); align-items: end; }
	.playbook-head h1 { max-width: 11ch; margin: 1.2rem 0 0; }
	.playbook-head > p { margin: 0; padding-top: 1.2rem; border-top: 1px solid #17181b; color: #686a66; font-size: 1.05rem; line-height: 1.7; }
	.featured { display: grid; grid-template-columns: .75fr 1.25fr; min-height: 33rem; margin-top: clamp(4rem, 8vw, 7rem); color: white; background: #2738d6; }.feature-mark { position: relative; display: flex; justify-content: space-between; padding: 1.5rem; overflow: hidden; border-right: 1px solid rgb(255 255 255 / .3); }.feature-mark span { font-size: .65rem; font-weight: 800; letter-spacing: .14em; }.feature-mark strong { font-family: Georgia, 'Times New Roman', serif; font-size: 1.4rem; font-weight: 500; }.feature-mark i { position: absolute; left: 50%; bottom: -4rem; transform: translateX(-50%); color: #ff6848; font-family: Georgia, 'Times New Roman', serif; font-size: 24rem; font-style: normal; line-height: .8; }
	.featured > a { display: flex; flex-direction: column; padding: clamp(2rem, 5vw, 4rem); color: white; text-decoration: none; }.meta { display: flex; justify-content: space-between; gap: 1rem; color: #c7cbff; font-size: .7rem; }.featured h2 { max-width: 13ch; margin: auto 0 1.2rem; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(2.7rem, 5vw, 5rem); font-weight: 500; line-height: .93; letter-spacing: -.05em; }.featured p { max-width: 36rem; margin: 0 0 2rem; color: #d8daff; line-height: 1.65; }.read { display: flex; justify-content: space-between; padding-top: 1rem; border-top: 1px solid rgb(255 255 255 / .35); font-size: .78rem; font-weight: 760; }.read b { color: #ffb7a6; font-size: 1.1rem; }
	.topic-nav{margin-top:5rem}.topic-nav>div{display:grid;grid-template-columns:repeat(5,minmax(0,1fr));margin-top:1rem;border-top:1px solid #bbb5aa;border-left:1px solid #bbb5aa}.topic-nav a{display:grid;gap:.7rem;min-height:7rem;padding:1rem;border-right:1px solid #bbb5aa;border-bottom:1px solid #bbb5aa;color:#17181b;text-decoration:none}.topic-nav a span{color:#6b6862;font-size:.65rem;text-transform:uppercase}.topic-nav a strong{font-family:Georgia,'Times New Roman',serif;font-size:1.05rem;font-weight:500}.topic-nav a i{align-self:end;color:#2738d6;font-style:normal}.topic-nav a:hover{background:#eef0ff}
	.archive { margin-top: 7rem; }.archive-head { display: flex; justify-content: space-between; gap: 2rem; padding-bottom: 1rem; border-bottom: 3px solid #17181b; font-size: .68rem; font-weight: 780; letter-spacing: .1em; text-transform: uppercase; }.archive-head p { margin: 0; }.archive-head span { color: #62645f; }.post-row { display: grid; grid-template-columns: 3rem 1fr 8rem 2rem; gap: clamp(1rem, 4vw, 4rem); align-items: start; padding: 2.4rem 0; border-bottom: 1px solid #cec9bf; color: #17181b; text-decoration: none; }.issue { color: #2738d6; font-size: .68rem; font-weight: 800; }.post-row h2 { max-width: 25ch; margin: 0 0 .7rem; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(1.6rem, 3vw, 2.5rem); font-weight: 500; line-height: 1.05; letter-spacing: -.035em; }.post-row p { max-width: 45rem; margin: 0; color: #686a66; line-height: 1.65; }.post-meta { color: #62645f; font-size: .68rem; line-height: 1.6; }.post-row i { color: #2738d6; font-size: 1.1rem; font-style: normal; }.post-row:hover h2 { color: #2738d6; }
	.library{display:grid;grid-template-columns:1fr 1fr;gap:2rem;align-items:end;margin-top:7rem;padding-bottom:1.5rem;border-bottom:3px solid #17181b}.library h2{margin:.4rem 0;font-family:Georgia,'Times New Roman',serif;font-size:clamp(2rem,5vw,3.6rem);font-weight:500}.filters{display:grid;grid-template-columns:1.5fr 1fr;gap:.7rem}.filters label{display:grid;gap:.35rem;font-size:.75rem;font-weight:750}.filters input,.filters select{box-sizing:border-box;width:100%;min-height:3rem;padding:.7rem;border:1px solid #928f87;background:#fff;font:inherit}.archive{margin-top:1.5rem}.post-row small{display:block;margin-bottom:.6rem;color:#2738d6;font-size:.67rem;font-weight:800;text-transform:uppercase;letter-spacing:.08em}.no-results{padding:3rem;text-align:center;background:#ebe7de}.no-results button{padding:.7rem 1rem;border:0;background:#2738d6;color:#fff;font-weight:750}
	.playbook-end { display: flex; justify-content: space-between; gap: 2rem; align-items: center; margin-top: 5rem; padding-top: 1.3rem; border-top: 3px solid #17181b; }.playbook-end p { margin: 0; font-family: Georgia, 'Times New Roman', serif; font-size: 1.3rem; }.playbook-end a { display: inline-flex; gap: 2rem; color: #2738d6; font-weight: 760; text-decoration: none; }
	@media (max-width: 780px) { .playbook-head,.library { grid-template-columns: 1fr; gap: 2.5rem; }.featured { grid-template-columns: 1fr; }.feature-mark { min-height: 13rem; border-right: 0; border-bottom: 1px solid rgb(255 255 255 / .3); }.feature-mark i { bottom: -8rem; font-size: 20rem; }.featured > a { min-height: 25rem; }.topic-nav>div{grid-template-columns:repeat(2,minmax(0,1fr))}.post-row { grid-template-columns: 2rem 1fr 2rem; }.post-meta { grid-column: 2; grid-row: 2; }.post-row i { grid-column: 3; grid-row: 1; }.filters{grid-template-columns:1fr} }
	@media (max-width: 520px) { .featured { margin-inline: -1.25rem; }.archive { margin-top: 5rem; }.archive-head span { display: none; }.post-row { gap: .8rem; }.post-row p { display: none; }.playbook-end { align-items: flex-start; flex-direction: column; } }
</style>
