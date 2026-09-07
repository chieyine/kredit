<script lang="ts">
	import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
	import { onMount } from 'svelte';
	import { adminGet, adminPost, localTime } from '$lib/admin-client';

	type Setting = {
		key: string;
		category: string;
		value: any;
		is_secret: boolean;
		secret_fingerprint?: string;
		description: string;
		version: number;
		updated_at: string;
		updated_by?: string;
		reason?: string;
	};

	type Governance = {
		mode: 'solo_owner' | 'delegated_team';
		updated_at: string;
		updated_by?: string;
		reason: string;
	};

	type SettingHistory = {
		id: string;
		key: string;
		old_value?: any;
		new_value: any;
		version: number;
		action: string;
		actor_id?: string;
		reason: string;
		recorded_at: string;
	};

	const categories = [
		{ id: 'all', label: 'All Settings' },
		{ id: 'launch', label: 'Launch & Visibility' },
		{ id: 'features', label: 'Features & Switches' },
		{ id: 'financial_caps', label: 'Financial & Policy Caps' },
		{ id: 'risk_limits', label: 'Risk Limits' },
		{ id: 'integrations', label: 'Integrations & Credentials' },
		{ id: 'security', label: 'Security & Auth' },
		{ id: 'legal', label: 'Legal & Compliance' },
		{ id: 'notifications', label: 'Notifications' }
	];

	let settings: Setting[] = $state([]);
	let governance: Governance = $state({ mode: 'solo_owner', updated_at: '', reason: '' });
	let selectedCategory = $state('all');
	let searchQuery = $state('');
	let loading = $state(true);
	let busy = $state(false);
	let error = $state('');
	let message = $state('');
	let isOwner = $state(false);

	// Modals & Drafts
	let editingSetting: Setting | null = $state(null);
	let editDraftValue: any = $state(null);
	let editReason = $state('');
	let previewDiffModal = $state(false);

	let secretRotateSetting: Setting | null = $state(null);
	let newSecretValue = $state('');
	let secretRotateReason = $state('');

	let historySettingKey: string | null = $state(null);
	let historyEntries: SettingHistory[] = $state([]);
	let historyLoading = $state(false);

	let governanceModal = $state(false);
	let newGovMode: 'solo_owner' | 'delegated_team' = $state('solo_owner');
	let govReason = $state('');

	let filteredSettings = $derived.by(() => {
		return settings.filter(s => {
			const matchCat = selectedCategory === 'all' || s.category === selectedCategory;
			const q = searchQuery.toLowerCase().trim();
			const matchQuery = !q || s.key.toLowerCase().includes(q) || s.description.toLowerCase().includes(q);
			return matchCat && matchQuery;
		});
	});

	async function load() {
		loading = true;
		error = '';
		try {
			const [caps, res] = await Promise.all([
				adminGet('/api/v1/ops/capabilities'),
				adminGet('/api/v1/ops/platform-settings')
			]);
			isOwner = caps.roles?.includes('platform_owner') || false;
			settings = res.settings || [];
			if (res.governance) {
				governance = res.governance;
				newGovMode = res.governance.mode;
			}
		} catch (e: any) {
			error = e.message || 'Failed to load platform settings';
		} finally {
			loading = false;
		}
	}

	function openEdit(setting: Setting) {
		editingSetting = setting;
		editDraftValue = JSON.parse(JSON.stringify(setting.value));
		editReason = '';
		previewDiffModal = true;
	}

	async function submitSettingUpdate() {
		if (!editingSetting) return;
		busy = true;
		error = '';
		message = '';
		try {
			await adminPost('/api/v1/ops/platform-settings', {
				key: editingSetting.key,
				value: editDraftValue,
				reason: editReason
			});
			message = `Setting ${editingSetting.key} updated successfully.`;
			previewDiffModal = false;
			editingSetting = null;
			await load();
		} catch (e: any) {
			error = e.message || 'Failed to update setting';
		} finally {
			busy = false;
		}
	}

	function openSecretRotate(setting: Setting) {
		secretRotateSetting = setting;
		newSecretValue = '';
		secretRotateReason = '';
	}

	async function submitSecretRotate() {
		if (!secretRotateSetting) return;
		busy = true;
		error = '';
		message = '';
		try {
			await adminPost('/api/v1/ops/platform-settings/secret', {
				key: secretRotateSetting.key,
				secret: newSecretValue,
				reason: secretRotateReason
			});
			message = `Secret for ${secretRotateSetting.key} rotated securely.`;
			secretRotateSetting = null;
			newSecretValue = '';
			secretRotateReason = '';
			await load();
		} catch (e: any) {
			error = e.message || 'Failed to rotate secret';
		} finally {
			busy = false;
		}
	}

	async function openHistory(key: string) {
		historySettingKey = key;
		historyLoading = true;
		historyEntries = [];
		try {
			const res = await adminGet(`/api/v1/ops/platform-settings/history?key=${encodeURIComponent(key)}`);
			historyEntries = res.history || [];
		} catch (e: any) {
			error = e.message || 'Failed to load setting history';
		} finally {
			historyLoading = false;
		}
	}

	async function submitGovernanceChange() {
		busy = true;
		error = '';
		message = '';
		try {
			await adminPost('/api/v1/ops/governance', {
				mode: newGovMode,
				reason: govReason
			});
			message = `Platform governance mode changed to ${newGovMode === 'solo_owner' ? 'Solo Owner Mode' : 'Delegated Team Mode'}.`;
			governanceModal = false;
			govReason = '';
			await load();
		} catch (e: any) {
			error = e.message || 'Failed to update governance mode';
		} finally {
			busy = false;
		}
	}

	let transferModal = $state(false);
	let transferTargetUserId = $state('');
	let transferReason = $state('');
	let transferConfirmed = $state(false);

	async function submitTransferOwnership() {
		busy = true;
		error = '';
		message = '';
		try {
			await adminPost('/api/v1/ops/ownership/transfer', {
				target_user_id: transferTargetUserId,
				reason: transferReason,
				confirm: transferConfirmed
			});
			message = 'Platform ownership transferred successfully.';
			transferModal = false;
			transferTargetUserId = '';
			transferReason = '';
			transferConfirmed = false;
			await load();
		} catch (e: any) {
			error = e.message || 'Failed to transfer ownership';
		} finally {
			busy = false;
		}
	}

	onMount(load);
</script>

<svelte:head>
	<title>Platform Settings & Launch Controls — Kredit Admin</title>
</svelte:head>

<main class="shell workspace">
	<div class="header-row">
		<div>
			<p class="eyebrow">Super Admin / Platform Controls</p>
			<h1>Platform Settings & Launch Registry</h1>
		</div>
		<div class="gov-pill-container">
			<div class="gov-badge {governance.mode}">
				<span class="dot"></span>
				<strong>{governance.mode === 'solo_owner' ? 'Solo-Owner Mode' : 'Delegated Team Mode'}</strong>
			</div>
			{#if isOwner}
				<button class="small-btn" onclick={() => { governanceModal = true; govReason = ''; }}>Change Mode</button>
				<button class="small-btn danger" onclick={() => { transferModal = true; transferReason = ''; transferConfirmed = false; transferTargetUserId = ''; }}>Transfer Ownership</button>
			{/if}
		</div>
	</div>

	<VerifyIdentity />

	<p class="subhead">
		Comprehensive database-backed configuration for launch stages, feature gates, risk parameters, integration secrets, and operational policies. All mutations require step-up MFA, are strictly typed, and maintain an immutable versioned audit history.
	</p>

	{#if error}<p role="alert" class="alert error">{error}</p>{/if}
	{#if message}<p role="status" class="alert success">{message}</p>{/if}

	<!-- Governance & Mode Banner -->
	<div class="banner {governance.mode === 'solo_owner' ? 'banner-warning' : 'banner-info'}">
		<div class="banner-title">
			<strong>{governance.mode === 'solo_owner' ? 'Single-Owner Autonomous Mode Active' : 'Delegated Team Mode Active'}</strong>
		</div>
		<p>
			{#if governance.mode === 'solo_owner'}
				Platform Owner is permitted to execute self-approvals on administrative and policy changes with explicit confirmation and mandatory audit justifications. Maker-checker is maintained for all non-owner roles.
			{:else}
				Strict four-eyes separation enforced across all administrative and policy mutations. No administrator or owner may approve their own proposed changes.
			{/if}
		</p>
	</div>

	<!-- Controls & Category Filter -->
	<div class="toolbar">
		<div class="search-box">
			<input type="search" placeholder="Search settings or description..." bind:value={searchQuery} />
		</div>
		<div class="category-tabs">
			{#each categories as cat}
				<button 
					class="tab-btn {selectedCategory === cat.id ? 'active' : ''}" 
					onclick={() => selectedCategory = cat.id}
				>
					{cat.label}
				</button>
			{/each}
		</div>
	</div>

	<!-- Settings Table -->
	{#if loading}
		<div class="loading-state">Loading platform registry settings...</div>
	{:else if filteredSettings.length === 0}
		<div class="empty-state">No settings match the current filter or search criteria.</div>
	{:else}
		<div class="table-wrap">
			<table class="settings-table">
				<thead>
					<tr>
						<th>Key & Description</th>
						<th>Category</th>
						<th>Current Value / Status</th>
						<th>Version</th>
						<th>Last Updated</th>
						<th>Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each filteredSettings as s (s.key)}
						<tr class="setting-row">
							<td class="key-col">
								<span class="setting-key">{s.key}</span>
								<p class="setting-desc">{s.description}</p>
							</td>
							<td>
								<span class="category-tag">{s.category}</span>
							</td>
							<td class="val-col">
								{#if s.is_secret}
									<div class="secret-val">
										<code>{typeof s.value === 'string' ? s.value : '••••••••'}</code>
										{#if s.secret_fingerprint}
											<span class="fingerprint" title="SHA-256 Fingerprint">SHA: {s.secret_fingerprint.slice(0, 10)}…</span>
										{/if}
									</div>
								{:else if typeof s.value === 'boolean'}
									<span class="bool-tag {s.value ? 'enabled' : 'disabled'}">
										{s.value ? 'Enabled' : 'Disabled'}
									</span>
								{:else if typeof s.value === 'object'}
									<code class="json-snippet">{JSON.stringify(s.value)}</code>
								{:else}
									<code>{String(s.value)}</code>
								{/if}
							</td>
							<td class="center">v{s.version}</td>
							<td>
								<span class="date">{localTime(s.updated_at)}</span>
								{#if s.reason}<p class="reason-hint">"{s.reason}"</p>{/if}
							</td>
							<td class="actions-col">
								{#if s.is_secret}
									<button class="action-btn" onclick={() => openSecretRotate(s)} disabled={!isOwner}>
										Rotate Secret
									</button>
								{:else}
									<button class="action-btn" onclick={() => openEdit(s)} disabled={!isOwner}>
										Edit Value
									</button>
								{/if}
								<button class="action-btn text-btn" onclick={() => openHistory(s.key)}>
									History
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<!-- Edit Modal with Diff Preview -->
	{#if previewDiffModal && editingSetting}
		<div class="modal-overlay" role="dialog" aria-modal="true">
			<div class="modal-card">
				<h2>Edit Platform Setting</h2>
				<p class="mono">{editingSetting.key}</p>
				<p class="desc">{editingSetting.description}</p>

				<form onsubmit={(e) => { e.preventDefault(); submitSettingUpdate(); }}>
					<div class="form-group">
						<label for="edit-val">New Value</label>
						{#if typeof editingSetting.value === 'boolean'}
							<select id="edit-val" bind:value={editDraftValue}>
								<option value={true}>true (Enabled)</option>
								<option value={false}>false (Disabled)</option>
							</select>
						{:else if typeof editingSetting.value === 'number'}
							<input id="edit-val" type="number" step="any" bind:value={editDraftValue} required />
						{:else if typeof editingSetting.value === 'string'}
							<input id="edit-val" type="text" bind:value={editDraftValue} required />
						{:else}
							<textarea id="edit-val" rows="4" bind:value={editDraftValue}></textarea>
						{/if}
					</div>

					<div class="diff-preview">
						<h3>Change Diff Preview</h3>
						<div class="diff-row before">
							<span class="diff-label">Current:</span>
							<code>{JSON.stringify(editingSetting.value)}</code>
						</div>
						<div class="diff-row after">
							<span class="diff-label">Proposed:</span>
							<code>{JSON.stringify(editDraftValue)}</code>
						</div>
					</div>

					<div class="form-group">
						<label for="edit-reason">Audit Reason (Mandatory, min 4 characters)</label>
						<input id="edit-reason" type="text" bind:value={editReason} placeholder="State the reason for this operational change" required minlength="4" />
					</div>

					<div class="modal-actions">
						<button type="button" class="cancel-btn" onclick={() => previewDiffModal = false} disabled={busy}>Cancel</button>
						<button type="submit" class="save-btn" disabled={busy || editReason.trim().length < 4 || JSON.stringify(editingSetting.value) === JSON.stringify(editDraftValue)}>
							{busy ? 'Saving...' : 'Apply Setting Change'}
						</button>
					</div>
				</form>
			</div>
		</div>
	{/if}

	<!-- Secret Rotation Modal -->
	{#if secretRotateSetting}
		<div class="modal-overlay" role="dialog" aria-modal="true">
			<div class="modal-card">
				<h2>Rotate Integration Credential</h2>
				<p class="mono">{secretRotateSetting.key}</p>
				<p class="desc">{secretRotateSetting.description}</p>
				<div class="security-note">
					Secret will be encrypted with AES-256-GCM prior to storage in PostgreSQL. The plain value will never be viewable again; only masked suffixes and SHA-256 fingerprints are preserved for verification.
				</div>

				<form onsubmit={(e) => { e.preventDefault(); submitSecretRotate(); }}>
					<div class="form-group">
						<label for="new-sec">New Plaintext Secret Value</label>
						<input id="new-sec" type="password" autocomplete="new-password" bind:value={newSecretValue} placeholder="Enter raw API key or token..." required />
					</div>

					<div class="form-group">
						<label for="sec-reason">Rotation Justification (Mandatory)</label>
						<input id="sec-reason" type="text" bind:value={secretRotateReason} placeholder="e.g., Scheduled quarterly credential rotation" required minlength="4" />
					</div>

					<div class="modal-actions">
						<button type="button" class="cancel-btn" onclick={() => secretRotateSetting = null} disabled={busy}>Cancel</button>
						<button type="submit" class="save-btn warn" disabled={busy || !newSecretValue || secretRotateReason.trim().length < 4}>
							{busy ? 'Rotating...' : 'Encrypt & Rotate Secret'}
						</button>
					</div>
				</form>
			</div>
		</div>
	{/if}

	<!-- Governance Switcher Modal -->
	{#if governanceModal}
		<div class="modal-overlay" role="dialog" aria-modal="true">
			<div class="modal-card">
				<h2>Platform Governance Mode</h2>
				<p class="desc">Configure maker-checker constraints for solo owner operations vs delegated team operations.</p>

				<form onsubmit={(e) => { e.preventDefault(); submitGovernanceChange(); }}>
					<div class="form-group">
						<span class="group-label">Select Operational Mode</span>
						<div class="mode-options">
							<label class="mode-radio">
								<input type="radio" name="gov-mode" value="solo_owner" bind:group={newGovMode} />
								<div>
									<strong>Solo-Owner Mode</strong>
									<p>Allows platform owner self-approval with explicit confirmation and audit justifications.</p>
								</div>
							</label>
							<label class="mode-radio">
								<input type="radio" name="gov-mode" value="delegated_team" bind:group={newGovMode} />
								<div>
									<strong>Delegated Team Mode</strong>
									<p>Enforces strict four-eyes separation. Every change requires an independent second administrator.</p>
								</div>
							</label>
						</div>
					</div>

					<div class="form-group">
						<label for="gov-reason">Audit Reason for Mode Change (Mandatory)</label>
						<input id="gov-reason" type="text" bind:value={govReason} placeholder="e.g., Transitioning to multi-operator production governance" required minlength="4" />
					</div>

					<div class="modal-actions">
						<button type="button" class="cancel-btn" onclick={() => governanceModal = false} disabled={busy}>Cancel</button>
						<button type="submit" class="save-btn" disabled={busy || govReason.trim().length < 4}>
							{busy ? 'Saving...' : 'Confirm Governance Change'}
						</button>
					</div>
				</form>
			</div>
		</div>
	{/if}

	<!-- History Drawer / Modal -->
	{#if historySettingKey}
		<div class="modal-overlay" role="dialog" aria-modal="true">
			<div class="modal-card wide">
				<div class="modal-header">
					<div>
						<h2>Audit History</h2>
						<p class="mono">{historySettingKey}</p>
					</div>
					<button class="close-btn" onclick={() => historySettingKey = null}>✕</button>
				</div>

				{#if historyLoading}
					<div class="loading-state">Loading version history...</div>
				{:else if historyEntries.length === 0}
					<div class="empty-state">No recorded history entries for this key.</div>
				{:else}
					<div class="table-wrap">
						<table class="history-table">
							<thead>
								<tr>
									<th>Version</th>
									<th>Action</th>
									<th>New Value</th>
									<th>Recorded At</th>
									<th>Reason & Actor</th>
								</tr>
							</thead>
							<tbody>
								{#each historyEntries as h}
									<tr>
										<td>v{h.version}</td>
										<td><span class="action-tag">{h.action}</span></td>
										<td>
											<code class="json-snippet">{JSON.stringify(h.new_value)}</code>
										</td>
										<td>{localTime(h.recorded_at)}</td>
										<td>
											<strong>{h.reason}</strong>
											{#if h.actor_id}<p class="actor-sub">Actor: {h.actor_id}</p>{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</div>
		</div>
	{/if}

	<!-- Ownership Transfer Modal -->
	{#if transferModal}
		<div class="modal-overlay" role="dialog" aria-modal="true">
			<div class="modal-card">
				<div class="modal-header">
					<h2>Audited Ownership Transfer</h2>
					<button class="close-btn" onclick={() => transferModal = false}>✕</button>
				</div>
				<p class="modal-desc">
					Transfer platform owner credentials to another active registered operator. This grants them primary ownership authority and revokes your platform owner role, recording an immutable audit trail.
				</p>

				<form onsubmit={(e) => { e.preventDefault(); submitTransferOwnership(); }}>
					<div class="form-group">
						<label for="transfer-target">Target User ID (UUID)</label>
						<input id="transfer-target" type="text" bind:value={transferTargetUserId} placeholder="e.g. 00000000-0000-0000-0000-000000000000" required />
					</div>

					<div class="form-group">
						<label for="transfer-reason">Audit Justification</label>
						<input id="transfer-reason" type="text" bind:value={transferReason} placeholder="e.g. Planned operator succession" required minlength="8" />
					</div>

					<div class="form-group checkbox-group">
						<label class="checkbox-label">
							<input type="checkbox" bind:checked={transferConfirmed} required />
							<span>I confirm I wish to transfer platform ownership. This action is audited and takes effect immediately.</span>
						</label>
					</div>

					<div class="modal-actions">
						<button type="button" class="cancel-btn" onclick={() => transferModal = false} disabled={busy}>Cancel</button>
						<button type="submit" class="danger-btn" disabled={busy || !transferConfirmed || transferReason.trim().length < 8 || !transferTargetUserId.trim()}>
							{busy ? 'Transferring...' : 'Confirm Ownership Transfer'}
						</button>
					</div>
				</form>
			</div>
		</div>
	{/if}
</main>

<style>
	main {
		max-width: 1200px;
		margin: 0 auto;
		padding: 2rem 1rem;
	}

	.header-row {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		flex-wrap: wrap;
		gap: 1rem;
		margin-bottom: 1rem;
	}

	.subhead {
		color: #4a4a4a;
		font-size: 0.95rem;
		line-height: 1.5;
		margin-bottom: 1.5rem;
	}

	.gov-pill-container {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.gov-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.4rem 0.8rem;
		border-radius: 2rem;
		font-size: 0.85rem;
		font-weight: 600;
	}

	.gov-badge.solo_owner {
		background: #fef3c7;
		color: #92400e;
		border: 1px solid #fde68a;
	}

	.gov-badge.delegated_team {
		background: #e0e7ff;
		color: #3730a3;
		border: 1px solid #c7d2fe;
	}

	.dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: currentColor;
	}

	.banner {
		padding: 1rem 1.25rem;
		border-radius: 0.6rem;
		margin-bottom: 1.5rem;
		font-size: 0.9rem;
		line-height: 1.4;
	}

	.banner-warning {
		background: #fffbeb;
		border: 1px solid #fcd34d;
		color: #78350f;
	}

	.banner-info {
		background: #eff6ff;
		border: 1px solid #93c5fd;
		color: #1e3a8a;
	}

	.toolbar {
		margin-bottom: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.search-box input {
		width: 100%;
		padding: 0.75rem 1rem;
		border: 1px solid #d1d5db;
		border-radius: 0.5rem;
		font-size: 0.95rem;
	}

	.category-tabs {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.tab-btn {
		background: #f3f4f6;
		border: 1px solid #e5e7eb;
		padding: 0.4rem 0.8rem;
		border-radius: 0.4rem;
		font-size: 0.85rem;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s;
	}

	.tab-btn:hover {
		background: #e5e7eb;
	}

	.tab-btn.active {
		background: #111827;
		color: #ffffff;
		border-color: #111827;
	}

	.table-wrap {
		overflow-x: auto;
		background: #ffffff;
		border: 1px solid #e5e7eb;
		border-radius: 0.75rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
	}

	.settings-table, .history-table {
		width: 100%;
		border-collapse: collapse;
		text-align: left;
		font-size: 0.9rem;
	}

	th {
		background: #f9fafb;
		padding: 0.8rem 1rem;
		border-bottom: 1px solid #e5e7eb;
		font-weight: 600;
		color: #374151;
	}

	td {
		padding: 0.8rem 1rem;
		border-bottom: 1px solid #f3f4f6;
		vertical-align: top;
	}

	.key-col {
		max-width: 280px;
	}

	.setting-key {
		font-family: monospace;
		font-weight: 600;
		color: #111827;
		display: block;
		font-size: 0.9rem;
	}

	.setting-desc {
		color: #6b7280;
		font-size: 0.8rem;
		margin-top: 0.2rem;
		line-height: 1.3;
	}

	.category-tag {
		display: inline-block;
		padding: 0.2rem 0.5rem;
		background: #f3f4f6;
		color: #4b5563;
		border-radius: 0.3rem;
		font-size: 0.75rem;
		text-transform: capitalize;
	}

	.val-col code {
		background: #f3f4f6;
		padding: 0.2rem 0.4rem;
		border-radius: 0.25rem;
		font-size: 0.85rem;
	}

	.secret-val {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.fingerprint {
		font-size: 0.75rem;
		color: #6b7280;
		font-family: monospace;
	}

	.bool-tag {
		display: inline-block;
		padding: 0.2rem 0.5rem;
		border-radius: 0.25rem;
		font-weight: 600;
		font-size: 0.8rem;
	}

	.bool-tag.enabled {
		background: #dcfce7;
		color: #166534;
	}

	.bool-tag.disabled {
		background: #fee2e2;
		color: #991b1b;
	}

	.actions-col {
		display: flex;
		gap: 0.4rem;
		align-items: center;
	}

	.action-btn {
		padding: 0.35rem 0.7rem;
		font-size: 0.8rem;
		font-weight: 600;
		border-radius: 0.35rem;
		border: 1px solid #d1d5db;
		background: #ffffff;
		cursor: pointer;
	}

	.action-btn:hover:not(:disabled) {
		background: #f9fafb;
		border-color: #9ca3af;
	}

	.text-btn {
		border: none;
		background: none;
		color: #2563eb;
	}

	.text-btn:hover {
		text-decoration: underline;
		background: none;
	}

	.date {
		font-size: 0.8rem;
		color: #6b7280;
		display: block;
	}

	.reason-hint {
		font-size: 0.75rem;
		color: #9ca3af;
		font-style: italic;
		margin-top: 0.2rem;
	}

	/* Modals */
	.modal-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
		padding: 1rem;
	}

	.modal-card {
		background: #ffffff;
		border-radius: 0.8rem;
		padding: 1.5rem;
		max-width: 540px;
		width: 100%;
		max-height: 90vh;
		overflow-y: auto;
		box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
	}

	.modal-card.wide {
		max-width: 850px;
	}

	.modal-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: 1rem;
	}

	.close-btn {
		background: none;
		border: none;
		font-size: 1.25rem;
		cursor: pointer;
		padding: 0.2rem 0.5rem;
	}

	.form-group {
		margin: 1.25rem 0;
	}

	.form-group label {
		display: block;
		font-weight: 600;
		margin-bottom: 0.4rem;
		font-size: 0.85rem;
	}

	.form-group input, .form-group select, .form-group textarea {
		width: 100%;
		padding: 0.65rem;
		border: 1px solid #d1d5db;
		border-radius: 0.4rem;
		font-size: 0.9rem;
	}

	.diff-preview {
		background: #f9fafb;
		padding: 1rem;
		border-radius: 0.5rem;
		border: 1px solid #e5e7eb;
		margin: 1rem 0;
	}

	.diff-preview h3 {
		font-size: 0.85rem;
		margin-bottom: 0.5rem;
		text-transform: uppercase;
		color: #6b7280;
	}

	.diff-row {
		display: flex;
		gap: 0.5rem;
		padding: 0.35rem 0;
		font-size: 0.85rem;
	}

	.diff-row.before {
		color: #dc2626;
	}

	.diff-row.after {
		color: #16a34a;
	}

	.security-note {
		background: #eff6ff;
		border: 1px solid #bfdbfe;
		color: #1e40af;
		padding: 0.75rem;
		border-radius: 0.4rem;
		font-size: 0.8rem;
		margin: 1rem 0;
		line-height: 1.4;
	}

	.mode-options {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.mode-radio {
		display: flex;
		gap: 0.75rem;
		align-items: flex-start;
		border: 1px solid #e5e7eb;
		padding: 0.75rem;
		border-radius: 0.5rem;
		cursor: pointer;
	}

	.mode-radio p {
		font-size: 0.8rem;
		color: #6b7280;
		margin-top: 0.2rem;
	}

	.modal-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
		margin-top: 1.5rem;
	}

	.save-btn {
		background: #111827;
		color: #ffffff;
		border: none;
		padding: 0.6rem 1.2rem;
		border-radius: 0.4rem;
		font-weight: 600;
		cursor: pointer;
	}

	.save-btn.warn {
		background: #b91c1c;
	}

	.save-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.cancel-btn {
		background: #f3f4f6;
		border: 1px solid #d1d5db;
		padding: 0.6rem 1.2rem;
		border-radius: 0.4rem;
		cursor: pointer;
	}

	.small-btn {
		background: #f3f4f6;
		border: 1px solid #d1d5db;
		padding: 0.35rem 0.75rem;
		border-radius: 0.375rem;
		font-size: 0.8rem;
		font-weight: 500;
		cursor: pointer;
	}

	.small-btn.danger {
		background: #fee2e2;
		color: #991b1b;
		border-color: #fca5a5;
	}

	.danger-btn {
		background: #dc2626;
		color: #ffffff;
		border: none;
		padding: 0.6rem 1.2rem;
		border-radius: 0.4rem;
		font-weight: 600;
		cursor: pointer;
	}

	.danger-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.checkbox-group {
		margin: 1rem 0;
	}

	.checkbox-label {
		display: flex;
		align-items: flex-start;
		gap: 0.5rem;
		font-size: 0.85rem;
		color: #374151;
		cursor: pointer;
	}

	.checkbox-label input {
		width: auto;
		margin-top: 0.2rem;
	}

	.alert {
		padding: 0.75rem 1rem;
		border-radius: 0.4rem;
		margin-bottom: 1rem;
	}

	.alert.error {
		background: #fee2e2;
		color: #991b1b;
		border: 1px solid #fca5a5;
	}

	.alert.success {
		background: #dcfce7;
		color: #166534;
		border: 1px solid #86efac;
	}

	.mono {
		font-family: monospace;
		font-weight: 600;
	}

	.desc {
		color: #6b7280;
		font-size: 0.85rem;
	}

	.empty-state, .loading-state {
		text-align: center;
		padding: 3rem 1rem;
		color: #6b7280;
	}
</style>
