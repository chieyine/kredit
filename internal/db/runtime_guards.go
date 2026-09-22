package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

const maxTimeoutMilliseconds int64 = 2147483647

var timeoutValuePattern = regexp.MustCompile(`^([0-9]+)\s*(ms|s|min|h|d)?$`)

// Runtime URLs take precedence over environment defaults, but all explicit
// values must be valid. Canonical decimal milliseconds are sent to PostgreSQL.
// Startup options are refused because a second -c channel can override both
// the selected runtime role and the already-validated timeout parameters.
func configureRuntimeTimeouts(params map[string]string) error {
	if strings.TrimSpace(params["options"]) != "" {
		return errors.New("database startup options are unsupported; use named connection parameters instead of options or PGOPTIONS")
	}
	statement := "30000"
	if raw := os.Getenv("DATABASE_STATEMENT_TIMEOUT_MS"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 1 || n > maxTimeoutMilliseconds {
			return errors.New("DATABASE_STATEMENT_TIMEOUT_MS must be between 1 and 2147483647 milliseconds")
		}
		statement = strconv.FormatInt(n, 10)
	}
	for _, setting := range []struct{ name, fallback string }{
		{"statement_timeout", statement},
		{"lock_timeout", "5000"},
		{"idle_in_transaction_session_timeout", "30000"},
	} {
		setRuntimeDefault(params, setting.name, setting.fallback)
		n, err := timeoutMilliseconds(params[setting.name])
		if err != nil {
			return fmt.Errorf("%s must be a positive whole-number duration (ms, s, min, h or d), at most 2147483647 milliseconds", setting.name)
		}
		params[setting.name] = strconv.FormatInt(n, 10)
	}
	return nil
}

func timeoutMilliseconds(value string) (int64, error) {
	parts := timeoutValuePattern.FindStringSubmatch(strings.TrimSpace(value))
	if parts == nil {
		return 0, errors.New("invalid timeout")
	}
	n, err := strconv.ParseInt(parts[1], 10, 64)
	unit := int64(1)
	switch parts[2] {
	case "s":
		unit = 1000
	case "min":
		unit = 60000
	case "h":
		unit = 3600000
	case "d":
		unit = 86400000
	}
	if err != nil || n < 1 || n > maxTimeoutMilliseconds/unit {
		return 0, errors.New("timeout out of range")
	}
	return n * unit, nil
}

// Check the server's effective settings, not merely the configuration sent by
// the client. AfterConnect invokes this for every replacement pool connection.
func verifyRuntimeTimeouts(ctx context.Context, conn *pgx.Conn, expected map[string]string) error {
	for _, name := range []string{"statement_timeout", "lock_timeout", "idle_in_transaction_session_timeout"} {
		var actual string
		if err := conn.QueryRow(ctx, `SELECT setting FROM pg_settings WHERE name=$1`, name).Scan(&actual); err != nil {
			return fmt.Errorf("verify database timeout %s: %w", name, err)
		}
		if actual != expected[name] {
			return fmt.Errorf("database timeout %s does not match the required connection setting", name)
		}
	}
	return nil
}

// Both session_user (reachable after RESET ROLE) and current_user must satisfy
// the runtime contract. Memberships that can reach a privileged or owner role
// are refused too; the deliberately privileged backup role is never altered.
func verifyRuntimeRole(ctx context.Context, conn *pgx.Conn, runtimeRole string) error {
	var sessionUser, currentUser string
	var unsafeSession, unsafeRuntime, runtimeCanLogin, unsafeMembership bool
	err := conn.QueryRow(ctx, `
		WITH privileged AS (
			SELECT r.oid FROM pg_roles r
			WHERE r.rolsuper OR r.rolbypassrls OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication
			   OR EXISTS (SELECT 1 FROM pg_database d WHERE d.datname=current_database() AND d.datdba=r.oid)
			   OR EXISTS (SELECT 1 FROM pg_namespace n WHERE n.nspowner=r.oid AND n.nspname IN ('app','ledger','jobs'))
			   OR EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relowner=r.oid AND n.nspname IN ('app','ledger','jobs'))
			   OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE p.proowner=r.oid AND n.nspname IN ('app','ledger','jobs'))
		)
		SELECT session_user, current_user,
		       EXISTS (SELECT 1 FROM privileged WHERE oid=s.oid),
		       EXISTS (SELECT 1 FROM privileged WHERE oid=r.oid),
		       r.rolcanlogin,
		       EXISTS (SELECT 1 FROM privileged p WHERE pg_has_role(s.oid,p.oid,'MEMBER') OR pg_has_role(r.oid,p.oid,'MEMBER'))
		FROM pg_roles s CROSS JOIN pg_roles r
		WHERE s.rolname=session_user AND r.rolname=current_user`).Scan(
		&sessionUser, &currentUser, &unsafeSession, &unsafeRuntime, &runtimeCanLogin, &unsafeMembership,
	)
	if err != nil {
		return fmt.Errorf("verify postgres runtime role: %w", err)
	}
	if currentUser != runtimeRole {
		return fmt.Errorf("postgres runtime role is %q, require %q", currentUser, runtimeRole)
	}
	if unsafeSession || unsafeRuntime || runtimeCanLogin || unsafeMembership {
		return fmt.Errorf("postgres login %q and runtime role %q must be unprivileged, non-owning roles without privileged memberships; the runtime role must be NOLOGIN", sessionUser, currentUser)
	}
	return nil
}
