package web

import (
	"errors"
	"net/http"
	"strings"

	"kredit/internal/access"
	"kredit/internal/audit"
	"kredit/internal/credit"
	"kredit/internal/creditapproval"

	"github.com/jackc/pgx/v5"
)

func (s *Server) businessCreditApprovals(w http.ResponseWriter, r *http.Request) {
	org, err := pathID(r, "organizationID")
	if err != nil {
		writeProblem(w, 400, "invalid_business", err.Error())
		return
	}
	permission := access.PermissionReadOrganization
	if r.Method != "GET" {
		permission = access.PermissionManageOrganization
		if r.PathValue("requestID") != "" || r.PathValue("drawdownID") != "" {
			permission = access.PermissionCreateCredit
		} else if r.PathValue("approvalID") != "" {
			permission = access.PermissionApproveBusinessCredit
		}
	}
	_, user, _, ok := s.requireOrganizationAccess(w, r, org, permission)
	if !ok {
		return
	}
	if r.Method != "GET" && !s.requireCSRF(w, r) {
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "approvals_unavailable", "Credit approvals require the business database.")
		return
	}
	service := creditapproval.Store{Pool: s.runtime.Database.Raw()}
	fail := func(err error) {
		switch {
		case credit.ReviewerCeilingExceeded(err):
			writeProblem(w, 403, "approval_ceiling_exceeded", "This offer exceeds your current approval limit. Ask a reviewer with sufficient authority.")
		case errors.Is(err, creditapproval.ErrAuthority):
			writeProblem(w, 403, "approval_authority_required", "A different authorized owner or finance reviewer must make this decision.")
		case errors.Is(err, creditapproval.ErrConflict):
			writeProblem(w, 409, "approval_changed", "Refresh the offer and approval before continuing.")
		case errors.Is(err, creditapproval.ErrInvalid):
			writeProblem(w, 422, "invalid_approval", "Check the threshold or provide a decision reason between 3 and 1000 characters.")
		case errors.Is(err, pgx.ErrNoRows):
			writeProblem(w, 404, "approval_not_found", "This offer or approval is not available to your business.")
		default:
			writeProblem(w, 503, "approval_unavailable", "The approval could not be confirmed. Refresh the record before retrying.")
		}
	}
	action, resource := "", ""
	if r.PathValue("reviewerID") != "" {
		target, e := pathID(r, "reviewerID")
		if e != nil {
			writeProblem(w, 400, "invalid_reviewer", "Choose a valid reviewer.")
			return
		}
		var in struct {
			Ceiling *int64 `json:"ceiling_kobo"`
			Version *int64 `json:"version"`
		}
		if e = decodeJSON(w, r, &in); e != nil {
			writeProblem(w, 400, "invalid_request", e.Error())
			return
		}
		if in.Ceiling == nil || in.Version == nil {
			writeProblem(w, 422, "invalid_approval", "Provide both the reviewer limit and its current version.")
			return
		}
		next, e := service.SetLimit(r.Context(), user.ID, org, target, *in.Ceiling, *in.Version)
		if e != nil {
			fail(e)
			return
		}
		s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: org, Action: "credit.reviewer_limit.updated", ResourceType: "credit_reviewer_limit", ResourceID: target, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
		writeJSON(w, 200, next)
		return
	}
	if r.Method == "GET" {
		result, err := service.Read(r.Context(), user.ID, org)
		if err != nil {
			fail(err)
			return
		}
		writeJSON(w, 200, result)
		return
	}
	if request := r.PathValue("requestID"); request != "" {
		if _, err = pathID(r, "requestID"); err != nil {
			writeProblem(w, 400, "invalid_request", err.Error())
			return
		}
		id, err := service.Request(r.Context(), user.ID, org, request)
		if err != nil {
			fail(err)
			return
		}
		action = "credit.approval.requested"
		resource = id
	} else if drawdown := r.PathValue("drawdownID"); drawdown != "" {
		if _, err = pathID(r, "drawdownID"); err != nil {
			writeProblem(w, 400, "invalid_drawdown", err.Error())
			return
		}
		id, err := service.RequestDrawdownApproval(r.Context(), user.ID, org, drawdown)
		if err != nil {
			fail(err)
			return
		}
		action = "drawdown.approval.requested"
		resource = id
	} else if id := r.PathValue("approvalID"); id != "" {
		if _, err = pathID(r, "approvalID"); err != nil {
			writeProblem(w, 400, "invalid_approval", err.Error())
			return
		}
		var in struct {
			Decision string `json:"decision"`
			Reason   string `json:"reason"`
		}
		if err = decodeJSON(w, r, &in); err != nil {
			writeProblem(w, 400, "invalid_request", err.Error())
			return
		}
		err = service.Decide(r.Context(), user.ID, org, id, in.Decision, strings.TrimSpace(in.Reason))
		if errors.Is(err, pgx.ErrNoRows) {
			err = service.DecideDrawdown(r.Context(), user.ID, org, id, in.Decision, strings.TrimSpace(in.Reason))
		}
		if err != nil {
			fail(err)
			return
		}
		action = "credit.approval." + in.Decision
		resource = id
	} else {
		var in creditapproval.Controls
		if err = decodeJSON(w, r, &in); err != nil {
			writeProblem(w, 400, "invalid_request", err.Error())
			return
		}
		next, err := service.SetControls(r.Context(), user.ID, org, in)
		if err != nil {
			fail(err)
			return
		}
		s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: org, Action: "credit.approval_policy.updated", ResourceType: "business_credit_controls", ResourceID: org, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
		writeJSON(w, 200, next)
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, OrganizationID: org, Action: action, ResourceType: "credit_offer_approval", ResourceID: resource, Outcome: "success", RequestID: requestIDFromContext(r.Context())})
	writeJSON(w, 200, map[string]string{"id": resource, "status": "recorded"})
}
