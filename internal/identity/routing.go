package identity

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
)

// Router selects the current provider only for new requests. Saved work must
// explicitly select its original provider; an absent account never falls back.
type Router struct {
	IdentityProvider
	accounts map[string]IdentityProvider
}

func NewRouter(active IdentityProvider, retained ...IdentityProvider) (*Router, error) {
	if active == nil {
		return nil, errors.New("active identity provider is required")
	}
	r := &Router{IdentityProvider: active, accounts: map[string]IdentityProvider{active.Name(): active}}
	for _, p := range retained {
		if p == nil || p.Name() == "" {
			return nil, errors.New("saved identity account is invalid")
		}
		if _, exists := r.accounts[p.Name()]; exists {
			return nil, errors.New("duplicate identity account name")
		}
		r.accounts[p.Name()] = p
	}
	return r, nil
}

func Resolve(provider IdentityProvider, name string) (IdentityProvider, error) {
	if router, ok := provider.(*Router); ok {
		if saved := router.accounts[name]; saved != nil {
			return saved, nil
		}
	} else if provider != nil && provider.Name() == name {
		return provider, nil
	}
	return nil, errors.New("the original verification account is unavailable; restore its saved connection")
}

func GetFrom(ctx context.Context, provider IdentityProvider, name, reference string) (ProviderVerification, error) {
	selected, err := Resolve(provider, name)
	if err != nil {
		return ProviderVerification{}, err
	}
	return selected.GetVerification(ctx, reference)
}

// RoutedReference is stored only by the server after creating a supplier case.
// Encoding keeps account selection separate from the provider's opaque ID.
func RoutedReference(name, reference string) string {
	return "kr1." + base64.RawURLEncoding.EncodeToString([]byte(name)) + "." + base64.RawURLEncoding.EncodeToString([]byte(reference))
}

func GetRouted(ctx context.Context, provider IdentityProvider, reference string) (ProviderVerification, error) {
	parts := strings.Split(reference, ".")
	if len(parts) != 3 || parts[0] != "kr1" {
		return ProviderVerification{}, errors.New("verification account was not recorded; support must reconcile the original case")
	}
	name, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ProviderVerification{}, errors.New("invalid saved verification account")
	}
	id, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(id) == 0 {
		return ProviderVerification{}, errors.New("invalid saved verification reference")
	}
	result, err := GetFrom(ctx, provider, string(name), string(id))
	if err != nil {
		return ProviderVerification{}, err
	}
	if result.ProviderID != string(id) {
		return ProviderVerification{}, errors.New("verification reference does not match")
	}
	result.ProviderID = reference
	return result, nil
}

func NativeAccounts(provider IdentityProvider) []*NativeLookup {
	accounts := []*NativeLookup{}
	if router, ok := provider.(*Router); ok {
		for _, p := range router.accounts {
			if native, ok := p.(*NativeLookup); ok {
				accounts = append(accounts, native)
			}
		}
	} else if native, ok := provider.(*NativeLookup); ok {
		accounts = append(accounts, native)
	}
	return accounts
}
