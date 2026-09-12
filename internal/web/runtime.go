package web

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"kredit/internal/audit"
	"kredit/internal/auth"
	"kredit/internal/billing"
	"kredit/internal/businesspolicy"
	"kredit/internal/buyers"
	"kredit/internal/collections"
	"kredit/internal/config"
	"kredit/internal/corrections"
	"kredit/internal/credit"
	"kredit/internal/db"
	"kredit/internal/disputes"
	"kredit/internal/documents"
	"kredit/internal/feedback"
	"kredit/internal/idempotency"
	"kredit/internal/identity"
	"kredit/internal/jobs"
	"kredit/internal/ledger"
	"kredit/internal/legalpublication"
	"kredit/internal/mandates"
	"kredit/internal/notifications"
	"kredit/internal/observability"
	"kredit/internal/onboarding"
	"kredit/internal/operations"
	"kredit/internal/organizations"
	"kredit/internal/outbox"
	"kredit/internal/paymentclaims"
	"kredit/internal/payments"
	"kredit/internal/platformops"
	"kredit/internal/platformsettings"
	"kredit/internal/providers/mono"
	"kredit/internal/readiness"
	"kredit/internal/relationships"
	"kredit/internal/reports"
	"kredit/internal/schedules"
	"kredit/internal/settlement"
	"kredit/internal/support"
	"kredit/internal/tradelines"
	"kredit/internal/usercontrol"
	"kredit/internal/whatsapp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Runtime struct {
	Settlement                settlement.Provider
	BusinessPolicies          *businesspolicy.Store
	policyInitializationError error
	Mono                      *mono.Client
	MonoAccounts              map[string]*mono.Client
	FeeBilling                *billing.FeeService
	WebhookJobs               *jobs.Client
	Database                  *db.Pool
	Persistence               PersistenceStatus
	Idempotency               idempotency.Service
	Auth                      auth.Service
	Organizations             organizations.Service
	Onboarding                onboarding.Service
	Audit                     audit.Service
	Identity                  identity.IdentityProvider
	Buyers                    buyers.Service
	Mandates                  mandates.Provider
	Ledger                    ledger.Service
	Credit                    credit.Service
	Payments                  payments.Service
	PaymentClaims             paymentclaims.Service
	PaymentClaimsEnabled      bool
	Schedules                 *schedules.Store
	TradeLines                tradelines.Service
	Collections               collections.Service
	Disputes                  disputes.Service
	Documents                 *documents.Store
	DocumentScanner           documents.Scanner
	Relationships             relationships.Service
	Support                   *support.Store
	Operations                operations.Service
	Reports                   *reports.Store
	Corrections               corrections.Service
	Readiness                 readiness.Report
	Metrics                   *observability.Store
	Tracer                    *observability.Tracer
	Notifications             *notifications.Store
	WhatsApp                  *whatsapp.Handler
	Outbox                    *outbox.Store
	PlatformOps               *platformops.Store
	PlatformSettings          platformsettings.Service
	UserControl               *usercontrol.Store
	Feedback                  *feedback.Store
	// ProviderFailures records adapters that were configured but failed to
	// construct. Readiness reports these; they are never silently ignored.
	ProviderFailures []string
}

// PersistenceStatus describes which runtime boundaries are actually backed
// by PostgreSQL. It is intentionally capability-oriented so readiness cannot
// infer durability from the mere presence of a database connection.
type PersistenceStatus struct {
	DatabaseConfigured      bool
	AuthDurable             bool
	BuyerDurable            bool
	CreditDurable           bool
	LedgerDurable           bool
	AuditDurable            bool
	IdempotencyDurable      bool
	DocumentsDurable        bool
	SupportDurable          bool
	DomainAggregatesDurable bool
}

// DurableDomainReady reports whether all state-bearing domain aggregates are
// safe to run across multiple API/worker instances. Development intentionally
// uses process-local adapters; a database-backed runtime selects the durable
// implementations for every aggregate and the transactional outbox.
func (r *Runtime) DurableDomainReady() bool {
	return r != nil && r.Persistence.DatabaseConfigured &&
		r.policyInitializationError == nil && r.Persistence.AuthDurable && r.Persistence.LedgerDurable && r.Persistence.AuditDurable &&
		r.Persistence.IdempotencyDurable && r.Persistence.DomainAggregatesDurable
}

func NewRuntime(cfg config.Config) *Runtime {
	return NewRuntimeWithDB(cfg, nil)
}

func NewRuntimeWithDB(cfg config.Config, database *db.Pool) *Runtime {
	// A provider that fails to construct must be visible. Discarding the error
	// leaves an adapter silently absent, which is indistinguishable from an
	// adapter that is deliberately switched off.
	var providerFailures []string
	var identityProvider identity.IdentityProvider = identity.NewMockProvider()
	if cfg.Environment != "development" {
		identityProvider = identity.NewUnavailableProvider("certified identity provider adapter is not configured")
		if cfg.RealIdentity {
			var provider identity.IdentityProvider
			var err error
			if cfg.IdentityAdapter == "mono" && database != nil {
				provider, err = identity.NewNativeLookup(database.Raw(), cfg.IdentityProvider, cfg.IdentityProviderEndpoint, cfg.IdentityProviderToken)
			} else if cfg.IdentityAdapter == "mono" {
				err = errors.New("native identity requires durable storage")
			} else {
				provider, err = identity.NewWebhookProvider(cfg.IdentityProvider, cfg.IdentityProviderEndpoint, cfg.IdentityProviderToken, cfg.IdentityWebhookSecret)
			}
			if err != nil {
				providerFailures = append(providerFailures, fmt.Sprintf("identity provider unavailable: %v", err))
			} else {
				identityProvider = provider
			}
		}
	}

	retainedIdentity, retainedIdentityErr := cfg.RetainedIdentities()
	if retainedIdentityErr != nil {
		providerFailures = append(providerFailures, retainedIdentityErr.Error())
	}
	var retainedIdentityAdapters []identity.IdentityProvider
	for _, account := range retainedIdentity {
		if account.Name == identityProvider.Name() {
			continue
		}
		var adapter identity.IdentityProvider
		var err error
		if account.Adapter == "mono" && database != nil {
			adapter, err = identity.NewNativeLookup(database.Raw(), account.Name, account.Endpoint, account.Token)
		} else if account.Adapter == "mono" {
			err = errors.New("native identity requires durable storage")
		} else {
			adapter, err = identity.NewWebhookProvider(account.Name, account.Endpoint, account.Token, account.WebhookSecret)
		}
		if err != nil {
			providerFailures = append(providerFailures, "saved identity account is unavailable")
			continue
		}
		retainedIdentityAdapters = append(retainedIdentityAdapters, adapter)
	}
	if router, err := identity.NewRouter(identityProvider, retainedIdentityAdapters...); err == nil {
		identityProvider = router
	} else {
		providerFailures = append(providerFailures, err.Error())
	}
	var monoClient *mono.Client
	monoAccounts := map[string]*mono.Client{}
	var settlementProvider settlement.Provider
	var webhookJobs *jobs.Client
	if database != nil {
		var jobsErr error
		if webhookJobs, jobsErr = jobs.NewEnqueueClient(database.Raw()); jobsErr != nil {
			providerFailures = append(providerFailures, fmt.Sprintf("webhook job client unavailable: %v", jobsErr))
		}
		if cfg.MonoSecretKey != "" {
			resolveCustomer := func(ctx context.Context, userID, businessID string) (string, error) {
				var reference string
				err := database.Raw().QueryRow(ctx, `SELECT provider_customer_reference FROM app.provider_customer_bindings WHERE provider=$3 AND buyer_user_id=$1::uuid AND buyer_business_id=$2::uuid`, userID, businessID, cfg.MonoAccount()).Scan(&reference)
				return reference, err
			}
			// The key decides the environment, and configuration has already
			// refused a sandbox key in production and a live key outside it. A
			// construction failure is recorded rather than discarded, so a bad
			// redirect URL surfaces in readiness instead of silently leaving the
			// provider absent.
			sandboxKey := strings.HasPrefix(cfg.MonoSecretKey, "test_sk_")
			build := mono.NewLive
			if sandboxKey {
				build = mono.New
			}
			client, monoErr := build(mono.DefaultBaseURL, cfg.MonoSecretKey, cfg.MonoWebhookSecret, cfg.MonoRedirectURL, cfg.PartialSweepEnabled, resolveCustomer)
			switch {
			case monoErr != nil:
				providerFailures = append(providerFailures, fmt.Sprintf("mono client unavailable: %v", monoErr))
			case sandboxKey && cfg.Environment == "production":
				providerFailures = append(providerFailures, "mono sandbox key refused in production")
			case !sandboxKey && cfg.Environment != "production":
				providerFailures = append(providerFailures, "mono live key refused outside production")
			default:
				monoClient = client.WithAccountName(cfg.MonoAccount())
				monoAccounts[monoClient.Name()] = monoClient
			}
		}
	}

	if database != nil {
		saved, err := cfg.RetainedCollections()
		if err != nil {
			providerFailures = append(providerFailures, err.Error())
		}
		for _, account := range saved {
			if account.Adapter != "mono" {
				continue
			}
			if _, exists := monoAccounts[account.Name]; exists {
				continue
			}
			build := mono.New
			if cfg.Environment == "production" {
				build = mono.NewLive
			}
			client, err := build(account.Endpoint, account.Token, account.WebhookSecret, "https://kredit.ng/buyer", account.Partial, func(context.Context, string, string) (string, error) {
				return "", errors.New("new customer registration is disabled on this saved account")
			})
			if err != nil {
				providerFailures = append(providerFailures, "saved Mono account unavailable")
				continue
			}
			monoAccounts[account.Name] = client.WithAccountName(account.Name).ReconciliationOnly()
		}
	}
	if cfg.SettlementEnabled {
		if cfg.SettlementProvider == cfg.MonoAccount() {
			if monoClient != nil {
				settlementProvider = monoClient
			} else {
				providerFailures = append(providerFailures, "seller bank registration is unavailable")
			}
		} else {
			connector, err := settlement.NewConnector(cfg.SettlementProvider, cfg.SettlementEndpoint, cfg.SettlementToken)
			if err != nil {
				providerFailures = append(providerFailures, "seller bank connector is unavailable")
			} else {
				settlementProvider = connector
			}
		}
	}
	mandateProvider := mandates.NewMockProvider()
	var mandateRuntime mandates.Provider = mandateProvider
	if database != nil {
		mandateRuntime = mandates.NewPostgresProvider(database.Raw(), cfg.CollectionProvider)
		if cfg.Environment != "development" && cfg.RealCollections && cfg.CollectionProvider != cfg.MonoAccount() {
			remote, err := mandates.NewWebhookProvider(cfg.CollectionProvider, cfg.CollectionProviderEndpoint, cfg.CollectionProviderToken)
			if err != nil {
				providerFailures = append(providerFailures, fmt.Sprintf("mandate connector unavailable: %v", err))
			} else {
				mandateRuntime = mandates.NewPostgresProviderWithRemote(database.Raw(), remote)
			}
		}
	}
	if cfg.CollectionProvider == cfg.MonoAccount() {
		mandateRuntime = mandates.NewUnavailableProvider(cfg.MonoAccount())
	}
	if monoClient != nil {
		if !cfg.MonoSweepEnabled {
			monoClient = monoClient.ReconciliationOnly()
		}
		if cfg.CollectionProvider == cfg.MonoAccount() {
			mandateRuntime = mandates.NewPostgresProviderWithRemote(database.Raw(), monoClient)
		}
	}

	if database != nil {
		retained, err := cfg.RetainedCollections()
		if err != nil {
			providerFailures = append(providerFailures, err.Error())
		}
		var savedMandates []*mandates.PostgresProvider
		if monoClient != nil && monoClient.Name() != mandateRuntime.Name() {
			savedMandates = append(savedMandates, mandates.NewPostgresProviderWithRemote(database.Raw(), monoClient.ReconciliationOnly()))
		}
		_, activeMandateConnected := mandateRuntime.(*mandates.PostgresProvider)
		for _, account := range retained {
			if (account.Name == mandateRuntime.Name() && activeMandateConnected) || (monoClient != nil && account.Name == monoClient.Name()) {
				continue
			}
			var remote mandates.Provider
			var err error
			if account.Adapter == "mono" {
				client := monoAccounts[account.Name]
				if client == nil {
					err = errors.New("saved Mono mandate account unavailable")
				} else {
					remote = client
				}
			} else {
				remote, err = mandates.NewWebhookProvider(account.Name, account.Endpoint, account.Token)
			}
			if err != nil {
				providerFailures = append(providerFailures, "saved mandate account unavailable")
				continue
			}
			savedMandates = append(savedMandates, mandates.NewPostgresProviderWithRemote(database.Raw(), remote))
		}
		if routed, err := mandates.NewRouter(database.Raw(), mandateRuntime, savedMandates...); err != nil {
			providerFailures = append(providerFailures, err.Error())
		} else {
			mandateRuntime = routed
		}
	}
	var outboxStore *outbox.Store
	var platformOpsStore *platformops.Store
	var platformSettingsStore platformsettings.Service
	var policies *businesspolicy.Store
	var policyError error
	feedbackStore := feedback.NewStore()
	var ledgerStore ledger.Service = ledger.NewStore()
	if database != nil {
		outboxStore = outbox.NewStore(database.Raw())
		platformOpsStore = platformops.NewStore(database.Raw())
		platformSettingsStore = platformsettings.NewPostgresStore(database.Raw(), platformsettings.NewEncryptor(cfg.SettingsEncryptionKey), nil)
		policies = businesspolicy.NewStore(database.Raw(), cfg)
		policyError = policies.ValidateStartup(context.Background())
		feedbackStore = feedback.NewPostgresStore(database.Raw())
		ledgerStore = ledger.NewPostgresStoreWithOutbox(database.Raw(), outboxStore)
	}
	scheduleStore := schedules.NewStore()
	var tradeLineStore tradelines.Service = tradelines.NewStore()
	var tradeLinePostgres *tradelines.PostgresStore
	if database != nil {
		scheduleStore = schedules.NewPostgresStore(database.Raw())
		tradeLinePostgres = tradelines.NewPostgresStoreWithOutbox(database.Raw(), outboxStore)
		tradeLineStore = tradeLinePostgres
	}
	if configurable, ok := tradeLineStore.(interface{ SetLegalReader(legalpublication.Reader) }); ok {
		configurable.SetLegalReader(legalpublication.SettingsReader(platformSettingsStore))
	}
	if cfg.PilotMaxActiveExposureKobo > 0 {
		tradeLineStore.SetMaxActiveExposure(ledger.Money(cfg.PilotMaxActiveExposureKobo))
		tradeLineStore.SetLineGuard(func(input tradelines.CreateLineInput) error {
			if int64(input.ApprovedLimitKobo) > cfg.PilotMaxActiveExposureKobo {
				return fmt.Errorf("trade-line limit exceeds configured pilot exposure limit")
			}
			return nil
		})
	}
	if cfg.PilotMaxDrawdownsPerLineDay > 0 {
		tradeLineStore.SetMaxDrawdownsPerLineDay(int(cfg.PilotMaxDrawdownsPerLineDay))
	}
	creditStore := credit.NewStore(mandateRuntime, ledgerStore)
	creditStore.SetLegalReader(legalpublication.SettingsReader(platformSettingsStore))
	if cfg.PilotEnhancedReviewKobo > 0 {
		creditStore.SetEnhancedReviewThreshold(ledger.Money(cfg.PilotEnhancedReviewKobo))
	}
	if cfg.PilotMaxActiveExposureKobo > 0 {
		creditStore.SetMaxActiveExposure(ledger.Money(cfg.PilotMaxActiveExposureKobo))
	}
	readinessReport := readiness.Evaluate(cfg)
	tracer := observability.NewNoopTracer()
	if cfg.Environment != "development" {
		if configuredTracer, err := observability.NewTracer(context.Background(), cfg.OTelEndpoint, "kredit-api"); err == nil {
			tracer = configuredTracer
		}
	}
	if cfg.PilotMaxPrincipalKobo > 0 {
		creditStore.SetCreationGuard(func(input credit.CreateInput) error {
			if int64(input.PrincipalKobo) > cfg.PilotMaxPrincipalKobo {
				return fmt.Errorf("principal exceeds configured pilot limit")
			}
			return nil
		})
	}
	creditStore.SetActivationHook(func(request credit.CreditRequest, obligation credit.Obligation) {
		location, locationErr := time.LoadLocation("Africa/Lagos")
		if locationErr != nil {
			location = time.FixedZone("WAT", 60*60)
		}
		startDate, parseErr := time.ParseInLocation("2006-01-02", request.DueDate, location)
		if parseErr != nil {
			return
		}
		if request.ScheduleType == "one_time" || request.ScheduleType == "" {
			_, _, _ = scheduleStore.Create(schedules.CreateInput{FirstCollectionAt: request.CollectionAt, ObligationID: obligation.ID, PrincipalKobo: obligation.PrincipalKobo, ScheduleType: schedules.TypeEqual, Count: 1, StartDate: startDate, DueHour: request.CollectionAt.In(location).Hour(), DueMinute: request.CollectionAt.In(location).Minute(), Timezone: "Africa/Lagos", GraceHours: request.GraceHours, Cadence: schedules.CadenceCustom, AllocationPolicy: "due_date_order"})
			return
		}
		input := schedules.CreateInput{FirstCollectionAt: request.CollectionAt, ObligationID: obligation.ID, PrincipalKobo: obligation.PrincipalKobo, ScheduleType: request.ScheduleType, Count: request.ScheduleCount, StartDate: startDate, DueHour: request.CollectionAt.In(location).Hour(), DueMinute: request.CollectionAt.In(location).Minute(), Timezone: "Africa/Lagos", GraceHours: request.GraceHours, Cadence: request.ScheduleCadence, MonthEndPolicy: request.MonthEndPolicy, AllocationPolicy: "due_date_order"}
		for _, item := range request.CustomScheduleItems {
			due, err := time.ParseInLocation("2006-01-02", item.DueDate, location)
			if err != nil {
				return
			}
			input.CustomItems = append(input.CustomItems, schedules.CustomItem{AmountKobo: item.AmountKobo, DueDate: due})
		}
		_, _, _ = scheduleStore.Create(input)
	})
	var creditRuntime credit.Service = creditStore
	var creditPostgres *credit.PostgresStore
	if database != nil {
		creditPostgres = credit.NewPostgresStore(database.Raw(), creditStore)
		if cfg.DeemedAcceptanceMinHours > 0 {
			creditPostgres.SetDeemedAcceptanceNotice(time.Duration(cfg.DeemedAcceptanceMinHours) * time.Hour)
		}
		creditRuntime = creditPostgres
	}
	tradeLineStore.SetActivationHandler(func(input tradelines.ActivationInput) (string, error) {
		view, _, err := creditRuntime.ActivateTradeLineDrawdown(credit.TradeLineActivationInput{
			FeeTerms: input.Drawdown.FeeTerms.Clone(), DrawdownID: input.Drawdown.ID, TradeLineID: input.Line.ID,
			SupplierOrganizationID: input.Line.SupplierOrganizationID,
			BuyerUserID:            input.Line.BuyerUserID, BuyerBusinessID: input.Line.BuyerBusinessID,
			MandateID: input.Line.MandateID, PrincipalKobo: input.Drawdown.PrincipalKobo,
			GoodsDescription: input.Drawdown.GoodsDescription, InvoiceReference: input.Drawdown.InvoiceReference,
			InvoiceDocumentHash: input.Drawdown.InvoiceDocumentHash, DueDate: input.Drawdown.DueDate,
			GraceHours: input.Drawdown.GraceHours, CollectionAt: input.Drawdown.CollectionAt,
			LegalVersions: input.Drawdown.LegalVersions, TermsVersion: input.Drawdown.TermsVersion, DrawdownAgreementHash: input.Drawdown.AgreementHash,
			BuyerConfirmedAt: input.Drawdown.BuyerConfirmedAt, ReleaseActorID: input.Drawdown.ReleaseActorID,
			DeliveryMethod: input.Drawdown.DeliveryMethod, ReleaseNotes: input.Drawdown.ReleaseNotes,
			ReleasedAt: input.Drawdown.ReleasedAt, ReceiptActorID: input.Drawdown.ReceiptActorID,
			ReceiptAt: input.Drawdown.ReceiptAt,
		})
		if err != nil {
			return "", err
		}
		if view.Obligation == nil {
			return "", fmt.Errorf("drawdown obligation was not created")
		}
		return view.Obligation.ID, nil
	})
	if tradeLinePostgres != nil && creditPostgres != nil {
		tradeLinePostgres.SetTransactionalActivationHandler(func(ctx context.Context, tx pgx.Tx, input tradelines.ActivationInput) (string, func(), error) {
			view, _, finalize, err := creditPostgres.ActivateTradeLineDrawdownTx(ctx, tx, credit.TradeLineActivationInput{
				FeeTerms: input.Drawdown.FeeTerms.Clone(), DrawdownID: input.Drawdown.ID, TradeLineID: input.Line.ID,
				SupplierOrganizationID: input.Line.SupplierOrganizationID,
				BuyerUserID:            input.Line.BuyerUserID, BuyerBusinessID: input.Line.BuyerBusinessID,
				MandateID: input.Line.MandateID, PrincipalKobo: input.Drawdown.PrincipalKobo,
				GoodsDescription: input.Drawdown.GoodsDescription, InvoiceReference: input.Drawdown.InvoiceReference,
				InvoiceDocumentHash: input.Drawdown.InvoiceDocumentHash, DueDate: input.Drawdown.DueDate,
				GraceHours: input.Drawdown.GraceHours, CollectionAt: input.Drawdown.CollectionAt,
				LegalVersions: input.Drawdown.LegalVersions, TermsVersion: input.Drawdown.TermsVersion, DrawdownAgreementHash: input.Drawdown.AgreementHash,
				BuyerConfirmedAt: input.Drawdown.BuyerConfirmedAt, ReleaseActorID: input.Drawdown.ReleaseActorID,
				DeliveryMethod: input.Drawdown.DeliveryMethod, ReleaseNotes: input.Drawdown.ReleaseNotes,
				ReleasedAt: input.Drawdown.ReleasedAt, ReceiptActorID: input.Drawdown.ReceiptActorID,
				ReceiptAt: input.Drawdown.ReceiptAt,
			})
			if err != nil {
				return "", nil, err
			}
			if view.Obligation == nil {
				return "", nil, fmt.Errorf("drawdown obligation was not created")
			}
			location, locationErr := time.LoadLocation("Africa/Lagos")
			if locationErr != nil {
				location = time.FixedZone("WAT", 60*60)
			}
			startDate, err := time.ParseInLocation("2006-01-02", view.Request.DueDate, location)
			if err != nil {
				return "", nil, err
			}
			if _, _, err := scheduleStore.CreateTx(ctx, tx, schedules.CreateInput{FirstCollectionAt: view.Request.CollectionAt, ObligationID: view.Obligation.ID, PrincipalKobo: view.Obligation.PrincipalKobo, ScheduleType: schedules.TypeEqual, Count: 1, StartDate: startDate, DueHour: view.Request.CollectionAt.In(location).Hour(), DueMinute: view.Request.CollectionAt.In(location).Minute(), Timezone: "Africa/Lagos", GraceHours: view.Request.GraceHours, Cadence: schedules.CadenceCustom, AllocationPolicy: "due_date_order"}); err != nil {
				return "", nil, err
			}
			return view.Obligation.ID, finalize, nil
		})
	}
	allocation := func(obligationID string, amount ledger.Money) ([]payments.AllocationTarget, error) {
		targets, err := scheduleStore.Allocate(obligationID, amount)
		out := make([]payments.AllocationTarget, 0, len(targets))
		for _, target := range targets {
			out = append(out, payments.AllocationTarget{ScheduleItemID: target.ScheduleItemID, AmountKobo: target.AmountKobo})
		}
		return out, err
	}
	reallocate := func(targets []payments.AllocationTarget) error {
		converted := make([]schedules.AllocationTarget, 0, len(targets))
		for _, target := range targets {
			converted = append(converted, schedules.AllocationTarget{ScheduleItemID: target.ScheduleItemID, AmountKobo: target.AmountKobo})
		}
		return scheduleStore.ReverseAllocations(converted)
	}
	applyPayment := func(id string, delta ledger.Money) error {
		apply := func() error { return creditRuntime.ApplyPayment(id, delta) }
		if memory, ok := tradeLineStore.(*tradelines.Store); ok {
			return memory.ApplyObligationDelta(id, delta, apply)
		}
		return apply()
	}
	memoryPaymentStore := payments.NewStoreWithAllocator(ledgerStore, creditRuntime.PaymentSnapshot, applyPayment, allocation, reallocate)
	memoryPaymentStore.SetCollectedMarker(scheduleStore.MarkCollected)
	memoryPaymentStore.SetCollectedReversalMarker(scheduleStore.ReverseCollected)
	var paymentStore payments.Service = memoryPaymentStore
	if database != nil {
		var invalidate func(string)
		if durableCredit, ok := creditRuntime.(*credit.PostgresStore); ok {
			invalidate = durableCredit.InvalidateObligation
		}
		paymentStore = payments.NewPostgresStore(database.Raw(), outboxStore, invalidate)
	}
	claimSnapshot := func(obligationID string) (paymentclaims.ObligationSnapshot, error) {
		snapshot, err := creditRuntime.PaymentSnapshot(obligationID)
		if err != nil {
			return paymentclaims.ObligationSnapshot{}, err
		}
		return paymentclaims.ObligationSnapshot{ID: snapshot.ID, BuyerUserID: snapshot.BuyerUserID, SupplierOrganizationID: snapshot.SupplierOrganizationID, OutstandingKobo: snapshot.OutstandingKobo, Currency: snapshot.Currency}, nil
	}
	var paymentClaimStore paymentclaims.Service = paymentclaims.NewStore(claimSnapshot)
	if database != nil {
		paymentClaimStore = paymentclaims.NewPostgresStore(database.Raw())
	}
	disputeSnapshot := func(obligationID string) (disputes.ObligationSnapshot, error) {
		state, err := creditRuntime.CollectionState(obligationID)
		if err != nil {
			return disputes.ObligationSnapshot{}, err
		}
		return disputes.ObligationSnapshot{OutstandingKobo: state.OutstandingKobo, SupplierOrganizationID: state.SupplierOrganizationID, BuyerUserID: state.BuyerUserID}, nil
	}
	applyAdjustment := func(id string, amount ledger.Money, resolvingDispute bool) error {
		if amount <= 0 {
			return fmt.Errorf("adjustment must be positive")
		}
		apply := func() error {
			snapshot, err := creditRuntime.PaymentSnapshot(id)
			if err != nil {
				return err
			}
			return scheduleStore.ReducePrincipal(id, snapshot.OutstandingKobo, amount, resolvingDispute, func() error { return creditRuntime.ApplyPayment(id, -amount) })
		}
		if memory, ok := tradeLineStore.(*tradelines.Store); ok {
			return memory.ApplyObligationDelta(id, -amount, apply)
		}
		return fmt.Errorf("durable adjustments require a database transaction")
	}
	var disputeStore disputes.Service = disputes.NewStore(disputeSnapshot, ledgerStore, func(id string, amount ledger.Money) error { return applyAdjustment(id, amount, true) })
	if database != nil {
		var invalidate func(string)
		if durableCredit, ok := creditRuntime.(*credit.PostgresStore); ok {
			invalidate = durableCredit.InvalidateObligation
		}
		disputeStore = disputes.NewPostgresStore(database.Raw(), invalidate)
	}
	var operationStore operations.Service = operations.NewStore(ledgerStore, func(id string, amount ledger.Money) error { return applyAdjustment(id, amount, false) })
	if database != nil {
		var invalidate func(string)
		if durableCredit, ok := creditRuntime.(*credit.PostgresStore); ok {
			invalidate = durableCredit.InvalidateObligation
		}
		operationStore = operations.NewPostgresStore(database.Raw(), outboxStore, invalidate)
	}
	feeWaivers := func(ctx context.Context, org string) (map[string]ledger.Money, error) {
		result := map[string]ledger.Money{}
		actions, err := operationStore.ListForOrganization(ctx, org)
		if err != nil {
			return nil, err
		}
		for _, action := range actions {
			if action.ActionType == "fee_waiver" {
				result[action.ObligationID], err = ledger.CheckedAdd(result[action.ObligationID], action.AmountKobo)
				if err != nil {
					return nil, err
				}
			}
		}
		return result, nil
	}
	reportStore := reports.NewStore(reports.Source{FeeWaivers: feeWaivers, SupplierViews: creditRuntime.ListForSupplier, BuyerViews: creditRuntime.ListForBuyer, Payments: paymentStore.List, Schedule: scheduleStore.GetForObligation, Disputes: disputeStore.ListForObligation})
	if database != nil {
		reportStore = reports.NewPostgresStore(database.Raw(), reports.Source{SupplierViews: creditRuntime.ListForSupplier, BuyerViews: creditRuntime.ListForBuyer, Payments: paymentStore.List, Schedule: scheduleStore.GetForObligation, Disputes: disputeStore.ListForObligation})
	}
	var correctionStore corrections.Service = corrections.NewStore()
	var auditStore audit.Service = audit.NewStore()
	var idempotencyStore idempotency.Service = idempotency.NewMemoryStore()
	sessionKey := runtimeDomainKey(cfg.SessionSigningKey, cfg.TokenHashKey, "sessions")
	otpKey := runtimeDomainKey(cfg.OTPHMACKey, cfg.TokenHashKey, "otp")
	fieldKey := runtimeDomainKey(cfg.FieldEncryptionKey, cfg.TokenHashKey, "field-encryption:"+cfg.FieldEncryptionKeyID)
	var authStore auth.Service = auth.NewStoreWithKeys(sessionKey, otpKey)
	var organizationStore organizations.Service = organizations.NewStore()
	var onboardingStore onboarding.Service = onboarding.NewStore()
	var buyerStore buyers.Service = buyers.NewStore(runtimeDomainKey(fieldKey, "", "buyers"), identityProvider)
	allowedIndustries := csvSet(cfg.PilotAllowedIndustries)
	if database != nil {
		auditStore = audit.NewPostgresStore(database.Raw())
		idempotencyStore = idempotency.NewPostgresStore(database.Raw())
		authStore = auth.NewPostgresStoreWithKeys(database.Raw(), sessionKey, otpKey, runtimeDomainKey(fieldKey, "", "auth"))
		organizationStore = organizations.NewPostgresStore(database.Raw(), runtimeDomainKey(fieldKey, "", "organizations"))
		onboardingStore = onboarding.NewPostgresStore(database.Raw())
		buyerStore = buyers.NewPostgresStore(database.Raw(), runtimeDomainKey(fieldKey, "", "buyers"), identityProvider)
		correctionStore = corrections.NewPostgresStore(database.Raw())
	}
	if configurable, ok := onboardingStore.(interface{ SetLegalReader(legalpublication.Reader) }); ok {
		configurable.SetLegalReader(legalpublication.SettingsReader(platformSettingsStore))
	}
	// Apply pilot guards after selecting the runtime adapter so a durable
	// deployment cannot silently lose its buyer limits during adapter switch.
	buyerStore.SetInvitationGuard(func(input buyers.CreateInvitationInput) error {
		if len(allowedIndustries) > 0 && !allowedIndustries[strings.ToLower(strings.TrimSpace(input.Industry))] {
			return fmt.Errorf("industry is not enabled for the pilot")
		}
		return nil
	})
	durableBuyerLimit := false
	if limited, ok := buyerStore.(interface{ SetBusinessLimit(int64) }); ok {
		limited.SetBusinessLimit(cfg.PilotMaxBuyerBusinesses)
		durableBuyerLimit = true
	}
	buyerStore.SetAcceptanceGuard(func(input buyers.AcceptInput) error {
		if !durableBuyerLimit && cfg.PilotMaxBuyerBusinesses > 0 && int64(buyerStore.CountBusinesses()) >= cfg.PilotMaxBuyerBusinesses {
			return fmt.Errorf("pilot buyer business limit reached")
		}
		if len(allowedIndustries) > 0 && strings.TrimSpace(input.Industry) != "" && !allowedIndustries[strings.ToLower(strings.TrimSpace(input.Industry))] {
			return fmt.Errorf("industry is not enabled for the pilot")
		}
		return nil
	})
	// Apply the pilot guard after selecting the runtime adapter.  The database
	// adapter is intentionally created after the in-memory development store;
	// setting the guard only before that switch would silently remove the
	// supplier-organisation cap in staging and production.
	durableOrganizationLimit := false
	if limited, ok := organizationStore.(interface{ SetOrganizationLimit(int64) }); ok {
		limited.SetOrganizationLimit(cfg.PilotMaxSupplierOrganizations)
		durableOrganizationLimit = true
	}
	organizationStore.SetCreateGuard(func(_ string, input organizations.CreateInput) error {
		if !durableOrganizationLimit && cfg.PilotMaxSupplierOrganizations > 0 && int64(organizationStore.Count()) >= cfg.PilotMaxSupplierOrganizations {
			return fmt.Errorf("pilot supplier organization limit reached")
		}
		if len(allowedIndustries) > 0 && !allowedIndustries[strings.ToLower(strings.TrimSpace(input.Industry))] {
			return fmt.Errorf("industry is not enabled for the pilot")
		}
		return nil
	})
	var objectStore documents.ObjectStore = documents.NewMemoryObjectStore()
	if cfg.Environment != "development" || (database != nil && cfg.ObjectStorageEndpoint != "") {
		configured, err := documents.NewS3ObjectStore(context.Background(), cfg.ObjectStorageEndpoint, cfg.ObjectStorageRegion, cfg.ObjectStorageAccessKey, cfg.ObjectStorageSecretKey, cfg.ObjectStorageBucket)
		if err != nil {
			if cfg.Environment != "development" {
				// Never silently fall back to process memory in staging/production;
				// uploads must fail closed instead of creating non-durable evidence.
				objectStore = documents.NewUnavailableObjectStore(err)
			}
		} else {
			objectStore = configured
		}
	}
	documentStore := documents.NewStore(objectStore)
	var documentScanner documents.Scanner = documents.CleanDevelopmentScanner{}
	if cfg.Environment != "development" {
		documentScanner = nil
		if scanner, err := documents.NewWebhookScanner(cfg.DocumentScannerEndpoint, cfg.DocumentScannerToken); err == nil {
			documentScanner = scanner
		}
	}
	supportStore := support.NewStore()
	if database != nil {
		documentStore = documents.NewPostgresStore(database.Raw(), objectStore)
		supportStore = support.NewPostgresStore(database.Raw())
	}
	notificationStore := notifications.NewStore(runtimeDomainKey(cfg.TokenHashKey, sessionKey, "notifications"))
	userControlStore := usercontrol.NewStore(runtimeDomainKey(cfg.TokenHashKey, sessionKey, "user-control"))
	if database != nil {
		notificationStore = notifications.NewPostgresStore(database.Raw(), runtimeDomainKey(cfg.TokenHashKey, sessionKey, "notifications"))
		userControlStore = usercontrol.NewPostgresStore(database.Raw(), runtimeDomainKey(cfg.TokenHashKey, sessionKey, "user-control"))
	}
	notificationStore.SetBaseURL(cfg.PublicBaseURL)
	if cfg.Environment == "development" {
		notificationStore.RegisterProvider(notifications.NewMockProvider(notifications.ChannelWhatsApp))
		notificationStore.RegisterProvider(notifications.NewMockProvider(notifications.ChannelEmail))
		notificationStore.RegisterProvider(notifications.NewMockProvider(notifications.ChannelSMS))
	} else {
		for _, connector := range []struct{ channel, endpoint, token, from, adapter string }{
			{notifications.ChannelEmail, cfg.NotificationEmailEndpoint, cfg.NotificationEmailToken, cfg.NotificationEmailFrom, cfg.NotificationEmailAdapter},
			{notifications.ChannelSMS, cfg.NotificationSMSEndpoint, cfg.NotificationSMSToken, cfg.NotificationSMSFrom, cfg.NotificationSMSAdapter},
			{notifications.ChannelWhatsApp, cfg.NotificationWhatsAppEndpoint, cfg.NotificationWhatsAppToken, "", cfg.NotificationWhatsAppAdapter},
		} {
			var fallback notifications.Provider
			if connector.endpoint != "" && connector.token != "" {
				fallback, _ = notifications.NewAdapter(connector.channel, platformsettings.NotificationConnector{Adapter: connector.adapter, Endpoint: connector.endpoint, Token: connector.token, From: connector.from})
			}
			configured := notifications.NewConfiguredProvider(connector.channel, platformSettingsStore, fallback).WithDeployment(platformsettings.NotificationConnector{Adapter: connector.adapter, Enabled: connector.endpoint != "" && connector.token != "", Endpoint: connector.endpoint, Token: connector.token, From: connector.from}, cfg.SettingsEncryptionKey)
			if database != nil {
				configured.WithPersistence(database.Raw(), []byte(runtimeDomainKey(cfg.TokenHashKey, sessionKey, "message-submissions")))
			}
			notificationStore.RegisterProvider(configured)
		}
	}
	whatsAppKey := runtimeDomainKey(cfg.TokenHashKey, sessionKey, "whatsapp-webhook")
	whatsAppHandler := whatsapp.NewHandler(whatsAppKey)
	if database != nil {
		whatsAppHandler = whatsapp.NewPostgresHandler(database.Raw(), whatsAppKey)
	}
	var baseCollectionProvider collections.Provider = collections.NewMockProvider(runtimeDomainKey(cfg.TokenHashKey, sessionKey, "mock-collections"))
	collectionEnabled := cfg.Environment == "development" && !cfg.RealCollections
	if cfg.Environment != "development" && cfg.RealCollections && cfg.CollectionProvider != cfg.MonoAccount() {
		if connector, err := collections.NewWebhookProvider(cfg.CollectionProvider, cfg.CollectionProviderEndpoint, cfg.CollectionProviderToken, cfg.CollectionWebhookSecret); err == nil {
			approvedAt, timeErr := time.Parse(time.RFC3339, cfg.ProviderApprovedAt)
			if timeErr == nil {
				approval := collections.ApprovalRecord{ProviderName: cfg.CollectionProvider, WrittenReference: cfg.ProviderApprovalReference, ApprovedBy: cfg.ProviderApprovedBy, ApprovedAt: approvedAt, AllowedCapabilities: []collections.Capability{collections.CapabilityOneTime, collections.CapabilitySettlement, collections.CapabilityReversal}, PilotLimitKobo: cfg.PilotMaxPrincipalKobo}
				adapter := collections.NewApprovedAdapter(connector, approval, true)
				if adapter.Enabled() {
					baseCollectionProvider = adapter
					collectionEnabled = true
				}
			}
		}
	}
	if cfg.MonoSweepEnabled || cfg.CollectionProvider == cfg.MonoAccount() {
		collectionEnabled = false
	}
	if monoClient != nil && cfg.CollectionProvider == cfg.MonoAccount() {
		baseCollectionProvider = monoClient
		collectionEnabled = cfg.MonoSweepEnabled
	}
	collectionProvider := collections.NewResilientProvider(baseCollectionProvider, 3, time.Minute)
	collectionSnapshot := func(ctx context.Context, obligationID string) (collections.ObligationSnapshot, error) {
		state, err := creditRuntime.CollectionStateContext(ctx, obligationID)
		if err != nil {
			return collections.ObligationSnapshot{}, err
		}
		if state.MandateReference != "" {
			if state.MandateProvider == "" || state.MandateProvider != mandateRuntime.Name() {
				return collections.ObligationSnapshot{}, errors.New("the original bank authorization provider must be configured before collection")
			}
			mandate, lookupErr := mandateRuntime.GetMandate(mandates.WithProvider(ctx, state.MandateProvider), state.MandateReference)
			if lookupErr != nil {
				return collections.ObligationSnapshot{}, lookupErr
			}
			state.MandateActive = mandate.Status == mandates.Active && (mandate.StartsAt.IsZero() || !time.Now().Before(mandate.StartsAt)) && (mandate.EndsAt.IsZero() || time.Now().Before(mandate.EndsAt))
			if monoClient != nil && mandate.PartialRecovery && !cfg.PartialSweepEnabled {
				state.CollectionEnabled = false
			}
		}
		blocked, disputeErr := disputeStore.BlockedAmount(obligationID)
		if disputeErr != nil {
			return collections.ObligationSnapshot{}, disputeErr
		}
		claimHold := paymentClaimStore.ActiveHold(db.WithTenantContext(ctx, state.BuyerUserID, state.SupplierOrganizationID), obligationID, time.Now().UTC())
		_, supplierReadiness, readinessErr := onboardingStore.Get(state.SupplierOrganizationID)
		if readinessErr != nil || !supplierReadiness.Ready {
			state.CollectionEnabled = false
		}
		if platformOpsStore != nil {
			buyerHeld, holdErr := platformOpsStore.ActiveHold(ctx, "buyer", state.BuyerUserID, "collection")
			if holdErr != nil {
				return collections.ObligationSnapshot{}, holdErr
			}
			supplierHeld, holdErr := platformOpsStore.ActiveHold(ctx, "supplier", state.SupplierOrganizationID, "collection")
			if holdErr != nil {
				return collections.ObligationSnapshot{}, holdErr
			}
			state.ComplianceHold = state.ComplianceHold || buyerHeld || supplierHeld
		}
		return collections.ObligationSnapshot{ID: state.ID, BuyerUserID: state.BuyerUserID, Currency: state.Currency, Active: state.Active, CollectionPolicy: state.CollectionPolicy, OutstandingKobo: state.OutstandingKobo, MandateActive: state.MandateActive, MandateReference: state.MandateReference, MandateProvider: state.MandateProvider, MandateRemainingKobo: state.MandateRemainingKobo, CollectionEnabled: state.CollectionEnabled, ComplianceHold: state.ComplianceHold, BuyerPaymentHold: state.BuyerPaymentHold, BuyerPaymentHoldKobo: claimHold, ProviderSupported: state.ProviderSupported, DisputedBlockedKobo: blocked, Version: state.Version}, nil
	}
	collectionDue := func(obligationID string, now time.Time) (ledger.Money, error) {
		return scheduleStore.CollectionTarget(obligationID, now)
	}
	collectionEngine := collections.NewContextEngine(collectionProvider, paymentStore, collectionSnapshot, collectionDue)
	if monoClient != nil && collectionProvider.Name() != monoClient.Name() {
		if err := collectionEngine.RegisterRetainedProvider(monoClient.ReconciliationOnly()); err != nil {
			providerFailures = append(providerFailures, "original Mono reconciliation could not be installed")
		}
	}
	retainedConnections, retainedErr := cfg.RetainedCollections()
	if retainedErr != nil {
		providerFailures = append(providerFailures, "retained collection configuration is invalid")
	}
	for _, connection := range retainedConnections {
		if connection.Name == collectionProvider.Name() || (monoClient != nil && connection.Name == monoClient.Name()) {
			continue
		}
		var retained collections.Provider
		var err error
		if connection.Adapter == "mono" {
			client := monoAccounts[connection.Name]
			if client == nil {
				err = errors.New("saved Mono collection account unavailable")
			} else {
				retained = client
			}
		} else {
			retained, err = collections.NewWebhookProvider(connection.Name, connection.Endpoint, connection.Token, connection.WebhookSecret)
		}
		if err == nil {
			err = collectionEngine.RegisterRetainedProvider(retained)
		}
		if err != nil {
			providerFailures = append(providerFailures, "retained collection connection could not be installed")
		}
	}

	if allowedProviders := csvSet(cfg.PilotAllowedProviderAccounts); len(allowedProviders) > 0 && !allowedProviders[strings.ToLower(collectionProvider.Name())] {
		collectionEnabled = false
	}
	collectionEngine.SetFeatureEnabled(collectionEnabled)
	if cfg.PilotMaxCollectionRetries > 0 {
		collectionEngine.SetMaxRetries(int(cfg.PilotMaxCollectionRetries))
	}
	var collectionRuntime collections.Service = collectionEngine
	if database != nil {
		pgCollections := collections.NewPostgresEngine(database.Raw(), collectionEngine)
		pgCollections.RequireSettlementRoute()
		if cfg.RealCollections || cfg.MonoSweepEnabled {
			delay := cfg.CollectionNoticeMinHours
			if delay < 1 {
				delay = 24
			}
			pgCollections.RequirePriorNotice(time.Duration(delay) * time.Hour)
		}
		collectionRuntime = pgCollections
	}
	relationshipStore := relationships.Service(relationships.NewStore())
	if database != nil {
		relationshipStore = relationships.NewPostgresStore(database.Raw())
	}
	notificationStore.SetReminderConsent(relationshipStore.AllowsReminders)
	notificationStore.SetOptionalProcessing(userControlStore.AllowsOptionalProcessing)
	userControlStore.SetRecoveryDelivery(func(ctx context.Context, request usercontrol.RecoveryRequest, token string) error {
		user, err := authStore.UserByID(request.TargetUserID)
		if err != nil {
			return err
		}
		destination := user.Email
		if request.RequestedChannel == "phone" {
			destination = user.Phone
		}
		link := strings.TrimRight(cfg.PublicBaseURL, "/") + "/recover#request=" + request.ID
		if token != "" {
			link += "&token=" + token
		}
		return notificationStore.SendRecoveryInstructions(ctx, destination, request.RequestedChannel, link)
	})
	if memory, ok := authStore.(*auth.Store); ok {
		userControlStore.SetRecoveryReset(memory.ResetAfterRecovery)
	}
	var feeBilling *billing.FeeService
	if database != nil {
		feeProviders := map[string]billing.FeeProvider{}
		for name, client := range monoAccounts {
			feeProviders[name] = client
		}
		if cfg.RealCollections && cfg.CollectionProvider != cfg.MonoAccount() {
			connector, e := billing.NewConnector(cfg.CollectionProvider, cfg.CollectionProviderEndpoint, cfg.CollectionProviderToken, false)
			if e != nil {
				providerFailures = append(providerFailures, "fee billing connector: "+e.Error())
			} else {
				feeProviders[connector.Name()] = connector
			}
		}
		for _, saved := range retainedConnections {
			if saved.Adapter == "mono" || feeProviders[saved.Name] != nil {
				continue
			}
			connector, e := billing.NewConnector(saved.Name, saved.Endpoint, saved.Token, true)
			if e != nil {
				providerFailures = append(providerFailures, "fee billing connector: "+e.Error())
				continue
			}
			feeProviders[saved.Name] = connector
		}
		active := ""
		if cfg.MonoSweepEnabled {
			active = cfg.MonoAccount()
		} else if cfg.RealCollections && cfg.CollectionProvider != cfg.MonoAccount() {
			active = cfg.CollectionProvider
		}
		feeBilling = billing.NewFeeService(database.Raw(), active, feeProviders, func(value string) string {
			mac := hmac.New(sha256.New, []byte(cfg.TokenHashKey))
			_, _ = mac.Write([]byte("fee-customer:" + value))
			return hex.EncodeToString(mac.Sum(nil))
		})
	}
	return &Runtime{
		FeeBilling: feeBilling,
		Mono:       monoClient, MonoAccounts: monoAccounts, WebhookJobs: webhookJobs, Settlement: settlementProvider,
		Database: database,
		Persistence: PersistenceStatus{
			DatabaseConfigured:      database != nil,
			AuthDurable:             database != nil,
			BuyerDurable:            database != nil,
			CreditDurable:           database != nil,
			LedgerDurable:           database != nil,
			AuditDurable:            database != nil,
			IdempotencyDurable:      database != nil,
			DocumentsDurable:        database != nil,
			SupportDurable:          database != nil,
			DomainAggregatesDurable: database != nil,
		},
		Idempotency:          idempotencyStore,
		Auth:                 authStore,
		Organizations:        organizationStore,
		Onboarding:           onboardingStore,
		Audit:                auditStore,
		Identity:             identityProvider,
		Buyers:               buyerStore,
		Mandates:             mandateRuntime,
		Ledger:               ledgerStore,
		Credit:               creditRuntime,
		Payments:             paymentStore,
		PaymentClaims:        paymentClaimStore,
		PaymentClaimsEnabled: cfg.OffPlatformPaymentClaims,
		Schedules:            scheduleStore,
		TradeLines:           tradeLineStore,
		Collections:          collectionRuntime,
		Disputes:             disputeStore,
		Documents:            documentStore,
		DocumentScanner:      documentScanner,
		Relationships:        relationshipStore,
		Support:              supportStore,
		Operations:           operationStore,
		Reports:              reportStore,
		Corrections:          correctionStore,
		Readiness:            readinessReport,
		Metrics:              observability.NewStore(),
		Tracer:               tracer,
		Notifications:        notificationStore,
		WhatsApp:             whatsAppHandler,
		Outbox:               outboxStore,
		PlatformOps:          platformOpsStore,
		PlatformSettings:     platformSettingsStore,
		ProviderFailures:     providerFailures,
		BusinessPolicies:     policies, policyInitializationError: policyError,
		UserControl: userControlStore,
		Feedback:    feedbackStore,
	}
}

func csvSet(value string) map[string]bool {
	result := make(map[string]bool)
	for _, item := range strings.Split(value, ",") {
		item = strings.ToLower(strings.TrimSpace(item))
		if item != "" {
			result[item] = true
		}
	}
	return result
}

func runtimeDomainKey(primary, fallback, domain string) string {
	key := strings.TrimSpace(primary)
	if key == "" {
		key = strings.TrimSpace(fallback)
	}
	if key == "" {
		key = "development-only-change-me"
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(domain))
	return hex.EncodeToString(mac.Sum(nil))
}
