<script lang="ts">
	import { onMount } from 'svelte';
	import { idempotencyKey } from '$lib/api/client';
	import { adminPost } from '$lib/admin-client';
	import { checkedJSON, record, publicError } from '$lib/api/reliable';
	let p:any=null;
 let intent:{payload:string,key:string}|null=null;
	let message='',error='',busy=false,loading=true;
	function preferences(value:unknown){
		const row=record(record(value).preferences),channels=['whatsapp','email','sms'];
		if(!channels.includes(String(row.preferred_channel))||!channels.includes(String(row.fallback_channel))||typeof row.payment_reminders_enabled!=='boolean'||typeof row.product_updates_enabled!=='boolean'||!Number.isSafeInteger(row.version)||Number(row.version)<1||typeof row.timezone!=='string'||!row.timezone.trim())throw new Error('Incomplete message choices');
		for(const key of ['quiet_start_hour','quiet_end_hour'])if(!Number.isInteger(row[key])||Number(row[key])<0||Number(row[key])>23)throw new Error('Invalid quiet hours');
		new Intl.DateTimeFormat('en-NG',{timeZone:row.timezone as string}).format();
		if(row.preferred_channel===row.fallback_channel)throw new Error('Choose different message channels');
		return row;
	}
	async function load(){
		loading=true;error='';p=null;
		try{p=await checkedJSON('/api/v1/me/notification-preferences',preferences);}
		catch(cause){error=publicError(cause,'your message choices');}
		finally{loading=false;}
	}
	async function save(){
		if(busy||loading||!p||error)return;busy=true;message='';
		try{

   const payload={preferred_channel:p.preferred_channel,fallback_channel:p.fallback_channel,payment_reminders_enabled:p.payment_reminders_enabled,product_updates_enabled:p.product_updates_enabled,quiet_start_hour:p.quiet_start_hour,quiet_end_hour:p.quiet_end_hour,timezone:p.timezone,expected_version:p.version};
   preferences({preferences:{...payload,version:p.version}});
   const encoded=JSON.stringify(payload);
   if(!intent||intent.payload!==encoded)intent={payload:encoded,key:idempotencyKey()};
   const response=await adminPost('/api/v1/me/notification-preferences',payload,'PUT',intent.key);

			p=preferences(response);intent=null;message='Your message choices were saved.';
		}catch(cause){message=cause instanceof Error?cause.message:'Your choices could not be confirmed.';}
		finally{busy=false;}
	}
	onMount(load);
</script>
<svelte:head><title>Notification preferences — Kredit</title></svelte:head>
<main class="shell workspace settings"><p class="eyebrow">Settings / Messages</p><h1>Choose how Kredit reaches you.</h1><p class="lede">You can switch off reminders and news. Important account, sale and payment messages are required; delivery still depends on your network and the messaging provider.</p>{#if message}<p class="notice" role="status">{message}</p>{/if}{#if loading}<p role="status">Opening message choices…</p>{:else if error}<p class="error" role="alert">{error}</p><button onclick={load}>Try again</button>{:else if p}<section><h2>Where should we message you?</h2><label>Try this first<select disabled={busy} bind:value={p.preferred_channel}><option value="whatsapp">WhatsApp</option><option value="email">Email</option><option value="sms">SMS</option></select></label><label>If that fails, use<select disabled={busy} bind:value={p.fallback_channel}><option value="email">Email</option><option value="sms">SMS</option><option value="whatsapp">WhatsApp</option></select></label><div class="hours"><label>Do not disturb from<input disabled={busy} type="number" min="0" max="23" bind:value={p.quiet_start_hour}/></label><label>Start messaging again at<input disabled={busy} type="number" min="0" max="23" bind:value={p.quiet_end_hour}/></label></div><p>Use 24-hour time in {p.timezone}. An urgent safety or payment message may still reach you during these hours.</p></section><section><h2>Messages you can stop</h2><label class="check"><input disabled={busy} type="checkbox" bind:checked={p.payment_reminders_enabled}/> Remind me about payments</label><label class="check"><input disabled={busy} type="checkbox" bind:checked={p.product_updates_enabled}/> Tell me about new Kredit features</label></section><button onclick={save} disabled={loading||busy||p.preferred_channel===p.fallback_channel}>{busy?'Saving…':'Save my choices'}</button>{/if}</main>
<style>.settings{max-width:48rem}.settings h1{font-size:clamp(2.4rem,6vw,4.5rem);line-height:1}.settings section{display:grid;gap:.8rem;margin:1rem 0;padding:1.3rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}label{display:grid;gap:.35rem;font-weight:700}select,input,button{padding:.75rem;border:1px solid var(--color-border);border-radius:.65rem;font:inherit}.hours{display:grid;grid-template-columns:1fr 1fr;gap:1rem}.check{display:flex;align-items:center}.check input{width:1.2rem;height:1.2rem}button{background:var(--color-primary);color:white;font-weight:800}.notice{padding:1rem;background:#eef5ff;border-radius:.7rem}@media(max-width:560px){.hours{grid-template-columns:1fr}}</style>
