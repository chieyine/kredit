package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/auth"
	"kredit/internal/db"
	"kredit/internal/operations"
	"kredit/internal/platformsettings"
)

func (s *Server) isFeatureEnabled(ctx context.Context, key string, defaultVal bool) bool {
	if s.runtime.PlatformSettings != nil {
		return s.runtime.PlatformSettings.GetBool(ctx, key, defaultVal)
	}
	return defaultVal
}

func (s *Server) platformCapabilities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var launchMode = platformsettings.LaunchModePreLaunch
	var bannerEnabled = false
	var bannerText = "Welcome to Kredit. Launching soon in private beta."
	var waitlistEnabled = true
	var govMode = platformsettings.GovernanceSoloOwner

	features := map[string]bool{
		"trade_lines":                false,
		"drawdowns":                  false,
		"repayment_extensions":       true,
		"disputes":                   true,
		"early_settlement_discounts": false,
		"notifications_whatsapp":     false,
		"mono_direct_debit":          false,
	}

	if s.runtime.PlatformSettings != nil {
		launchMode = s.runtime.PlatformSettings.GetString(ctx, "launch.mode", platformsettings.LaunchModePreLaunch)
		bannerEnabled = s.runtime.PlatformSettings.GetBool(ctx, "launch.banner_enabled", false)
		bannerText = s.runtime.PlatformSettings.GetString(ctx, "launch.banner_text", bannerText)
		waitlistEnabled = s.runtime.PlatformSettings.GetBool(ctx, "launch.waitlist_enabled", true)
		for k := range features {
			features[k] = s.runtime.PlatformSettings.GetBool(ctx, "features."+k, features[k])
		}
		if gov, err := s.runtime.PlatformSettings.GetGovernance(ctx); err == nil {
			govMode = gov.Mode
		}
	}

	isOwner := false
	if token := sessionTokenFromRequest(r); token != "" {
		if session, user, err := s.runtime.Auth.SessionFromToken(token); err == nil && session.AuthenticationLevel == auth.AAL2 {
			roles, _ := s.adminRoles(r, user.ID)
			for _, role := range roles {
				if role == access.PlatformOwner {
					isOwner = true
					break
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"launch_mode": launchMode,
		"banner": map[string]any{
			"enabled": bannerEnabled,
			"text":    bannerText,
		},
		"waitlist_enabled": waitlistEnabled,
		"features":         features,
		"governance_mode":  govMode,
		"is_owner":         isOwner,
	})
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

	gov, _ := s.runtime.PlatformSettings.GetGovernance(r.Context())

	s.auditPlatformRead(r, user.ID, "platform_settings.viewed", "platform_settings", category)
	writeJSON(w, http.StatusOK, map[string]any{
		"settings":   filtered,
		"governance": gov,
	})
}

type updateSettingInput struct {
	Key    string          `json:"key"`
	Value  json.RawMessage `json:"value"`
	Reason string          `json:"reason"`
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
	in.Reason = strings.TrimSpace(in.Reason)
	if in.Key == "" {
		writeProblem(w, http.StatusBadRequest, "key_required", "Setting key is required")
		return
	}
	if len(in.Reason) < 4 {
		writeProblem(w, http.StatusBadRequest, "reason_required", "Provide a reason of at least 4 characters")
		return
	}

	updated, err := s.runtime.PlatformSettings.Update(r.Context(), user.ID, in.Key, in.Value, in.Reason)
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

type rotateSecretInput struct {
	Key    string `json:"key"`
	Secret string `json:"secret"`
	Reason string `json:"reason"`
}

func (s *Server) rotatePlatformSecret(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformSettings)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	if s.runtime.PlatformSettings == nil {
		writeProblem(w, http.StatusServiceUnavailable, "settings_unavailable", "platform settings store is unavailable")
		return
	}

	var in rotateSecretInput
	if !decodeJSONRequest(w, r, &in) {
		return
	}

	in.Key = strings.TrimSpace(in.Key)
	in.Reason = strings.TrimSpace(in.Reason)
	if in.Key == "" {
		writeProblem(w, http.StatusBadRequest, "key_required", "Setting key is required")
		return
	}
	if len(in.Reason) < 4 {
		writeProblem(w, http.StatusBadRequest, "reason_required", "Provide a reason of at least 4 characters")
		return
	}

	updated, err := s.runtime.PlatformSettings.RotateSecret(r.Context(), user.ID, in.Key, in.Secret, in.Reason)
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "secret_invalid", err.Error())
		return
	}

	s.runtime.Audit.Append(audit.Event{
		ActorUserID:  user.ID,
		Action:       "platform_settings.secret_rotated",
		ResourceType: "platform_settings",
		ResourceID:   in.Key,
		Outcome:      "success",
		Severity:     "warning",
		RequestID:    requestIDFromContext(r.Context()),
		Metadata: map[string]string{
			"key":         in.Key,
			"version":     fmt.Sprintf("%d", updated.Version),
			"fingerprint": updated.SecretFingerprint,
			"reason":      in.Reason,
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

	var targetActive bool
	err = tx.QueryRow(r.Context(), `SELECT (status = 'active') FROM app.users WHERE id=$1::uuid`, in.TargetUserID).Scan(&targetActive)
	if err != nil || !targetActive {
		writeProblem(w, http.StatusBadRequest, "target_user_inactive", "Target user must be an active registered user")
		return
	}

	// 1. Grant platform_owner to target user first (ensuring at least one active owner always exists)
	_, err = tx.Exec(r.Context(), `
		INSERT INTO app.platform_role_assignments(user_id, role, granted_by, reason)
		VALUES($1::uuid, 'platform_owner', $2::uuid, $3)
		ON CONFLICT(user_id, role) WHERE revoked_at IS NULL
		DO UPDATE SET reason=EXCLUDED.reason
	`, in.TargetUserID, user.ID, in.Reason)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "grant_failed", err.Error())
		return
	}

	// Also grant platform_admin to target user
	_, err = tx.Exec(r.Context(), `
		INSERT INTO app.platform_role_assignments(user_id, role, granted_by, reason)
		VALUES($1::uuid, 'platform_admin', $2::uuid, $3)
		ON CONFLICT(user_id, role) WHERE revoked_at IS NULL
		DO UPDATE SET reason=EXCLUDED.reason
	`, in.TargetUserID, user.ID, in.Reason)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "grant_admin_failed", err.Error())
		return
	}

	// 2. Revoke platform_owner from caller
	_, err = tx.Exec(r.Context(), `
		UPDATE app.platform_role_assignments
		SET revoked_at=now(), revoked_by=$2::uuid
		WHERE user_id=$1::uuid AND role='platform_owner' AND revoked_at IS NULL
	`, user.ID, user.ID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "revoke_failed", err.Error())
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
