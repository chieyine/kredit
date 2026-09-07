<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { checkedJSON, normalizeNigerianPhone, record, RequestError, safeNext, text } from '$lib/api/reliable';
  let identifier = $state(''), challengeID = $state(''), code = $state(''), developmentCode = $state(''), error = $state('');
  let channel = $state<'email' | 'sms'>('sms');
  let busy = $state(false), sentTo = $state(''), expiresAt = $state(0), resendAt = $state(0), now = $state(Date.now());
  let checking = $state(true), sessionWarning = $state('');
  const countdown = $derived(Math.max(0, Math.ceil((resendAt - now) / 1000)));
  const expired = $derived(challengeID !== '' && now >= expiresAt);
  function destination() { return safeNext(page.url.searchParams.get('next'), location.origin); }
  function masked(value: string) {
    const at = value.indexOf('@');
    return at >= 0 ? `${value.slice(0, 2)}…${value.slice(at)}` : `${value.slice(0, 4)} •••• ${value.slice(-4)}`;
  }
  async function checkSession(signal: AbortSignal | undefined = undefined) {
    checking = true; sessionWarning = '';
    try { await checkedJSON('/api/v1/me', value => text(record(record(value).user).id), { signal }); await goto(destination()); }
    catch (cause) { if (!signal?.aborted && !(cause instanceof RequestError && cause.status === 401)) sessionWarning = 'We could not check your current session. You can try the check again or sign in below.'; }
    finally { checking = false; }
  }
  async function requestCode() {
    if (busy || (challengeID && countdown > 0)) return;
    busy = true; error = '';
    try {
      const target = channel === 'sms' ? normalizeNigerianPhone(identifier) : identifier.trim();
      const body = await checkedJSON('/api/v1/auth/otp/challenges', value => {
        const result = record(value); const expiry = Date.parse(text(result.expires_at));
        if (!Number.isFinite(expiry) || expiry <= Date.now()) throw new RequestError('The code expiry was not confirmed.');
        return { id: text(result.challenge_id), expiry, developmentCode: typeof result.development_code === 'string' ? result.development_code : '' };
      }, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ identifier: target, channel, purpose: 'login' }) });
      identifier = target; sentTo = masked(target); challengeID = body.id; expiresAt = body.expiry;
      developmentCode = body.developmentCode; code = ''; resendAt = Date.now() + 60000;
    } catch (cause) {
      error = cause instanceof RequestError && cause.code === 'invalid_phone' ? cause.message : cause instanceof RequestError && cause.status === 429 ? 'Please wait before requesting another code. The previous code may still arrive.' : 'We could not confirm that a code was sent. Check your connection, then try again.';
      if (cause instanceof RequestError && cause.status === 429) resendAt = Date.now() + 60000;
    } finally { busy = false; }
  }
  async function verifyCode() {
    if (busy || expired || !/^\d{6}$/.test(code)) return;
    busy = true; error = '';
    try {
      await checkedJSON('/api/v1/auth/otp/verify', value => { const result = record(value); text(record(result.user).id); return result; }, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ challenge_id: challengeID, code, device_label: navigator.userAgent.slice(0, 120) })
      });
      await goto(destination());
    } catch (cause) {
      if (cause instanceof RequestError && cause.status === 401) error = 'That code is incorrect or has expired. Check the six digits, or request a new code.';
      else error = 'We could not confirm sign-in. Check your connection and try again. Do not share this code with anyone.';
    } finally { busy = false; }
  }
  function editIdentifier() { if (busy) return; challengeID = ''; code = ''; developmentCode = ''; error = ''; expiresAt = 0; }
  onMount(() => {
    const controller = new AbortController();
    void checkSession(controller.signal);
    const timer = setInterval(() => now = Date.now(), 1000);
    return () => { clearInterval(timer); controller.abort(); };
  });
</script>
<svelte:head><title>Start or sign in — Kredit</title></svelte:head>
<main class="shell auth-page">
  <section class="auth-copy"><p class="eyebrow">Your Kredit account</p><h1>Start or sign in.</h1><p class="lede">Keep your sales and payments in one place. We will send you a code—there is no password to remember.</p><p class="privacy-note">New here? Start with your phone number or email. Your business details come next.</p></section>
  <form class="card auth-card" onsubmit={(event) => { event.preventDefault(); void (challengeID ? verifyCode() : requestCode()); }}>
    <h2>{challengeID ? 'Enter your six-digit code' : 'Open your account'}</h2>
    {#if page.url.searchParams.get('signed_out') === '1'}<p class="inline-notice" role="status">You are signed out.</p>{/if}
    {#if sessionWarning}<div class="inline-notice"><p>{sessionWarning}</p><button type="button" class="text-button" disabled={checking} onclick={() => void checkSession()}>Check session again</button></div>{/if}
    {#if !challengeID}
      <fieldset class="channel"><legend>Where should we send your code?</legend><label><input type="radio" value="sms" bind:group={channel} disabled={busy} />Phone (SMS)</label><label><input type="radio" value="email" bind:group={channel} disabled={busy} />Email</label></fieldset>
      <label>{channel === 'sms' ? 'Phone number' : 'Email address'}<input bind:value={identifier} type={channel === 'sms' ? 'tel' : 'email'} autocomplete={channel === 'sms' ? 'tel' : 'email'} placeholder={channel === 'sms' ? '0801 234 5678' : 'you@example.com'} disabled={busy} required /></label>
      {#if channel === 'sms'}<p class="field-help">Use your Nigerian number, starting with 0 or +234.</p>{/if}
    {:else}
      <p>A code was sent to <strong>{sentTo}</strong>.</p>
      <label>Six-digit code<input bind:value={code} inputmode="numeric" autocomplete="one-time-code" pattern="[0-9]{6}" minlength="6" maxlength="6" disabled={busy || expired} required /></label>
      <p class:expired>{expired ? 'This code has expired. Request a new one below.' : `This code expires in ${Math.max(1, Math.ceil((expiresAt - now) / 60000))} minute(s).`}</p>
      {#if developmentCode}<p class="notice">Development code: <strong>{developmentCode}</strong></p>{/if}
    {/if}
    {#if error}<p class="error" role="alert">{error}</p>{/if}
    <button class="primary" disabled={busy || (challengeID ? expired || !/^\d{6}$/.test(code) : !identifier.trim())}>{busy ? 'Checking…' : challengeID ? 'Open my account' : 'Send me a code'}</button>
    {#if challengeID}<div class="code-actions"><button class="secondary" type="button" onclick={requestCode} disabled={busy || countdown > 0}>{countdown > 0 ? `Resend in ${countdown}s` : 'Send a new code'}</button><button class="text-button" type="button" onclick={editIdentifier} disabled={busy}>Change email or number</button></div>{/if}
    <p class="security-note">Kredit support will never ask you to share your sign-in code.</p>
  </form>
</main>
<style>
  .auth-page{display:grid;grid-template-columns:1fr minmax(18rem,28rem);gap:clamp(2rem,7vw,6rem);align-items:center;min-height:calc(100svh - 5rem);padding-block:3rem}.auth-copy h1{font-family:Georgia,serif;font-weight:500;font-size:clamp(2.8rem,6vw,5rem);line-height:1.04;letter-spacing:-.045em;margin:.5rem 0 1.5rem}.privacy-note,.security-note,.field-help{color:var(--color-muted);font-size:.9rem;line-height:1.6}.auth-card{display:grid;gap:1rem;padding:clamp(1.25rem,4vw,2rem);border-top:3px solid var(--color-primary)}.auth-card h2,.auth-card p{margin:0}.auth-card label{display:grid;gap:.45rem;font-weight:650}.auth-card input:not([type=radio]){box-sizing:border-box;width:100%;padding:.8rem;border:1px solid var(--color-border);border-radius:.35rem;font:inherit;background:var(--color-surface)}.channel{display:flex;gap:1rem;flex-wrap:wrap;padding:0;border:0;margin:0}.channel legend{margin-bottom:.5rem;font-weight:650}.channel label{display:flex;align-items:center;gap:.5rem;min-height:2.75rem}.channel input{min-height:1.2rem;width:1.2rem;height:1.2rem;accent-color:var(--color-primary)}.code-actions{display:grid;gap:.5rem}.secondary{background:transparent;border:1px solid var(--color-border);padding:.75rem;border-radius:.35rem}.text-button{border:0;background:transparent;color:var(--color-primary);text-decoration:underline;cursor:pointer}.error,.expired{color:var(--color-destructive);line-height:1.6}.security-note{border-top:1px solid var(--color-border);padding-top:1rem}.inline-notice{padding:.8rem;background:#eef0ff;border-left:3px solid var(--color-primary)}@media(max-width:760px){.auth-page{grid-template-columns:1fr;gap:1.5rem;align-items:start;min-height:auto;padding-block:2rem}.auth-copy h1{font-size:2.6rem}.auth-copy .lede{font-size:1rem}}
</style>
