package legalpublication

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"kredit/internal/platformsettings"

	"github.com/jackc/pgx/v5"
)

type Versions struct {
	Terms   string `json:"terms_version"`
	Privacy string `json:"privacy_version"`
}

func Initial() Versions { return Versions{Terms: TermsVersion, Privacy: PrivacyVersion} }

type Reader func() (Versions, error)

// SettingsReader changes only the versions selected for new consent/offer
// records. Existing evidence retains its previously recorded version.
func SettingsReader(settings platformsettings.Service) Reader {
	return func() (Versions, error) {
		result := Initial()
		if settings == nil {
			return result, nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, page := range []string{"terms", "privacy"} {
			setting, err := settings.Get(ctx, platformsettings.WebsitePrefix+page, false)
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			if err != nil {
				return Versions{}, err
			}
			var content platformsettings.WebsiteContent
			if err = json.Unmarshal(setting.Value, &content); err != nil {
				return Versions{}, err
			}
			if content.Published == nil {
				continue
			}
			publication := content.Published
			if err := platformsettings.ValidateWebsitePublication(page, *publication); err != nil {
				return Versions{}, err
			}
			if page == "terms" {
				result.Terms = publication.DocumentVersion
			} else {
				result.Privacy = publication.DocumentVersion
			}
		}
		return result, nil
	}
}

func Resolve(reader Reader) (Versions, error) {
	if reader == nil {
		return Initial(), nil
	}
	versions, err := reader()
	if err != nil {
		return Versions{}, err
	}
	if versions.Terms == "" || versions.Privacy == "" {
		return Versions{}, errors.New("current legal documents are unavailable")
	}
	return versions, nil
}
