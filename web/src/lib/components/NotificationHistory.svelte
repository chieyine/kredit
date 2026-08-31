<script lang="ts">
	import { onMount } from 'svelte';
	import { productLabel } from '$lib/product-language';
	let { preferencesHref = '/app/settings/notifications' } = $props<{ preferencesHref?: string }>();
	let items: any[] = $state([]), loading = $state(true), error = $state('');
	const channel = (value: string) => ({ whatsapp: 'WhatsApp', email: 'Email', sms: 'SMS' } as Record<string,string>)[value] ?? productLabel(value);
	const messageName = (value: string) => ({ PaymentDueSoon: 'Payment reminder', PaymentRecorded: 'Payment received', CollectionSubmitted: 'Bank debit started', AccountRecoveryRequested: 'Account recovery started', AccountRecoveryCompleted: 'Account recovery completed', PrivacyRequestReceived: 'Information request received', PrivacyExportReady: 'Information copy ready', NotificationPreferencesChanged: 'Message choices changed' } as Record<string,string>)[value] ?? productLabel(value);
	onMount(async () => {
		const response = await fetch('/api/v1/me/notifications', { credentials: 'include' });
		const data = await response.json().catch(() => ({})); loading = false;
		if (!response.ok) { error = data.detail ?? 'We could not open your message history.'; return; }
		items = (data.notifications ?? []).sort((a: any, b: any) => new Date(b.sent_at || b.scheduled_at || 0).getTime() - new Date(a.sent_at || a.scheduled_at || 0).getTime());
	});
</script>

<main class="shell workspace messages"><p class="eyebrow">Message history</p><h1>Messages Kredit sent you.</h1><p class="lede">See where each message was sent and whether it went through. Private codes are never shown here.</p>{#if loading}<p role="status">Opening your messages…</p>{:else if error}<p class="error" role="alert">{error}</p>{:else if items.length}<section class="list">{#each items as item}<article><div><strong>{messageName(item.template)}</strong><span>{productLabel(item.state)}</span></div><p>{item.body || 'Kredit sent an account message.'}</p><small>{channel(item.channel)} · {new Date(item.sent_at || item.scheduled_at || item.failed_at).toLocaleString('en-NG')}</small>{#if item.failure_reason}<p class="failed">It did not go through: {item.failure_reason}</p>{/if}</article>{/each}</section>{:else}<section class="empty-state"><h2>No messages yet</h2><p>Account, payment and reminder messages will appear here after Kredit sends them.</p></section>{/if}<p><a href={preferencesHref}>Change where messages go →</a></p></main>
<style>.messages{max-width:58rem}.list{display:grid;gap:.75rem;margin-top:2rem}.list article{padding:1rem;border:1px solid var(--color-border);background:var(--color-surface)}.list article>div{display:flex;justify-content:space-between;gap:1rem}.list span{font-size:.8rem;font-weight:750}.list p{margin:.5rem 0}.list small{color:var(--color-muted)}.failed,.error{color:var(--color-destructive)}</style>
