<script lang="ts">
	let { accept = '', disabled = false, onselect = undefined } = $props<{ accept?: string; disabled?: boolean; onselect?: (file: File) => void }>();
	function selectFile(event: Event) {
		const file = (event.currentTarget as HTMLInputElement).files?.[0];
		if (file) onselect?.(file);
	}
</script>

<label class="field">Add a receipt or a photo <small>optional</small><input type="file" {accept} {disabled} onchange={selectFile} /></label>
<small>We check every file for viruses before anybody can open it.</small>
