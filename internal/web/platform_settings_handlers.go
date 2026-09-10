package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/config"
	"kredit/internal/db"
	"kredit/internal/operations"
	"kredit/internal/platformsettings"

	"github.com/google/uuid"
)

func (s *Server) isFeatureEnabled(ctx context.Context, key string, defaultVal bool) bool {
	if s.runtime.PlatformSettings != nil {
		return s.runtime.PlatformSettings.GetBool(ctx, key, defaultVal)
	}
	return defaultVal
}

func (s *Server) platformCapabilities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	features := map[string]bool{
		"trade_lines": false, "drawdowns": false, "repayment_extensions": false,
		"disputes": true, "early_settlement_discounts": false,
		"notifications_whatsapp": false, "mono_direct_debit": false,
	}
	if s.runtime.PlatformSettings != nil {
		for key := range features {
			// Match the defaults used by the corresponding action handlers. Public
			// capabilities must not claim a switched-off action is available.
			features[key] = s.runtime.PlatformSettings.GetBool(ctx, "features."+key, key == "disputes")
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"features": features})
}

func (s *Server) listPlatformSettings(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformSettings)
	if !ok || !s.requireFreshMFA(w, session) {
		return
	}
	if s.runtime.PlatformSettings == nil {
		writeProblem(w, http.StatusServiceUnavailable, "settings_unavailable", "platform settings store is unavailable")
		return
	}

	all, err := s.runtime.PlatformSettings.GetAll(r.Context(), false)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "settings_load_failed", err.Error())
		return
	}

	// Website drafts belong in the owner editor, never the generic value table.
	visible := all[:0]
	for _, item := range all {
		if !strings.HasPrefix(item.Key, platformsettings.WebsitePrefix) {
			visible = append(visible, item)
		}
	}
	all = visible

	// List every consumed connection even before the first save.
	present := make(map[string]bool)
	for _, item := range all {
		present[item.Key] = true
	}
	keys := []string{"integrations.notifications.email", "integrations.notifications.sms", "integrations.notifications.whatsapp"}
	for key := range platformsettings.RuntimeConnections {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !present[key] {
			meta := platformsettings.KnownSettings[key]
			all = append(all, platformsettings.Setting{Key: key, Category: meta.Category, IsSecret: true, Value: json.RawMessage(`""`), Description: meta.Description})
		}
	}
	for i := range all {
		definition, ok := platformsettings.RuntimeConnections[all[i].Key]
		if !ok {
			continue
		}
		all[i].RequiresRestart = true
		all[i].ConnectionFields = definition.Fields
		all[i].AppliedVersion = s.config.AdminConnectionVersions[all[i].Key]
		all[i].ConnectionValues = config.PublicConnectionValues(s.config, all[i].Key)
		if all[i].Version > 0 {
			saved, readErr := s.runtime.PlatformSettings.Get(r.Context(), all[i].Key, true)
			if readErr != nil {
				all[i].ConnectionState = "unavailable"
				continue
			}
			values, decodeErr := platformsettings.DecodeRuntimeConnection(all[i].Key, saved.Value)
			if decodeErr != nil {
				all[i].ConnectionState = "unavailable"
				continue
			}
			for _, field := range definition.Fields {
				if field.Kind == "password" {
					continue
				}
				var value any
				if json.Unmarshal(values[field.Key], &value) == nil {
					all[i].ConnectionValues[field.Key] = value
				}
			}
			all[i].ConnectionState = "restart_required"
			if all[i].AppliedVersion == all[i].Version {
				all[i].ConnectionState = "applied_unverified"
			}
		}
	}

	category := strings.TrimSpace(r.URL.Query().Get("category"))
	var filtered []platformsettings.Setting
	for _, item := range all {
		if category == "" || item.Category == category {
			filtered = append(filtered, item)
		}
	}
	if filtered == nil {
		filtered = []platformsettings.Setting{}
	}

	gov, err := s.runtime.PlatformSettings.GetGovernance(r.Context())
	if err != nil {
		writeProblem(w, http.StatusServiceUnavailable, "governance_unavailable", "The approval rules could not be verified.")
		return
	}

	s.auditPlatformRead(r, user.ID, "platform_settings.viewed", "platform_settings", category)
	writeJSON(w, http.StatusOK, map[string]any{
		"settings":   filtered,
		"governance": gov,
	})
}

type updateSettingInput struct {
	ClearCredentials bool            `json:"clear_credentials"`
	ExpectedVersion  *int            `json:"expected_version"`
	Key              string          `json:"key"`
	Value            json.RawMessage `json:"value"`
	Reason           string          `json:"reason"`
}

func (s *Server) updatePlatformSetting(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformSettings)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	if s.runtime.PlatformSettings == nil {
		writeProblem(w, http.StatusServiceUnavailable, "settings_unavailable", "platform settings store is unavailable")
		return
	}

	var in updateSettingInput
	if !decodeJSONRequest(w, r, &in) {
		return
	}

	in.Key = strings.TrimSpace(in.Key)
	if strings.HasPrefix(in.Key, platformsettings.WebsitePrefix) {
		writeProblem(w, 400, "use_website_editor", "Use the website editor to save or publish page content.")
		return
	}
	in.Reason = strings.TrimSpace(in.Reason)
	if in.Key == "" {
		writeProblem(w, http.StatusBadRequest, "key_required", "Setting key is required")
		return
	}
	if len(in.Reason) < 4 {
		writeProblem(w, http.StatusBadRequest, "reason_required", "Provide a reason of at least 4 characters")
		return
	}

	if in.ExpectedVersion == nil {
		writeProblem(w, 400, "version_required", "Reload this setting before saving.")
		return
	}
	var updated platformsettings.Setting
	var err error
	if _, runtimeConnection := platformsettings.RuntimeConnections[in.Key]; runtimeConnection {
		updater, ok := s.runtime.PlatformSettings.(platformsettings.ValidatedUpdater)
		if !ok {
			writeProblem(w, http.StatusServiceUnavailable, "settings_unavailable", "Connection validation is unavailable.")
			return
		}
		updated, err = updater.UpdateValidated(r.Context(), user.ID, in.Key, in.Value, in.Reason, *in.ExpectedVersion, func(snapshot platformsettings.Service, raw json.RawMessage) (json.RawMessage, error) {
			prepared, err := config.PrepareConnectionUpdate(r.Context(), s.config, snapshot, in.Key, raw, in.ClearCredentials)
			if err != nil {
				return nil, err
			}
			if _, err = config.ApplyStoredConnections(r.Context(), s.config, snapshot, in.Key, prepared); err != nil {
				return nil, err
			}
			return prepared, nil
		})
	} else {
		updated, err = s.runtime.PlatformSettings.Update(r.Context(), user.ID, in.Key, in.Value, in.Reason, *in.ExpectedVersion)
	}

	if errors.Is(err, platformsettings.ErrVersionConflict) {
		writeProblem(w, 409, "setting_conflict", err.Error())
		return
	}
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "setting_invalid", err.Error())
		return
	}

	s.runtime.Audit.Append(audit.Event{
		ActorUserID:  user.ID,
		Action:       "platform_settings.updated",
		ResourceType: "platform_settings",
		ResourceID:   in.Key,
		Outcome:      "success",
		Severity:     "info",
		RequestID:    requestIDFromContext(r.Context()),
		Metadata: map[string]string{
			"key":     in.Key,
			"version": fmt.Sprintf("%d", updated.Version),
			"reason":  in.Reason,
		},
	})

	writeJSON(w, http.StatusOK, map[string]any{"setting": updated})
}

func (s *Server) platformSettingsHistory(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformSettings)
	if !ok || !s.requireFreshMFA(w, session) {
		return
	}
	if s.runtime.PlatformSettings == nil {
		writeProblem(w, http.StatusServiceUnavailable, "settings_unavailable", "platform settings store is unavailable")
		return
	}

	key := strings.TrimSpace(r.URL.Query().Get("key"))
	history, err := s.runtime.PlatformSettings.GetHistory(r.Context(), key, 50, 0)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "history_load_failed", err.Error())
		return
	}
	if history == nil {
		history = []platformsettings.SettingHistory{}
	}

	s.auditPlatformRead(r, user.ID, "platform_settings.history_viewed", "platform_settings", key)
	writeJSON(w, http.StatusOK, map[string]any{"history": history})
}

func (s *Server) getPlatformGovernance(w http.ResponseWriter, r *http.Request) {
	session, _, _, ok := s.requirePlatformAccess(w, r, "")
	if !ok || !s.requireFreshMFA(w, session) {
		return
	}
	if s.runtime.PlatformSettings == nil {
		writeProblem(w, http.StatusServiceUnavailable, "settings_unavailable", "platform settings store is unavailable")
		return
	}

	gov, err := s.runtime.PlatformSettings.GetGovernance(r.Context())
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "governance_load_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"governance": gov})
}

type setGovernanceInput struct {
	Mode   string `json:"mode"`
	Reason string `json:"reason"`
}

func (s *Server) setPlatformGovernance(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformOwner)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	if s.runtime.PlatformSettings == nil {
		writeProblem(w, http.StatusServiceUnavailable, "settings_unavailable", "platform settings store is unavailable")
		return
	}

	var in setGovernanceInput
	if !decodeJSONRequest(w, r, &in) {
		return
	}

	gov, err := s.runtime.PlatformSettings.SetGovernance(r.Context(), user.ID, in.Mode, in.Reason)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "governance_invalid", err.Error())
		return
	}

	s.runtime.Audit.Append(audit.Event{
		ActorUserID:  user.ID,
		Action:       "governance_mode.updated",
		ResourceType: "platform_governance",
		Outcome:      "success",
		Severity:     "warning",
		RequestID:    requestIDFromContext(r.Context()),
		Metadata: map[string]string{
			"mode":   gov.Mode,
			"reason": in.Reason,
		},
	})

	writeJSON(w, http.StatusOK, map[string]any{"governance": gov})
}

type soloOwnerApproveInput struct {
	TargetType string `json:"target_type"` // "financial_change" | "policy"
	TargetID   string `json:"target_id"`
	Reason     string `json:"reason"`
	Confirm    bool   `json:"confirm"`
}

func (s *Server) soloOwnerApprove(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformOwner)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	if s.runtime.PlatformSettings == nil {
		writeProblem(w, http.StatusServiceUnavailable, "settings_unavailable", "platform settings store is unavailable")
		return
	}

	var in soloOwnerApproveInput
	if !decodeJSONRequest(w, r, &in) {
		return
	}

	if !in.Confirm {
		writeProblem(w, http.StatusBadRequest, "confirmation_required", "Explicit solo-owner self-approval confirmation is required")
		return
	}

	in.Reason = strings.TrimSpace(in.Reason)
	if len(in.Reason) < 8 {
		writeProblem(w, http.StatusBadRequest, "reason_required", "Provide a detailed justification of at least 8 characters")
		return
	}

	gov, err := s.runtime.PlatformSettings.GetGovernance(r.Context())
	if err != nil || gov.Mode != platformsettings.GovernanceSoloOwner {
		writeProblem(w, http.StatusForbidden, "solo_owner_mode_required", "Self-approval is only permitted in solo_owner governance mode")
		return
	}

	switch in.TargetType {
	case "financial_change":
		store, ok := s.runtime.Operations.(*operations.PostgresStore)
		if !ok {
			writeProblem(w, http.StatusServiceUnavailable, "workflow_unavailable", "operations store unavailable")
			return
		}
		var supplierOrgID string
		_ = s.runtime.Database.Raw().QueryRow(r.Context(), `SELECT o.supplier_organization_id::text FROM app.admin_change_requests c JOIN app.obligations o ON o.id=c.obligation_id WHERE c.id=$1::uuid`, in.TargetID).Scan(&supplierOrgID)
		reqCtx := r.Context()
		if supplierOrgID != "" {
			reqCtx = db.WithTenantContext(reqCtx, user.ID, supplierOrgID)
		}
		if err := store.DecideChange(reqCtx, in.TargetID, user.ID, "approve", in.Reason, false); err != nil {
			policyFailure(w, err)
			return
		}

	case "policy":
		if s.runtime.BusinessPolicies == nil {
			writeProblem(w, http.StatusServiceUnavailable, "policy_unavailable", "policy store unavailable")
			return
		}
		if err := s.runtime.BusinessPolicies.Decide(r.Context(), in.TargetID, user.ID, "approve", in.Reason); err != nil {
			policyFailure(w, err)
			return
		}

	default:
		writeProblem(w, http.StatusBadRequest, "invalid_target_type", "Target type must be 'financial_change' or 'policy'")
		return
	}

	s.runtime.Audit.Append(audit.Event{
		ActorUserID:  user.ID,
		Action:       "solo_owner.self_approval",
		ResourceType: in.TargetType,
		ResourceID:   in.TargetID,
		Outcome:      "success",
		Severity:     "warning",
		RequestID:    requestIDFromContext(r.Context()),
		Metadata: map[string]string{
			"target_type": in.TargetType,
			"target_id":   in.TargetID,
			"reason":      in.Reason,
		},
	})

	writeJSON(w, http.StatusOK, map[string]any{"approved": true})
}

type transferOwnershipInput struct {
	TargetUserID string `json:"target_user_id"`
	Reason       string `json:"reason"`
	Confirm      bool   `json:"confirm"`
}

func (s *Server) transferOwnership(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformOwner)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}

	var in transferOwnershipInput
	if !decodeJSONRequest(w, r, &in) {
		return
	}

	if !in.Confirm {
		writeProblem(w, http.StatusBadRequest, "confirmation_required", "Explicit confirmation is required to transfer ownership")
		return
	}

	in.TargetUserID = strings.TrimSpace(in.TargetUserID)
	targetID, parseErr := uuid.Parse(in.TargetUserID)
	if parseErr != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_target_user", "Target user must be a valid Kredit user reference")
		return
	}
	in.TargetUserID = targetID.String()
	in.Reason = strings.TrimSpace(in.Reason)
	if in.TargetUserID == "" || in.TargetUserID == user.ID {
		writeProblem(w, http.StatusBadRequest, "invalid_target_user", "Target user must be a different active user")
		return
	}
	if len(in.Reason) < 8 {
		writeProblem(w, http.StatusBadRequest, "reason_required", "Provide a reason of at least 8 characters")
		return
	}

	tx, err := s.runtime.Database.Raw().Begin(r.Context())
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	if err := transferOwnershipTx(r.Context(), tx, user.ID, in.TargetUserID, in.Reason); err != nil {
		if errors.Is(err, errOwnershipChanged) {
			writeProblem(w, http.StatusConflict, "ownership_changed", err.Error())
		} else {
			writeProblem(w, http.StatusServiceUnavailable, "transfer_unavailable", "Ownership transfer could not be completed. Reload and check the owner before retrying.")
		}
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeProblem(w, http.StatusInternalServerError, "commit_failed", err.Error())
		return
	}

	s.runtime.Audit.Append(audit.Event{
		ActorUserID:  user.ID,
		Action:       "platform_owner.transferred",
		ResourceType: "user",
		ResourceID:   in.TargetUserID,
		Outcome:      "success",
		Severity:     "critical",
		RequestID:    requestIDFromContext(r.Context()),
		Metadata: map[string]string{
			"from_user_id": user.ID,
			"to_user_id":   in.TargetUserID,
			"reason":       in.Reason,
		},
	})

	writeJSON(w, http.StatusOK, map[string]any{"transferred": true, "new_owner_user_id": in.TargetUserID})
}
