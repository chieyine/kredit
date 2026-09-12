package web

import (
	"kredit/internal/access"
	"kredit/internal/notifications"
	"net/http"
)

func (s *Server) providerWork(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations)
	if !ok {
		return
	}
	if s.runtime.PlatformOps == nil {
		writeProblem(w, 503, "provider_work_unavailable", "Provider work requires the database connection.")
		return
	}
	work, err := s.runtime.PlatformOps.ProviderWork(r.Context())
	if err != nil {
		writeProblem(w, 503, "provider_work_unavailable", "Provider work could not be loaded.")
		return
	}
	connections := []map[string]string{}
	for _, cfg := range []struct{ channel, name, endpoint, token, adapter string }{{"email", "Sendly", s.config.NotificationEmailEndpoint, s.config.NotificationEmailToken, s.config.NotificationEmailAdapter}, {"sms", "Mesaj", s.config.NotificationSMSEndpoint, s.config.NotificationSMSToken, s.config.NotificationSMSAdapter}, {"whatsapp", "WhatsApp Cloud API", s.config.NotificationWhatsAppEndpoint, s.config.NotificationWhatsAppToken, s.config.NotificationWhatsAppAdapter}} {
		state := "not_configured"
		adapter := cfg.adapter
		if adapter == "" {
			adapter = notifications.DefaultAdapter(cfg.channel)
		}
		if cfg.endpoint != "" && cfg.token != "" {
			state = "configured_unverified"
		}
		current, e := notifications.ResolveConnector(r.Context(), s.runtime.PlatformSettings, cfg.channel)
		if e != nil {
			state = "configuration_unavailable"
		} else if current != nil {
			state = "disabled"
			adapter = current.Adapter
			if adapter == "" {
				adapter = notifications.DefaultAdapter(cfg.channel)
			}
			if current.Enabled {
				state = "configured_unverified"
			}
		}
		connections = append(connections, map[string]string{"provider": cfg.channel + " / " + adapter, "state": state, "action": "Review provider settings and recent delivery records."})
	}
	state := "not_configured"
	if s.runtime.Mono != nil {
		state = "configured_unverified"
	}
	connections = append(connections, map[string]string{"provider": "Mono", "state": state, "action": "Review bank authorization, debit and settlement evidence separately."})
	s.auditPlatformRead(r, user.ID, "operations.provider_work.viewed", "provider_work", "")
	writeJSON(w, 200, map[string]any{"work": work, "connections": connections})
}
