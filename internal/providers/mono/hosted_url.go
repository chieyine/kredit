package mono

import (
	"errors"
	"net/url"
)

func validateHostedAuthorizationURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "authorise.mono.co" || parsed.Port() != "" || parsed.User != nil || parsed.Path == "" || parsed.Path == "/" {
		return errors.New("unapproved hosted authorization URL")
	}
	return nil
}
