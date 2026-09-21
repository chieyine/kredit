// Types for the platform settings screen, shared with the dialogs split out
// of it. Kept here so a child component cannot drift from the page's shape.

export type Setting = {
	key: string;
	is_secret?: boolean;
	connection_state?: string;
	requires_restart?: boolean;
	applied_version?: number;
	connection_fields?: { key: string; label: string; kind: string }[];
	connection_values?: Record<string, string | number | boolean>;
	category: string;
	value: unknown;
	description: string;
	version: number;
	updated_at: string;
	updated_by?: string;
	reason?: string;
};

export type Governance = {
	mode: 'solo_owner' | 'delegated_team';
	updated_at: string;
	updated_by?: string;
	reason: string;
};

export type SettingHistory = {
	id: string;
	key: string;
	old_value?: unknown;
	new_value: unknown;
	version: number;
	action: string;
	actor_id?: string;
	reason: string;
	recorded_at: string;
};
