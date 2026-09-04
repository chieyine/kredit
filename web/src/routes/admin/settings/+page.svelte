<script lang="ts">
 import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
 import {onMount} from 'svelte';
 import {csrfHeaders,idempotencyKey} from '$lib/api/client';
 import {formatKobo,parseNaira} from '$lib/money';
 type Values=Record<string,number|boolean|string>;
 type Field={key:string;label:string;group:string;kind:string;min:number;max:number;help:string};
 type Change={id:string;revision:number;base_revision:number;values:Values;before_values?:Values;proposed_by:string;reason:string;effective_at:string;created_at:string;state:string;decided_by:string|null};
 type Event={change_id:string;actor_id:string;action:string;reason:string;occurred_at:string};
 type Data={current:{revision:number;values:Values};fields:Field[];changes:Change[];events:Event[];actor_id:string;can_propose?:boolean;can_approve?:boolean;actors:Record<string,string>;deployment_limits:Values};
 let preview:any=$state(null);let units:Record<string,string>=$state({});
 let data:Data|null=$state(null),draft:Values=$state({}),reason=$state(''),effective=$state(''),notes:Record<string,string>=$state({});
 let loading=$state(true),busy=$state(false),error=$state(''),message=$state(''),proposalID=$state(crypto.randomUUID());
 let changed=$derived.by(()=>data?data.fields.filter(f=>draft[f.key]!==data?.current.values[f.key]):[]);
 let blocked=$derived.by(()=>data?.changes.some(c=>c.state==='pending'||(c.state==='approved'&&new Date(c.effective_at).getTime()>Date.now()))??false);
 function decimal(value:Values[string]){const n=BigInt(Number(value));return `${n/100n}.${(n%100n).toString().padStart(2,'0')}`}
 function scaled(f:Field){return f.kind==='money'||f.key.endsWith('_bps')}
 function label(f:Field){return f.label.replace('(kobo)','(₦)').replace('(basis points)','(%)')}
 function actor(id:string){return id===data?.actor_id?'you':data?.actors?.[id]||'Administrator'}
 async function impact(values:Values=draft,base=data?.current.revision){busy=true;error='';try{const r=await fetch('/api/v1/ops/business-policies/preview',{method:'POST',credentials:'include',headers:{'Content-Type':'application/json','Idempotency-Key':idempotencyKey(),...csrfHeaders()},body:JSON.stringify({values,base_revision:base})});const b=await r.json();if(!r.ok)throw new Error(b.detail||'Preview unavailable');preview=b}catch(e){error=e instanceof Error?e.message:'Preview unavailable'}finally{busy=false}}
 function display(f:Field,value:Values[string]|undefined){if(value===undefined)return 'Unavailable';if(f.kind==='boolean')return value?'Enabled':'Disabled';if(f.kind==='money')return Number(value)===0?'No additional cap':formatKobo(Number(value));if(f.key.endsWith('_bps'))return `${Number(value)/100}%`;return String(value)||'No additional restriction'}
 function when(value:string){return new Intl.DateTimeFormat('en-NG',{dateStyle:'medium',timeStyle:'short',timeZone:'Africa/Lagos'}).format(new Date(value))}
 function status(c:Change){if(c.state!=='approved')return c.state;return c.revision===data?.current.revision?'active':new Date(c.effective_at).getTime()>Date.now()?'scheduled':'superseded'}
 function reset(){if(!data)return;draft={...data.current.values};units=Object.fromEntries(data.fields.filter(scaled).map(f=>[f.key,decimal(draft[f.key])]));preview=null;reason='';effective='';proposalID=crypto.randomUUID()}
 async function load(){loading=true;error='';try{const r=await fetch('/api/v1/ops/business-policies',{credentials:'include'});const b=await r.json();if(!r.ok)throw new Error(b.detail||'Settings could not be loaded');data=b;reset()}catch(e){error=e instanceof Error?e.message:'Settings could not be loaded'}finally{loading=false}}
 async function post(path:string,body:unknown){const r=await fetch(path,{method:'POST',credentials:'include',headers:{'Content-Type':'application/json','Idempotency-Key':idempotencyKey(),...csrfHeaders()},body:JSON.stringify(body)});const b=await r.json();if(!r.ok)throw new Error(b.detail||'Change could not be saved')}
 async function propose(){if(!data)return;busy=true;error='';message='';try{for(const f of data.fields){if(f.kind==='number'||f.kind==='money'){const n=draft[f.key];if(typeof n!=='number'||!Number.isSafeInteger(n)||n<f.min||n>f.max)throw new Error(`Enter a whole number between ${f.min} and ${f.max} for ${label(f)}`)}}if(!effective)throw new Error('Choose an effective date');await post('/api/v1/ops/business-policies',{id:proposalID,base_revision:data.current.revision,values:draft,reason,effective_at:new Date(`${effective}:00+01:00`).toISOString()});await load();message='Proposal saved. Another platform administrator must approve it before its effective date.'}catch(e){error=e instanceof Error?e.message:'Proposal could not be saved'}finally{busy=false}}
 async function decide(c:Change,action:string){busy=true;error='';message='';try{await post(`/api/v1/ops/business-policies/${c.id}/decision`,{action,reason:notes[c.id]||''});await load();message='Decision recorded.'}catch(e){error=e instanceof Error?e.message:'Decision could not be saved'}finally{busy=false}}
 onMount(load);
</script>
<svelte:head><title>Business settings — Kredit admin</title></svelte:head>
<main class="shell workspace">
 <p class="eyebrow">Administration / Business settings</p><h1>Business settings</h1><VerifyIdentity/>
 <p>Review the current policy, propose changes, and schedule when they take effect. Every change needs approval from another platform administrator.</p>
 <p class="notice">Existing offers retain their recorded fee terms. Provider approvals and deployment limits still apply. Reconciliation continues when new collections are paused.</p>
 {#if error}<p role="alert" class="error">{error}</p>{/if}{#if message}<p role="status">{message}</p>{/if}
 <button onclick={load} disabled={loading||busy}>Refresh settings</button>
 {#if loading}<p>Loading settings…</p>{:else if data}
 <p>Current policy: <strong>{data.current.revision===0?'Initial deployment settings':`Revision ${data.current.revision}`}</strong>. Times are shown in Lagos time.</p>
 {#if blocked}<p class="notice">A change is awaiting approval or its effective date. Resolve or cancel it before proposing another change.</p>{/if}
 <form onsubmit={(e)=>{e.preventDefault();propose()}}>
 {#each ['Collections','Limits','Fees','Notices'] as group}
 <fieldset disabled={busy||blocked||data.can_propose===false}><legend>{group}</legend>
 {#each data.fields.filter(f=>f.group===group) as f}
 <div class="setting">
 <label for={f.key}>{label(f)}</label>
 {#if f.kind==='boolean'}<select id={f.key} value={String(draft[f.key])} onchange={(e)=>{draft[f.key]=e.currentTarget.value==='true';preview=null}}><option value="true">Enabled</option><option value="false">Disabled</option></select>
 {:else if f.kind==='text'}<input id={f.key} type="text" maxlength="2000" value={String(draft[f.key])} oninput={(e)=>{draft[f.key]=e.currentTarget.value;preview=null}}/>
 {:else if scaled(f)}<input id={f.key} type="text" inputmode="decimal" required value={units[f.key]} oninput={(e)=>{units[f.key]=e.currentTarget.value;draft[f.key]=parseNaira(e.currentTarget.value);preview=null}}/><small>{f.kind==='money'?'Enter naira, with up to two decimal places.':'Enter a percentage, with up to two decimal places.'}</small>
 {:else}<input id={f.key} type="number" min={f.min} max={f.max} step="1" required value={Number(draft[f.key])} oninput={(e)=>{draft[f.key]=e.currentTarget.value===''?NaN:Number(e.currentTarget.value);preview=null}}/>{/if}
 <small>{f.help}</small><p class="current">Current: {display(f,data.current.values[f.key])}</p>
 </div>{/each}
 </fieldset>{/each}
 <fieldset disabled={busy||blocked||data.can_propose===false}><legend>Review your proposal</legend>
 {#if changed.length}<ul>{#each changed as f}<li><strong>{label(f)}:</strong> {display(f,data.current.values[f.key])} → {display(f,draft[f.key])}</li>{/each}</ul>{:else}<p>No changes selected.</p>{/if}
 <label for="effective">Effective date and time (Lagos)</label><input id="effective" type="datetime-local" bind:value={effective} required/>
 <label for="reason">Reason for the change</label><textarea id="reason" bind:value={reason} required minlength="8" maxlength="2000" placeholder="Explain the business reason and supporting approval or evidence"></textarea>
 <button type="submit" disabled={!changed.length||reason.trim().length<8||!effective}>Submit for independent approval</button>
 </fieldset></form>
 <section aria-label="Policy impact"><h2>Impact preview</h2><button disabled={busy||!changed.length} onclick={()=>impact()}>Preview draft impact</button>{#if preview}<p>Preview against policy revision {preview.base_revision}.</p><p>{preview.note}</p><ul>{#each preview.effects as effect}<li>{effect}</li>{/each}</ul><dl>{#each Object.entries(preview.counts) as [key,value]}<dt>{key.replaceAll('_',' ')}</dt><dd>{String(value)}</dd>{/each}</dl>{/if}</section>
 <h2>Changes and decisions</h2><p><a href="/admin/history?kind=policy">Search and export the complete change history</a></p>
 {#if !data.changes.length}<p>No changes have been proposed.</p>{/if}
 {#each data.changes as c (c.id)}<article>
 <h3>Revision {c.revision} · {status(c)}</h3><p>Effective: {when(c.effective_at)}</p><p>{c.reason}</p><p>Proposed by {actor(c.proposed_by)}{#if c.decided_by} · Decision by {actor(c.decided_by)}{/if}</p>
 <button disabled={busy} onclick={()=>impact(c.values,c.base_revision)}>Preview this proposal’s impact</button><details open={c.state==='pending'}><summary>Review proposed values</summary><div class="table-wrap"><table><thead><tr><th>Setting</th><th>Previous</th><th>Proposed</th></tr></thead><tbody>{#each data.fields as f}<tr class:changed={c.values[f.key]!==(c.before_values??data.current.values)[f.key]}><th>{label(f)}</th><td>{display(f,(c.before_values??data.current.values)[f.key])}</td><td>{display(f,c.values[f.key])}</td></tr>{/each}</tbody></table></div></details>
 {#each data.events.filter(e=>e.change_id===c.id) as event}<p class="history">{when(event.occurred_at)} · {actor(event.actor_id)} · {event.action}: {event.reason}</p>{/each}
 {#if c.state==='pending'||status(c)==='scheduled'}
 <label for={`decision-${c.id}`}>Decision notes</label><textarea id={`decision-${c.id}`} bind:value={notes[c.id]} minlength="8" maxlength="2000"></textarea>
 {#if c.state==='pending'&&c.proposed_by!==data.actor_id&&data.can_approve!==false}<button disabled={busy||(notes[c.id]||'').trim().length<8} onclick={()=>decide(c,'approve')}>Approve exact proposal</button><button disabled={busy||(notes[c.id]||'').trim().length<8} onclick={()=>decide(c,'reject')}>Reject proposal</button>{/if}
 {#if c.state==='pending'&&c.proposed_by===data.actor_id}<p>Another platform administrator must approve your proposal.</p>{/if}
 <button disabled={busy||(notes[c.id]||'').trim().length<8} onclick={()=>decide(c,'cancel')}>Cancel change</button>
 {/if}</article>{/each}
 <details><summary>Protected deployment controls</summary><p>Provider connections, credentials, certification evidence, live-money enablement, identity integrations, retention approval, currency, and accounting safeguards are managed through deployment and approval processes. Large corrections and accepted-schedule amendments require their separate supported workflows.</p></details>
 {/if}
</main>
<style>
 main{max-width:1100px;margin:auto;padding:2rem 1rem}fieldset,article{border:1px solid var(--color-border,#ccc);border-radius:1rem;padding:1.25rem;margin:1.5rem 0}legend,h3{font-weight:700}label{display:block;font-weight:600;margin:.8rem 0 .35rem}input,select,textarea{width:100%;max-width:36rem;padding:.7rem;border:1px solid #bbb;border-radius:.4rem;background:white;color:#222}textarea{display:block;min-height:5rem}small{display:block;max-width:48rem;margin:.4rem 0;color:#555}.setting{padding:.5rem 0;border-bottom:1px solid #ddd}.current,.history{font-size:.9rem;color:#555}.notice{padding:1rem;background:#e7e6dc;border-radius:.5rem}button{padding:.7rem 1rem;margin:.7rem .5rem .4rem 0;border:1px solid #999;border-radius:.5rem}button:disabled{opacity:.5}.table-wrap{overflow:auto}table{border-collapse:collapse;width:100%;margin:1rem 0}th,td{text-align:left;padding:.7rem;border-bottom:1px solid #ddd}.changed{background:#fff7da}.error{color:#9c2525}article p{overflow-wrap:anywhere}
</style>
