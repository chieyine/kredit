<script lang="ts">
	let { children } = $props();
</script>

<div class="public-action-shell">
	<a class="wordmark action-brand" href="/" aria-label="Kredit home"><span>K</span><b>Kredit</b></a>
	<div class="action-card">{@render children()}</div>
	<p class="action-help">
		Questions about this link? Ask the business that sent it to you, or <a href="/contact">contact Kredit</a>. We will
		never ask for your sign-in code or bank PIN.
	</p>
</div>
