# Break-glass access

Owner: security incident commander. Break-glass requires an incident ID,
two-person approval, time-bounded credentials, least privilege, and a written
purpose. Use the audited access workflow; never share a database password or
disable RLS.

Record every command and accessed resource. Revoke credentials immediately at
incident close, review the audit trail with security, and rotate any exposed
secret. Emergency access is read-only by default; financial changes require a
separate approved compensating action.
