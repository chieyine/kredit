package notifications

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"kredit/internal/identifier"
	"kredit/internal/platform/logging"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ChannelWhatsApp  = "whatsapp"
	ChannelEmail     = "email"
	ChannelSMS       = "sms"
	PriorityCritical = "critical"
	PriorityRoutine  = "routine"
	StateScheduled   = "scheduled"
	StateSending     = "sending"
	StateSent        = "sent"
	StateDelivered   = "delivered"
	StateFailed      = "failed"
	StateSuppressed  = "suppressed"
)

type Event struct {
	DeferDelivery  bool
	ID             string
	Type           string
	RecipientID    string
	Phone          string
	Email          string
	OrganizationID string
	Priority       string
	AmountKobo     int64
	Currency       string
	Date           time.Time
	Reference      string
	NextAction     string
	SupportLink    string
	SecurePath     string
}
type Message struct {
	AuthenticationCode string
	EventID            string
	RecipientID        string
	Channel            string
	Template           string
	TemplateVersion    string
	Body               string
	SecureLink         string
	Destination        string
}
type Delivery struct {
	ID                string    `json:"id"`
	EventID           string    `json:"event_id"`
	RecipientID       string    `json:"recipient_id"`
	Channel           string    `json:"channel"`
	Template          string    `json:"template"`
	TemplateVersion   string    `json:"template_version"`
	State             string    `json:"state"`
	ProviderMessageID string    `json:"provider_message_id,omitempty"`
	Body              string    `json:"body"`
	ScheduledAt       time.Time `json:"scheduled_at,omitempty"`
	SentAt            time.Time `json:"sent_at,omitempty"`
	FailedAt          time.Time `json:"failed_at,omitempty"`
	FailureReason     string    `json:"failure_reason,omitempty"`
	SecureLink        string    `json:"secure_link,omitempty"`
}
type Preferences struct {
	PreferredChannel        string `json:"preferred_channel"`
	FallbackChannel         string `json:"fallback_channel"`
	OptedOut                bool   `json:"opted_out"`
	PaymentRemindersEnabled bool   `json:"payment_reminders_enabled"`
	ProductUpdatesEnabled   bool   `json:"product_updates_enabled"`
	QuietStart              int    `json:"quiet_start_hour"`
	QuietEnd                int    `json:"quiet_end_hour"`
	Timezone                string `json:"timezone"`
	Version                 int64  `json:"version"`
}
type Provider interface {
	Channel() string
	Send(context.Context, Message) (string, error)
}

// SendOTP delivers an authentication code through the requested channel. OTP
// delivery deliberately bypasses user notification preferences and quiet hours:
// it is an authentication response, not a marketing or routine notification.
func (s *Store) SendRecoveryInstructions(ctx context.Context, recipient, channel, link string) error {
	recipient = strings.TrimSpace(recipient)
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel == "phone" {
		channel = ChannelSMS
	}
	if recipient == "" || !validChannel(channel) {
		return errors.New("valid recovery destination and channel are required")
	}
	s.mu.Lock()
	provider := s.providers[channel]
	s.mu.Unlock()
	if provider == nil {
		return errors.New("recovery delivery provider is unavailable")
	}
	body := "Your Kredit account recovery request is under review. If you did not request this, contact support."
	if link != "" {
		body = "Continue your Kredit account recovery securely: " + link + ". If you did not request this, contact support."
	}
	_, err := provider.Send(ctx, Message{EventID: "recovery-" + fmt.Sprintf("%x", sha256.Sum256([]byte(link))), RecipientID: recipient, Destination: recipient, Channel: channel, Template: "AccountRecoveryContinuation", TemplateVersion: "v1", Body: body, SecureLink: link})
	return err
}

func (s *Store) SendOTP(ctx context.Context, recipient, channel, code string) error {
	recipient = strings.TrimSpace(recipient)
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel == "phone" {
		channel = ChannelSMS
	}
	code = strings.TrimSpace(code)
	if recipient == "" || code == "" {
		return errors.New("OTP recipient and code are required")
	}
	if channel != ChannelEmail && channel != ChannelSMS && channel != ChannelWhatsApp {
		return errors.New("unsupported OTP delivery channel")
	}
	s.mu.Lock()
	provider := s.providers[channel]
	s.mu.Unlock()
	if provider == nil {
		return fmt.Errorf("%s OTP provider is unavailable", channel)
	}
	_, err := provider.Send(ctx, Message{
		EventID:            s.newID(),
		RecipientID:        recipient,
		Destination:        recipient,
		Channel:            channel,
		Template:           "AuthenticationCode",
		AuthenticationCode: code,
		TemplateVersion:    "v1",
		Body:               "Your Kredit verification code is " + code + ". It expires shortly. Never share this code.",
	})
	return err
}

// SendInvitation delivers the single-use onboarding link to the exact target
// that the supplier invited. Failure is returned to the caller so the UI can
// offer the same link for an explicit manual handoff instead of claiming that
// delivery succeeded.
func (s *Store) SendInvitation(ctx context.Context, recipient, targetType, invitationURL string) error {
	channel := ChannelSMS
	if targetType == "email" {
		channel = ChannelEmail
	} else if targetType != "phone" {
		return errors.New("invitation target type must be email or phone")
	}
	if strings.TrimSpace(recipient) == "" || strings.TrimSpace(invitationURL) == "" {
		return errors.New("invitation recipient and URL are required")
	}
	s.mu.Lock()
	provider := s.providers[channel]
	s.mu.Unlock()
	if provider == nil {
		return fmt.Errorf("%s invitation provider is unavailable", channel)
	}
	_, err := provider.Send(ctx, Message{EventID: "buyer-invitation:" + s.newID(), RecipientID: recipient, Destination: recipient, Channel: channel, Template: "BuyerInvitation", TemplateVersion: "v1", Body: "You have been invited to Kredit. Review and verify your business using this private link: " + invitationURL, SecureLink: invitationURL})
	return err
}

type MockProvider struct {
	mu       sync.Mutex
	channel  string
	messages []Message
	fail     bool
	nextID   int
}

func NewMockProvider(channel string) *MockProvider { return &MockProvider{channel: channel} }
func (p *MockProvider) Channel() string            { return p.channel }
func (p *MockProvider) SetFail(fail bool)          { p.mu.Lock(); defer p.mu.Unlock(); p.fail = fail }
func (p *MockProvider) Send(_ context.Context, message Message) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.fail {
		return "", errors.New("mock notification provider failed")
	}
	p.nextID++
	p.messages = append(p.messages, message)
	return fmt.Sprintf("%s-message-%d", p.channel, p.nextID), nil
}
func (p *MockProvider) Messages() []Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]Message(nil), p.messages...)
}

type Store struct {
	mu                 sync.Mutex
	secret             []byte
	providers          map[string]Provider
	templates          map[string]string
	preferences        map[string]Preferences
	deliveries         map[string]*Delivery
	dedupe             map[string]string
	eventFingerprints  map[string]string
	now                func() time.Time
	newID              func() string
	baseURL            string
	pool               *pgxpool.Pool
	encryption         []byte
	reminderConsent    func(context.Context, string, string) (bool, error)
	optionalProcessing func(context.Context, string) (bool, error)
}

func NewStore(secret string) *Store {
	return &Store{secret: []byte(secret), providers: map[string]Provider{}, templates: map[string]string{}, preferences: map[string]Preferences{}, deliveries: map[string]*Delivery{}, dedupe: map[string]string{}, eventFingerprints: map[string]string{}, now: func() time.Time { return time.Now().UTC() }, newID: newIdentifier, baseURL: "https://app.kredit.ng"}
}
func NewPostgresStore(pool *pgxpool.Pool, secret string) *Store {
	store := NewStore(secret)
	key := sha256.Sum256([]byte("kredit-notifications:" + secret))
	store.pool = pool
	store.encryption = key[:]
	return store
}
func (s *Store) SetBaseURL(baseURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(baseURL) != "" {
		s.baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	}
}
func (s *Store) RegisterProvider(provider Provider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[provider.Channel()] = provider
}
func (s *Store) SetTemplate(eventType, channel, version, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.templates[eventType+"|"+channel+"|"+version] = body
	if s.pool != nil {
		_, _ = s.pool.Exec(context.Background(), `INSERT INTO app.notification_templates(event_type,channel,version,body,active) VALUES($1,$2,$3,$4,true) ON CONFLICT(event_type,channel,version) DO UPDATE SET body=EXCLUDED.body,active=true`, eventType, channel, version, body)
	}
}
func (s *Store) SetPreferences(recipient string, prefs Preferences) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Callers predating category controls expressed only routing/quiet-hours.
	// Preserve that behavior while authenticated updates use an explicit version.
	if prefs.Version == 0 && !prefs.OptedOut {
		prefs.PaymentRemindersEnabled = true
		prefs.Version = 1
	}
	s.preferences[recipient] = prefs
	if s.pool != nil {
		_, _ = s.pool.Exec(context.Background(), `INSERT INTO app.notification_preferences(recipient_id,preferred_channel,fallback_channel,opted_out,payment_reminders_enabled,product_updates_enabled,quiet_start_hour,quiet_end_hour,timezone) VALUES($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(recipient_id) DO UPDATE SET preferred_channel=EXCLUDED.preferred_channel,fallback_channel=EXCLUDED.fallback_channel,opted_out=EXCLUDED.opted_out,payment_reminders_enabled=EXCLUDED.payment_reminders_enabled,product_updates_enabled=EXCLUDED.product_updates_enabled,quiet_start_hour=EXCLUDED.quiet_start_hour,quiet_end_hour=EXCLUDED.quiet_end_hour,timezone=EXCLUDED.timezone,version=app.notification_preferences.version+1,updated_at=now()`, recipient, prefs.PreferredChannel, prefs.FallbackChannel, prefs.OptedOut, prefs.PaymentRemindersEnabled, prefs.ProductUpdatesEnabled, prefs.QuietStart, prefs.QuietEnd, prefs.Timezone)
	}
}

func DefaultPreferences() Preferences {
	return Preferences{PreferredChannel: ChannelWhatsApp, FallbackChannel: ChannelEmail, PaymentRemindersEnabled: true, QuietStart: 22, QuietEnd: 7, Timezone: "Africa/Lagos", Version: 1}
}

func (s *Store) GetPreferences(ctx context.Context, recipient string) (Preferences, error) {
	prefs := DefaultPreferences()
	if s.pool != nil {
		err := s.pool.QueryRow(ctx, `SELECT preferred_channel,fallback_channel,opted_out,payment_reminders_enabled,product_updates_enabled,quiet_start_hour,quiet_end_hour,timezone,version FROM app.notification_preferences WHERE recipient_id=$1::uuid`, recipient).Scan(&prefs.PreferredChannel, &prefs.FallbackChannel, &prefs.OptedOut, &prefs.PaymentRemindersEnabled, &prefs.ProductUpdatesEnabled, &prefs.QuietStart, &prefs.QuietEnd, &prefs.Timezone, &prefs.Version)
		if errors.Is(err, pgx.ErrNoRows) {
			return prefs, nil
		}
		return prefs, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if stored, ok := s.preferences[recipient]; ok {
		prefs = stored
		if prefs.Version == 0 {
			prefs.Version = 1
		}
	}
	return prefs, nil
}

func (s *Store) UpdatePreferences(ctx context.Context, recipient string, prefs Preferences, expectedVersion int64) (Preferences, error) {
	if recipient == "" || !validChannel(prefs.PreferredChannel) || !validChannel(prefs.FallbackChannel) || prefs.PreferredChannel == prefs.FallbackChannel || prefs.QuietStart < 0 || prefs.QuietStart > 23 || prefs.QuietEnd < 0 || prefs.QuietEnd > 23 || prefs.Timezone != "Africa/Lagos" {
		return Preferences{}, errors.New("notification preference is invalid")
	}
	if s.pool != nil {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return Preferences{}, err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		defaults := DefaultPreferences()
		if _, err = tx.Exec(ctx, `INSERT INTO app.notification_preferences(recipient_id,preferred_channel,fallback_channel,opted_out,payment_reminders_enabled,product_updates_enabled,quiet_start_hour,quiet_end_hour,timezone,version) VALUES($1::uuid,$2,$3,false,true,false,$4,$5,$6,1) ON CONFLICT(recipient_id) DO NOTHING`, recipient, defaults.PreferredChannel, defaults.FallbackChannel, defaults.QuietStart, defaults.QuietEnd, defaults.Timezone); err != nil {
			return Preferences{}, err
		}
		var out Preferences
		err = tx.QueryRow(ctx, `UPDATE app.notification_preferences SET preferred_channel=$2,fallback_channel=$3,opted_out=$4,payment_reminders_enabled=$5,product_updates_enabled=$6,quiet_start_hour=$7,quiet_end_hour=$8,timezone=$9,version=version+1,updated_at=now() WHERE recipient_id=$1::uuid AND version=$10 RETURNING preferred_channel,fallback_channel,opted_out,payment_reminders_enabled,product_updates_enabled,quiet_start_hour,quiet_end_hour,timezone,version`, recipient, prefs.PreferredChannel, prefs.FallbackChannel, prefs.OptedOut, prefs.PaymentRemindersEnabled, prefs.ProductUpdatesEnabled, prefs.QuietStart, prefs.QuietEnd, prefs.Timezone, expectedVersion).Scan(&out.PreferredChannel, &out.FallbackChannel, &out.OptedOut, &out.PaymentRemindersEnabled, &out.ProductUpdatesEnabled, &out.QuietStart, &out.QuietEnd, &out.Timezone, &out.Version)
		if errors.Is(err, pgx.ErrNoRows) {
			return Preferences{}, errors.New("notification preference version conflict")
		}
		if err != nil {
			return Preferences{}, err
		}
		if err = tx.Commit(ctx); err != nil {
			return Preferences{}, err
		}
		return out, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.preferences[recipient]
	if !ok {
		current = DefaultPreferences()
	}
	if expectedVersion != current.Version {
		return Preferences{}, errors.New("notification preference version conflict")
	}
	prefs.Version = current.Version + 1
	s.preferences[recipient] = prefs
	return prefs, nil
}

func validChannel(channel string) bool {
	return channel == ChannelEmail || channel == ChannelSMS || channel == ChannelWhatsApp
}

func (s *Store) SetReminderConsent(check func(context.Context, string, string) (bool, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reminderConsent = check
}
func (s *Store) SetOptionalProcessing(check func(context.Context, string) (bool, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.optionalProcessing = check
}
func (s *Store) reminderAllowed(ctx context.Context, event Event) (bool, error) {
	if event.Priority == PriorityCritical {
		return true, nil
	}
	s.mu.Lock()
	optional, check := s.optionalProcessing, s.reminderConsent
	s.mu.Unlock()
	if optional != nil {
		allowed, err := optional(ctx, event.RecipientID)
		if err != nil || !allowed {
			return false, err
		}
	}
	if event.Type != "PaymentDueSoon" || check == nil {
		return true, nil
	}
	if event.OrganizationID == "" {
		return false, nil
	}
	return check(ctx, event.RecipientID, event.OrganizationID)
}

func (s *Store) Emit(ctx context.Context, event Event) ([]Delivery, error) {
	event.Priority = defaultPriority(event.Priority)
	if event.ID == "" || event.Type == "" || event.RecipientID == "" {
		return nil, errors.New("event, type, and recipient are required")
	}
	allowed, err := s.reminderAllowed(ctx, event)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return []Delivery{}, nil
	}
	prefs, err := s.GetPreferences(ctx, event.RecipientID)
	if err != nil {
		return nil, err
	}
	channels := s.channelsFor(event, prefs)
	deliveries := []Delivery{}
	for _, channel := range channels {
		if event.Email != "" || event.Phone != "" {
			if channel == ChannelEmail && event.Email == "" || channel != ChannelEmail && event.Phone == "" {
				continue
			}
		}
		delivery, err := s.deliver(ctx, event, channel, prefs)
		if err != nil {
			return deliveries, err
		}
		deliveries = append(deliveries, delivery)
	}
	return deliveries, nil
}
func (s *Store) channelsFor(event Event, prefs Preferences) []string {
	preferred := prefs.PreferredChannel
	if preferred == "" {
		preferred = ChannelWhatsApp
	}
	fallback := prefs.FallbackChannel
	if fallback == "" {
		fallback = ChannelEmail
	}
	if event.Priority == PriorityCritical {
		return uniqueChannels([]string{ChannelWhatsApp, preferred, ChannelEmail, ChannelSMS})
	}
	if prefs.OptedOut || (event.Type == "PaymentDueSoon" && !prefs.PaymentRemindersEnabled) || (event.Type == "ProductUpdate" && !prefs.ProductUpdatesEnabled) {
		return []string{}
	}
	return uniqueChannels([]string{preferred, fallback})
}
func (s *Store) deliver(ctx context.Context, event Event, channel string, prefs Preferences) (Delivery, error) {
	if s.pool != nil {
		return s.deliverPostgres(ctx, event, channel, prefs)
	}
	key := event.ID + "|" + channel
	fingerprint := s.eventFingerprint(event)
	s.mu.Lock()
	if existing := s.dedupe[key]; existing != "" {
		if s.eventFingerprints[key] != fingerprint {
			s.mu.Unlock()
			return Delivery{}, errors.New("notification event belongs to a different intent")
		}
		delivery := *s.deliveries[existing]
		s.mu.Unlock()
		return delivery, nil
	}
	templateVersion := "v1"
	body := s.templates[event.Type+"|"+channel+"|"+templateVersion]
	if body == "" {
		body = defaultTemplate(event.Type)
	}
	body = render(body, event)
	now := s.now()
	delivery := &Delivery{ID: s.newID(), EventID: event.ID, RecipientID: event.RecipientID, Channel: channel, Template: event.Type, TemplateVersion: templateVersion, Body: body}
	if event.DeferDelivery || (event.Priority != PriorityCritical && inQuietHours(now, prefs)) {
		delivery.State = StateScheduled
		delivery.ScheduledAt = now
		if event.Priority != PriorityCritical && inQuietHours(now, prefs) {
			delivery.ScheduledAt = nextQuietEnd(now, prefs)
		}
		s.deliveries[delivery.ID] = delivery
		s.dedupe[key] = delivery.ID
		s.eventFingerprints[key] = fingerprint
		s.mu.Unlock()
		return cloneDelivery(*delivery), nil
	}
	provider := s.providers[channel]
	if provider == nil {
		delivery.State = StateFailed
		delivery.FailedAt = now
		delivery.FailureReason = "provider unavailable"
		s.deliveries[delivery.ID] = delivery
		s.dedupe[key] = delivery.ID
		s.eventFingerprints[key] = fingerprint
		s.mu.Unlock()
		return cloneDelivery(*delivery), nil
	}
	delivery.State = StateSending
	s.deliveries[delivery.ID] = delivery
	s.dedupe[key] = delivery.ID
	s.eventFingerprints[key] = fingerprint
	working := *delivery
	delivery = &working
	s.mu.Unlock()
	if event.SecurePath != "" {
		delivery.SecureLink = s.SecureLink(event.SecurePath, now.Add(15*time.Minute))
		delivery.Body += " Review securely: " + delivery.SecureLink
	}
	destination := event.Email
	if channel == ChannelSMS || channel == ChannelWhatsApp {
		destination = event.Phone
	}
	providerID, err := provider.Send(ctx, Message{EventID: event.ID, RecipientID: event.RecipientID, Destination: destination, Channel: channel, Template: event.Type, TemplateVersion: templateVersion, Body: delivery.Body, SecureLink: delivery.SecureLink})
	s.mu.Lock()
	defer s.mu.Unlock()
	stored := s.deliveries[delivery.ID]
	if stored == nil {
		stored = delivery
		s.deliveries[delivery.ID] = stored
	}
	stored.Body, stored.SecureLink = delivery.Body, delivery.SecureLink
	if err != nil {
		stored.State = deliveryFailureState(err)
		stored.FailedAt = s.now()
		stored.FailureReason = logging.Redact(err.Error())
	} else {
		stored.State = StateSent
		stored.ProviderMessageID = providerID
		stored.SentAt = s.now()
	}
	s.dedupe[key] = stored.ID
	return cloneDelivery(*stored), nil
}
func (s *Store) SecureLink(path string, expires time.Time) string {
	payload := path + "|" + strconv.FormatInt(expires.Unix(), 10)
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(payload))
	s.mu.Lock()
	baseURL := s.baseURL
	s.mu.Unlock()
	return baseURL + "/secure?path=" + hex.EncodeToString([]byte(path)) + "&exp=" + strconv.FormatInt(expires.Unix(), 10) + "&sig=" + hex.EncodeToString(mac.Sum(nil))
}
func (s *Store) VerifySecureLink(path string, expires time.Time, signature string) bool {
	if !time.Now().Before(expires) {
		return false
	}
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(path + "|" + strconv.FormatInt(expires.Unix(), 10)))
	return hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(signature))
}
func (s *Store) ListDeliveries(recipient string) []Delivery {
	if s.pool != nil {
		rows, err := s.pool.Query(context.Background(), `SELECT id::text,event_reference,recipient_id::text,channel,template,template_version,state,COALESCE(provider_message_id,''),body,scheduled_at,sent_at,failed_at,COALESCE(failure_reason,''),COALESCE(secure_link,'') FROM app.notifications WHERE recipient_id=$1::uuid ORDER BY COALESCE(sent_at,scheduled_at) DESC NULLS LAST`, recipient)
		if err != nil {
			return []Delivery{}
		}
		defer rows.Close()
		out := []Delivery{}
		for rows.Next() {
			delivery, scanErr := scanPersistentDelivery(rows)
			if scanErr != nil {
				return []Delivery{}
			}
			out = append(out, delivery)
		}
		if rows.Err() != nil {
			return []Delivery{}
		}
		return out
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Delivery{}
	for _, delivery := range s.deliveries {
		if delivery.RecipientID == recipient {
			out = append(out, cloneDelivery(*delivery))
		}
	}
	return out
}
func render(template string, event Event) string {
	return strings.NewReplacer("{{amount}}", formatAmount(event.AmountKobo, event.Currency), "{{date}}", event.Date.In(time.FixedZone("Africa/Lagos", 3600)).Format("2 Jan 2006"), "{{reference}}", event.Reference, "{{next_action}}", event.NextAction, "{{support_link}}", event.SupportLink).Replace(template)
}
func defaultTemplate(eventType string) string {
	switch eventType {
	case "MandateRevoked":
		return "Your bank debit permission is now off. Kredit can no longer take money from your account. Open Kredit to arrange how you will pay."
	case "CollectionRetryScheduled":
		return "We may try your bank again after {{date}} for money still unpaid. Check what you owe in Kredit."
	case "ObligationAccepted":
		return "You have agreed to this sale. Your payment days are saved in Kredit. Open it any time to see them."
	case "SystemAcceptance":
		return "The waiting period after your delivery notice has ended without a receipt confirmation or problem report. This sale is now active under its agreed delivery terms. This is a system record, not a confirmation from you. Open Kredit to review or report a problem."
	case "GoodsReleased":
		return "Your seller sent goods worth {{amount}}. Confirm they arrived or report a problem in Kredit. If we hear nothing by {{date}}, we record them as received."
	case "ObligationRepaid":
		return "You have finished paying this sale. Nothing is left to pay. The full payment record is in Kredit."
	case "MandateExpiring":
		return "Your bank debit permission ends on {{date}}. Open Kredit to renew it, or arrange another way to pay."
	case "CollectionUncertain":
		return "We are still checking with the bank about {{amount}}. Do not pay again until you have checked this sale in Kredit."
	case "CollectionFailed":
		return "The bank debit did not go through. Open Kredit to see what is still owed and what to do next."
	case "CollectionCancelled":
		return "The bank debit was cancelled. Open Kredit to see if anything is still owed."
	case "PaymentRecorded":
		return "{{amount}} received and recorded. Reference: {{reference}}."
	case "PriorDebitNotice":
		return "If unpaid, we may debit your bank up to {{amount}} on or after {{date}}, as you agreed. Open Kredit first to check your payment days or report a problem."
	case "PaymentDueSoon":
		return "Your {{amount}} payment is due on {{date}}. {{next_action}}"
	case "CollectionSubmitted":
		return "{{amount}} was still unpaid after the extra days agreed, so we have asked your bank for it. Reference: {{reference}}."
	case "PaymentReversed":
		return "A payment of {{amount}} has come back, so that money is owed again. Check your balance in Kredit. Reference: {{reference}}. {{next_action}}"
	case "FinancialAdjustmentRecorded":
		return "We have adjusted this sale by {{amount}}. Open Kredit to see the new balance. Reference: {{reference}}."
	case "DisputeUpdated":
		return "There is an update on the problem about {{amount}}. Open Kredit to see the decision and what is left. Reference: {{reference}}. {{next_action}}"
	case "DisputeOpened":
		return "A problem was reported about {{amount}}. See what happens next: {{support_link}}"
	case "FeeInvoiceIssued":
		return "Your Kredit fee bill is ready. Open Kredit to see the charges and payment instructions. Reference: {{reference}}. {{next_action}}"
	case "FeeInvoiceRefundRecorded":
		return "A refund of Kredit fees has been recorded. Open your fee bill to see the bank record and remaining credit. Reference: {{reference}}. {{next_action}}"
	case "FeeInvoicePaymentRecorded":
		return "A payment toward your Kredit fee bill has been recorded. Open Kredit to see the remaining balance. Reference: {{reference}}. {{next_action}}"
	case "SupplierSensitiveSettingChanged":
		return "An important setting on your Kredit account was changed: {{reference}}. If this was not you, contact us now."
	case "ScheduleAmendment":
		return "Somebody wants to change your payment days. Nothing changes until you agree. {{next_action}}. Reference: {{reference}}"
	case "OperationsControlApplied":
		return "A protected control was applied to this account: {{reference}}. Next: {{next_action}}"
	case "SupplierVerificationOutcome":
		return "Your business check is now {{reference}}. {{next_action}}"
	case "SupplierPilotReady":
		return "Your Kredit account is ready to use. {{next_action}}"
	case "SupplierOnboardingRequirementExpired":
		return "One of your setup steps has expired. Finish it in Kredit before money can move again."
	case "NotificationPreferencesChanged":
		return "Your Kredit message settings were changed. If this was not you, secure your account now."
	case "AccountRecoveryRequested":
		return "Somebody asked to recover this Kredit account. If it was not you, cancel it now from a device where you are still signed in."
	case "AccountRecoveryCoolingOff":
		return "Your account recovery was approved. There is a short safety wait before it finishes. Until then, money changes stay blocked."
	case "AccountRecoveryCancelled":
		return "The account recovery request was cancelled."
	case "AccountRecoveryCompleted":
		return "Your account recovery is done. Anyone signed in before has been signed out, and your old recovery codes no longer work."
	case "PrivacyRequestReceived":
		return "We have received your request about your information. Reference: {{reference}}."
	case "PrivacyClarificationRequired":
		return "We need a little more detail about your information request. Open Kredit to answer."
	case "PrivacyRequestDecided":
		return "There is a decision on your information request. Reference: {{reference}}."
	case "PrivacyExportReady":
		return "Your copy of your information is ready to download. The link works for a short time only."
	case "PrivacyRequestCompleted":
		return "Your information request is finished. Reference: {{reference}}."
	default:
		return "Kredit update: {{reference}}. {{next_action}}"
	}
}

// formatAmount renders a kobo amount for a notification body.
// Thousands separators are added so the figure is readable on a phone, and a
// whole amount drops the ".00" tail. The currency stays as an ISO code rather
// than the naira sign on purpose: the sign is outside the GSM-7 alphabet, so
// including it would force SMS bodies into UCS-2 and cut a paid segment from
// 160 characters to 70.
func formatAmount(amount int64, currency string) string {
	if currency == "" {
		currency = "NGN"
	}
	sign, whole, fraction := "", amount/100, amount%100
	if amount < 0 {
		sign, whole, fraction = "-", -whole, -fraction
	}
	digits := strconv.FormatInt(whole, 10)
	var grouped strings.Builder
	for index, digit := range digits {
		if index > 0 && (len(digits)-index)%3 == 0 {
			grouped.WriteByte(',')
		}
		grouped.WriteRune(digit)
	}
	if fraction == 0 {
		return fmt.Sprintf("%s %s%s", currency, sign, grouped.String())
	}
	return fmt.Sprintf("%s %s%s.%02d", currency, sign, grouped.String(), fraction)
}
func inQuietHours(now time.Time, prefs Preferences) bool {
	if prefs.QuietStart == prefs.QuietEnd {
		return false
	}
	location, err := time.LoadLocation(prefs.Timezone)
	if err != nil {
		location = time.FixedZone("Africa/Lagos", 3600)
	}
	hour, _ := strconv.Atoi(now.In(location).Format("15"))
	if prefs.QuietStart < prefs.QuietEnd {
		return hour >= prefs.QuietStart && hour < prefs.QuietEnd
	}
	return hour >= prefs.QuietStart || hour < prefs.QuietEnd
}
func nextQuietEnd(now time.Time, prefs Preferences) time.Time {
	location, err := time.LoadLocation(prefs.Timezone)
	if err != nil {
		location = time.FixedZone("Africa/Lagos", 3600)
	}
	local := now.In(location)
	end := time.Date(local.Year(), local.Month(), local.Day(), prefs.QuietEnd, 0, 0, 0, location)
	if !end.After(local) {
		end = end.Add(24 * time.Hour)
	}
	return end.UTC()
}
func uniqueChannels(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
func cloneDelivery(v Delivery) Delivery { return v }
func newIdentifier() string             { return identifier.New() }

func (s *Store) deliverPostgres(ctx context.Context, event Event, channel string, prefs Preferences) (Delivery, error) {
	templateVersion := "v1"
	s.mu.Lock()
	body := s.templates[event.Type+"|"+channel+"|"+templateVersion]
	provider := s.providers[channel]
	s.mu.Unlock()
	if body == "" {
		var persisted string
		err := s.pool.QueryRow(ctx, `SELECT body FROM app.notification_templates WHERE event_type=$1 AND channel=$2 AND version=$3 AND active=true`, event.Type, channel, templateVersion).Scan(&persisted)
		if err == nil {
			body = persisted
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return Delivery{}, err
		}
	}
	if body == "" {
		body = defaultTemplate(event.Type)
	}
	body = render(body, event)
	now := s.now()
	delivery := Delivery{ID: s.newID(), EventID: event.ID, RecipientID: event.RecipientID, Channel: channel, Template: event.Type, TemplateVersion: templateVersion, Body: body}
	if event.SecurePath != "" {
		delivery.SecureLink = s.SecureLink(event.SecurePath, now.Add(15*time.Minute))
		delivery.Body += " Review securely: " + delivery.SecureLink
	}
	destination := event.Email
	if channel == ChannelSMS || channel == ChannelWhatsApp {
		destination = event.Phone
	}
	ciphertext, err := s.encryptDestination([]byte(destination))
	if err != nil {
		return Delivery{}, err
	}
	shouldSend := true
	if event.DeferDelivery || (event.Priority != PriorityCritical && inQuietHours(now, prefs)) {
		delivery.State = StateScheduled
		delivery.ScheduledAt = now
		if event.Priority != PriorityCritical && inQuietHours(now, prefs) {
			delivery.ScheduledAt = nextQuietEnd(now, prefs)
		}
		shouldSend = false
	} else if provider == nil {
		delivery.State = StateFailed
		delivery.FailedAt = now
		delivery.FailureReason = "provider unavailable"
		shouldSend = false
	} else {
		delivery.State = StateSending
	}
	inserted := false
	err = s.pool.QueryRow(ctx, `INSERT INTO app.notifications(id,recipient_id,channel,template,template_version,event_reference,state,body,scheduled_at,failed_at,failure_reason,secure_link,destination_ciphertext,lease_expires_at,supplier_organization_id,priority,send_started_at,event_fingerprint) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,''),NULLIF($12,''),$13,$14,NULLIF($15,'')::uuid,$16,CASE WHEN $17 THEN now() END,$18) ON CONFLICT(event_reference,channel) DO NOTHING RETURNING true`, delivery.ID, delivery.RecipientID, delivery.Channel, delivery.Template, delivery.TemplateVersion, delivery.EventID, delivery.State, delivery.Body, nullableTime(delivery.ScheduledAt), nullableTime(delivery.FailedAt), delivery.FailureReason, delivery.SecureLink, ciphertext, leaseTime(shouldSend, now), event.OrganizationID, defaultPriority(event.Priority), shouldSend, s.eventFingerprint(event)).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		var fingerprint string
		if err := s.pool.QueryRow(ctx, `SELECT COALESCE(event_fingerprint,'') FROM app.notifications WHERE event_reference=$1 AND channel=$2`, event.ID, channel).Scan(&fingerprint); err != nil {
			return Delivery{}, err
		}
		if fingerprint != s.eventFingerprint(event) {
			return Delivery{}, errors.New("notification event belongs to a different or unverifiable intent")
		}
		existing, loadErr := s.loadPersistentDelivery(ctx, event.ID, channel)
		return existing, loadErr
	}
	if err != nil {
		return Delivery{}, err
	}
	if !shouldSend {
		return delivery, nil
	}
	providerID, sendErr := provider.Send(ctx, Message{EventID: event.ID, RecipientID: event.RecipientID, Destination: destination, Channel: channel, Template: event.Type, TemplateVersion: templateVersion, Body: delivery.Body, SecureLink: delivery.SecureLink})
	if sendErr != nil {
		delivery.State = deliveryFailureState(sendErr)
		delivery.FailedAt = s.now()
		delivery.FailureReason = logging.Redact(sendErr.Error())
		_, err = s.pool.Exec(ctx, `UPDATE app.notifications SET state=$4,failed_at=$2,failure_reason=$3,lease_expires_at=NULL,delivery_attempts=CASE WHEN $5 THEN 8 ELSE delivery_attempts END,updated_at=now() WHERE id=$1::uuid AND state='sending' AND delivery_attempts=0`, delivery.ID, delivery.FailedAt, delivery.FailureReason, delivery.State, stopAutomaticDeliveryRetry(sendErr))
		return delivery, err
	}
	delivery.State = StateSent
	delivery.ProviderMessageID = providerID
	delivery.SentAt = s.now()
	_, err = s.pool.Exec(ctx, `UPDATE app.notifications SET state='sent',provider_message_id=$2,sent_at=$3,lease_expires_at=NULL,updated_at=now() WHERE id=$1::uuid AND state='sending' AND delivery_attempts=0`, delivery.ID, providerID, delivery.SentAt)
	return delivery, err
}
func (s *Store) loadPersistentDelivery(ctx context.Context, eventID, channel string) (Delivery, error) {
	return scanPersistentDelivery(s.pool.QueryRow(ctx, `SELECT id::text,event_reference,recipient_id::text,channel,template,template_version,state,COALESCE(provider_message_id,''),body,scheduled_at,sent_at,failed_at,COALESCE(failure_reason,''),COALESCE(secure_link,'') FROM app.notifications WHERE event_reference=$1 AND channel=$2`, eventID, channel))
}

// DueDeliveryIDs returns persisted work that is ready to be put on the
// durable notification queue. Expired sending leases are included so a worker
// crash cannot strand a delivery forever. The connector receives a stable
// idempotency key, making these recovery attempts safe.
func (s *Store) DueDeliveryIDs(ctx context.Context, limit int) ([]string, error) {
	if s.pool == nil {
		return nil, errors.New("notification database is not configured")
	}
	if limit <= 0 || limit > 500 {
		return nil, errors.New("notification delivery limit must be between 1 and 500")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text
		FROM app.notifications
		WHERE delivery_attempts < 8 AND (
			(state = 'scheduled' AND scheduled_at <= now()) OR
			(state = 'failed' AND COALESCE(next_attempt_at, now()) <= now()) OR
			(state = 'sending' AND lease_expires_at <= now())
		)
		ORDER BY COALESCE(next_attempt_at, scheduled_at, lease_expires_at, updated_at)
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// DeliverScheduled claims and sends one persisted notification. The UPDATE is
// the ownership boundary: only one worker can acquire an eligible row, while
// expired leases remain recoverable after a process failure.
func (s *Store) DeliverScheduled(ctx context.Context, id string) error {
	if s.pool == nil || strings.TrimSpace(id) == "" {
		return errors.New("notification database and id are required")
	}
	var message Message
	var ciphertext []byte
	var organizationID, priority string
	var attempt int
	var sendStarted *time.Time
	err := s.pool.QueryRow(ctx, `
		UPDATE app.notifications
		SET state='sending', lease_expires_at=now()+interval '10 minutes',
			delivery_attempts=delivery_attempts+1, next_attempt_at=NULL,
			failed_at=NULL, failure_reason=NULL, updated_at=now()
		WHERE id=$1::uuid AND delivery_attempts < 8 AND (
			(state='scheduled' AND scheduled_at<=now()) OR
			(state='failed' AND COALESCE(next_attempt_at,now())<=now()) OR
			(state='sending' AND lease_expires_at<=now())
		)
		RETURNING event_reference,recipient_id::text,channel,template,
			template_version,body,COALESCE(secure_link,''),destination_ciphertext,COALESCE(supplier_organization_id::text,''),priority,delivery_attempts,send_started_at`, id).
		Scan(&message.EventID, &message.RecipientID, &message.Channel, &message.Template,
			&message.TemplateVersion, &message.Body, &message.SecureLink, &ciphertext, &organizationID, &priority, &attempt, &sendStarted)
	if errors.Is(err, pgx.ErrNoRows) {
		// A sent row or a lease owned by another worker is already complete from
		// this job's perspective.
		return nil
	}
	if err != nil {
		return err
	}
	prefs, err := s.GetPreferences(ctx, message.RecipientID)
	if err != nil {
		return s.failDelivery(ctx, id, attempt, err)
	}
	allowed, err := s.reminderAllowed(ctx, Event{RecipientID: message.RecipientID, OrganizationID: organizationID, Type: message.Template, Priority: priority})
	if err != nil {
		return s.failDelivery(ctx, id, attempt, err)
	}
	channelAllowed := false
	for _, channel := range s.channelsFor(Event{Type: message.Template, Priority: priority}, prefs) {
		if channel == message.Channel {
			channelAllowed = true
		}
	}
	if !allowed || !channelAllowed {
		_, err = s.pool.Exec(ctx, "UPDATE app.notifications SET state='suppressed',lease_expires_at=NULL,next_attempt_at=NULL,updated_at=now() WHERE id=$1::uuid AND state='sending' AND delivery_attempts=$2", id, attempt)
		return err
	}
	if priority != PriorityCritical && inQuietHours(s.now(), prefs) {
		_, err = s.pool.Exec(ctx, `UPDATE app.notifications SET state='scheduled',scheduled_at=$3,lease_expires_at=NULL,next_attempt_at=NULL,delivery_attempts=delivery_attempts-1,updated_at=now() WHERE id=$1::uuid AND state='sending' AND delivery_attempts=$2`, id, attempt, nextQuietEnd(s.now(), prefs))
		return err
	}
	destination, err := s.decryptDestination(ciphertext)
	if err != nil {
		return s.failDelivery(ctx, id, attempt, err)
	}
	message.Destination = string(destination)
	s.mu.Lock()
	provider := s.providers[message.Channel]
	s.mu.Unlock()
	if provider == nil {
		return s.failDelivery(ctx, id, attempt, errors.New("notification provider is unavailable"))
	}
	if sendStarted == nil {
		// Prepare the short-lived link when a queued message is actually sent.
		// Persist it before contacting the provider so retries keep the exact
		// same request identity and body, even after an unknown outcome.
		if message.SecureLink != "" {
			parsed, parseErr := url.Parse(message.SecureLink)
			if parseErr != nil {
				return s.failDelivery(ctx, id, attempt, errors.New("queued secure link is invalid"))
			}
			path, decodeErr := hex.DecodeString(parsed.Query().Get("path"))
			if decodeErr != nil || len(path) == 0 || !strings.HasPrefix(string(path), "/") {
				return s.failDelivery(ctx, id, attempt, errors.New("queued secure link path is invalid"))
			}
			fresh := s.SecureLink(string(path), s.now().Add(15*time.Minute))
			message.Body = strings.ReplaceAll(message.Body, message.SecureLink, fresh)
			message.SecureLink = fresh
		}
		saved, saveErr := s.pool.Exec(ctx, `UPDATE app.notifications SET body=$3,secure_link=NULLIF($4,''),send_started_at=now() WHERE id=$1::uuid AND state='sending' AND delivery_attempts=$2 AND send_started_at IS NULL`, id, attempt, message.Body, message.SecureLink)
		if saveErr != nil {
			return saveErr
		}
		if saved.RowsAffected() != 1 {
			return errors.New("notification delivery lease was lost")
		}
	}
	sendCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	providerID, err := provider.Send(sendCtx, message)
	cancel()

	if err != nil {
		return s.failDelivery(ctx, id, attempt, err)
	}
	command, err := s.pool.Exec(ctx, `UPDATE app.notifications SET state='sent',provider_message_id=$2,sent_at=now(),lease_expires_at=NULL,next_attempt_at=NULL,updated_at=now() WHERE id=$1::uuid AND state='sending' AND delivery_attempts=$3`, id, providerID, attempt)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return errors.New("notification delivery lease was lost")
	}
	return nil
}

func (s *Store) failDelivery(ctx context.Context, id string, attempt int, deliveryErr error) error {
	if deliveryErr == nil {
		deliveryErr = errors.New("notification delivery failed")
	}
	_, updateErr := s.pool.Exec(ctx, `UPDATE app.notifications SET state=$4,failed_at=now(),failure_reason=$2,lease_expires_at=NULL,next_attempt_at=CASE WHEN $5 THEN NULL ELSE now() + LEAST(interval '1 hour', interval '30 seconds' * power(2, GREATEST(delivery_attempts-1,0))) END,delivery_attempts=CASE WHEN $5 THEN GREATEST(delivery_attempts,8) ELSE delivery_attempts END,updated_at=now() WHERE id=$1::uuid AND state='sending' AND delivery_attempts=$3`, id, logging.Redact(deliveryErr.Error()), attempt, deliveryFailureState(deliveryErr), stopAutomaticDeliveryRetry(deliveryErr))
	if updateErr != nil {
		return errors.Join(deliveryErr, updateErr)
	}
	return deliveryErr
}

type notificationScanner interface{ Scan(...any) error }

func scanPersistentDelivery(row notificationScanner) (Delivery, error) {
	var d Delivery
	var scheduled, sent, failed *time.Time
	err := row.Scan(&d.ID, &d.EventID, &d.RecipientID, &d.Channel, &d.Template, &d.TemplateVersion, &d.State, &d.ProviderMessageID, &d.Body, &scheduled, &sent, &failed, &d.FailureReason, &d.SecureLink)
	if scheduled != nil {
		d.ScheduledAt = *scheduled
	}
	if sent != nil {
		d.SentAt = *sent
	}
	if failed != nil {
		d.FailedAt = *failed
	}
	return d, err
}
func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
func leaseTime(send bool, now time.Time) any {
	if !send {
		return nil
	}
	return now.Add(10 * time.Minute)
}
func (s *Store) encryptDestination(value []byte) ([]byte, error) {
	if len(value) == 0 {
		return nil, nil
	}
	block, err := aes.NewCipher(s.encryption)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, value, nil), nil
}

func (s *Store) decryptDestination(value []byte) ([]byte, error) {
	if len(value) == 0 {
		return nil, errors.New("notification destination is unavailable")
	}
	block, err := aes.NewCipher(s.encryption)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(value) < gcm.NonceSize() {
		return nil, errors.New("notification destination is invalid")
	}
	nonce, ciphertext := value[:gcm.NonceSize()], value[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func defaultPriority(value string) string {
	if value == PriorityRoutine {
		return PriorityRoutine
	}
	return PriorityCritical
}

// Match the actual day shown in the message, not incidental caller nanoseconds.
// HMAC keeps contact details out of persistent evidence and offline guessing.
func (s *Store) eventFingerprint(event Event) string {
	encoded, _ := json.Marshal([]any{event.ID, event.Type, event.RecipientID, event.OrganizationID, event.Email, event.Phone, defaultPriority(event.Priority), event.AmountKobo, event.Currency, event.Date.In(time.FixedZone("Africa/Lagos", 3600)).Format("2006-01-02"), event.Reference, event.NextAction, event.SupportLink, event.SecurePath})
	hash := hmac.New(sha256.New, s.secret)
	_, _ = hash.Write(encoded)
	return hex.EncodeToString(hash.Sum(nil))
}
