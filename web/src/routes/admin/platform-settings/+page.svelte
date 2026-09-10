<script lang="ts">
	import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';
	import OwnerDialog from '$lib/components/OwnerDialog.svelte';
	import { MutationIntent } from '$lib/api/mutation';
	import { record, text } from '$lib/api/reliable';
	import { onMount } from 'svelte';
	import { adminGet, localTime } from '$lib/admin-client';

	type Setting = {
		key: string;
        is_secret?: boolean;
        connection_state?: string;
        requires_restart?: boolean;
        applied_version?: number;
        connection_fields?: {key:string;label:string;kind:string}[];
        connection_values?: Record<string,string|number|boolean>;
		category: string;
		value: any;
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

 const categories = [{ id: 'all', label: 'All settings' }, { id: 'features', label: 'Features' }, { id: 'integrations', label: 'Connections' }];
 let clearRuntimeSecrets = $state(false);
 let runtimeDraft = $state<Record<string,string|number|boolean>>({});
 let connectorEnabled = $state(true);
 let connectorEndpoint = $state('');
 let connectorToken = $state('');
 function clearConnector() { connectorEndpoint = ''; connectorToken = ''; runtimeDraft = {}; clearRuntimeSecrets=false; }

	let settingIntent: MutationIntent | null = null;
 let governanceIntent: MutationIntent | null = null;
 let transferIntent: MutationIntent | null = null;
 let settings: Setting[] = $state([]);
	let governance: Governance | null = $state(null);
	let selectedCategory = $state('all');
	let searchQuery = $state('');
	let loading = $state(true);
	let busy = $state(false);
	let error = $state('');
	let message = $state('');
	let isOwner = $state(false);

	// Modals & Drafts
	// One place decides how a stored value reads on screen, so the table, the
	// diff and the history cannot describe the same value three different ways.
	function readable(value: unknown): string {
		if (typeof value === 'boolean') return value ? 'On' : 'Off';
		if (value === null || value === undefined) return 'Not set';
		if (typeof value === 'object') return 'Structured data';
		return String(value);
	}

	let editingSetting: Setting | null = $state(null);
	let editDraftValue: any = $state(null);
	let editReason = $state('');
	let previewDiffModal = $state(false);

	let historySettingKey: string | null = $state(null);
	let historyEntries: SettingHistory[] = $state([]);
	let historyLoading = $state(false);
    let historyGeneration = 0;
    let historyError = $state('');

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
        governance = null; isOwner = false; settings = [];
		try {
			const [caps, res] = await Promise.all([
				adminGet('/api/v1/ops/capabilities'),
				adminGet('/api/v1/ops/platform-settings')
			]);
			if(!Array.isArray(caps.roles)||caps.roles.some((role:unknown)=>typeof role!=='string'))throw new Error('Admin permissions could not be verified.');
			isOwner = caps.roles.includes('platform_owner');
			if (!Array.isArray(res.settings) || !['solo_owner', 'delegated_team'].includes(res.governance?.mode)) throw new Error('Settings and approval rules could not be verified. Try again.');
            const keys=new Set<string>();
            settings = res.settings.map((value:unknown)=>{
             const item=record(value);
             for(const key of ['key','description','category'])text(item[key]);
             if(keys.has(item.key as string)||!Number.isSafeInteger(item.version)||Number(item.version)<0||!('value' in item))throw new Error('Settings could not be verified.');
             keys.add(item.key as string);
             if(item.requires_restart){
              if(!Array.isArray(item.connection_fields))throw new Error('Connection fields unavailable.');
              for(const value of item.connection_fields){const field=record(value);text(field.key);text(field.label);if(!['password','boolean','number','url','text'].includes(text(field.kind)))throw new Error('Unsupported connection field.');}
             }
             return item as Setting;
            });
			if (res.governance) {
				governance = res.governance;
				newGovMode = res.governance?.mode;
			}
		} catch (e: any) {
			governance=null;isOwner=false;settings=[];
			error = e.message || 'We could not open the settings. Try again.';
		} finally {
			loading = false;
		}
	}

	function openEdit(setting: Setting) {
		editingSetting = setting;
        settingIntent = new MutationIntent(`platform-setting:${setting.key}`, '/api/v1/ops/platform-settings');
        clearConnector(); connectorEnabled = true;
        if(setting.requires_restart){runtimeDraft=Object.fromEntries((setting.connection_fields||[]).map(field=>[field.key,field.kind==='password'?'':setting.connection_values?.[field.key]??(field.kind==='boolean'?false:field.kind==='number'?0:'')]))}
		editDraftValue = JSON.parse(JSON.stringify(setting.value));
		editReason = '';
		previewDiffModal = true;
	}

	async function submitSettingUpdate() {
		if (!editingSetting || !settingIntent || busy) return;
		const expectedSetting=editingSetting;
		busy = true;
		error = '';
		message = '';
		try {
			await settingIntent.run({
				key: editingSetting.key,
                clear_credentials: editingSetting.requires_restart && clearRuntimeSecrets,
 expected_version: editingSetting.version,
				value: editingSetting.requires_restart ? JSON.stringify(runtimeDraft) : editingSetting.is_secret ? JSON.stringify({ enabled: connectorEnabled, endpoint: connectorEnabled ? connectorEndpoint.trim() : "", token: connectorEnabled ? connectorToken.trim() : "" }) : editDraftValue,
				reason: editReason
			}, value => {const item=record(record(value).setting);if(item.key!==expectedSetting.key||!Number.isSafeInteger(item.version)||Number(item.version)<=expectedSetting.version)throw new Error('Setting save was not confirmed');return item;});
			message = editingSetting.requires_restart ? 'Connection saved. Restart the API and worker to apply it; live operation still needs verification.' : `${editingSetting.description} — saved.`;
			previewDiffModal = false;
            clearConnector();
			editingSetting = null;
			await load();
		} catch (e: any) {
			error = e.message || 'That change was not saved.';
		} finally {
			busy = false;
		}
	}


	async function openHistory(key: string) {
        const generation = ++historyGeneration;
		historySettingKey = key;
		historyLoading = true;
		historyEntries = []; historyError = '';
		try {
			const res = await adminGet(`/api/v1/ops/platform-settings/history?key=${encodeURIComponent(key)}`);
			if (generation !== historyGeneration || historySettingKey !== key) return;
			if (!Array.isArray(res.history)) throw new Error('History could not be verified.');
            historyEntries = res.history;
		} catch (e: any) {
			if (generation === historyGeneration && historySettingKey === key) historyError = e.message || 'We could not open the history for this setting.';
		} finally {
			if (generation === historyGeneration) historyLoading = false;
		}
	}

	async function submitGovernanceChange() {
        if (busy) return;
		busy = true;
		error = '';
		message = '';
		try {
			governanceIntent ??= new MutationIntent('platform-governance','/api/v1/ops/governance');
			await governanceIntent.run({
				mode: newGovMode,
				reason: govReason
			},value=>{const gov=record(record(value).governance);if(gov.mode!==newGovMode)throw new Error('Approval rule change was not confirmed');return gov;});
			message = newGovMode === 'solo_owner' ? 'You can now approve your own changes.' : 'Changes now need a second administrator.';
			governanceModal = false;
			govReason = '';
			await load();
		} catch (e: any) {
			error = e.message || 'That change was not saved.';
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
			transferIntent ??= new MutationIntent('platform-transfer','/api/v1/ops/ownership/transfer');
			await transferIntent.run({
				target_user_id: targetUserId,
				reason: transferReason,
				confirm: transferConfirmed
			},value=>{const result=record(value);if(result.transferred!==true||result.new_owner_user_id!==targetUserId)throw new Error('Ownership transfer was not confirmed');return result;});
			message = 'Ownership handed over.';
			transferModal = false;
			transferTargetUserId = '';
			transferReason = '';
			transferConfirmed = false;
			await load();
		} catch (e: any) {
			error = e.message || 'Ownership was not handed over.';
		} finally {
			busy = false;
		}
	}

	onMount(load);
</script>

<svelte:head>
	<title>Platform settings — Kredit admin</title>
</svelte:head>

<main class="shell workspace">
	<div class="header-row">
		<div>
			<p class="eyebrow">Admin / Platform</p>
			<h1>Platform settings</h1>
		</div>
		<div class="gov-pill-container">
			<div class="gov-badge {governance?.mode}">
				<span class="dot"></span>
				<strong>{!governance ? 'Approval rules unavailable' : governance.mode === 'solo_owner' ? 'Solo owner' : 'Delegated team'}</strong>
			</div>
			{#if isOwner}
				<button class="small-btn" onclick={() => { governanceModal = true; govReason = ''; }}>Change this</button>
				<button class="small-btn danger" onclick={() => { transferModal = true; transferReason = ''; transferConfirmed = false; transferTargetUserId = ''; }}>Hand over ownership</button>
			{/if}
		</div>
	</div>

	<VerifyIdentity />

	<p class="subhead">
		Manage product features and provider connections. Every change needs a fresh authenticator code and a reason, and the history cannot be edited afterwards.
	</p>

	{#if error}<p role="alert" class="alert error">{error}</p>{/if}
	{#if message}<p role="status" class="alert success">{message}</p>{/if}

	{#if governance}<div class="banner banner-info">
		<div class="banner-title">
			<strong>{governance?.mode === 'solo_owner' ? 'You run Kredit alone' : 'Changes need a second person'}</strong>
		</div>
		<p>
			{#if governance?.mode === 'solo_owner'}
				You can approve your own financial and business-policy proposals. Each approval needs a fresh authenticator code and a written reason. Platform settings and provider connections are managed directly by the owner.
			{:else}
				Financial and business-policy proposals need a different administrator to approve them. Platform settings and provider connections are managed directly by the owner.
			{/if}
		</p>
	</div>

    {/if}
	<!-- Controls & Category Filter -->
	<div class="toolbar">
		<div class="search-box">
			<input type="search" placeholder="Search settings" bind:value={searchQuery} aria-label="Search settings" />
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
		<div class="loading-state">Opening settings…</div>
	{:else if !governance}<button type="button" onclick={load}>Try again</button>
	{:else if filteredSettings.length === 0}
		<div class="empty-state">No setting matches that search.</div>
	{:else}
		<div class="table-wrap">
			<table class="settings-table">
				<thead>
					<tr>
						<th>Setting</th>
						<th>Category</th>
						<th>Now</th>
						<th>Version</th>
						<th>Last changed</th>
						<th></th>
					</tr>
				</thead>
				<tbody>
					{#each filteredSettings as s (s.key)}
						<tr class="setting-row">
							<td class="key-col">
								<span class="setting-key">{s.description}</span>
								<details class="setting-desc"><summary>Technical details</summary><code>{s.key}</code></details>
							</td>
							<td>
								<span class="category-tag">{s.category}</span>
							</td>
							<td class="val-col">
								{#if s.is_secret}<span>{!s.version ? 'No admin override' : s.connection_state === 'restart_required' ? 'Saved · restart API and worker' : s.connection_state === 'applied_unverified' ? 'Applied to API · verify worker and provider' : s.connection_state === 'disabled' ? 'Disabled' : s.connection_state === 'enabled_unverified' ? 'Enabled · delivery not verified' : 'Configuration needs checking'}</span>
                                {:else if typeof s.value === 'boolean'}
									<span class="bool-tag {s.value ? 'enabled' : 'disabled'}">
										{s.value ? 'Enabled' : 'Disabled'}
									</span>
								{:else}
									<span>{readable(s.value)}</span>
								{/if}
							</td>
							<td class="center">v{s.version}</td>
							<td>
								<span class="date">{s.version ? localTime(s.updated_at) : '—'}</span>
								{#if s.reason}<p class="reason-hint">"{s.reason}"</p>{/if}
							</td>
							<td class="actions-col">
								<button class="action-btn" onclick={() => openEdit(s)} disabled={!isOwner}>
									Change
								</button>
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
		<OwnerDialog {busy} bind:open={previewDiffModal} title="Change this setting" description={editingSetting.description} onclose={() => { editingSetting = null; clearConnector(); }}>
				<details class="setting-desc"><summary>Technical name</summary><code>{editingSetting.key}</code></details>

				<form onsubmit={(e) => { e.preventDefault(); submitSettingUpdate(); }}><fieldset disabled={busy}>
					<div class="form-group">
						{#if !editingSetting.requires_restart}<label for="edit-val">New value</label>{/if}
						{#if editingSetting.requires_restart}
                            <p>Changes apply after restarting the API and worker. Saved access tokens and signing secrets stay hidden; leave them blank to keep the current value. A changed connection must be verified with the provider before launch.</p>
                            {#if editingSetting.connection_fields?.some(field=>field.kind==='password')}
                                <label class="credential-clear"><input type="checkbox" bind:checked={clearRuntimeSecrets} disabled={busy} onchange={()=>{if(clearRuntimeSecrets)for(const field of editingSetting?.connection_fields||[])if(field.kind==='password')runtimeDraft[field.key]=''}} />Remove current credentials instead of keeping blank fields</label>
                                {#if clearRuntimeSecrets}<p>Removing credentials disconnects the provider after restart. Finish pending verification and bank operations before doing this.</p>{/if}
                            {/if}
                            {#each editingSetting.connection_fields || [] as field (field.key)}
                                <label for={`connection-${field.key}`}>{field.label}</label>
                                {#if field.kind === 'boolean'}
                                    <select id={`connection-${field.key}`} bind:value={runtimeDraft[field.key]} disabled={busy}><option value={true}>Enabled</option><option value={false}>Disabled</option></select>
                                {:else if field.kind === 'number'}
                                    <input id={`connection-${field.key}`} type="number" min="0" max="9007199254740991" step="1" bind:value={runtimeDraft[field.key]} disabled={busy} required />
                                {:else if field.kind === 'password'}
                                    <input id={`connection-${field.key}`} type="password" bind:value={runtimeDraft[field.key]} autocomplete="new-password" disabled={busy} placeholder="Leave blank to keep current" />
                                {:else if field.kind === 'url'}
                                    <input id={`connection-${field.key}`} type="url" bind:value={runtimeDraft[field.key]} disabled={busy} />
                                {:else}<input id={`connection-${field.key}`} type="text" bind:value={runtimeDraft[field.key]} disabled={busy} />{/if}
                            {/each}
                        {:else if editingSetting.is_secret}
                            <select id="edit-val" bind:value={connectorEnabled} disabled={busy}>
                                <option value={true}>Connect or replace credentials</option>
                                <option value={false}>Disable this channel</option>
                            </select>
                            <p>Use a service that supports Kredit’s notification connector format. A vendor API key alone may not work. Changes apply to the next delivery in staging and production; development keeps using test delivery.</p>
                            {#if connectorEnabled}
                                <label for="connector-endpoint">Connector HTTPS address</label>
                                <input id="connector-endpoint" type="url" bind:value={connectorEndpoint} placeholder="https://your-connector.example/send" required disabled={busy} />
                                <label for="connector-token">Connector access token</label>
                                <input id="connector-token" type="password" bind:value={connectorToken} autocomplete="new-password" required disabled={busy} />
                                <p>Enter both fields to replace the connection. Saved credentials are encrypted and cannot be revealed here.</p>
                            {:else}<p>This stops delivery through this channel, including sign-in codes. Make sure you have another working sign-in channel before saving.</p>{/if}
                        {:else if typeof editingSetting.value === 'boolean'}
							<select id="edit-val" bind:value={editDraftValue}>
								<option value={true}>On</option>
								<option value={false}>Off</option>
							</select>
						{:else if typeof editingSetting.value === 'number'}
							<input id="edit-val" type="number" step="any" bind:value={editDraftValue} required />
						{:else if typeof editingSetting.value === 'string'}
							<input id="edit-val" type="text" bind:value={editDraftValue} required />
						{:else}
							<p class="not-editable">This setting holds structured data. It is changed in a release, not on this screen.</p>
						{/if}
					</div>

					{#if editingSetting.requires_restart}
                        <div class="diff-preview"><h3>Fields being changed</h3>
                            {#each editingSetting.connection_fields || [] as field}
                                {#if field.kind === 'password' ? clearRuntimeSecrets || runtimeDraft[field.key] !== '' : runtimeDraft[field.key] !== editingSetting.connection_values?.[field.key]}
                                    <p><strong>{field.label}:</strong> {field.kind === 'password' ? (runtimeDraft[field.key] ? 'Replace hidden value' : 'Remove current value') : `${readable(editingSetting.connection_values?.[field.key])} → ${readable(runtimeDraft[field.key])}`}</p>
                                {/if}
                            {/each}
                        </div>
                    {/if}
                    <div class="diff-preview">
						<h3>What changes</h3>
						<div class="diff-row before"><span class="diff-label">Now</span><strong>{editingSetting.is_secret ? (editingSetting.version ? 'Saved configuration' : 'Deployment configuration, if available') : readable(editingSetting.value)}</strong></div>
						<div class="diff-row after"><span class="diff-label">After</span><strong>{editingSetting.requires_restart ? 'Saved configuration; applies after API and worker restart' : editingSetting.is_secret ? (connectorEnabled ? 'Replace connection; delivery still needs verification' : 'Channel disabled') : readable(editDraftValue)}</strong></div>
					</div>

					<div class="form-group">
						<label for="edit-reason">Why are you making this change?</label>
						<textarea id="edit-reason" rows="3" bind:value={editReason} placeholder="This is recorded permanently and cannot be edited later." required minlength="8" maxlength="2000"></textarea>
					</div>

					{#if error}<p role="alert" class="error">{error}</p>{/if}
                    <div class="modal-actions">
						<button type="button" class="cancel-btn" onclick={() => { previewDiffModal = false; editingSetting = null; clearConnector(); }} disabled={busy}>Cancel</button>
						<button type="submit" class="save-btn" disabled={busy || editReason.trim().length < 8 || (editingSetting.requires_restart ? false : editingSetting.is_secret ? (connectorEnabled && (!connectorEndpoint.trim() || !connectorToken.trim())) : JSON.stringify(editingSetting.value) === JSON.stringify(editDraftValue))}>
							{busy ? 'Saving…' : 'Save this change'}
						</button>
					</div>
				</fieldset></form>
		</OwnerDialog>
	{/if}

	<!-- Governance Switcher Modal -->
	{#if governanceModal}
		<OwnerDialog {busy} open={governanceModal} title="Who approves changes" description="This decides whether you can approve your own changes, or whether a second administrator has to." onclose={() => (governanceModal = false)}>

				<form onsubmit={(e) => { e.preventDefault(); submitGovernanceChange(); }}><fieldset disabled={busy}>
					<div class="form-group">
						<span class="group-label">Choose one</span>
						<div class="mode-options">
							<label class="mode-radio">
								<input type="radio" name="gov-mode" value="solo_owner" bind:group={newGovMode} />
								<div>
									<strong>I run Kredit alone</strong>
									<p>You approve your own changes. Each one still needs a fresh authenticator code and a written reason.</p>
								</div>
							</label>
							<label class="mode-radio">
								<input type="radio" name="gov-mode" value="delegated_team" bind:group={newGovMode} />
								<div>
									<strong>A second person approves</strong>
									<p>Nobody approves their own change, including you. Choose this once you have a second administrator.</p>
								</div>
							</label>
						</div>
					</div>

					<div class="form-group">
						<label for="gov-reason">Why are you changing this?</label>
						<textarea id="gov-reason" rows="3" bind:value={govReason} placeholder="This is recorded permanently and cannot be edited later." required minlength="8" maxlength="2000"></textarea>
					</div>

					{#if error}<p role="alert" class="error">{error}</p>{/if}
                    <div class="modal-actions">
						<button type="button" class="cancel-btn" onclick={() => governanceModal = false} disabled={busy}>Cancel</button>
						<button type="submit" class="save-btn" disabled={busy || govReason.trim().length < 8}>
							{busy ? 'Saving…' : 'Save this'}
						</button>
					</div>
				</fieldset></form>
		</OwnerDialog>
	{/if}

	<!-- History Drawer / Modal -->
	{#if historySettingKey}
		<OwnerDialog open={true} title="History" description="Every change to this setting, oldest last. This record cannot be edited." onclose={() => (historySettingKey = null)}>
				{#if historyLoading}
					<div class="loading-state">Opening history…</div>
				{:else if historyError}<p role="alert" class="error">{historyError}</p><button type="button" onclick={() => openHistory(historySettingKey!)}>Try again</button>
				{:else if historyEntries.length === 0}
					<div class="empty-state">This setting has not been changed yet.</div>
				{:else}
					<div class="table-wrap">
						<table class="history-table">
							<thead>
								<tr>
									<th>Version</th>
									<th>Action</th>
									<th>Changed to</th>
									<th>When</th>
									<th>Reason and who</th>
								</tr>
							</thead>
							<tbody>
								{#each historyEntries as h}
									<tr>
										<td>v{h.version}</td>
										<td><span class="action-tag">{h.action}</span></td>
										<td>{readable(h.new_value)}</td>
										<td>{localTime(h.recorded_at)}</td>
										<td>
											<strong>{h.reason}</strong>
											{#if h.actor_id}<p class="actor-sub">Changed by {h.actor_id}</p>{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
		</OwnerDialog>
	{/if}

	<!-- Ownership Transfer Modal -->
	{#if transferModal}
		<OwnerDialog {busy} open={transferModal} title="Hand over ownership" description="The person you choose becomes the owner of Kredit and you stop being the owner. It happens straight away and it is recorded permanently." onclose={() => (transferModal = false)}>

				<form onsubmit={(e) => { e.preventDefault(); submitTransferOwnership(); }}><fieldset disabled={busy}>
					<div class="form-group">
						<label for="transfer-target">Who takes over</label>
						<input id="transfer-target" type="text" bind:value={transferTargetUserId} placeholder="Their Kredit user reference" aria-describedby="transfer-target-help" required />
						<small id="transfer-target-help">Find it on the Users page, under the person's name.</small>
					</div>

					<div class="form-group">
						<label for="transfer-reason">Why are you handing it over?</label>
						<textarea id="transfer-reason" rows="3" bind:value={transferReason} placeholder="This is recorded permanently and cannot be edited later." required minlength="8" maxlength="2000"></textarea>
					</div>

					<div class="form-group checkbox-group">
						<label class="checkbox-label">
							<input type="checkbox" bind:checked={transferConfirmed} required />
							<span>I understand that I will no longer be the owner of Kredit.</span>
						</label>
					</div>

					{#if error}<p role="alert" class="error">{error}</p>{/if}
                    <div class="modal-actions">
						<button type="button" class="cancel-btn" onclick={() => transferModal = false} disabled={busy}>Cancel</button>
						<button type="submit" class="danger-btn" disabled={busy || !transferConfirmed || transferReason.trim().length < 8 || !transferTargetUserId.trim()}>
							{busy ? 'Handing over…' : 'Hand over ownership'}
						</button>
					</div>
				</fieldset></form>
		</OwnerDialog>
	{/if}
</main>

<style>
 fieldset{border:0;margin:0;padding:0;min-width:0;}
 .gov-pill-container{flex-wrap:wrap;}
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

	.credential-clear { display:flex !important; gap:.5rem; align-items:start; }
 .credential-clear input { width:auto !important; flex-shrink:0; }
	.form-group {
		margin: 1.25rem 0;
	}

	.form-group label {
		display: block;
		font-weight: 600;
		margin-bottom: 0.4rem;
		font-size: 0.85rem;
	}

	.form-group input + label, .form-group select + label { margin-top: 1rem; }
	.form-group input, .form-group select, .form-group textarea {
        box-sizing: border-box;
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
		color: #166534;
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
