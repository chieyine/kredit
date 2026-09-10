<script lang="ts">
	import { onMount } from 'svelte';
	import { productLabel } from '$lib/product-language';
 import { checkedJSON, LatestRequest, publicError, record, rows, text } from '$lib/api/reliable';
	let { preferencesHref = '/app/settings/notifications' } = $props<{ preferencesHref?: string }>();
	type HistoryItem = { id: string; channel: string; template: string; state: string; body: string; sent_at: string; scheduled_at: string; failed_at: string; failure_reason: string; recipient_hint: string };
 const requests = new LatestRequest();
 let items: HistoryItem[] = $state([]), loading = $state(true), error = $state(''), filter = $state('all');
	const channel = (value: string) => ({ whatsapp: 'WhatsApp', email: 'Email', sms: 'SMS' } as Record<string,string>)[String(value).toLowerCase()] ?? productLabel(value);
	const messageName = (value: string) => ({
		PaymentDueSoon: 'Payment due soon', PaymentOverdue: 'Payment overdue', PaymentRecorded: 'Payment recorded', PaymentReceipt: 'Payment receipt',
		CreditRequestSent: 'Sale sent to customer', CreditRequestAccepted: 'Customer accepted the sale', GoodsReleased: 'Goods sent out', GoodsReceived: 'Goods received',
		CollectionSubmitted: 'Bank debit requested', CollectionSucceeded: 'Bank debit completed', CollectionFailed: 'Bank debit failed',
		DisputeOpened: 'Problem reported', DisputeUpdated: 'Problem updated',
		AccountRecoveryRequested: 'Account recovery started', AccountRecoveryCompleted: 'Account recovery completed',
		PrivacyRequestReceived: 'Information request received', PrivacyExportReady: 'Your information copy is ready', NotificationPreferencesChanged: 'Message settings changed'
	} as Record<string,string>)[value] ?? productLabel(value);
	const visible = $derived(filter === 'all' ? items : filter === 'failed' ? items.filter((item)=>String(item.state).toLowerCase()==='failed'||item.failure_reason) : items.filter((item)=>String(item.channel).toLowerCase()===filter));
	const failures = $derived(items.filter((item)=>String(item.state).toLowerCase()==='failed'||item.failure_reason).length);
	const sentCount = $derived(items.filter((item)=>['sent','delivered','recognized','completed'].includes(String(item.state).toLowerCase())).length);
	function timestamp(item: HistoryItem): number {
		for (const value of [item.sent_at, item.failed_at, item.scheduled_at]) {
			if (value && !value.startsWith('0001-')) {
				const result = Date.parse(value);
				if (Number.isFinite(result)) return result;
			}
		}
		return 0;
	}
	function sentTime(item: HistoryItem): string {
		const value = timestamp(item);
		return value ? new Date(value).toLocaleString('en-NG', { dateStyle: 'medium', timeStyle: 'short' }) : 'Time unavailable';
	}
	async function load() {
		const request = requests.begin();
		loading = true; error = '';
		try {
			const data = await checkedJSON('/api/v1/me/notifications', rows('notifications', value => {
				const item = record(value);
				const optional = (key: string) => item[key] == null ? '' : text(item[key]);
				return { id: text(item.id), channel: text(item.channel), template: text(item.template), state: text(item.state), body: optional('body'), sent_at: optional('sent_at'), scheduled_at: optional('scheduled_at'), failed_at: optional('failed_at'), failure_reason: optional('failure_reason'), recipient_hint: optional('recipient_hint') };
			}), { signal: request.signal });
			if (request.current()) items = data.sort((a, b) => timestamp(b) - timestamp(a));
		} catch (cause) {
			if (request.current()) error = publicError(cause, 'your message history');
		} finally {
			if (request.current()) loading = false;
		}
	}
	onMount(() => { void load(); return () => requests.cancel(); });
</script>

<main class="shell workspace messages"><header><p class="eyebrow">Message history</p><h1>Know exactly what Kredit sent.<br />Know whether it arrived.</h1><p class="lede">Every reminder, payment message and account notice, with its latest delivery status. We never show private codes here.</p></header>
{#if loading}<p role="status">Opening your messages…</p>{:else if error}<p class="error" role="alert">{error}</p><button type="button" onclick={load}>Try again</button>{:else}
<section class="summary"><article><span>Total messages</span><strong>{items.length}</strong></article><article><span>Sent or delivered</span><strong>{sentCount}</strong></article><article class:danger={failures>0}><span>Delivery failed</span><strong>{failures}</strong></article></section>
<div class="toolbar" role="group" aria-label="Filter message history"><button class:active={filter==='all'} onclick={()=>filter='all'}>All</button><button class:active={filter==='whatsapp'} onclick={()=>filter='whatsapp'}>WhatsApp</button><button class:active={filter==='email'} onclick={()=>filter='email'}>Email</button><button class:active={filter==='sms'} onclick={()=>filter='sms'}>SMS</button><button class:active={filter==='failed'} onclick={()=>filter='failed'}>Failed</button><a href={preferencesHref}>Notification settings →</a></div>
{#if visible.length}<section class="list">{#each visible as item}<article class:failed-card={item.failure_reason}><div class="message-head"><div><span class="channel">{channel(item.channel)}</span><strong>{messageName(item.template)}</strong></div><span class="state">{productLabel(item.state)}</span></div><p class="body">{item.body || 'Kredit sent an account message.'}</p><div class="meta"><time>{sentTime(item)}</time>{#if item.recipient_hint}<span>{item.recipient_hint}</span>{/if}</div>{#if item.failure_reason}<p class="failed"><strong>What went wrong:</strong> {item.failure_reason}</p>{/if}</article>{/each}</section>{:else}<section class="empty-state"><h2>{items.length?'No messages match this filter':'No messages yet'}</h2><p>{items.length?'Choose another channel to see more messages.':'Account, payment and reminder messages will appear here after Kredit sends them.'}</p></section>{/if}
<section class="explain"><strong>Why this page matters</strong><p>Check the delivery status before following up. “Sent” means the delivery provider accepted the message; it does not confirm the customer received or read it.</p></section>
{/if}</main>
<style>.messages{max-width:64rem}.messages header{max-width:52rem}.messages h1{max-width:15ch;font-family:var(--font-serif);font-size:clamp(3rem,6vw,5.2rem);font-weight:500;line-height:.94;letter-spacing:-.055em}.summary{display:grid;grid-template-columns:repeat(3,1fr);margin:2rem 0;border-top:1px solid var(--color-border);border-left:1px solid var(--color-border)}.summary article{display:grid;gap:.4rem;padding:1rem;border-right:1px solid var(--color-border);border-bottom:1px solid var(--color-border);background:#fffdf8}.summary span{color:var(--color-muted);font-size:.75rem}.summary strong{font-family:var(--font-serif);font-size:2rem;font-weight:500}.summary article.danger{border-top:4px solid #b42318}.toolbar{display:flex;align-items:center;gap:.4rem;flex-wrap:wrap;margin:1.5rem 0;padding:.6rem;background:#17181b}.toolbar button{min-height:2.6rem;padding:.55rem .8rem;border:1px solid #4b4c51;background:transparent;color:#d5d5d1;font:inherit;font-size:.75rem;font-weight:750}.toolbar button.active{border-color:#e85f3d;background:#e85f3d;color:#17181b}.toolbar a{margin-left:auto;color:#fff;font-size:.76rem;font-weight:750}.list{display:grid;gap:.75rem;margin-top:2rem}.list article{padding:1rem 1.1rem;border:1px solid var(--color-border);border-left:4px solid #2738d6;background:var(--color-surface)}.list article.failed-card{border-left-color:#b42318}.message-head{display:flex;justify-content:space-between;gap:1rem}.message-head>div{display:grid;gap:.25rem}.channel{color:#2738d6;font-size:.65rem;font-weight:850;letter-spacing:.08em;text-transform:uppercase}.state{font-size:.75rem;font-weight:750}.body{margin:.75rem 0;color:#343733;line-height:1.6}.meta{display:flex;justify-content:space-between;gap:1rem;color:var(--color-muted);font-size:.72rem}.failed,.error{color:var(--color-destructive)}.failed{margin:.75rem 0 0;padding:.7rem;background:#fff0ed}.empty-state{padding:3rem;background:#2738d6;color:#fff}.empty-state p{color:#d8dbff}.explain{margin-top:2rem;padding:1rem;border-left:4px solid #16794e;background:#edf8f2}.explain p{margin:.4rem 0;color:#59615c;line-height:1.6}@media(max-width:650px){.summary{grid-template-columns:1fr}.toolbar{align-items:stretch}.toolbar button{flex:1}.toolbar a{width:100%;margin:0;padding:.6rem}.message-head,.meta{align-items:flex-start;flex-direction:column;gap:.45rem}}</style>
