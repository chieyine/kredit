# Deployment environments

This OpenTofu/Terraform root deploys the immutable API, worker, and web images
to a Kubernetes-compatible platform. Managed PostgreSQL, S3-compatible object
storage, provider credentials, keys, and signed approvals are injected through
a pre-created secret; they are deliberately not stored in Terraform state.

Use a protected remote state backend in the environment wrapper, review every
plan, and apply staging before production. The runtime secret must contain the
validated variables documented in `README.md` and the deployment must attach
the monitoring resources in `infra/monitoring`.

For the selected Vercel/VPS deployment, use `docs/operations/PRODUCTION-DEPLOYMENT-GUIDE.md`. Compose requires both `.env.production` (deployment database passwords) and `.env.runtime` (application settings). Certificates remain outside version control.

Legal publication metadata is source-controlled and later versions use the website-content publication workflow. Legacy Terraform legal environment values do not change the current pages. Follow `docs/release/go-live-runbook.md` for release evidence and cutover.
