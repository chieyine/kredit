package organizations

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"kredit/internal/access"
)

type Organization struct {
	ID               string    `json:"id"`
	LegalName        string    `json:"legal_name"`
	TradingName      string    `json:"trading_name"`
	BusinessType     string    `json:"business_type"`
	RegistrationInfo string    `json:"registration_info,omitempty"`
	BusinessAddress  string    `json:"business_address"`
	Industry         string    `json:"industry"`
	Status           string    `json:"status"`
	DefaultTimezone  string    `json:"default_timezone"`
	DefaultCurrency  string    `json:"default_currency"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Version          int64     `json:"version"`
}

type Membership struct {
	ID             string      `json:"id"`
	OrganizationID string      `json:"organization_id"`
	UserID         string      `json:"user_id"`
	Role           access.Role `json:"role"`
	Status         string      `json:"status"`
	CreatedAt      time.Time   `json:"created_at"`
}

type CreateInput struct {
	LegalName        string
	TradingName      string
	BusinessType     string
	RegistrationInfo string
	BusinessAddress  string
	Industry         string
	Timezone         string
	Currency         string
}

type Store struct {
	mu            sync.RWMutex
	organizations map[string]*Organization
	memberships   map[string]*Membership
	byOrg         map[string][]string
	byUser        map[string][]string
	newID         func() string
	now           func() time.Time
	createGuard   func(string, CreateInput) error
}

// Service is the organization boundary consumed by the API. The in-memory
// implementation remains useful for development while the PostgreSQL
// implementation provides tenant-scoped persistence in deployed runtimes.
type Service interface {
	SetCreateGuard(func(string, CreateInput) error)
	Count() int
	Create(string, CreateInput) (Organization, Membership, error)
	Get(string) (Organization, bool)
	ListForUser(string) []Organization
	ReadForUser(context.Context, string) ([]Organization, error)
	Membership(string, string) (Membership, bool)
	ReadMembership(context.Context, string, string) (Membership, error)
	ReadOwner(context.Context, string) (Membership, error)
}

var _ Service = (*Store)(nil)

func (s *Store) SetCreateGuard(guard func(string, CreateInput) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.createGuard = guard
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.organizations)
}

func NewStore() *Store {
	return &Store{
		organizations: make(map[string]*Organization),
		memberships:   make(map[string]*Membership),
		byOrg:         make(map[string][]string),
		byUser:        make(map[string][]string),
		newID:         newIdentifier,
		now:           func() time.Time { return time.Now().UTC() },
	}
}

func (s *Store) Create(ownerUserID string, input CreateInput) (Organization, Membership, error) {
	if strings.TrimSpace(ownerUserID) == "" {
		return Organization{}, Membership{}, errors.New("owner user is required")
	}
	if err := validateCreateInput(input); err != nil {
		return Organization{}, Membership{}, err
	}
	s.mu.RLock()
	guard := s.createGuard
	s.mu.RUnlock()
	if guard != nil {
		if err := guard(ownerUserID, input); err != nil {
			return Organization{}, Membership{}, err
		}
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	organization := &Organization{
		ID:               s.newID(),
		LegalName:        strings.TrimSpace(input.LegalName),
		TradingName:      strings.TrimSpace(input.TradingName),
		BusinessType:     strings.TrimSpace(input.BusinessType),
		RegistrationInfo: strings.TrimSpace(input.RegistrationInfo),
		BusinessAddress:  strings.TrimSpace(input.BusinessAddress),
		Industry:         strings.TrimSpace(input.Industry),
		Status:           "onboarding",
		DefaultTimezone:  defaultOr(input.Timezone, "Africa/Lagos"),
		DefaultCurrency:  defaultOr(input.Currency, "NGN"),
		CreatedAt:        now,
		UpdatedAt:        now,
		Version:          1,
	}
	membership := &Membership{ID: s.newID(), OrganizationID: organization.ID, UserID: ownerUserID, Role: access.RoleOwner, Status: "active", CreatedAt: now}
	s.organizations[organization.ID] = organization
	s.addMembership(membership)
	return cloneOrganization(*organization), cloneMembership(*membership), nil
}

func (s *Store) Get(organizationID string) (Organization, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	organization, ok := s.organizations[organizationID]
	if !ok {
		return Organization{}, false
	}
	return cloneOrganization(*organization), true
}

func (s *Store) ReadForUser(ctx context.Context, userID string) ([]Organization, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.ListForUser(userID), nil
}

func (s *Store) ListForUser(userID string) []Organization {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Organization, 0)
	for _, membershipID := range s.byUser[userID] {
		membership := s.memberships[membershipID]
		if membership.Status != "active" || membership.Role != access.RoleOwner {
			continue
		}
		if organization, ok := s.organizations[membership.OrganizationID]; ok {
			result = append(result, cloneOrganization(*organization))
		}
	}
	return result
}

// ErrMembershipNotFound means the person genuinely is not a member. Anything
// else returned by ReadMembership is an infrastructure failure, and the caller
// must not read it as a refusal: telling someone they have no access to their
// own business because a query failed is a wrong answer, and recording it as
// an authorization denial puts a decision in the audit trail that nobody made.

var ErrMembershipNotFound = errors.New("membership not found")

// ReadMembership separates "not a member" from "could not tell".
func (s *Store) ReadMembership(ctx context.Context, organizationID, userID string) (Membership, error) {
	if err := ctx.Err(); err != nil {
		return Membership{}, err
	}
	membership, ok := s.Membership(organizationID, userID)
	if !ok {
		return Membership{}, ErrMembershipNotFound
	}
	return membership, nil
}

func (s *Store) Membership(organizationID, userID string) (Membership, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, membershipID := range s.byOrg[organizationID] {
		membership := s.memberships[membershipID]
		if membership.UserID == userID && membership.Status == "active" && membership.Role == access.RoleOwner {
			return cloneMembership(*membership), true
		}
	}
	return Membership{}, false
}

func (s *Store) addMembership(membership *Membership) {
	s.memberships[membership.ID] = membership
	s.byOrg[membership.OrganizationID] = append(s.byOrg[membership.OrganizationID], membership.ID)
	s.byUser[membership.UserID] = append(s.byUser[membership.UserID], membership.ID)
}

func validateCreateInput(input CreateInput) error {
	if strings.TrimSpace(input.LegalName) == "" || strings.TrimSpace(input.BusinessType) == "" || strings.TrimSpace(input.BusinessAddress) == "" || strings.TrimSpace(input.Industry) == "" {
		return errors.New("legal name, business type, address, and industry are required")
	}
	if input.Currency != "" && input.Currency != "NGN" {
		return errors.New("organisation currency must be NGN")
	}
	return nil
}

func defaultOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func newIdentifier() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err == nil {
		return hex.EncodeToString(value[:])
	}
	return time.Now().UTC().Format("20060102150405.000000000")
}

func cloneOrganization(value Organization) Organization { return value }
func cloneMembership(value Membership) Membership       { return value }

// ReadOwner returns the sole active operator for business-level notifications and setup.
func (s *Store) ReadOwner(ctx context.Context, organizationID string) (Membership, error) {
	if err := ctx.Err(); err != nil {
		return Membership{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, id := range s.byOrg[organizationID] {
		member := s.memberships[id]
		if member.Role == access.RoleOwner && member.Status == "active" {
			return *member, nil
		}
	}
	return Membership{}, ErrMembershipNotFound
}
