<script lang="ts">
  import { onMount, setContext, type Snippet } from 'svelte';
  import { ACCOUNT_CONTEXT, type AccountContext } from '$lib/account-context';
  import { checkedJSON, clearPrivateBrowserData, LatestRequest, record, RequestError, text } from '$lib/api/reliable';
  let { children, area = 'account' }: { children: Snippet; area?: string } = $props();
  let gateStatus = $state<'checking' | 'ready' | 'error'>('checking');
  let userID = $state('');
  const requests = new LatestRequest();
  setContext<AccountContext>(ACCOUNT_CONTEXT, { get userID() { return userID; } });
  async function verify() {
    const request = requests.begin();
    gateStatus = 'checking'; userID = '';
    try {
      const id = await checkedJSON('/api/v1/me', value => text(record(record(value).user).id), { signal: request.signal });
      if (!request.current()) return;
      if (!id) throw new RequestError('Account identity was not returned.');
      userID = id; gateStatus = 'ready';
    } catch (error) {
      if (!request.current()) return;
      if (error instanceof RequestError && error.status === 401) {
        clearPrivateBrowserData();
        location.replace(`/app?next=${encodeURIComponent(location.pathname + location.search)}`);
        return;
      }
      gateStatus = 'error';
    }
  }
  onMount(() => { void verify(); return () => requests.cancel(); });
</script>

{#if gateStatus === 'ready'}
  {@render children()}
{:else}
  <main class="account-gate" aria-live="polite">
    <a class="gate-brand" href="/" aria-label="Kredit home"><span aria-hidden="true">K</span>Kredit</a>
    <section>
      <p class="eyebrow">Private {area}</p>
      {#if gateStatus === 'checking'}
        <h1>Checking your account…</h1><p>Your private records will open after this check.</p>
      {:else}
        <h1>We cannot open your account.</h1><p>The account check did not finish. Check your connection and try again.</p>
        <div class="gate-actions"><button type="button" onclick={verify}>Try again</button><a href="/">Go to the home page</a></div>
      {/if}
    </section>
  </main>
{/if}
<style>
  .account-gate{min-height:100svh;box-sizing:border-box;padding:clamp(1.25rem,4vw,3rem);background:var(--color-background);color:var(--color-foreground)}.gate-brand{display:inline-flex;align-items:center;gap:.65rem;text-decoration:none;font-weight:750}.gate-brand span{display:grid;place-items:center;width:2rem;height:2rem;background:var(--color-primary);color:#fff}.account-gate section{max-width:34rem;margin:clamp(3rem,15vh,9rem) auto}.account-gate h1{font-family:inherit;font-size:clamp(1.8rem,4vw,2.5rem);line-height:1.2;letter-spacing:-.03em}.account-gate p{line-height:1.6;color:var(--color-muted)}.gate-actions{display:flex;gap:1rem;flex-wrap:wrap}.gate-actions button,.gate-actions a{display:inline-flex;align-items:center;justify-content:center;padding:.75rem 1rem;border:1px solid var(--color-border);border-radius:.4rem;font:inherit;text-decoration:none}.gate-actions button{background:var(--color-primary);color:#fff;border-color:var(--color-primary)}
</style>
