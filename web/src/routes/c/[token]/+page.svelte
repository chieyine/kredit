<script lang="ts">import { page } from '$app/state';</script>
<svelte:head><title>Your private sale link — Kredit</title></svelte:head>
<main class="shell prose-page"><p class="eyebrow">Your private link</p><h1>Read the sale before you say yes.</h1><p class="lede">We are opening the private link that was sent to you. First we will send a code to your phone or email, to be sure it is really you.</p><p class="sr-only">Invitation number {page.params.token}</p></main>
