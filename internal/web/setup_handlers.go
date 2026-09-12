package web

import (
	"encoding/json"
	"kredit/internal/access"
	"kredit/internal/notifications"
	"kredit/internal/platformsettings"
	"kredit/internal/readiness"
	"net/http"
	"net/url"
	"time"
)

type setupTask struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	State  string `json:"state"`
	Detail string `json:"detail"`
	URL    string `json:"url"`
}

func (s *Server) ownerSetup(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionPlatformSettings)
	if !ok || !s.requireFreshMFA(w, session) {
		return
	}
	if s.runtime.Database == nil || s.runtime.PlatformSettings == nil {
		writeProblem(w, 503, "setup_unavailable", "Setup requires the database connection.")
		return
	}
	tasks := []setupTask{}
	link := func(key string) string { return "/admin/platform-settings?connection=" + url.QueryEscape(key) }
	add := func(id, label string, configured bool, detail, path string) {
		state := "needs_setup"
		if configured {
			state = "configured_unverified"
		}
		tasks = append(tasks, setupTask{id, label, state, detail, path})
	}
	for _, channel := range []string{notifications.ChannelEmail, notifications.ChannelSMS, notifications.ChannelWhatsApp} {
		configured := false
		switch channel {
		case notifications.ChannelEmail:
			configured = s.config.NotificationEmailEndpoint != "" && s.config.NotificationEmailToken != ""
		case notifications.ChannelSMS:
			configured = s.config.NotificationSMSEndpoint != "" && s.config.NotificationSMSToken != ""
		case notifications.ChannelWhatsApp:
			configured = s.config.NotificationWhatsAppEndpoint != "" && s.config.NotificationWhatsAppToken != ""
		}
		connection, err := notifications.ResolveConnector(r.Context(), s.runtime.PlatformSettings, channel)
		if err != nil {
			writeProblem(w, 503, "setup_unavailable", "A saved messaging connection could not be read.")
			return
		}
		if connection != nil {
			configured = connection.Enabled
		}
		add(channel, map[string]string{"email": "Email delivery", "sms": "SMS delivery", "whatsapp": "WhatsApp delivery"}[channel], configured, "Enter the provider connection and sender details. Confirm delivery with the provider before launch.", link("integrations.notifications."+channel))
	}
	add("identity", "Identity and business verification", s.config.RealIdentity && s.config.IdentityProviderEndpoint != "", "Choose an approved identity connection and enter its credentials.", link("integrations.runtime.identity"))
	add("collections", "Bank authorization and collection", s.runtime.Mono != nil || (s.config.RealCollections && s.config.CollectionProviderEndpoint != ""), "Configure Mono or another collection provider. Bank permission and provider approval are separate requirements.", link("integrations.runtime.mono"))
	add("settlement", "Seller bank accounts", s.runtime.Settlement != nil, "Choose the bank registration provider. Review business ownership before activating a destination.", link("integrations.runtime.settlement"))
	add("storage", "Private file storage", s.config.ObjectStorageEndpoint != "" && s.config.ObjectStorageAccessKey != "", "Connect the private storage bucket used for invoices and evidence.", link("integrations.runtime.storage"))
	add("scanner", "Document safety checks", s.config.DocumentScannerEndpoint != "", "Connect the document scanner. Files stay unavailable until their scan passes.", link("integrations.runtime.scanner"))
	report := readiness.Evaluate(s.config)
	add("approvals", "Launch evidence and operating limits", report.Ready, "Record genuine review references and operating limits. Entering a reference does not perform the review.", link("integrations.runtime.launch"))
	tasks = append(tasks, setupTask{"legal", "Website and legal documents", "review_required", "Review and publish your business details, terms and privacy policy.", "/admin/website"})
	tasks = append(tasks, setupTask{"team", "Super-admin security and team access", "review_required", "Keep recovery methods current and grant only the access each person needs.", "/admin/team"})
	rows, err := s.runtime.Database.Raw().Query(r.Context(), `SELECT process,versions,state,updated_at FROM app.runtime_process_status ORDER BY process`)
	if err != nil {
		writeProblem(w, 503, "setup_unavailable", "Service configuration status could not be read.")
		return
	}
	type processStatus struct {
		Process   string          `json:"process"`
		Versions  json.RawMessage `json:"versions"`
		State     string          `json:"state"`
		UpdatedAt time.Time       `json:"updated_at"`
		Fresh     bool            `json:"fresh"`
	}
	processes := []processStatus{}
	for rows.Next() {
		var item processStatus
		if err = rows.Scan(&item.Process, &item.Versions, &item.State, &item.UpdatedAt); err != nil {
			rows.Close()
			writeProblem(w, 503, "setup_unavailable", "Service configuration status could not be read.")
			return
		}
		item.Fresh = time.Since(item.UpdatedAt) < 45*time.Second
		processes = append(processes, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		writeProblem(w, 503, "setup_unavailable", "Service configuration status could not be read.")
		return
	}
	all, err := s.runtime.PlatformSettings.GetAll(r.Context(), false)
	if err != nil {
		writeProblem(w, 503, "setup_unavailable", "Saved settings could not be read.")
		return
	}
	wanted := map[string]int{}
	for _, item := range all {
		if _, ok := platformsettings.RuntimeConnections[item.Key]; ok {
			wanted[item.Key] = item.Version
		}
	}
	s.auditPlatformRead(r, user.ID, "platform_setup.viewed", "platform_setup", "")
	writeJSON(w, 200, map[string]any{"tasks": tasks, "processes": processes, "saved_versions": wanted, "auto_apply": s.config.AdminConfigAutoApply, "configuration_gates": report.Gates, "provider_initialization_failures": len(s.runtime.ProviderFailures), "readiness_note": "Configuration status is separate from completed launch verification.", "checked_at": time.Now().UTC()})
}
