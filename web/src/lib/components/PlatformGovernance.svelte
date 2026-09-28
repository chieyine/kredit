<script lang="ts">
	import OwnerDialog from './OwnerDialog.svelte';
	import { MutationIntent } from '$lib/api/mutation';
	import { record } from '$lib/api/reliable';
	import { errorMessage } from '$lib/admin-client';
	import type { Governance } from '$lib/admin/platform-settings';
	let {
		governance,
		isOwner,
		busy = $bindable(false),
		error = $bindable(''),
		message = $bindable(''),
		onchanged
	}: {
		governance: Governance | null;
		isOwner: boolean;
		busy?: boolean;
		error?: string;
		message?: string;
		onchanged: () => Promise<void>;
	} = $props();
	let governanceIntent: MutationIntent | null = null,
		transferIntent: MutationIntent | null = null;
	let governanceModal = $state(false),
		govReason = $state('');
	let newGovMode: 'solo_owner' | 'delegated_team' = $state('solo_owner');
	async function submitGovernanceChange() {
		if (busy) return;
		busy = true;
		error = '';
		message = '';
		try {
			governanceIntent ??= new MutationIntent('platform-governance', '/api/v1/ops/governance');
			await governanceIntent.run(
				{
					mode: newGovMode,
					reason: govReason
				},
				(value) => {
					const gov = record(record(value).governance);
					if (gov.mode !== newGovMode) throw new Error('Approval rule change was not confirmed');
					return gov;
				}
			);
			message =
				newGovMode === 'solo_owner'
					? 'You can now approve your own changes.'
					: 'Changes now need a second administrator.';
			governanceModal = false;
			govReason = '';
			await onchanged();
		} catch (e) {
			error = errorMessage(e, 'That change was not saved.');
		} finally {
			busy = false;
		}
	}

	let transferModal = $state(false);
	let transferTargetUserId = $state('');
	let transferReason = $state('');
	let transferConfirmed = $state(false);

	async function submitTransferOwnership() {
		if (busy) return;
		const targetUserId = transferTargetUserId.trim().toLowerCase();
		busy = true;
		error = '';
		message = '';
		try {
			transferIntent ??= new MutationIntent('platform-transfer', '/api/v1/ops/ownership/transfer');
			await transferIntent.run(
				{
					target_user_id: targetUserId,
					reason: transferReason,
					confirm: transferConfirmed
				},
				(value) => {
					const result = record(value);
					if (result.transferred !== true || result.new_owner_user_id !== targetUserId)
						throw new Error('Ownership transfer was not confirmed');
					return result;
				}
			);
			message = 'Ownership handed over.';
			transferModal = false;
			transferTargetUserId = '';
			transferReason = '';
			transferConfirmed = false;
			await onchanged();
		} catch (e) {
			error = errorMessage(e, 'Ownership was not handed over.');
		} finally {
			busy = false;
		}
	}
</script>

<div class="gov-pill-container">
	<div class="gov-badge {governance?.mode}">
		<span class="dot"></span>
		<strong
			>{!governance
				? 'Approval rules unavailable'
				: governance.mode === 'solo_owner'
					? 'Solo owner'
					: 'Delegated team'}</strong
		>
	</div>
	{#if isOwner}
		<button
			class="small-btn"
			disabled={busy || !governance}
			onclick={() => {
				if (!governance) return;
				newGovMode = governance.mode;
				governanceModal = true;
				govReason = '';
			}}>Change this</button
		>
		<button
			class="small-btn danger"
			disabled={busy}
			onclick={() => {
				transferModal = true;
				transferReason = '';
				transferConfirmed = false;
				transferTargetUserId = '';
			}}>Hand over ownership</button
		>
	{/if}
</div>
{#if governanceModal}
	<OwnerDialog
		{busy}
		open={governanceModal}
		title="Who approves changes"
		description="This decides whether you can approve your own changes, or whether a second administrator has to."
		onclose={() => (governanceModal = false)}
	>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				submitGovernanceChange();
			}}
		>
			<fieldset disabled={busy}>
				<div class="form-group">
					<span class="group-label">Choose one</span>
					<div class="mode-options">
						<label class="mode-radio">
							<input type="radio" name="gov-mode" value="solo_owner" bind:group={newGovMode} />
							<div>
								<strong>I run Kredit alone</strong>
								<p>
									You approve your own changes. Each one still needs a fresh authenticator code and a written reason.
								</p>
							</div>
						</label>
						<label class="mode-radio">
							<input type="radio" name="gov-mode" value="delegated_team" bind:group={newGovMode} />
							<div>
								<strong>A second person approves</strong>
								<p>
									Nobody approves their own change, including you. Choose this once you have a second administrator.
								</p>
							</div>
						</label>
					</div>
				</div>

				<div class="form-group">
					<label for="gov-reason">Why are you changing this?</label>
					<textarea
						id="gov-reason"
						rows="3"
						bind:value={govReason}
						placeholder="This is recorded permanently and cannot be edited later."
						required
						minlength="8"
						maxlength="2000"
					></textarea>
				</div>

				{#if error}<p role="alert" class="error">{error}</p>{/if}
				<div class="modal-actions">
					<button type="button" class="cancel-btn" onclick={() => (governanceModal = false)} disabled={busy}
						>Cancel</button
					>
					<button type="submit" class="save-btn" disabled={busy || govReason.trim().length < 8}>
						{busy ? 'Saving…' : 'Save this'}
					</button>
				</div>
			</fieldset>
		</form>
	</OwnerDialog>
{/if}

<!-- Ownership Transfer Modal -->
{#if transferModal}
	<OwnerDialog
		{busy}
		open={transferModal}
		title="Hand over ownership"
		description="The person you choose becomes the owner of Kredit and you stop being the owner. It happens straight away and it is recorded permanently."
		onclose={() => (transferModal = false)}
	>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				submitTransferOwnership();
			}}
		>
			<fieldset disabled={busy}>
				<div class="form-group">
					<label for="transfer-target">Who takes over</label>
					<input
						id="transfer-target"
						type="text"
						bind:value={transferTargetUserId}
						placeholder="Their Kredit user reference"
						aria-describedby="transfer-target-help"
						required
					/>
					<small id="transfer-target-help">Find it on the Users page, under the person's name.</small>
				</div>

				<div class="form-group">
					<label for="transfer-reason">Why are you handing it over?</label>
					<textarea
						id="transfer-reason"
						rows="3"
						bind:value={transferReason}
						placeholder="This is recorded permanently and cannot be edited later."
						required
						minlength="8"
						maxlength="2000"
					></textarea>
				</div>

				<div class="form-group checkbox-group">
					<label class="checkbox-label">
						<input type="checkbox" bind:checked={transferConfirmed} required />
						<span>I understand that I will no longer be the owner of Kredit.</span>
					</label>
				</div>

				{#if error}<p role="alert" class="error">{error}</p>{/if}
				<div class="modal-actions">
					<button type="button" class="cancel-btn" onclick={() => (transferModal = false)} disabled={busy}
						>Cancel</button
					>
					<button
						type="submit"
						class="danger-btn"
						disabled={busy || !transferConfirmed || transferReason.trim().length < 8 || !transferTargetUserId.trim()}
					>
						{busy ? 'Handing over…' : 'Hand over ownership'}
					</button>
				</div>
			</fieldset>
		</form>
	</OwnerDialog>
{/if}

<style>
	.gov-pill-container {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.5rem;
	}
	.gov-badge {
		padding: 0.5rem 0.75rem;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
	}
	fieldset {
		border: 0;
		padding: 0;
		display: grid;
		gap: 1rem;
		min-width: 0;
	}
	.form-group {
		display: grid;
		gap: 0.5rem;
	}
	.mode-options {
		display: grid;
		gap: 1rem;
	}
	.mode-radio,
	.checkbox-label {
		display: flex;
		gap: 0.75rem;
		align-items: start;
	}
	.mode-radio p {
		margin: 0.25rem 0 0;
	}
	.modal-actions {
		display: flex;
		justify-content: flex-end;
		flex-wrap: wrap;
		gap: 0.75rem;
	}
	input:not([type='radio']):not([type='checkbox']),
	textarea {
		width: 100%;
	}
	button {
		min-height: 44px;
	}
	.danger,
	.danger-btn {
		color: var(--color-error);
	}
</style>
