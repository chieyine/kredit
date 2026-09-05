from pathlib import Path


def one(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected 1 match, found {count}")
    return text.replace(old, new, 1)


# Add reusable context-aware collection reads alongside the other financial
# adapter helpers. In-memory adapters keep working through the fallback path.
p = Path("internal/web/financial_reads.go")
text = p.read_text()
old = '''func (r *Runtime) readCollectionsAttempts(id string) ([]collections.Attempt, error) {
\treturn r.readCollectionsAttemptsContext(context.Background(), id)
}
'''
new = '''func (r *Runtime) collectionEligibilityContext(ctx context.Context, id string, now time.Time) (collections.Eligibility, error) {
\tif source, ok := r.Collections.(interface {
\t\tEligibilityContext(context.Context, string, time.Time) (collections.Eligibility, error)
\t}); ok {
\t\treturn source.EligibilityContext(ctx, id, now)
\t}
\treturn r.Collections.Eligibility(id, now)
}
func (r *Runtime) getCollectionAttemptContext(ctx context.Context, id string) (collections.Attempt, bool) {
\tif source, ok := r.Collections.(interface {
\t\tGetAttemptContext(context.Context, string) (collections.Attempt, bool)
\t}); ok {
\t\treturn source.GetAttemptContext(ctx, id)
\t}
\treturn r.Collections.GetAttempt(id)
}
func (r *Runtime) readCollectionsAttempts(id string) ([]collections.Attempt, error) {
\treturn r.readCollectionsAttemptsContext(context.Background(), id)
}
'''
text = one(text, old, new, "financial collection context helpers")
if '"time"' not in text.split(')')[0]:
    text = one(text, '"net/http"\n', '"net/http"\n\t"time"\n', "financial_reads time import")
p.write_text(text)


p = Path("internal/web/credit_handlers.go")
text = p.read_text()

# Eligibility: the org authorization is authoritative; the DB sees the same org.
old = '''\tv, err := s.runtime.Credit.GetForSupplier(requestID, orgID)
\tif err != nil || v.Obligation == nil {
\t\twriteProblem(w, 404, "obligation_not_found", "obligation was not found")
\t\treturn
\t}
\teligibility, err := s.runtime.Collections.Eligibility(v.Obligation.ID, time.Now().UTC())
'''
new = '''\tv, err := s.runtime.Credit.GetForSupplier(requestID, orgID)
\tif err != nil || v.Obligation == nil {
\t\twriteProblem(w, 404, "obligation_not_found", "obligation was not found")
\t\treturn
\t}
\tctx := db.WithTenantContext(r.Context(), "", orgID)
\teligibility, err := s.runtime.collectionEligibilityContext(ctx, v.Obligation.ID, time.Now().UTC())
'''
text = one(text, old, new, "collection eligibility tenant context")

# Start: carry authenticated actor + org into every downstream financial call.
old = '''\t_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
\tif !ok {
\t\treturn
\t}
\tif !s.requireSupplierReady(w, orgID, user.ID, "starting a live collection") {
'''
new = '''\t_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
\tif !ok {
\t\treturn
\t}
\tr = r.WithContext(db.WithTenantContext(r.Context(), user.ID, orgID))
\tif !s.requireSupplierReady(w, orgID, user.ID, "starting a live collection") {
'''
text = one(text, old, new, "start collection tenant context")

# Retry: query and mutation both run under the authorized org. The application
# ownership check remains as defence in depth.
old = '''\t_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
\tif !ok {
\t\treturn
\t}
\tif !s.requireCSRF(w, r) {
\t\treturn
\t}
\tattempt, exists := s.runtime.Collections.GetAttempt(attemptID)
\tif !exists || !s.runtime.Credit.ObligationBelongsToOrganization(attempt.ObligationID, orgID) {
\t\twriteProblem(w, 404, "collection_not_found", "collection attempt was not found")
\t\treturn
\t}
\tretried, err := s.runtime.Collections.Retry(r.Context(), attemptID, time.Now().UTC())
'''
new = '''\t_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
\tif !ok {
\t\treturn
\t}
\tr = r.WithContext(db.WithTenantContext(r.Context(), user.ID, orgID))
\tif !s.requireCSRF(w, r) {
\t\treturn
\t}
\tattempt, exists := s.runtime.getCollectionAttemptContext(r.Context(), attemptID)
\tif !exists || !s.runtime.Credit.ObligationBelongsToOrganization(attempt.ObligationID, orgID) {
\t\twriteProblem(w, 404, "collection_not_found", "collection attempt was not found")
\t\treturn
\t}
\tretried, err := s.runtime.Collections.Retry(r.Context(), attemptID, time.Now().UTC())
'''
text = one(text, old, new, "retry collection tenant context")

# Reconcile: same fail-closed tenant boundary as retry.
old = '''\t_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
\tif !ok {
\t\treturn
\t}
\tif !s.requireCSRF(w, r) {
\t\treturn
\t}
\tattempt, exists := s.runtime.Collections.GetAttempt(attemptID)
\tif !exists || !s.runtime.Credit.ObligationBelongsToOrganization(attempt.ObligationID, orgID) {
\t\twriteProblem(w, 404, "collection_not_found", "collection attempt was not found")
\t\treturn
\t}
\tresolved, err := s.runtime.Collections.Reconcile(r.Context(), attemptID)
'''
new = '''\t_, user, _, ok := s.requireOrganizationAccess(w, r, orgID, access.PermissionManageFinancial)
\tif !ok {
\t\treturn
\t}
\tr = r.WithContext(db.WithTenantContext(r.Context(), user.ID, orgID))
\tif !s.requireCSRF(w, r) {
\t\treturn
\t}
\tattempt, exists := s.runtime.getCollectionAttemptContext(r.Context(), attemptID)
\tif !exists || !s.runtime.Credit.ObligationBelongsToOrganization(attempt.ObligationID, orgID) {
\t\twriteProblem(w, 404, "collection_not_found", "collection attempt was not found")
\t\treturn
\t}
\tresolved, err := s.runtime.Collections.Reconcile(r.Context(), attemptID)
'''
text = one(text, old, new, "reconcile collection tenant context")
p.write_text(text)
