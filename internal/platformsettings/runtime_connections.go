package platformsettings

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ConnectionField describes only explicitly supported configuration fields.
// It never contains a saved credential or accepts an arbitrary environment key.
type ConnectionField struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
}

type RuntimeConnection struct {
	Title  string
	Fields []ConnectionField
}

var RuntimeConnections = map[string]RuntimeConnection{
	"integrations.runtime.identity": {Title: "Identity verification", Fields: []ConnectionField{
		{"RealIdentity", "Enable identity verification", "boolean"},
		{"IdentityProvider", "Provider name", "text"},
		{"IdentityProviderEndpoint", "Kredit-compatible connector HTTPS address", "url"},
		{"IdentityProviderToken", "Connector access token", "password"},
		{"IdentityWebhookSecret", "Webhook signing secret", "password"},
		{"IdentityApprovalReference", "Provider approval reference", "text"},
	}},
	"integrations.runtime.mono": {Title: "Mono bank collections", Fields: []ConnectionField{
		{"MonoSweepEnabled", "Enable new Mono collections", "boolean"},
		{"PartialSweepEnabled", "Allow partial collections", "boolean"},
		{"MonoSecretKey", "Mono secret key", "password"},
		{"MonoWebhookSecret", "Mono webhook secret", "password"},
		{"MonoRedirectURL", "Customer return HTTPS address", "url"},
	}},
	"integrations.runtime.scanner": {Title: "Document scanning", Fields: []ConnectionField{
		{"DocumentScannerEnabled", "Enable document scanning", "boolean"},
		{"DocumentScannerEndpoint", "Scanner HTTPS address", "url"},
		{"DocumentScannerToken", "Scanner access token", "password"},
	}},
	"integrations.runtime.collections": {Title: "Other bank collection connector", Fields: []ConnectionField{
		{"RealCollections", "Enable this collection connector", "boolean"},
		{"CollectionProvider", "Provider name", "text"},
		{"CollectionProviderEndpoint", "Kredit-compatible connector HTTPS address", "url"},
		{"CollectionProviderToken", "Connector access token", "password"},
		{"CollectionWebhookSecret", "Webhook signing secret", "password"},
		{"ProviderApprovalReference", "Written provider approval reference", "text"},
		{"ProviderApprovedBy", "Approved by", "text"},
		{"ProviderApprovedAt", "Approval time (for example 2026-09-08T10:00:00+01:00)", "text"},
	}},
	"integrations.runtime.launch": {Title: "Launch approvals and provider limits", Fields: []ConnectionField{
		{"ProductionPilot", "Enable approved production pilot", "boolean"},
		{"PilotApprovalReference", "Pilot approval reference", "text"},
		{"ProviderCertificationReference", "Provider certification reference", "text"},
		{"SecurityReviewReference", "Security review reference", "text"},
		{"DPIAReference", "Data protection review reference", "text"},
		{"LegalApprovalReference", "Legal approval reference", "text"},
		{"PenTestReference", "Penetration test reference", "text"},
		{"BackupRestoreReference", "Backup restore evidence reference", "text"},
		{"SupportTrainingReference", "Support training reference", "text"},
		{"LaunchApprovalReference", "Launch approval reference", "text"},
		{"PilotAllowedProviderAccounts", "Allowed provider names, separated by commas", "text"},
		{"PilotAllowedIndustries", "Allowed industries, separated by commas", "text"},
		{"PilotMaxSupplierOrganizations", "Maximum supplier businesses", "number"},
		{"PilotMaxBuyerBusinesses", "Maximum buyer businesses", "number"},
		{"PilotMaxPrincipalKobo", "Maximum principal (kobo)", "number"},
		{"PilotMaxActiveExposureKobo", "Maximum active exposure (kobo)", "number"},
		{"PilotMaxDrawdownsPerLineDay", "Maximum daily drawdowns per limit", "number"},
		{"PilotMaxCollectionRetries", "Maximum collection retries", "number"},
		{"PilotEnhancedReviewKobo", "Enhanced review threshold (kobo)", "number"},
	}},
}

func init() {
	for key, definition := range RuntimeConnections {
		KnownSettings[key] = SettingMeta{Category: "integrations", IsSecret: true, Description: definition.Title + " (applies after API and worker restart)", Validate: func(raw json.RawMessage) error { _, err := DecodeRuntimeConnection(key, raw); return err }}
	}
}

// DecodeRuntimeConnection refuses partial/unknown fields so a replacement is
// reviewable and cannot unexpectedly inherit a credential from the environment.
func DecodeRuntimeConnection(key string, raw json.RawMessage) (map[string]json.RawMessage, error) {
	definition, ok := RuntimeConnections[key]
	if !ok {
		return nil, errors.New("unsupported runtime connection")
	}
	var encoded string
	if json.Unmarshal(raw, &encoded) != nil {
		return nil, errors.New("connection must be encoded configuration")
	}
	var values map[string]json.RawMessage
	if json.Unmarshal([]byte(encoded), &values) != nil || len(values) != len(definition.Fields) {
		return nil, errors.New("provide every field in this connection")
	}
	for _, field := range definition.Fields {
		value, ok := values[field.Key]
		if !ok || string(value) == "null" {
			return nil, fmt.Errorf("%s is required", field.Label)
		}
		var err error
		switch field.Kind {
		case "boolean":
			var v bool
			err = json.Unmarshal(value, &v)
		case "number":
			var v int64
			err = json.Unmarshal(value, &v)
			if v > 9007199254740991 {
				return nil, fmt.Errorf("%s exceeds the supported maximum", field.Label)
			}
			if v < 0 {
				return nil, fmt.Errorf("%s cannot be negative", field.Label)
			}
		default:
			var v string
			err = json.Unmarshal(value, &v)
		}
		if err != nil {
			return nil, fmt.Errorf("invalid value for %s", field.Label)
		}
	}
	return values, nil
}
