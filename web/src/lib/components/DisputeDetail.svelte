<script lang="ts">
  import { getContext } from 'svelte';
  import { ACCOUNT_CONTEXT, type AccountContext } from '$lib/account-context';
  import { MutationIntent } from '$lib/api/mutation';
  import { checkedJSON, LatestRequest, publicError, record, rows, text } from '$lib/api/reliable';
  import { formatKobo, nairaInput, parseNaira, type KoboValue } from '$lib/money';
  import { productLabel } from '$lib/product-language';
  import { kobo, timeLabel } from '$lib/records';
  let { endpoint, backHref, canDecide = false }: { endpoint: string; backHref: string; canDecide?: boolean } = $props();
  const account = getContext<AccountContext>(ACCOUNT_CONTEXT);
  const requests = new LatestRequest();
  const evidenceIntent = $derived(new MutationIntent(account.userID, `${endpoint}/evidence`));
  const decisionIntent = $derived(new MutationIntent(account.userID, `${endpoint}/decide`));
  let dispute: any = $state(null), evidence: any[] = $state([]), decisions: any[] = $state([]);
  let loading = $state(true), busy = $state(false), error = $state(''), notice = $state('');
  let statement = $state(''), documentID = $state(''), outcome = $state('UPHELD'), valid = $state('0'), adjustment = $state('0'), remaining = $state('0'), reason = $state('');
  const money = (value: KoboValue) => formatKobo(value);
  async function load() {
    const request = requests.begin();
    loading = true; error = '';
    try {
      const data = await checkedJSON(endpoint, value => {
        const response = record(value), item = record(response.dispute);
        text(item.id); text(item.state); kobo(item.total_disputed_kobo); kobo(item.remaining_disputed_kobo);
        return { dispute: item, evidence: rows('evidence', record)(response), decisions: rows('decisions', record)(response) };
      }, { signal: request.signal });
      if (!request.current()) return;
      dispute = data.dispute; evidence = data.evidence; decisions = data.decisions;
      remaining = nairaInput(dispute.remaining_disputed_kobo);
    } catch (cause) {
      if (request.current()) { dispute = null; error = publicError(cause, 'this problem'); }
    } finally { if (request.current()) loading = false; }
  }
  async function submitEvidence(event: SubmitEvent) {
    event.preventDefault();
    if (busy || !statement.trim()) return;
    const scope = endpoint;
    busy = true; error = ''; notice = '';
    try {
      await evidenceIntent.run({ document_id: documentID, statement }, value => text(record(record(value).evidence).id));
      if (scope !== endpoint) return;
      statement = ''; documentID = ''; notice = 'Information added.';
      await load();
    } catch (cause) {
      if (scope === endpoint) error = cause instanceof Error ? cause.message : 'We could not confirm that the information was added.';
    } finally { if (scope === endpoint) busy = false; }
  }
  async function decide(event: SubmitEvent) {
    event.preventDefault();
    if (busy || !canDecide) return;
    const scope = endpoint;
    const body = { outcome, valid_principal_kobo: parseNaira(valid), adjustment_kobo: parseNaira(adjustment), remaining_disputed_kobo: parseNaira(remaining), reason };
    if ([body.valid_principal_kobo, body.adjustment_kobo, body.remaining_disputed_kobo].some(amount => amount < 0)) { error = 'Enter valid amounts with no more than two decimal places.'; return; }
    busy = true; error = ''; notice = '';
    try {
      await decisionIntent.run(body, value => text(record(record(value).dispute).id));
      if (scope !== endpoint) return;
      notice = 'Decision saved.';
      await load();
    } catch (cause) {
      if (scope === endpoint) error = cause instanceof Error ? cause.message : 'We could not confirm that the decision was saved.';
    } finally { if (scope === endpoint) busy = false; }
  }
  $effect(() => {
    endpoint;
    dispute = null; evidence = []; decisions = []; busy = false; notice = ''; statement = ''; documentID = ''; reason = '';
    void load();
    return () => requests.cancel();
  });
</script>
<a href={backHref}>← All problems</a>
{#if loading}<p role="status">Opening this problem…</p>{:else if error && !dispute}<p class="error" role="alert">{error} <button onclick={load}>Try again</button></p>{:else if dispute}
	{#if error}<p class="error" role="alert">{error}</p>{/if}{#if notice}<p class="notice" role="status">{notice}</p>{/if}
	<section class="summary" aria-label="What the problem is"><article><span>Now</span><strong>{productLabel(dispute.state)}</strong></article><article><span>Money first reported</span><strong>{money(dispute.total_disputed_kobo)}</strong></article><article><span>Money still in question</span><strong>{money(dispute.remaining_disputed_kobo)}</strong></article><article><span>What happens to bank debit</span><strong>{productLabel(dispute.collection_effect)}</strong></article></section>
	<article class="card"><h2>{dispute.reason}</h2><p>{dispute.explanation||'No other details were added.'}</p><small>Reported {timeLabel(dispute.opened_at)}</small></article>
	<div class="columns"><section class="card"><h2>What has been added</h2>{#if evidence.length}<ol>{#each evidence as item}<li><p>{item.statement||'A document was added.'}</p>{#if item.document_id}<details><summary>Document number</summary><code>{item.document_id}</code></details>{/if}<small>{timeLabel(item.submitted_at)}</small></li>{/each}</ol>{:else}<p>Nothing has been added yet.</p>{/if}<form onsubmit={submitEvidence}><h3>Add more information</h3><label>What else should we know?<textarea bind:value={statement} required rows="4" disabled={busy}></textarea></label><label>Document number <small>optional</small><input bind:value={documentID} disabled={busy}/></label><button class="primary" disabled={busy||!statement.trim()}>{busy?'Adding…':'Add this information'}</button></form></section>
	<section class="card"><h2>Decisions so far</h2>{#if decisions.length}<ol>{#each decisions as item}<li><strong>{productLabel(item.outcome)}</strong><p>{item.reason}</p><small>{timeLabel(item.decided_at)}</small></li>{/each}</ol>{:else}<p>No decision yet.</p>{/if}
	{#if canDecide && !['RESOLVED','WITHDRAWN'].includes(dispute.state)}<form onsubmit={decide}><h3>Make a decision</h3><label>Decision<select bind:value={outcome} disabled={busy}><option value="UPHELD">Accept the report in full</option><option value="PARTIALLY_UPHELD">Accept part of the report</option><option value="REJECTED">Reject the report</option></select></label><label>Correct sale amount (₦)<input bind:value={valid} inputmode="decimal" disabled={busy}/></label><label>Amount to remove from the balance (₦)<input bind:value={adjustment} inputmode="decimal" disabled={busy}/></label><label>Amount still in question (₦)<input bind:value={remaining} inputmode="decimal" disabled={busy}/></label><label>Why did you decide this?<textarea bind:value={reason} required rows="3" disabled={busy}></textarea></label><button class="primary" disabled={busy||!reason.trim()}>{busy?'Saving…':'Save this decision'}</button></form>{/if}</section></div>
{/if}
<style>.summary{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:1rem;margin:1.5rem 0}.summary article{padding:1rem;border:1px solid var(--color-border);border-radius:1rem;background:var(--color-surface)}.summary span,.summary strong{display:block}.columns{display:grid;grid-template-columns:1fr 1fr;gap:1rem}.card{padding:1.25rem}ol{display:grid;gap:.75rem;padding:0;list-style:none}li{padding:.8rem;border-left:3px solid var(--color-primary);background:var(--color-surface-muted)}li p{margin:.25rem 0}form,label{display:grid;gap:.45rem}form{gap:.75rem;margin-top:1rem;padding-top:1rem;border-top:1px solid var(--color-border)}input,select,textarea{box-sizing:border-box;width:100%;padding:.7rem;border:1px solid var(--color-border);border-radius:.55rem;background:var(--color-surface);color:inherit}.notice{padding:.8rem;border-left:4px solid var(--color-positive);background:var(--color-surface-muted)}@media(max-width:760px){.summary,.columns{grid-template-columns:1fr}}</style>
