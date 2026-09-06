package mono

import "testing"

func TestHostedAuthorizationURLRejectsUnapprovedDestinations(t *testing.T) {
	for _, value := range []string{"", "javascript:alert(1)", "http://authorise.mono.co/a", "https://authorise.mono.co.evil.test/a", "https://user@authorise.mono.co/a", "https://authorise.mono.co:8443/a", "https://authorise.mono.co/", "https://evil.test/a"} {
		if err := validateHostedAuthorizationURL(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
	if err := validateHostedAuthorizationURL("https://authorise.mono.co/test-authorization"); err != nil {
		t.Fatal(err)
	}
}
