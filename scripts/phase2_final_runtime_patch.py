from pathlib import Path


def one(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected 1 match, found {count}")
    return text.replace(old, new, 1)


p = Path("internal/web/collection_jobs.go")
text = p.read_text()
text = one(
    text,
    '''func (r *Runtime) HandleCollectionJob(ctx context.Context, cfg config.Config, args jobs.CollectionArgs) error {
\toperation, id := args.Operation, args.ResourceID
\tif args.OrganizationID != "" {
''',
    '''func (r *Runtime) HandleCollectionJob(ctx context.Context, cfg config.Config, args jobs.CollectionArgs) error {
\toperation, id := args.Operation, args.ResourceID
\tif (operation == jobs.OpReconcileProvider || operation == "collect_due") && args.OrganizationID == "" {
\t\treturn errors.New("collection tenant context is required")
\t}
\tif args.OrganizationID != "" {
''',
    "collection job tenant requirement",
)
p.write_text(text)

p = Path("internal/web/credit_handlers.go")
text = p.read_text()
text = one(
    text,
    's.runtime.readCollectionsAttempts(v.Obligation.ID)',
    's.runtime.readCollectionsAttemptsContext(db.WithTenantContext(r.Context(), "", orgID), v.Obligation.ID)',
    "supplier collection attempt read",
)
p.write_text(text)
