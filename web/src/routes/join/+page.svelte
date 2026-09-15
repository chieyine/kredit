<script lang="ts">
 import {onMount} from 'svelte';import {read} from '$lib/consumer';import {record} from '$lib/api/reliable';
 let code=$state(''),name=$state(''),error=$state('');onMount(()=>{code=new URLSearchParams(location.search).get('ref')??'';void read('/api/v1/dsa/code/'+encodeURIComponent(code)).then(v=>{name=String(record(v).name)}).catch(()=>error='This referral link is unavailable. Ask the agent to check their code.');});
</script>
<svelte:head><title>Join Kredit through a field agent</title></svelte:head>
<main class="shell"><h1>Keep your business sales and payments organised.</h1>{#if error}<p role="alert">{error}</p>{:else if name}<p>{name} invited your business to Kredit. They may earn a reward if you join and use Kredit.</p><p>Sign in yourself, then confirm the introduction. Never share your sign-in code with an agent.</p><a class="primary" href={`/app?next=${encodeURIComponent('/app/referral?code='+encodeURIComponent(code))}`}>Start or sign in</a>{:else}<p>Checking your invitation…</p>{/if}</main>
