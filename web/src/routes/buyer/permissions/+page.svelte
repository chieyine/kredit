<script lang="ts">
	import { getContext, onMount } from 'svelte';
 import {ACCOUNT_CONTEXT,type AccountContext} from '$lib/account-context';
 const account=getContext<AccountContext>(ACCOUNT_CONTEXT);
	import { MutationIntent } from '$lib/api/mutation';
	import { checkedJSON, LatestRequest, rows, record, text, publicError } from '$lib/api/reliable';
	let sellers:any[]=$state([]),consents:any[]=$state([]),loading=$state(true),error=$state(''),notice=$state(''),busy=$state('');
	const reads = new LatestRequest(), intents = new Map<string, MutationIntent>();
	function consentRecord(value: unknown) {
		const row = record(value);
		for (const key of ['id','supplier_organization_id','consent_type','created_at']) if (!text(row[key])) throw new Error('Incomplete consent');
		if (typeof row.granted !== 'boolean' || !Number.isFinite(Date.parse(String(row.created_at)))) throw new Error('Unverified consent');
		return row;
	}
	function current(orgID: string) { return consents.filter(item => item.supplier_organization_id === orgID && item.consent_type === 'payment_reminders').sort((a,b) => Date.parse(b.created_at)-Date.parse(a.created_at) || String(b.id).localeCompare(String(a.id)))[0]; }
	async function load() {
		const read = reads.begin(); loading = true; error = '';
		try {
			const result = await checkedJSON('/api/v1/buyer/relationships/consents', value=>{
    const permissions=rows('consents',consentRecord)(value);
    const suppliers=rows('suppliers',value=>{const supplier=record(value);const id=text(supplier.id),name=text(supplier.trading_name)||text(supplier.legal_name);if(!id||!name)throw new Error('Incomplete seller directory');return {id,name}})(value);
    if(new Set(suppliers.map(item=>item.id)).size!==suppliers.length)throw new Error('Duplicate seller directory');
    return {permissions,suppliers};
   },{signal:read.signal});
   if(read.current()){sellers=result.suppliers;consents=result.permissions;}
		} catch (cause) { if (read.current()) error = publicError(cause, 'your seller permissions'); }
		finally { if (read.current()) loading = false; }
	}
	async function save(seller: any, granted: boolean) {
		if (busy || loading || error) return;
		busy = seller.id; notice = '';
		try {
			const words = `payment_reminders|v1|${seller.id}|${granted}`;
			const bytes = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(words));
			const evidence_hash = [...new Uint8Array(bytes)].map(value => value.toString(16).padStart(2,'0')).join('');
			if (!intents.has(seller.id)) intents.set(seller.id, new MutationIntent(`${account.userID}:buyer-seller-permission:${seller.id}`, '/api/v1/buyer/relationships/consents'));
			await intents.get(seller.id)!.run({supplier_organization_id:seller.id,consent_type:'payment_reminders',version:'v1',evidence_hash,granted}, value => {
				const consent = consentRecord(record(value).consent);
				if (consent.supplier_organization_id !== seller.id || consent.consent_type !== 'payment_reminders' || consent.granted !== granted) throw new Error('Permission not confirmed');
				return consent;
			});
			notice = granted ? `${seller.name} may send payment reminders.` : `${seller.name} may no longer send optional payment reminders.`;
			await load();
		} catch(cause) { notice = cause instanceof Error ? cause.message : 'We could not confirm this permission.'; }
		finally { busy = ''; }
	}
	onMount(() => { void load(); return () => reads.cancel(); });
</script>
<svelte:head><title>Seller permissions — Kredit</title></svelte:head>
<main class="shell workspace permissions"><p class="eyebrow">Seller permissions</p><h1>Choose which sellers may message you.</h1><p class="lede">This only covers extra reminders a seller may want to send. Important sale, payment and safety messages always come from Kredit, and you cannot switch those off.</p>{#if error}<p class="error" role="alert">{error}</p>{/if}{#if notice}<p class="notice" role="status">{notice}</p>{/if}{#if loading}<p role="status">Opening permissions…</p>{:else if error}<button onclick={load}>Try again</button>{:else if sellers.length}<section>{#each sellers as seller}{@const allowed=current(seller.id)?.granted===true}<article><div><strong>{seller.name}</strong><span>{allowed?'Reminders allowed':'Reminders stopped'}</span></div><button disabled={busy!==''} onclick={()=>save(seller,!allowed)}>{busy===seller.id?'Saving…':allowed?'Stop optional reminders':'Allow payment reminders'}</button></article>{/each}</section>{:else}<section class="empty-state"><h2>No sellers yet</h2><p>This list fills up once you start buying from a seller on Kredit.</p></section>{/if}</main>
<style>.permissions{max-width:54rem}.permissions section{display:grid;gap:.75rem;margin-top:2rem}.permissions article{display:flex;justify-content:space-between;align-items:center;gap:1rem;padding:1rem;border:1px solid var(--color-border);background:var(--color-surface)}.permissions article div,.permissions article span{display:block}.permissions article span{margin-top:.3rem;color:var(--color-muted)}.permissions button{padding:.7rem;border:1px solid var(--color-primary);background:var(--color-primary);color:#fff;font:inherit;font-weight:750}.notice{padding:.8rem;border-left:4px solid var(--color-positive)}.error{color:var(--color-destructive)}@media(max-width:600px){.permissions article{align-items:stretch;flex-direction:column}}
</style>
