<script lang="ts">let { title, href = '' } = $props<{ title: string; href?: string }>();</script>
<section class="card"><h2>{title}</h2>{#if href}<a href={href} rel="noreferrer">Open this file</a>{:else}<p>We are checking this file. You can open it when the check is complete.</p>{/if}</section>
