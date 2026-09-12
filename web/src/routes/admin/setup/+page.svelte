<script lang="ts">
 import { onMount } from 'svelte';
 import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
 import { adminGet, localTime } from '$lib/admin-client';
 import { record, text, publicError } from '$lib/api/reliable';
 type Gate={name:string;passed:boolean;detail:string};
 type Task={id:string;label:string;state:string;detail:string;url:string};
 type Process={process:string;versions:Record<string,number>;state:string;updated_at:string;fresh:boolean};
 let gates=$state<Gate[]>([]);
 let tasks=$state<Task[]>([]),processes=$state<Process[]>([]),versions=$state<Record<string,number>>({});
 let loading=$state(true),error=$state(''),autoApply=$state(false),checkedAt=$state(''),failures=$state(0);
 let busy=false;
 const gateLabels:Record<string,string>={security_review:'Security review reference',dpia:'Data protection review reference',legal_approval:'Legal approval reference',penetration_test:'Security test evidence',backup_restore:'Backup recovery evidence',provider_certification:'Provider certification reference',support_training:'Support training evidence',launch_approval:'Launch approval reference',pilot_provider_accounts:'Allowed provider accounts',pilot_industries:'Allowed industries',real_identity_provider:'Identity provider enabled',real_collection_provider:'Collection provider enabled',pilot_max_supplier_organizations:'Maximum seller businesses',pilot_max_buyer_businesses:'Maximum buyer businesses',pilot_max_principal_kobo:'Maximum sale amount',pilot_max_active_exposure_kobo:'Maximum outstanding credit',pilot_max_drawdowns_per_line_day:'Daily purchases per credit limit',pilot_max_collection_retries:'Collection retry limit',pilot_enhanced_review_kobo:'Extra review threshold'};
 const labels:Record<string,string>={needs_setup:'Needs setup',configured_unverified:'Configured · delivery or operation still needs confirmation',review_required:'Needs your review'};
 function applied(item:Process){return item.fresh&&item.state==='current'&&Object.entries(versions).every(([key,version])=>item.versions[key]===version)&&Object.keys(item.versions).every(key=>versions[key]===item.versions[key])}
 async function load(){if(busy)return;busy=true;try{
  const data=record(await adminGet('/api/v1/ops/setup'));
  if(!Array.isArray(data.tasks)||!Array.isArray(data.processes)||typeof data.auto_apply!=='boolean')throw new Error('Setup information could not be verified.');
  tasks=data.tasks.map(value=>{const item=record(value);const url=text(item.url);if(!url.startsWith('/admin/'))throw new Error('Invalid setup link');return{id:text(item.id),label:text(item.label),state:text(item.state),detail:text(item.detail),url}});
  const saved=record(data.saved_versions);for(const value of Object.values(saved))if(!Number.isSafeInteger(value)||Number(value)<1)throw new Error('Saved versions could not be verified.');versions=saved as Record<string,number>;
  processes=data.processes.map(value=>{const item=record(value);return{process:text(item.process),versions:record(item.versions) as Record<string,number>,state:text(item.state),updated_at:text(item.updated_at),fresh:item.fresh===true}});
  if(!Array.isArray(data.configuration_gates))throw new Error('Launch requirements could not be verified.');gates=data.configuration_gates.map(value=>{const item=record(value);if(typeof item.passed!=='boolean')throw new Error('Invalid launch requirement');return{name:text(item.name),passed:item.passed,detail:text(item.detail)}});
  autoApply=data.auto_apply;checkedAt=text(data.checked_at);failures=Number(data.provider_initialization_failures);error='';
 }catch(cause){error=publicError(cause,'super-admin setup')}finally{busy=false;loading=false}}
 onMount(()=>{void load();const timer=setInterval(()=>void load(),15000);return()=>clearInterval(timer)});
</script>
<svelte:head><title>Launch setup — Kredit</title></svelte:head>
<main class="shell workspace"><h1>Launch setup</h1><p>Complete your provider connections, business details and launch evidence here. Credentials are stored securely and are never shown after saving.</p>
<VerifyIdentity />
<button onclick={load} disabled={loading}>Refresh status</button>
{#if error}<p role="alert">{error}</p>{/if}
{#if loading}<p role="status">Loading setup…</p>{:else if !error}
<section class="service-status"><h2>Applying your settings</h2><p>{autoApply?'Saved service settings apply automatically. The API and background worker restart safely to pick them up.':'This deployment needs automatic configuration updates enabled. Until then, saved service settings require an operator restart.'}</p>
{#each ['api','worker'] as name}{@const item=processes.find(value=>value.process===name)}<p><strong>{name==='api'?'Website service':'Background worker'}:</strong> {item?(applied(item)?'Latest settings applied':!item.fresh?'Status is out of date':item.state==='saved_configuration_unavailable'?'Saved configuration could not be applied':'Applying saved settings'):'Waiting for service status'}</p>{/each}
{#if failures>0}<p role="alert">A configured provider could not be started. Review its settings before launch.</p>{/if}
<p>Applied settings do not prove that a provider has approved your account or completed a live operation.</p></section>
<section class="service-status"><h2>Launch requirements</h2><p>These show which configuration values are recorded. Evidence still needs to be genuine and reviewed.</p><ul>{#each gates as gate(gate.name)}<li>{gateLabels[gate.name]??gate.detail}: <strong>{gate.passed?'Recorded':'Missing'}</strong></li>{/each}</ul><a href="/admin/platform-settings?connection=integrations.runtime.launch">Update launch evidence and limits</a></section>
<div class="tasks">{#each tasks as task(task.id)}<article><h2>{task.label}</h2><strong>{labels[task.state]??task.state}</strong><p>{task.detail}</p><a href={task.url}>Open {task.label.toLowerCase()}</a></article>{/each}</div>
<p>Checked {localTime(checkedAt)}. Some approvals, domain verification and account activation must be completed with the provider; record their results here.</p>
<p><a href="/admin/platform-settings?connection=integrations.runtime.retained_collections">Manage saved collection accounts</a> · <a href="/admin/billing">Manage fee billing</a></p>
<p><a href="/admin/settlement-review">Review business bank accounts</a></p>
<a href="/admin/provider-work">Review pending provider work</a>
{/if}</main>
<style>main{padding-block:2rem}h1{font-size:2rem}h2{font-size:1.2rem}p{line-height:1.6}.tasks{display:grid;grid-template-columns:repeat(auto-fit,minmax(17rem,1fr));gap:1rem;margin-block:1.5rem}article,.service-status{padding:1.3rem;background:#fff;border:1px solid #d8dfd8;border-radius:.75rem}.service-status{margin-top:1rem}button{padding:.7rem 1rem;font:inherit}article strong{font-size:.9rem}article a{display:inline-block;margin-top:.5rem}</style>
