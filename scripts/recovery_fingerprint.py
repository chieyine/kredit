#!/usr/bin/env python3
"""Read-only data/security fingerprint for a QUIESCED isolated restore drill.

Raw rows stay in the local pipe; only counts and SHA-256 digests are written.
This is not an encrypted backup, PITR proof, or a live-database comparison tool.
Use the same PostgreSQL major version, paused writers and paused DDL throughout.
"""
import argparse
import hashlib
import json
import os
import subprocess
import sys
import tempfile
import urllib.parse
from pathlib import Path


class FingerprintError(ValueError):
    """A deliberately non-sensitive diagnostic suitable for the drill log."""


def connection_environment(raw: str) -> dict[str, str]:
    # PGDATABASE is a database NAME, not a portable place for a full URI.
    # Explicit libpq environment settings avoid credentials in command arguments
    # and avoid defaulting to the container's (possibly unnamed) operating user.
    parsed = urllib.parse.urlsplit(raw)
    if parsed.scheme not in ('postgres', 'postgresql') or not parsed.hostname or not parsed.username or not parsed.path or parsed.path == '/' or parsed.fragment:
        raise FingerprintError('an explicit single-host PostgreSQL URI is required')
    environment = {key: value for key, value in os.environ.items() if not key.startswith('PG')}
    environment.update(PGHOST=parsed.hostname, PGPORT=str(parsed.port or 5432),
                       PGUSER=urllib.parse.unquote(parsed.username),
                       PGPASSWORD=urllib.parse.unquote(parsed.password or ''),
                       PGDATABASE=urllib.parse.unquote(parsed.path[1:]),
                       PGCONNECT_TIMEOUT='10', PGCLIENTENCODING='UTF8', PGOPTIONS='-c statement_timeout=300000 -c lock_timeout=10000')
    settings = {'sslmode': 'PGSSLMODE', 'connect_timeout': 'PGCONNECT_TIMEOUT',
                'sslrootcert': 'PGSSLROOTCERT', 'sslcert': 'PGSSLCERT', 'sslkey': 'PGSSLKEY',
                'channel_binding': 'PGCHANNELBINDING', 'target_session_attrs': 'PGTARGETSESSIONATTRS',
                'application_name': 'PGAPPNAME', 'hostaddr': 'PGHOSTADDR'}
    seen = set()
    for key, value in urllib.parse.parse_qsl(parsed.query, keep_blank_values=True):
        if key not in settings or key in seen or not value:
            raise FingerprintError('unsupported or duplicate connection URI option')
        seen.add(key)
        environment[settings[key]] = value
    return environment


def psql(sql: str, environment: dict[str, str]) -> str:
    result = subprocess.run(['psql', '-XqAt', '-v', 'ON_ERROR_STOP=1'],
                            input=sql, text=True, capture_output=True, check=False, env=environment)
    if result.returncode:
        raise FingerprintError('fingerprint query failed; inspect restricted database logs')
    return result.stdout


def quote(value: str) -> str:
    return '"' + value.replace('"', '""') + '"'


def row_bytes(line: str) -> bytes:
    # Hash exact PostgreSQL JSON output, not Python float reserialization.
    return line.rstrip('\n').encode('utf-8') + b'\n'


def capture() -> dict:
    if not os.environ.get('DATABASE_URL'):
        raise FingerprintError('DATABASE_URL is required')
    environment = connection_environment(os.environ['DATABASE_URL'])
    catalog = json.loads(psql("SELECT COALESCE(json_agg(json_build_array(n.nspname,c.relname) ORDER BY n.nspname,c.relname),'[]') FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind IN ('r','p') AND (n.nspname IN ('app','ledger','jobs') OR (n.nspname='public' AND c.relname='goose_db_version'));", environment))
    names = {'.'.join(row) for row in catalog}
    if not {'app.obligations', 'app.payments', 'ledger.postings', 'public.goose_db_version'} <= names:
        raise FingerprintError('financial schema is incomplete')
    commands = ["BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY; SET LOCAL row_security=off; SET LOCAL timezone='UTC'; SET LOCAL datestyle='ISO, YMD'; SET LOCAL intervalstyle='postgres';"]
    for schema, table in catalog:
        name = schema + '.' + table
        label = "'" + name.replace("'", "''") + "'"
        commands.append(f"SELECT json_build_object('table',{label},'header',true);")
        commands.append(f"SELECT json_build_object('table',{label},'row',to_jsonb(t)) FROM {quote(schema)}.{quote(table)} t ORDER BY to_jsonb(t)::text COLLATE \"C\";")
    commands.extend([
        "SELECT json_build_object('table','@policies','header',true);",
        "SELECT json_build_object('table','@policies','row',to_jsonb(p)) FROM pg_policies p WHERE schemaname IN ('app','ledger','jobs') ORDER BY schemaname,tablename,policyname;",
        "SELECT json_build_object('table','@rls','header',true);",
        "SELECT json_build_object('table','@rls','row',json_build_array(n.nspname,c.relname,c.relrowsecurity,c.relforcerowsecurity)) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('app','ledger','jobs') AND c.relkind IN ('r','p') ORDER BY n.nspname,c.relname;",
        "SELECT json_build_object('table','@definer_public_execute','header',true);",
        "SELECT json_build_object('table','@definer_public_execute','row',json_build_array(n.nspname,p.proname,pg_get_function_identity_arguments(p.oid),EXISTS(SELECT 1 FROM aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a WHERE a.grantee=0 AND a.privilege_type='EXECUTE'))) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname IN ('app','ledger','jobs') AND p.prosecdef ORDER BY n.nspname,p.proname,pg_get_function_identity_arguments(p.oid);",
        "COMMIT;",
    ])
    digests, counts = {}, {}
    with tempfile.TemporaryFile(mode='w+t') as errors, tempfile.TemporaryFile(mode='w+t') as command_file:
        command_file.write('\n'.join(commands))
        command_file.seek(0)
        process = subprocess.Popen(['psql', '-XqAt', '-v', 'ON_ERROR_STOP=1'],
                                   stdin=command_file, stdout=subprocess.PIPE, stderr=errors, text=True, env=environment)
        try:
            assert process.stdout is not None
            for line in process.stdout:
                record = json.loads(line)
                name = record['table']
                if record.get('header'):
                    if name in digests:
                        raise FingerprintError('duplicate fingerprint section')
                    digests[name], counts[name] = hashlib.sha256(), 0
                else:
                    digests[name].update(row_bytes(line))
                    counts[name] += 1
            if process.wait() != 0:
                raise FingerprintError('fingerprint capture failed; no evidence was accepted')
        finally:
            if process.poll() is None:
                process.kill()
                process.wait()
            if process.stdout is not None:
                process.stdout.close()
    if set(digests) != names | {'@policies', '@rls', '@definer_public_execute'}:
        raise FingerprintError('fingerprint capture was incomplete')
    unbalanced = psql('SELECT count(*) FROM (SELECT t.id FROM ledger.transactions t LEFT JOIN ledger.postings p ON p.transaction_id=t.id GROUP BY t.id HAVING count(p.transaction_id)=0 OR sum(p.debit_kobo)<>sum(p.credit_kobo)) bad;', environment).strip()
    if unbalanced != '0':
        raise FingerprintError('restored/source ledger has unbalanced transactions')
    return {'schema_version': 1, 'scope': 'quiesced-data-rls-public-definer-acl',
            'sections': {name: {'rows': counts[name], 'sha256': digests[name].hexdigest()} for name in sorted(digests)}}


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('operation', choices=['capture', 'compare'])
    parser.add_argument('first', type=Path)
    parser.add_argument('second', type=Path, nargs='?')
    parser.add_argument('--quiesced', action='store_true')
    args = parser.parse_args()
    if args.operation == 'capture':
        if not args.quiesced:
            raise FingerprintError('paused writers and DDL must be acknowledged with --quiesced')
        payload = capture()
        fd = os.open(args.first, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, 'w', encoding='utf-8') as stream:
            json.dump(payload, stream, indent=2, sort_keys=True)
            stream.write('\n')
        print('recovery_fingerprint=captured')
    else:
        if args.second is None:
            raise FingerprintError('compare requires two fingerprints')
        before, after = json.loads(args.first.read_text()), json.loads(args.second.read_text())
        if before.get('schema_version') != 1 or not before.get('sections') or before != after:
            raise FingerprintError('restored data or security fingerprint does not match the source')
        print('data_and_security_fingerprint=matched')


if __name__ == '__main__':
    try:
        main()
    except FingerprintError as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
    except (OSError, ValueError, KeyError, TypeError, AssertionError):
        print('Recovery fingerprint failed; check arguments, schema, permissions and protected evidence.', file=sys.stderr)
        sys.exit(1)
