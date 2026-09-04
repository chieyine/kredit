package organizations

import (
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
	InvitedBy      string      `json:"invited_by,omitempty"`
	InvitedAt      time.Time   `json:"invited_at,omitempty"`
	AcceptedAt     time.Time   `json:"accepted_at,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
}

type Invitation struct {
	ID             string      `json:"id"`
	OrganizationID string      `json:"organization_id"`
	Target         string      `json:"target"`
	TargetType     string      `json:"target_type"`
	Role           access.Role `json:"role"`
	Status         string      `json:"status"`
	InvitedBy      string      `json:"invited_by"`
	ExpiresAt      time.Time   `json:"expires_at"`
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
	invitations   map[string]*Invitation
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
	Membership(string, string) (Membership, bool)
	ListMembers(string) []Membership
	Invite(string, string, string, string, string, access.Role) (Invitation, Membership, error)
	ActivateInvitations(string) []Membership
	ChangeRole(string, string, string, access.Role) (Membership, error)
	ChangeStatus(string, string, string, string) (Membership, error)
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
		invitations:   make(map[string]*Invitation),
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
	membership := &Membership{ID: s.newID(), OrganizationID: organization.ID, UserID: ownerUserID, Role: access.RoleOwner, Status: "active", AcceptedAt: now, CreatedAt: now}
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

func (s *Store) ListForUser(userID string) []Organization {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Organization, 0)
	for _, membershipID := range s.byUser[userID] {
		membership := s.memberships[membershipID]
		if membership.Status == "removed" || membership.Status == "suspended" {
			continue
		}
		if organization, ok := s.organizations[membership.OrganizationID]; ok {
			result = append(result, cloneOrganization(*organization))
		}
	}
	return result
}

func (s *Store) Membership(organizationID, userID string) (Membership, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, membershipID := range s.byOrg[organizationID] {
		membership := s.memberships[membershipID]
		if membership.UserID == userID && membership.Status != "removed" {
			return cloneMembership(*membership), true
		}
	}
	return Membership{}, false
}

func (s *Store) ListMembers(organizationID string) []Membership {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Membership, 0, len(s.byOrg[organizationID]))
	for _, membershipID := range s.byOrg[organizationID] {
		result = append(result, cloneMembership(*s.memberships[membershipID]))
	}
	return result
}

func (s *Store) Invite(actorUserID, organizationID, target, targetType string, targetUserID string, role access.Role) (Invitation, Membership, error) {
	if !role.Valid() || role == access.RoleOwner {
		return Invitation{}, Membership{}, errors.New("only non-owner roles may be invited")
	}
	if strings.TrimSpace(target) == "" || targetUserID == "" || (targetType != "phone" && targetType != "email") {
		return Invitation{}, Membership{}, errors.New("invitation target is required")
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.organizations[organizationID]; !ok {
		return Invitation{}, Membership{}, errors.New("organisation not found")
	}
	for _, membershipID := range s.byOrg[organizationID] {
		membership := s.memberships[membershipID]
		if membership.UserID == targetUserID && membership.Status != "removed" {
			return Invitation{}, Membership{}, errors.New("user already belongs to organisation")
		}
	}
	invitation := &Invitation{ID: s.newID(), OrganizationID: organizationID, Target: strings.TrimSpace(target), TargetType: targetType, Role: role, Status: "pending", InvitedBy: actorUserID, ExpiresAt: now.Add(7 * 24 * time.Hour), CreatedAt: now}
	membership := &Membership{ID: s.newID(), OrganizationID: organizationID, UserID: targetUserID, Role: role, Status: "invited", InvitedBy: actorUserID, InvitedAt: now, CreatedAt: now}
	s.invitations[invitation.ID] = invitation
	s.addMembership(membership)
	return cloneInvitation(*invitation), cloneMembership(*membership), nil
}

func (s *Store) ActivateInvitations(userID string) []Membership {
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	activated := make([]Membership, 0)
	for _, membershipID := range s.byUser[userID] {
		membership := s.memberships[membershipID]
		if membership.Status != "invited" {
			continue
		}
		membership.Status = "active"
		membership.AcceptedAt = now
		activated = append(activated, cloneMembership(*membership))
	}
	return activated
}

func (s *Store) ChangeRole(organizationID, actorUserID, targetUserID string, role access.Role) (Membership, error) {
	if !role.Valid() || role == access.RoleOwner {
		return Membership{}, errors.New("invalid target role")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var target *Membership
	for _, membershipID := range s.byOrg[organizationID] {
		membership := s.memberships[membershipID]
		if membership.UserID == targetUserID && membership.Status == "active" {
			target = membership
			break
		}
	}
	if target == nil {
		return Membership{}, errors.New("membership not found")
	}
	if target.Role == access.RoleOwner {
		return Membership{}, errors.New("owner role cannot be changed")
	}
	if target.UserID == actorUserID {
		return Membership{}, errors.New("members cannot change their own role")
	}
	target.Role = role
	return cloneMembership(*target), nil
}

func (s *Store) ChangeStatus(organizationID, actorUserID, targetUserID, status string) (Membership, error) {
	if status != "active" && status != "suspended" && status != "removed" {
		return Membership{}, errors.New("invalid membership status")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var target *Membership
	for _, membershipID := range s.byOrg[organizationID] {
		membership := s.memberships[membershipID]
		if membership.UserID == targetUserID && membership.Status != "removed" {
			target = membership
			break
		}
	}
	if target == nil {
		return Membership{}, errors.New("membership not found")
	}
	if target.Role == access.RoleOwner {
		return Membership{}, errors.New("the owner membership cannot be suspended or removed")
	}
	if target.Role == access.RoleOwner {
		return Membership{}, errors.New("owner role cannot be changed")
	}
	if target.UserID == actorUserID {
		return Membership{}, errors.New("members cannot change their own access status")
	}
	if status == "active" && target.Status == "invited" {
		return Membership{}, errors.New("an invitation must be accepted by the invited user")
	}
	target.Status = status
	return cloneMembership(*target), nil
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
func cloneInvitation(value Invitation) Invitation       { return value }
