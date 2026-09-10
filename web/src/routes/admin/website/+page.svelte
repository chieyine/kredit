<script lang="ts">
 import { articles, articleCategories } from '$lib/blog/articles';
 import { onMount } from 'svelte';
 import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
 import { adminGet, adminPost, localTime } from '$lib/admin-client';
 import { idempotencyKey } from '$lib/api/client';
 import { decodeWebsiteCopy, decodeWebsiteRecord, isLegalPage, isGuidePage, websiteDefaults, websitePages, type WebsitePage, type WebsiteCopy, type WebsiteRecord } from '$lib/website-editor-content';
 function cloneCopy(value:WebsiteCopy):WebsiteCopy{return JSON.parse(JSON.stringify(value));}
 let selected = $state<WebsitePage>('home');
 let current = $state<WebsiteRecord | null>(null);
 let copy = $state<WebsiteCopy>(cloneCopy(websiteDefaults.home));
 let loading = $state(true), busy = $state(false), error = $state(''), message = $state(''), reason = $state(''), preview = $state(false);
 let history = $state<{version:number;recorded_at:string;reason:string;copy:WebsiteCopy}[]>([]), historyError=$state('');
 let requestIdentity={body:'',key:''};
 let controller: AbortController | undefined;
 const savedCopy = $derived(current?.content?.draft ?? websiteDefaults[selected]);
 const dirty = $derived(JSON.stringify(copy)!==JSON.stringify(savedCopy));
 async function load() {
  controller?.abort(); controller=new AbortController(); const signal=controller.signal;
  loading=true; error=''; message=''; current=null; history=[]; historyError=''; preview=false;
  try {
   const value=decodeWebsiteRecord(await adminGet(`/api/v1/ops/website/${selected}`,signal));
   if(signal.aborted)return;
   current=value; copy=cloneCopy(value.content?.draft ?? websiteDefaults[selected]);
   try {
    const result=await adminGet(`/api/v1/ops/platform-settings/history?key=website.${selected}`,signal);
    if (!Array.isArray(result.history)) throw new Error('History was incomplete.');
    const rows=result.history.map((item:any)=>{
     if(item.key!==`website.${selected}` || !Number.isSafeInteger(item.version) || item.version<1 || typeof item.recorded_at!=='string' || !Number.isFinite(Date.parse(item.recorded_at)) || typeof item.reason!=='string')throw new Error('History was incomplete.');
     return {version:item.version,recorded_at:item.recorded_at,reason:item.reason,copy:decodeWebsiteCopy(item.new_value?.draft)};
    });
    if(!signal.aborted)history=rows;
   } catch(e) { if(!signal.aborted)historyError='Saved history could not be opened. Reload to try again.'; }
  } catch(e) { if(!signal.aborted)error=e instanceof Error?e.message:'The editor could not be opened.'; }
  finally { if(!signal.aborted)loading=false; }
 }
 async function save(action:'save'|'publish') {
  if(!current || busy || loading)return;
  busy=true; error=''; message='';
  try {
   decodeWebsiteCopy(copy);
   const payload={action,copy,expected_version:current.version,reason:reason.trim()};
   const body=JSON.stringify(payload);
   if(requestIdentity.body!==body)requestIdentity={body,key:idempotencyKey()};
   current=decodeWebsiteRecord(await adminPost(`/api/v1/ops/website/${selected}`,payload,'POST',requestIdentity.key));
   copy=cloneCopy(current.content!.draft); requestIdentity={body:'',key:''}; preview=false; reason='';
   message=action==='publish'?'Published. The public page now uses this copy.':'Draft saved. Preview it before publishing.';
  }catch(e){error=e instanceof Error?e.message:'The result could not be confirmed. Reload to check the saved version.';}
  finally{busy=false;}
 }
 function restore(item:{version:number;copy:WebsiteCopy}) {
  copy=cloneCopy(item.copy); reason=`Restore content from saved version ${item.version}`; preview=false;
  message='Earlier copy loaded into the editor. Save it as a new draft, then preview and publish.';
 }
 onMount(()=>{void load();return()=>controller?.abort();});
</script>
<svelte:head><title>Website content · Kredit admin</title><meta name="robots" content="noindex,nofollow"/></svelte:head>
<main class="editor">
 <header><p class="eyebrow">Website</p><h1>Make the words yours.</h1><p>Save a draft, preview the copy, then publish when it is ready.</p></header>
 <VerifyIdentity/>
 <div class="toolbar"><label>Page<select bind:value={selected} disabled={busy||loading||dirty} onchange={()=>load()}>{#each Object.entries(websitePages) as [key,label]}<option value={key}>{label}</option>{/each}</select></label><button type="button" disabled={busy||loading} onclick={()=>load()}>Reload saved copy</button></div>
 {#if loading}<p role="status">Opening the editor…</p>{/if}
 {#if error}<p class="notice error" role="alert">{error}</p>{/if}
 {#if message}<p class="notice" role="status">{message}</p>{/if}
 {#if current && !loading}
  {#if isLegalPage(selected) && current.content?.published?.document_version}<p><a href={`/legal/${selected}?version=${encodeURIComponent(current.content.published.document_version)}`} target="_blank" rel="noreferrer">Read published document · {current.content.published.document_version}</a></p>{/if}
  <p class="version">Draft version {current.version}{#if current.content?.published} · Published {localTime(current.content.published.published_at)} · Publication {current.content.published.version}{:else} · Public page uses the initial release copy{/if}</p>
  {#if isLegalPage(selected)}<p class="notice">Publishing takes effect today and creates a permanent document version. Earlier accepted records keep the document version they referenced.</p>{/if}
  <form onsubmit={event=>{event.preventDefault();void save('save')}}>
   <fieldset disabled={busy}>
    <label>Title<input bind:value={copy.title} required maxlength="160"/></label>
    {#if !isLegalPage(selected)}<label>Second line<input bind:value={copy.accent} maxlength="160"/></label>{/if}
    <label>Introduction<textarea bind:value={copy.introduction} required maxlength="2000" rows="4"></textarea></label>
    {#if selected==='faq'||isLegalPage(selected)||isGuidePage(selected)}
     {#each copy.sections as section,i}
      <section class="question"><label>{selected==='faq'?'Question':'Section heading'} {i+1}<input bind:value={section.heading} required maxlength="200"/></label><label>{selected==='faq'?'Answer':'Section text'}<textarea bind:value={section.body} required maxlength="6000" rows="5"></textarea></label>{#if isGuidePage(selected)}<label>Bullet points, one per line<textarea rows="4" value={section.points?.join("\n") ?? ""} oninput={event=>{section.points=event.currentTarget.value.split("\n").filter(s=>s.trim())}}></textarea></label>{/if}<button type="button" onclick={()=>{copy.sections=copy.sections.filter((_,index)=>index!==i);preview=false}}>Remove section {i+1}</button></section>
     {/each}
     <button type="button" disabled={copy.sections.length>=40} onclick={()=>{copy.sections=[...copy.sections,{heading:'',body:''}];preview=false}}>{selected==='faq'?'Add a question':'Add a section'}</button>
    {/if}
    {#if copy.guide}
     <label>Search description<textarea bind:value={copy.guide.description} required maxlength="2000"></textarea></label>
     <label>Topic<select bind:value={copy.guide.category}>{#each articleCategories as category}<option>{category}</option>{/each}</select></label>
     <label>Main search phrase<input bind:value={copy.guide.keyphrase} required maxlength="200"/></label>
     <h2>Questions and answers</h2>
     {#each copy.guide.faq as q,i}<section class="question"><label>Question {i+1}<input bind:value={q.heading} maxlength="200" required/></label><label>Answer<textarea bind:value={q.body} maxlength="6000" required rows="4"></textarea></label><button type="button" onclick={()=>{copy.guide!.faq=copy.guide!.faq.filter((_,j)=>j!==i)}}>Remove question {i+1}</button></section>{/each}
     <button type="button" disabled={copy.guide.faq.length>=40} onclick={()=>{copy.guide!.faq=[...copy.guide!.faq,{heading:'',body:''}]}}>Add guide question</button>
     <h2>Further reading</h2>
     {#each copy.guide.sources as source,i}<section class="question"><label>Source name<input bind:value={source.name} maxlength="200" required/></label><label>Source HTTPS address<input type="url" bind:value={source.url} maxlength="2000" required/></label><label>Source note<textarea bind:value={source.note} maxlength="2000"></textarea></label><button type="button" onclick={()=>{copy.guide!.sources=copy.guide!.sources.filter((_,j)=>j!==i)}}>Remove source {i+1}</button></section>{/each}
     <button type="button" disabled={copy.guide.sources.length>=40} onclick={()=>{copy.guide!.sources=[...copy.guide!.sources,{name:'',url:'',note:''}]}}>Add source</button>
     <label>Related guides<select multiple bind:value={copy.guide.related}>{#each articles.filter(a=>'guide-'+a.slug!==selected) as article}<option value={article.slug}>{article.title}</option>{/each}</select></label>
    {/if}
    {#if copy.contact}
     <label>Help email<input type="email" bind:value={copy.contact.support_email} required maxlength="254"/></label>
     <label>Privacy email<input type="email" bind:value={copy.contact.privacy_email} required maxlength="254"/></label>
     <label>Contact phone, optional<input type="tel" bind:value={copy.contact.phone} maxlength="40"/></label>
     <label>Contact address<textarea bind:value={copy.contact.address} required maxlength="2000"></textarea></label>
    {/if}
    <label>Reason for this change<input bind:value={reason} required minlength="4" maxlength="500" placeholder="For example, clarify how delivery proof works"/></label>
    <div class="actions"><button type="submit" disabled={reason.trim().length<4}>Save draft</button><button type="button" disabled={dirty||current.version===0} onclick={()=>preview=!preview}>Preview saved copy</button></div>
   </fieldset>
  </form>
  {#if dirty}<p>There are unsaved edits. Save the draft before switching pages or publishing.</p>{/if}
  {#if preview && !dirty}
   <section class="preview" aria-label="Saved copy preview"><p class="eyebrow">Copy preview · private</p><h2>{copy.title} <em>{copy.accent}</em></h2><p>{copy.introduction}</p>{#each copy.sections as section}<h3>{section.heading}</h3><p>{section.body}</p>{#if section.points}<ul>{#each section.points as point}<li>{point}</li>{/each}</ul>{/if}{/each}{#if copy.guide}<p>{copy.guide.description} · {copy.guide.category}</p>{#each copy.guide.faq as q}<h3>{q.heading}</h3><p>{q.body}</p>{/each}{#each copy.guide.sources as source}<p>{source.name} — {source.url}<br/>{source.note}</p>{/each}{/if}{#if copy.contact}<p>{copy.contact.support_email}<br/>{copy.contact.privacy_email}<br/>{copy.contact.phone}<br/>{copy.contact.address}</p>{/if}</section>
   <p>Publishing replaces this page’s supported text. Its links, layout and fee calculations keep their existing behaviour.</p><button class="publish" disabled={busy||reason.trim().length<4} onclick={()=>save('publish')}>Publish reviewed copy</button><p class="hint">Enter a publication reason above before publishing.</p>
  {/if}
  <section class="history"><h2>Earlier saved copy</h2><p>Restore an earlier version into a new draft. Published history is preserved.</p>{#if historyError}<p role="alert">{historyError}</p>{/if}{#each history as item}<article><div><strong>Version {item.version}</strong><p>{localTime(item.recorded_at)} · {item.reason}</p></div><button disabled={busy||dirty} onclick={()=>restore(item)}>Use this copy</button></article>{/each}{#if !history.length && !historyError}<p>No earlier saved copy.</p>{/if}</section>
 {/if}
</main>
<style>
 .editor{max-width:72rem;margin:auto;padding:3rem 1.5rem;color:#202220}header{max-width:48rem}h1{font-size:clamp(2.2rem,5vw,4rem);line-height:1.05;margin:.5rem 0 1rem}h2{font-size:1.8rem}p{line-height:1.65}label{display:grid;gap:.5rem;font-weight:600;margin:1rem 0}input,textarea,select{box-sizing:border-box;max-width:100%;width:100%;padding:.8rem;border:1px solid #8a8b80;border-radius:.25rem;background:#fff;font:inherit;color:inherit}textarea{resize:vertical}button{padding:.8rem 1rem;border:1px solid #202220;border-radius:.25rem;background:#fff;color:#202220;font:inherit;cursor:pointer}button:disabled{opacity:.55;cursor:not-allowed}button:focus-visible,input:focus-visible,textarea:focus-visible,select:focus-visible{outline:3px solid #303bdd;outline-offset:3px}.toolbar,.actions{display:flex;align-items:center;gap:1rem;flex-wrap:wrap}.toolbar label{min-width:16rem}.version,.hint{font-size:.9rem}fieldset{padding:0;border:0}.question{border-top:1px solid #c3c0b5;margin:2rem 0;padding-top:1rem}.notice{padding:1rem;border-left:3px solid #303bdd;background:#fff}.error{border-color:#a32716}.preview{margin-top:2rem;padding:clamp(1.2rem,4vw,3rem);background:#fff;border:1px solid #c3c0b5;overflow-wrap:anywhere}.preview p{white-space:pre-wrap;max-width:68ch}.preview h2{font-size:clamp(2rem,4vw,3rem)}.publish,.actions button:first-child{background:#2935ce;color:white;border-color:#2935ce}.history{border-top:1px solid #b6b3a9;margin-top:3rem;padding-top:2rem}.history article{display:flex;gap:1rem;justify-content:space-between;align-items:center;border-bottom:1px solid #d1cfc5;padding:1rem 0}.history article p{margin:.3rem 0}@media(max-width:600px){.editor{padding:2rem 1rem}.history article{align-items:flex-start;flex-direction:column}.toolbar>*{width:100%}}
</style>
