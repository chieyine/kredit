package agreementdocs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"time"

	"kredit/internal/credit"
	"kredit/internal/ledger"
	"kredit/internal/schedules"
	"kredit/internal/tradelines"
)

// DocumentData is the complete, non-authoritative printable projection of an
// activated agreement. The canonical agreement JSON and its stored digest
// remain the source of truth.
type DocumentData struct {
	View     credit.View
	Schedule schedules.Schedule
	Items    []schedules.Item
}

type DrawdownDocumentData struct {
	FeeDisclosure string
	Line          tradelines.TradeLine
	Drawdown      tradelines.Drawdown
}

func RenderDrawdownHTML(data DrawdownDocumentData) ([]byte, error) {
	if !tradelines.VerifyAgreementHash(data.Drawdown, data.Line) {
		return nil, errors.New("drawdown agreement hash does not match its immutable terms")
	}
	data.FeeDisclosure = tradelines.DrawdownFeeDisclosure(data.Drawdown, data.Line)
	var output bytes.Buffer
	if err := drawdownDocumentTemplate.Execute(&output, data); err != nil {
		return nil, fmt.Errorf("render drawdown agreement document: %w", err)
	}
	return output.Bytes(), nil
}

func RenderHTML(data DocumentData) ([]byte, error) {
	if data.View.Obligation == nil || data.View.Acceptance == nil || data.View.Release == nil {
		return nil, errors.New("agreement document is available only after activation evidence exists")
	}
	if len(data.View.Agreement.CanonicalJSON) == 0 || data.View.Agreement.DocumentHash == "" {
		return nil, errors.New("canonical agreement evidence is incomplete")
	}
	digest := sha256.Sum256(data.View.Agreement.CanonicalJSON)
	if hex.EncodeToString(digest[:]) != data.View.Agreement.DocumentHash {
		return nil, errors.New("canonical agreement hash does not match stored evidence")
	}
	var output bytes.Buffer
	if err := documentTemplate.Execute(&output, data); err != nil {
		return nil, fmt.Errorf("render agreement document: %w", err)
	}
	return output.Bytes(), nil
}

var documentTemplate = template.Must(template.New("agreement").Funcs(template.FuncMap{
	"money": func(value any) string {
		var kobo int64
		switch typed := value.(type) {
		case int64:
			kobo = typed
		case ledger.Money:
			kobo = int64(typed)
		default:
			return "NGN 0.00"
		}
		return fmt.Sprintf("NGN %d.%02d", kobo/100, kobo%100)
	},
	"dateTime": func(value time.Time) string {
		if value.IsZero() {
			return "Not recorded"
		}
		return value.In(lagosLocation()).Format("02 Jan 2006, 15:04 MST")
	},
}).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Kredit agreement {{.View.Request.ID}}</title>
<style>body{font:15px/1.5 system-ui,sans-serif;color:#17211b;max-width:850px;margin:40px auto;padding:0 24px}h1{font-size:2rem}h2{margin-top:2rem;border-bottom:1px solid #ccd5cf;padding-bottom:.35rem}dl{display:grid;grid-template-columns:minmax(150px,1fr) 2fr;gap:.45rem 1rem}dt{font-weight:700}dd{margin:0}table{width:100%;border-collapse:collapse}th,td{text-align:left;border-bottom:1px solid #dfe5e1;padding:.55rem}.hash{overflow-wrap:anywhere;font-family:ui-monospace,monospace;font-size:.85rem}.notice{background:#f1f5f2;padding:1rem;border-radius:.5rem}@media print{body{margin:0}.no-print{display:none}}</style></head><body>
<button class="no-print" onclick="window.print()">Print or save as PDF</button>
<h1>Kredit trade-credit agreement</h1><p class="notice">This document is a printable representation. The immutable canonical agreement identified by the cryptographic hash below remains authoritative.</p>
<h2>Parties and terms</h2><dl>
<dt>Transaction reference</dt><dd>{{.View.Request.ID}}</dd><dt>Supplier</dt><dd>{{.View.Request.SupplierLegalName}}</dd><dt>Buyer</dt><dd>{{.View.Request.BuyerLegalName}}</dd>
<dt>Principal</dt><dd>{{money .View.Request.PrincipalKobo}}</dd><dt>Goods or services</dt><dd>{{.View.Request.GoodsDescription}}</dd><dt>Invoice reference</dt><dd>{{if .View.Request.InvoiceReference}}{{.View.Request.InvoiceReference}}{{else}}Not supplied{{end}}</dd>
<dt>Due date</dt><dd>{{.View.Request.DueDate}}</dd><dt>Collection time</dt><dd>{{dateTime .View.Request.CollectionAt}}</dd><dt>Grace period</dt><dd>{{.View.Request.GraceHours}} hours</dd></dl>
<h2>Acceptance and mandate</h2><dl><dt>Accepted</dt><dd>{{dateTime .View.Acceptance.AcceptedAt}}</dd><dt>Method</dt><dd>{{.View.Acceptance.AcceptanceMethod}}</dd><dt>Authentication level</dt><dd>{{.View.Acceptance.AuthenticationLevel}}</dd><dt>Mandate provider</dt><dd>{{.View.Mandate.Provider}}</dd><dt>Mandate status at capture</dt><dd>{{.View.Mandate.Status}}</dd><dt>Mandate ceiling</dt><dd>{{money .View.Mandate.AmountCeiling}}</dd></dl>
<h2>Goods evidence</h2><dl><dt>Released</dt><dd>{{dateTime .View.Release.ReleasedAt}}</dd><dt>Delivery method</dt><dd>{{.View.Release.DeliveryMethod}}</dd><dt>Release notes</dt><dd>{{if .View.Release.Notes}}{{.View.Release.Notes}}{{else}}None{{end}}</dd><dt>Invoice document hash</dt><dd class="hash">{{if .View.Request.InvoiceDocumentHash}}{{.View.Request.InvoiceDocumentHash}}{{else}}No invoice document attached{{end}}</dd></dl>
<h2>Payment schedule</h2><table><thead><tr><th>Instalment</th><th>Amount</th><th>Due</th><th>Collection</th></tr></thead><tbody>{{range .Items}}<tr><td>{{.Sequence}}</td><td>{{money .PrincipalDueKobo}}</td><td>{{dateTime .DueAt}}</td><td>{{dateTime .CollectionAt}}</td></tr>{{else}}<tr><td colspan="4">No schedule items recorded.</td></tr>{{end}}</tbody></table>
<h2>Support and disputes</h2><p>Raise a structured issue from the buyer portal or contact Kredit support. A valid dispute blocks collection only for the contested amount while the evidence and decision history remain attached to the obligation.</p>
<h2>Integrity</h2><dl><dt>Agreement version</dt><dd>{{.View.Agreement.Version}}</dd><dt>Terms version</dt><dd>{{.View.Agreement.TermsVersion}}</dd><dt>Privacy version</dt><dd>{{.View.Agreement.PrivacyVersion}}</dd><dt>Cryptographic document hash</dt><dd class="hash">{{.View.Agreement.DocumentHash}}</dd></dl>
</body></html>`))

var drawdownDocumentTemplate = template.Must(template.New("drawdown-agreement").Funcs(template.FuncMap{
	"money":    func(value ledger.Money) string { return fmt.Sprintf("NGN %d.%02d", value/100, value%100) },
	"dateTime": func(value time.Time) string { return value.In(lagosLocation()).Format("02 Jan 2006, 15:04 MST") },
}).Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Kredit drawdown agreement {{.Drawdown.ID}}</title><style>body{font:15px/1.5 system-ui,sans-serif;color:#17211b;max-width:850px;margin:40px auto;padding:0 24px}h1{font-size:2rem}h2{margin-top:2rem;border-bottom:1px solid #ccd5cf;padding-bottom:.35rem}dl{display:grid;grid-template-columns:minmax(170px,1fr) 2fr;gap:.45rem 1rem}dt{font-weight:700}dd{margin:0}.hash{overflow-wrap:anywhere;font-family:ui-monospace,monospace;font-size:.85rem}.notice{background:#f1f5f2;padding:1rem;border-radius:.5rem}@media print{body{margin:0}.no-print{display:none}}</style></head><body><button class="no-print" onclick="window.print()">Print or save as PDF</button><h1>Kredit trade-line drawdown agreement</h1><p class="notice">The terms below were verified against the cryptographic hash before this printable representation was generated.</p><h2>Parties and purchase</h2><dl><dt>Drawdown reference</dt><dd>{{.Drawdown.ID}}</dd><dt>Trade-line reference</dt><dd>{{.Line.ID}}</dd><dt>Supplier organisation</dt><dd>{{.Line.SupplierOrganizationID}}</dd><dt>Buyer business</dt><dd>{{.Line.BuyerBusinessID}}</dd><dt>Principal</dt><dd>{{money .Drawdown.PrincipalKobo}}</dd><dt>Goods or services</dt><dd>{{.Drawdown.GoodsDescription}}</dd><dt>Invoice reference</dt><dd>{{if .Drawdown.InvoiceReference}}{{.Drawdown.InvoiceReference}}{{else}}Not supplied{{end}}</dd><dt>Invoice document hash</dt><dd class="hash">{{if .Drawdown.InvoiceDocumentHash}}{{.Drawdown.InvoiceDocumentHash}}{{else}}Not supplied{{end}}</dd></dl><h2>Repayment terms</h2><dl><dt>Due date</dt><dd>{{.Drawdown.DueDate}}</dd><dt>Collection time</dt><dd>{{dateTime .Drawdown.CollectionAt}}</dd><dt>Grace period</dt><dd>{{.Drawdown.GraceHours}} hours</dd><dt>Cadence</dt><dd>{{.Line.Cadence}}</dd><dt>Mandate reference</dt><dd>{{.Line.MandateID}}</dd><dt>Fee effect</dt><dd>{{.FeeDisclosure}}</dd></dl><h2>Evidence and integrity</h2><dl><dt>Terms version</dt><dd>{{.Drawdown.TermsVersion}}</dd><dt>Buyer confirmation</dt><dd>{{if .Drawdown.BuyerConfirmedAt.IsZero}}Not yet confirmed{{else}}{{dateTime .Drawdown.BuyerConfirmedAt}}{{end}}</dd><dt>Goods release</dt><dd>{{if .Drawdown.ReleasedAt.IsZero}}Not yet released{{else}}{{dateTime .Drawdown.ReleasedAt}} · {{.Drawdown.DeliveryMethod}}{{end}}</dd><dt>Buyer receipt</dt><dd>{{if .Drawdown.ReceiptAt.IsZero}}Not yet recorded{{else}}{{dateTime .Drawdown.ReceiptAt}} · {{.Drawdown.ReceiptState}}{{end}}</dd><dt>Agreement hash</dt><dd class="hash">{{.Drawdown.AgreementHash}}</dd></dl></body></html>`))

func lagosLocation() *time.Location {
	location, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		return time.FixedZone("WAT", 60*60)
	}
	return location
}
