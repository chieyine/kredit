package web

import (
	"context"
	"fmt"
	"kredit/internal/audit"
	"kredit/internal/auth"
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
	"kredit/internal/ledger"
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
	"kredit/internal/readiness"
	"kredit/internal/relationships"
	"kredit/internal/reports"
	"kredit/internal/schedules"
	"kredit/internal/support"
	"kredit/internal/tradelines"
	"kredit/internal/usercontrol"
	"kredit/internal/whatsapp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Runtime struct {
	Database             *db.Pool
	Persistence          PersistenceStatus
	Idempotency          idempotency.Service
	Auth                 auth.Service
	Organizations        organizations.Service
	Onboarding           onboarding.Service
	Audit                audit.Service
	Identity             identity.IdentityProvider
	Buyers               buyers.Service
	Mandates             mandates.Provider
	Ledger               ledger.Service
	Credit               credit.Service
	Payments             payments.Service
	PaymentClaims        paymentclaims.Service
	PaymentClaimsEnabled bool
	Schedules            *schedules.Store
	TradeLines           tradelines.Service
	Collections          collections.Service
	Disputes             disputes.Service
	Documents            *documents.Store
	DocumentScanner      documents.Scanner
	Relationships        relationships.Service
	Support              *support.Store
	Operations           operations.Service
	Reports              *reports.Store
	Corrections          corrections.Service
	Readiness            readiness.Report
	Metrics              *observability.Store
	Tracer               *observability.Tracer
	Notifications        *notifications.Store
	WhatsApp             *whatsapp.Handler
	Outbox               *outbox.Store
	PlatformOps          *platformops.Store
	UserControl          *usercontrol.Store
	Feedback             *feedback.Store
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
		r.Persistence.AuthDurable && r.Persistence.LedgerDurable && r.Persistence.AuditDurable &&
		r.Persistence.IdempotencyDurable && r.Persistence.DomainAggregatesDurable
}

func NewRuntime(cfg config.Config) *Runtime {
	return NewRuntimeWithDB(cfg, nil)
}

func NewRuntimeWithDB(cfg config.Config, database *db.Pool) *Runtime {
	var identityProvider identity.IdentityProvider = identity.NewMockProvider()
	if cfg.Environment != "development" {
		identityProvider = identity.NewUnavailableProvider("certified identity provider adapter is not configured")
		if provider, err := identity.NewWebhookProvider(cfg.IdentityProvider, cfg.IdentityProviderEndpoint, cfg.IdentityProviderToken, cfg.IdentityWebhookSecret); err == nil {
			identityProvider = provider
		}
	}
	mandateProvider := mandates.NewMockProvider()
	var mandateRuntime mandates.Provider = mandateProvider
	if database != nil {
		mandateRuntime = mandates.NewPostgresProvider(database.Raw(), cfg.CollectionProvider)
		if cfg.Environment != "development" && cfg.RealCollections {
			if remote, err := mandates.NewWebhookProvider(cfg.CollectionProvider, cfg.CollectionProviderEndpoint, cfg.CollectionProviderToken); err == nil {
				mandateRuntime = mandates.NewPostgresProviderWithRemote(database.Raw(), remote)
			}
		}
	}
	var outboxStore *outbox.Store
	var platformOpsStore *platformops.Store
	feedbackStore := feedback.NewStore()
	var ledgerStore ledger.Service = ledger.NewStore()
	if database != nil {
		outboxStore = outbox.NewStore(database.Raw())
		platformOpsStore = platformops.NewStore(database.Raw())
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
			_, _, _ = scheduleStore.Create(schedules.CreateInput{ObligationID: obligation.ID, PrincipalKobo: obligation.PrincipalKobo, ScheduleType: schedules.TypeEqual, Count: 1, StartDate: startDate, DueHour: request.CollectionAt.In(location).Hour(), DueMinute: request.CollectionAt.In(location).Minute(), Timezone: "Africa/Lagos", GraceHours: request.GraceHours, Cadence: schedules.CadenceCustom, AllocationPolicy: "due_date_order"})
			return
		}
		input := schedules.CreateInput{ObligationID: obligation.ID, PrincipalKobo: obligation.PrincipalKobo, ScheduleType: request.ScheduleType, Count: request.ScheduleCount, StartDate: startDate, DueHour: request.CollectionAt.In(location).Hour(), DueMinute: request.CollectionAt.In(location).Minute(), Timezone: "Africa/Lagos", GraceHours: request.GraceHours, Cadence: request.ScheduleCadence, MonthEndPolicy: request.MonthEndPolicy, AllocationPolicy: "due_date_order"}
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
		creditRuntime = creditPostgres
	}
	tradeLineStore.SetActivationHandler(func(input tradelines.ActivationInput) (string, error) {
		view, _, err := creditRuntime.ActivateTradeLineDrawdown(credit.TradeLineActivationInput{
			DrawdownID: input.Drawdown.ID, TradeLineID: input.Line.ID,
			SupplierOrganizationID: input.Line.SupplierOrganizationID,
			BuyerUserID:            input.Line.BuyerUserID, BuyerBusinessID: input.Line.BuyerBusinessID,
			MandateID: input.Line.MandateID, PrincipalKobo: input.Drawdown.PrincipalKobo,
			GoodsDescription: input.Drawdown.GoodsDescription, InvoiceReference: input.Drawdown.InvoiceReference,
			InvoiceDocumentHash: input.Drawdown.InvoiceDocumentHash, DueDate: input.Drawdown.DueDate,
			GraceHours: input.Drawdown.GraceHours, CollectionAt: input.Drawdown.CollectionAt,
			TermsVersion: input.Drawdown.TermsVersion, DrawdownAgreementHash: input.Drawdown.AgreementHash,
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
				DrawdownID: input.Drawdown.ID, TradeLineID: input.Line.ID,
				SupplierOrganizationID: input.Line.SupplierOrganizationID,
				BuyerUserID:            input.Line.BuyerUserID, BuyerBusinessID: input.Line.BuyerBusinessID,
				MandateID: input.Line.MandateID, PrincipalKobo: input.Drawdown.PrincipalKobo,
				GoodsDescription: input.Drawdown.GoodsDescription, InvoiceReference: input.Drawdown.InvoiceReference,
				InvoiceDocumentHash: input.Drawdown.InvoiceDocumentHash, DueDate: input.Drawdown.DueDate,
				GraceHours: input.Drawdown.GraceHours, CollectionAt: input.Drawdown.CollectionAt,
				TermsVersion: input.Drawdown.TermsVersion, DrawdownAgreementHash: input.Drawdown.AgreementHash,
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
			if _, _, err := scheduleStore.CreateTx(ctx, tx, schedules.CreateInput{ObligationID: view.Obligation.ID, PrincipalKobo: view.Obligation.PrincipalKobo, ScheduleType: schedules.TypeEqual, Count: 1, StartDate: startDate, DueHour: view.Request.CollectionAt.In(location).Hour(), DueMinute: view.Request.CollectionAt.In(location).Minute(), Timezone: "Africa/Lagos", GraceHours: view.Request.GraceHours, Cadence: schedules.CadenceCustom, AllocationPolicy: "due_date_order"}); err != nil {
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
	memoryPaymentStore := payments.NewStoreWithAllocator(ledgerStore, creditRuntime.PaymentSnapshot, creditRuntime.ApplyPayment, allocation, reallocate)
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
	var disputeStore disputes.Service = disputes.NewStore(disputeSnapshot, ledgerStore, creditRuntime.ApplyAdjustment)
	if database != nil {
		var invalidate func(string)
		if durableCredit, ok := creditRuntime.(*credit.PostgresStore); ok {
			invalidate = durableCredit.InvalidateObligation
		}
		disputeStore = disputes.NewPostgresStore(database.Raw(), invalidate)
	}
	var operationStore operations.Service = operations.NewStore(ledgerStore, creditRuntime.ApplyAdjustment)
	if database != nil {
		var invalidate func(string)
		if durableCredit, ok := creditRuntime.(*credit.PostgresStore); ok {
			invalidate = durableCredit.InvalidateObligation
		}
		operationStore = operations.NewPostgresStore(database.Raw(), outboxStore, invalidate)
	}
	reportStore := reports.NewStore(reports.Source{SupplierViews: creditRuntime.ListForSupplier, BuyerViews: creditRuntime.ListForBuyer, Payments: paymentStore.List, Schedule: scheduleStore.GetForObligation, Disputes: disputeStore.ListForObligation})
	if database != nil {
		reportStore = reports.NewPostgresStore(database.Raw(), reports.Source{SupplierViews: creditRuntime.ListForSupplier, BuyerViews: creditRuntime.ListForBuyer, Payments: paymentStore.List, Schedule: scheduleStore.GetForObligation, Disputes: disputeStore.ListForObligation})
	}
	var correctionStore corrections.Service = corrections.NewStore()
	var auditStore audit.Service = audit.NewStore()
	var idempotencyStore idempotency.Service = idempotency.NewMemoryStore()
	var authStore auth.Service = auth.NewStore(cfg.TokenHashKey)
	var organizationStore organizations.Service = organizations.NewStore()
	var onboardingStore onboarding.Service = onboarding.NewStore()
	var buyerStore buyers.Service = buyers.NewStore(cfg.TokenHashKey, identityProvider)
	allowedIndustries := csvSet(cfg.PilotAllowedIndustries)
	if database != nil {
		auditStore = audit.NewPostgresStore(database.Raw())
		idempotencyStore = idempotency.NewPostgresStore(database.Raw())
		authStore = auth.NewPostgresStore(database.Raw(), cfg.TokenHashKey)
		organizationStore = organizations.NewPostgresStore(database.Raw(), cfg.TokenHashKey)
		onboardingStore = onboarding.NewPostgresStore(database.Raw())
		buyerStore = buyers.NewPostgresStore(database.Raw(), cfg.TokenHashKey, identityProvider)
		correctionStore = corrections.NewPostgresStore(database.Raw())
	}
	// Apply pilot guards after selecting the runtime adapter so a durable
	// deployment cannot silently lose its buyer limits during adapter switch.
	buyerStore.SetInvitationGuard(func(input buyers.CreateInvitationInput) error {
		if len(allowedIndustries) > 0 && !allowedIndustries[strings.ToLower(strings.TrimSpace(input.Industry))] {
			return fmt.Errorf("industry is not enabled for the pilot")
		}
		return nil
	})
	buyerStore.SetAcceptanceGuard(func(input buyers.AcceptInput) error {
		if cfg.PilotMaxBuyerBusinesses > 0 && int64(buyerStore.CountBusinesses()) >= cfg.PilotMaxBuyerBusinesses {
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
	organizationStore.SetCreateGuard(func(_ string, input organizations.CreateInput) error {
		if cfg.PilotMaxSupplierOrganizations > 0 && int64(organizationStore.Count()) >= cfg.PilotMaxSupplierOrganizations {
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
	notificationStore := notifications.NewStore(cfg.TokenHashKey)
	userControlStore := usercontrol.NewStore(cfg.TokenHashKey)
	if database != nil {
		notificationStore = notifications.NewPostgresStore(database.Raw(), cfg.TokenHashKey)
		userControlStore = usercontrol.NewPostgresStore(database.Raw(), cfg.TokenHashKey)
	}
	notificationStore.SetBaseURL(cfg.PublicBaseURL)
	if cfg.Environment == "development" {
		notificationStore.RegisterProvider(notifications.NewMockProvider(notifications.ChannelWhatsApp))
		notificationStore.RegisterProvider(notifications.NewMockProvider(notifications.ChannelEmail))
		notificationStore.RegisterProvider(notifications.NewMockProvider(notifications.ChannelSMS))
	} else {
		for _, connector := range []struct{ channel, endpoint, token string }{
			{notifications.ChannelEmail, cfg.NotificationEmailEndpoint, cfg.NotificationEmailToken},
			{notifications.ChannelSMS, cfg.NotificationSMSEndpoint, cfg.NotificationSMSToken},
			{notifications.ChannelWhatsApp, cfg.NotificationWhatsAppEndpoint, cfg.NotificationWhatsAppToken},
		} {
			if connector.endpoint == "" || connector.token == "" {
				continue
			}
			if provider, err := notifications.NewWebhookProvider(connector.channel, connector.endpoint, connector.token); err == nil {
				notificationStore.RegisterProvider(provider)
			}
		}
	}
	whatsAppHandler := whatsapp.NewHandler(cfg.TokenHashKey)
	if database != nil {
		whatsAppHandler = whatsapp.NewPostgresHandler(database.Raw(), cfg.TokenHashKey)
	}
	var baseCollectionProvider collections.Provider = collections.NewMockProvider(cfg.TokenHashKey)
	collectionEnabled := cfg.Environment == "development" && !cfg.RealCollections
	if cfg.Environment != "development" && cfg.RealCollections {
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
	collectionProvider := collections.NewResilientProvider(baseCollectionProvider, 3, time.Minute)
	collectionSnapshot := func(obligationID string) (collections.ObligationSnapshot, error) {
		state, err := creditRuntime.CollectionState(obligationID)
		if err != nil {
			return collections.ObligationSnapshot{}, err
		}
		blocked, _ := disputeStore.BlockedAmount(obligationID)
		claimHold := paymentClaimStore.ActiveHold(context.Background(), obligationID, time.Now().UTC())
		_, supplierReadiness, readinessErr := onboardingStore.Get(state.SupplierOrganizationID)
		if readinessErr != nil || !supplierReadiness.Ready {
			state.CollectionEnabled = false
		}
		if platformOpsStore != nil {
			buyerHeld, _ := platformOpsStore.ActiveHold(context.Background(), "buyer", state.BuyerUserID, "collection")
			supplierHeld, _ := platformOpsStore.ActiveHold(context.Background(), "supplier", state.SupplierOrganizationID, "collection")
			state.ComplianceHold = state.ComplianceHold || buyerHeld || supplierHeld
		}
		return collections.ObligationSnapshot{ID: state.ID, BuyerUserID: state.BuyerUserID, Currency: state.Currency, Active: state.Active, OutstandingKobo: state.OutstandingKobo, MandateActive: state.MandateActive, MandateRemainingKobo: state.MandateRemainingKobo, CollectionEnabled: state.CollectionEnabled, ComplianceHold: state.ComplianceHold, BuyerPaymentHold: state.BuyerPaymentHold, BuyerPaymentHoldKobo: claimHold, ProviderSupported: state.ProviderSupported, DisputedBlockedKobo: blocked, Version: state.Version}, nil
	}
	collectionDue := func(obligationID string, now time.Time) (ledger.Money, error) {
		return scheduleStore.CollectionTarget(obligationID, now)
	}
	collectionEngine := collections.NewEngine(collectionProvider, paymentStore, collectionSnapshot, collectionDue)
	if allowedProviders := csvSet(cfg.PilotAllowedProviderAccounts); len(allowedProviders) > 0 && !allowedProviders[strings.ToLower(collectionProvider.Name())] {
		collectionEnabled = false
	}
	collectionEngine.SetFeatureEnabled(collectionEnabled)
	if cfg.PilotMaxCollectionRetries > 0 {
		collectionEngine.SetMaxRetries(int(cfg.PilotMaxCollectionRetries))
	}
	var collectionRuntime collections.Service = collectionEngine
	if database != nil {
		collectionRuntime = collections.NewPostgresEngine(database.Raw(), collectionEngine)
	}
	relationshipStore := relationships.Service(relationships.NewStore())
	if database != nil {
		relationshipStore = relationships.NewPostgresStore(database.Raw())
	}
	return &Runtime{
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
		UserControl:          userControlStore,
		Feedback:             feedbackStore,
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
