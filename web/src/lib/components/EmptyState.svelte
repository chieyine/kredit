<script lang="ts">let { title, detail } = $props<{ title: string; detail?: string }>();</script>
<section class="empty-state"><h2>{title}</h2>{#if detail}<p>{detail}</p>{/if}</section>
