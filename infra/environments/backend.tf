# Terraform state for this stack was previously local: no locking, no shared
# history, and a state file holding Kubernetes secret material on whichever
# machine last ran `apply`. Two operators applying at once would corrupt the
# environment with no record of it.
#
# This is a partial backend. Terraform does not allow variables here, so the
# bucket and credentials are supplied at init time:
#
#   terraform init -backend-config=backend.hcl
#
# Copy backend.hcl.example to backend.hcl and fill it in. backend.hcl is
# git-ignored because it names infrastructure and may carry an endpoint.
terraform {
  backend "s3" {}
}
