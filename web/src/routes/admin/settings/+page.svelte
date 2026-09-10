<script lang="ts">
 import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
 import {onMount} from 'svelte';
 import {adminGet,adminPost,localTime,lagosISO} from '$lib/admin-client';
 import {LatestRequest,record,rows,text} from '$lib/api/reliable';
 import {MutationIntent} from '$lib/api/mutation';
 const reads=new LatestRequest();
 const intents=new Map<string,MutationIntent>();
 function write(url:string,payload:unknown,decode:(value:unknown)=>unknown){let intent=intents.get(url);if(!intent){intent=new MutationIntent('admin-business-policy',url);intents.set(url,intent);}return intent.run(payload,decode);}
 function integer(value:unknown){if(typeof value!=='number'||!Number.isSafeInteger(value))throw new Error('Policy number was incomplete');return value;}
 function policyData(value:unknown):Data {
  const body=record(value),current=record(body.current);integer(current.revision);text(body.actor_id);
  const fields=rows('fields',value=>{const field=record(value);for(const key of ['key','label','group','kind','help'])text(field[key]);if(!['boolean','text','number','money'].includes(field.kind as string))throw new Error('Unknown setting type');integer(field.min);integer(field.max);return field as Field;})(body);
  if(new Set(fields.map(f=>f.key)).size!==fields.length)throw new Error('Duplicate settings');
  const validateValues=(value:unknown)=>{const values=record(value);for(const field of fields){const entry=values[field.key];if(field.kind==='boolean'){if(typeof entry!=='boolean')throw new Error('Policy switch was incomplete');}else if(field.kind==='text'){text(entry);}else{integer(entry);}}return values;};
  validateValues(current.values);record(body.deployment_limits);Object.values(record(body.actors)).forEach(text);
  for(const key of ['can_propose','can_approve'])if(body[key]!==undefined&&typeof body[key]!=='boolean')throw new Error('Policy permissions were incomplete');
  rows('changes',value=>{const change=record(value);for(const key of ['id','proposed_by','reason','effective_at','created_at','state'])text(change[key]);integer(change.revision);integer(change.base_revision);validateValues(change.values);validateValues(change.before_values);if(change.decided_by!==null)text(change.decided_by);return change;})(body);
  rows('events',value=>{const event=record(value);for(const key of ['change_id','actor_id','action','reason','occurred_at'])text(event[key]);return event;})(body);
  return body as Data;
 }
 function impactData(value:unknown){const body=record(value);integer(body.base_revision);text(body.note);if(!Array.isArray(body.effects))throw new Error('Policy effects were incomplete');body.effects.forEach(text);for(const count of Object.values(record(body.counts)))integer(count);return body;}
 import {formatKobo,parseNaira} from '$lib/money';
 type Values=Record<string,number|boolean|string>;
 type Field={key:string;label:string;group:string;kind:string;min:number;max:number;help:string};
 type Change={id:string;revision:number;base_revision:number;values:Values;before_values?:Values;proposed_by:string;reason:string;effective_at:string;created_at:string;state:string;decided_by:string|null};
 type Event={change_id:string;actor_id:string;action:string;reason:string;occurred_at:string};
 type Data={current:{revision:number;values:Values};fields:Field[];changes:Change[];events:Event[];actor_id:string;can_propose?:boolean;can_approve?:boolean;actors:Record<string,string>;deployment_limits:Values};
 let preview:any=$state(null);let units:Record<string,string>=$state({});
 let data:Data|null=$state(null),draft:Values=$state({}),reason=$state(''),effective=$state(''),notes:Record<string,string>=$state({});
 let loading=$state(true),busy=$state(false),error=$state(''),message=$state(''),proposalID=$state(crypto.randomUUID());
 let isPlatformOwner=$state(false),governanceMode=$state('unavailable');
 let changed=$derived.by(()=>data?data.fields.filter(f=>draft[f.key]!==data?.current.values[f.key]):[]);
 let blocked=$derived.by(()=>data?.changes.some(c=>c.state==='pending'||(c.state==='approved'&&new Date(c.effective_at).getTime()>Date.now()))??false);
 function decimal(value:Values[string]){const n=BigInt(Number(value));return `${n/100n}.${(n%100n).toString().padStart(2,'0')}`}
 function scaled(f:Field){return f.kind==='money'||f.key.endsWith('_bps')}
 function label(f:Field){return f.label.replace('(kobo)','(₦)').replace('(basis points)','(%)')}
 function actor(id:string){return id===data?.actor_id?'you':data?.actors?.[id]||'Administrator'}
 async function impact(values:Values=draft,base=data?.current.revision){if(busy)return;busy=true;error='';preview=null;try{preview=impactData(await adminPost('/api/v1/ops/business-policies/preview',{values,base_revision:base}))}catch(e){error=e instanceof Error?e.message:'Preview unavailable'}finally{busy=false}}
 function display(f:Field,value:Values[string]|undefined){if(value===undefined)return 'Unavailable';if(f.kind==='boolean')return value?'Enabled':'Disabled';if(f.kind==='money')return Number(value)===0?'No additional cap':formatKobo(Number(value));if(f.key.endsWith('_bps'))return `${Number(value)/100}%`;return String(value)||'No additional restriction'}
 function when(value:string){return localTime(value)}
 function status(c:Change){if(c.state!=='approved')return c.state;return c.revision===data?.current.revision?'active':new Date(c.effective_at).getTime()>Date.now()?'scheduled':'superseded'}
 function reset(){if(!data)return;draft={...data.current.values};units=Object.fromEntries(data.fields.filter(scaled).map(f=>[f.key,decimal(draft[f.key])]));preview=null;reason='';effective='';proposalID=crypto.randomUUID()}
 async function load(){const read=reads.begin();loading=true;error='';data=null;governanceMode='unavailable';isPlatformOwner=false;try{const [b,caps,gov]=await Promise.all([adminGet('/api/v1/ops/business-policies',read.signal),adminGet('/api/v1/ops/capabilities',read.signal),adminGet('/api/v1/ops/governance',read.signal)]);if(!read.current())return;if(!b.current?.values||!Array.isArray(b.fields)||!Array.isArray(b.changes)||!Array.isArray(b.events)||!['solo_owner','delegated_team'].includes(gov.governance?.mode))throw new Error('Settings and approval rules could not be verified.');if(!Array.isArray(caps.roles))throw new Error('Admin roles were incomplete');caps.roles.forEach(text);data=policyData(b);isPlatformOwner=caps.roles.includes('platform_owner');governanceMode=gov.governance.mode;reset()}catch(e){if(read.current())error=e instanceof Error?e.message:'Settings could not be loaded'}finally{if(read.current())loading=false}}
 async function soloApprove(c:Change){if(busy)return;busy=true;error='';message='';try{await write('/api/v1/ops/solo-owner/approve',{target_type:'policy',target_id:c.id,reason:notes[c.id]||'',confirm:true},value=>{if(record(value).approved!==true)throw new Error('Approval was not confirmed');return true;});await load();message='Approved. The change is recorded.'}catch(e){error=e instanceof Error?e.message:'That approval was not saved.'}finally{busy=false}}
 async function propose(){if(!data||busy)return;busy=true;error='';message='';try{for(const f of data.fields){if(f.kind==='number'||f.kind==='money'){const n=draft[f.key];if(typeof n!=='number'||!Number.isSafeInteger(n)||n<f.min||n>f.max)throw new Error(`Enter a whole number between ${f.min} and ${f.max} for ${label(f)}`)}}if(!effective)throw new Error('Choose an effective date');const expectedID=proposalID;await write('/api/v1/ops/business-policies',{id:expectedID,base_revision:data.current.revision,values:draft,reason,effective_at:lagosISO(effective)},value=>{if(record(value).id!==expectedID)throw new Error('Proposal was not confirmed');return true;});await load();message='Proposal saved.'+(governanceMode==='solo_owner'&&isPlatformOwner?' You can approve it yourself below.':' A second administrator must approve it before its effective date.')}catch(e){error=e instanceof Error?e.message:'Proposal could not be saved'}finally{busy=false}}
 async function decide(c:Change,action:string){if(busy)return;busy=true;error='';message='';try{await write(`/api/v1/ops/business-policies/${encodeURIComponent(c.id)}/decision`,{action,reason:notes[c.id]||''},value=>{if(record(value).status!=='recorded')throw new Error('Decision was not confirmed');return true;});await load();message='Decision recorded.'}catch(e){error=e instanceof Error?e.message:'Decision could not be saved'}finally{busy=false}}
 onMount(()=>{void load();return()=>reads.cancel();});
</script>
<svelte:head><title>Business settings — Kredit admin</title></svelte:head>
<main class="shell workspace">
 <p class="eyebrow">Administration / Business settings</p><h1>Business settings</h1><VerifyIdentity/>
 <p>Review the current policy, propose a change and set when it takes effect. {governanceMode === 'solo_owner' ? 'You approve your own proposals, with a fresh authenticator code and a written reason.' : governanceMode === 'delegated_team' ? 'A second administrator approves it before its effective date.' : 'Approval rules are not available yet.'}</p>
 <p class="notice">Existing offers keep the fee terms recorded with them. Provider approvals and deployment limits still apply. Reconciliation keeps running even when new collections are paused.</p>
 {#if error}<p role="alert" class="error">{error}</p>{/if}{#if message}<p role="status">{message}</p>{/if}
 <button onclick={load} disabled={loading||busy}>Refresh settings</button>
 {#if loading}<p>Loading settings…</p>{:else if data}
 <p>Current policy: <strong>{data.current.revision===0?'Initial deployment settings':`Revision ${data.current.revision}`}</strong>. Times are shown in Lagos time.</p>
 {#if blocked}<p class="notice">A change is still waiting for approval or for its effective date. Resolve or cancel it before you propose another.</p>{/if}
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
 <label for="reason">Reason for the change</label><textarea id="reason" bind:value={reason} required minlength="8" maxlength="2000" placeholder="Give the business reason, and the approval or evidence behind it"></textarea>
 <button type="submit" disabled={!changed.length||reason.trim().length<8||!effective}>Send for approval</button>
 </fieldset></form>
 <section aria-label="Policy impact"><h2>Impact preview</h2><button disabled={busy||!changed.length} onclick={()=>impact()}>Preview draft impact</button>{#if preview}<p>Preview against policy revision {preview.base_revision}.</p><p>{preview.note}</p><ul>{#each preview.effects as effect}<li>{effect}</li>{/each}</ul><dl>{#each Object.entries(preview.counts) as [key,value]}<dt>{key.replaceAll('_',' ')}</dt><dd>{String(value)}</dd>{/each}</dl>{/if}</section>
 <h2>Changes and decisions</h2><p><a href="/admin/history?kind=policy">Search and export the full change history</a></p>
 {#if !data.changes.length}<p>No changes have been proposed.</p>{/if}
 {#each data.changes as c (c.id)}<article>
 <h3>Revision {c.revision} · {status(c)}</h3><p>Effective: {when(c.effective_at)}</p><p>{c.reason}</p><p>Proposed by {actor(c.proposed_by)}{#if c.decided_by} · Decision by {actor(c.decided_by)}{/if}</p>
 <button disabled={busy} onclick={()=>impact(c.values,c.base_revision)}>Preview this proposal’s impact</button><details open={c.state==='pending'}><summary>Review proposed values</summary><div class="table-wrap"><table><thead><tr><th>Setting</th><th>Previous</th><th>Proposed</th></tr></thead><tbody>{#each data.fields as f}<tr class:changed={c.values[f.key]!==(c.before_values??data.current.values)[f.key]}><th>{label(f)}</th><td>{display(f,(c.before_values??data.current.values)[f.key])}</td><td>{display(f,c.values[f.key])}</td></tr>{/each}</tbody></table></div></details>
 {#each data.events.filter(e=>e.change_id===c.id) as event}<p class="history">{when(event.occurred_at)} · {actor(event.actor_id)} · {event.action}: {event.reason}</p>{/each}
 {#if c.state==='pending'||status(c)==='scheduled'}
 <label for={`decision-${c.id}`}>Decision notes</label><textarea disabled={busy} id={`decision-${c.id}`} bind:value={notes[c.id]} minlength="8" maxlength="2000"></textarea>
 {#if c.state==='pending'&&c.proposed_by!==data.actor_id&&data.can_approve!==false}<button disabled={busy||(notes[c.id]||'').trim().length<8} onclick={()=>decide(c,'approve')}>Approve exactly this</button><button disabled={busy||(notes[c.id]||'').trim().length<8} onclick={()=>decide(c,'reject')}>Reject proposal</button>{/if}
 {#if c.state==='pending'&&c.proposed_by===data.actor_id}
  {#if isPlatformOwner&&governanceMode==='solo_owner'}
   <button class="primary" disabled={busy||(notes[c.id]||'').trim().length<8} onclick={()=>soloApprove(c)}>Approve my own change</button>
  {:else}
   <p>Another platform administrator must approve your proposal.</p>
  {/if}
 {/if}
 <button disabled={busy||(notes[c.id]||'').trim().length<8} onclick={()=>decide(c,'cancel')}>Cancel change</button>
 {/if}</article>{/each}
 <details><summary>Protected deployment controls</summary><p>The platform owner manages provider connections, credentials and launch approval references in Platform settings. Runtime connection changes apply after the API and worker restart. Infrastructure, retention, currency and accounting safeguards require their separate deployment or review process. Large corrections and accepted-schedule amendments each have their own workflow.</p><a href="/admin/platform-settings">Open platform settings →</a></details>
 {/if}
</main>
<style>
 main{max-width:1100px;margin:auto;padding:2rem 1rem}fieldset,article{border:1px solid var(--color-border,#ccc);border-radius:1rem;padding:1.25rem;margin:1.5rem 0}legend,h3{font-weight:700}label{display:block;font-weight:600;margin:.8rem 0 .35rem}input,select,textarea{width:100%;max-width:36rem;padding:.7rem;border:1px solid #bbb;border-radius:.4rem;background:white;color:#222}textarea{display:block;min-height:5rem}small{display:block;max-width:48rem;margin:.4rem 0;color:#555}.setting{padding:.5rem 0;border-bottom:1px solid #ddd}.current,.history{font-size:.9rem;color:#555}.notice{padding:1rem;background:#e7e6dc;border-radius:.5rem}button{padding:.7rem 1rem;margin:.7rem .5rem .4rem 0;border:1px solid #999;border-radius:.5rem}button:disabled{opacity:.5}.table-wrap{overflow:auto}table{border-collapse:collapse;width:100%;margin:1rem 0}th,td{text-align:left;padding:.7rem;border-bottom:1px solid #ddd}.changed{background:#fff7da}.error{color:#9c2525}article p{overflow-wrap:anywhere}
</style>
