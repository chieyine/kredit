<script lang="ts">
	import { MutationIntent } from '$lib/api/mutation';
	import { page } from '$app/state';
	import { localTime } from '$lib/admin-client';
	import { checkedJSON, record, rows, text, LatestRequest, publicError } from '$lib/api/reliable';
	let item=$state<any>(null), timeline:any[]=$state([]), error=$state(''), notice=$state(''),loading=$state(true),busy=$state(false),nextState=$state('IN_PROGRESS'),note=$state('');
	const reads = new LatestRequest();
	let intent: MutationIntent | null = null;
	async function load(id: string) {
		const request = reads.begin(); loading=true; error='';
		try {
			const data = await checkedJSON(`/api/v1/ops/cases/${encodeURIComponent(id)}`, value => {
				const body=record(value), item=record(body.case);
				if(text(item.id)!==id) throw new Error('Case identity mismatch');
				text(item.state); text(item.subject_type); text(item.created_at);
				return { item, timeline: rows('timeline', value => { const event=record(value); text(event.action); text(event.created_at); return event; })(body) };
			}, { signal: request.signal });
			if (!request.current()) return;
			item=data.item; timeline=data.timeline;
			nextState=item.state==='OPEN'?'IN_PROGRESS':item.state==='IN_PROGRESS'?'RESOLVED':'CLOSED';
		} catch(cause) { if(request.current()) error=publicError(cause,'this case'); }
		finally { if(request.current()) loading=false; }
	}
	async function update(e:SubmitEvent) {
		e.preventDefault(); if(busy||!intent)return; busy=true; notice='';
		const id=page.params.id!;
		try {
			await intent.run({state:nextState,note}, value => {const body=record(value);if(text(record(body.case).id)!==id)throw new Error('Unconfirmed case update');return body;}, 'PATCH');
			if (page.params.id !== id) return;
			notice='Case updated. The note was added to its permanent history.'; note=''; await load(id);
		} catch(cause) { if(page.params.id!==id)return; notice=cause instanceof Error?cause.message:'The case update could not be confirmed.'; }
		finally { busy=false; }
	}
	$effect(() => { const id=page.params.id!; notice='';note='';intent=new MutationIntent('admin-case',`/api/v1/ops/cases/${encodeURIComponent(id)}`);void load(id);return()=>reads.cancel(); });
</script>
<svelte:head><title>Support case — Kredit</title></svelte:head>
<main class="shell workspace"><a href="/admin/cases">← All support cases</a><p class="eyebrow">Admin / Support case</p><h1>Support case</h1>{#if notice}<p class="notice" role="status">{notice}</p>{/if}{#if loading}<p>Loading case…</p>{:else if error}<p class="error" role="alert">{error}</p><button onclick={()=>load(page.params.id!)}>Try again</button>{:else}<section class="summary"><div><span>Status</span><strong>{item.state.replaceAll('_',' ')}</strong></div><div><span>Subject</span><strong>{item.subject_type.replaceAll('_',' ')}</strong></div><div><span>Reference</span><code>{item.subject_id}</code></div><div><span>Opened</span><strong>{localTime(item.created_at)}</strong></div></section><div class="columns"><section><h2>Case history</h2><ol>{#each timeline as event}<li><strong>{event.action.replaceAll('_',' ')}</strong><span>{localTime(event.created_at)}</span>{#if event.note}<p>{event.note}</p>{/if}</li>{/each}</ol></section>{#if item.state!=='CLOSED'}<section class="action"><h2>Move this case forward</h2><form onsubmit={update}><fieldset disabled={busy}><label>New status<select bind:value={nextState}><option value="OPEN">Open</option><option value="IN_PROGRESS">Being handled</option><option value="RESOLVED">Resolved</option><option value="CLOSED">Closed</option></select></label><label>What happened?<textarea bind:value={note} minlength="4" rows="5" required></textarea></label><button class="primary" disabled={busy}>{busy?'Saving…':'Save case update'}</button></fieldset></form></section>{/if}</div>{/if}</main>
<style>.action fieldset{display:grid;gap:.7rem;border:0;padding:0;margin:0;min-width:0}h1{font-family:var(--font-serif);font-size:clamp(2.5rem,6vw,4.5rem);font-weight:500}.summary{display:grid;grid-template-columns:repeat(auto-fit,minmax(12rem,1fr));gap:1rem;margin:1.5rem 0}.summary div,li{padding:1rem;border:1px solid var(--color-border);background:var(--color-surface)}.summary div{display:grid;gap:.3rem}.summary span,li span{color:var(--color-muted)}.columns{display:grid;grid-template-columns:1.3fr .7fr;gap:2rem}ol{display:grid;gap:.7rem;padding:0;list-style:none}li span{display:block;font-size:.9rem;margin-top:.25rem}.action{align-self:start;padding:1.2rem;background:#17181b;color:white}.action form,.action label{display:grid;gap:.7rem}.action label{gap:.3rem}.action select,.action textarea,.action button{padding:.7rem;border:1px solid #555;background:#242529;color:white;font:inherit}.action button{background:#ff6848;color:#17181b;font-weight:850}@media(max-width:760px){.columns{grid-template-columns:1fr}}</style>
