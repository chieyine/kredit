<script lang="ts">
 import DocumentLayout from './DocumentLayout.svelte';
 import type {WebsitePublication} from '$lib/website-content';
 let {publication}:{publication:WebsitePublication}=$props();
 const sections=$derived(publication.copy.sections.map((section,index)=>({id:`section-${index+1}`,title:section.heading})));
 function pieces(text:string){return text.split(/(https?:\/\/[^\s)]+|[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,})/gi).filter(Boolean).map(value=>({text:value,href:/^https?:\/\//i.test(value)?value:/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)?`mailto:${value}`:null}));}
</script>
<DocumentLayout title={publication.copy.title} description={publication.copy.introduction} version={publication.document_version!} effectiveDate={publication.effective_date!} {sections}>
 {#each publication.copy.sections as section,index}<section id={`section-${index+1}`}><h2>{section.heading}</h2>{#each section.body.split(/\n{2,}/) as paragraph}<p>{#each pieces(paragraph) as piece}{#if piece.href}<a href={piece.href} rel="noreferrer">{piece.text}</a>{:else}{piece.text}{/if}{/each}</p>{/each}</section>{/each}
</DocumentLayout>
<style>p{white-space:pre-line}</style>
