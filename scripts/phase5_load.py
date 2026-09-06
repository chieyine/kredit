#!/usr/bin/env python3
"""Bounded, authenticated API/database load baseline on an isolated local stack.

Never targets production. Uses real development OTP and real database reads,
not mocked sessions, a browser or an agent. Report is CI evidence, not capacity
certification for a production-shaped portfolio or an external provider.
"""
import argparse
import concurrent.futures
import http.cookiejar
import json
import math
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid
from pathlib import Path


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def validate_origin(base: str) -> str:
    url = urllib.parse.urlsplit(base)
    if url.scheme != 'http' or url.hostname not in ('127.0.0.1', '::1') or url.username or url.password or url.query or url.fragment or url.path not in ('', '/'):
        raise ValueError('load baseline accepts only an explicit loopback HTTP origin')
    if url.port is None or not 1 <= url.port <= 65535:
        raise ValueError('explicit local API port is required')
    return base.rstrip('/')


def percentile(values: list[float], percent: float) -> float:
    if not values or not 0 < percent <= 1:
        raise ValueError('nonempty samples and a valid percentile are required')
    ordered = sorted(values)
    return ordered[max(0, math.ceil(percent * len(ordered)) - 1)]


def read(opener, base: str, path: str, payload=None, cookie: str = '') -> tuple[int, dict]:
    headers = {'Accept': 'application/json', 'Origin': base}
    if cookie:
        headers['Cookie'] = cookie
    data = None
    if payload is not None:
        headers['Content-Type'] = 'application/json'
        data = json.dumps(payload).encode()
    request = urllib.request.Request(base + path, data=data, headers=headers)
    try:
        with opener.open(request, timeout=10) as response:
            body = response.read(2 * 1024 * 1024 + 1)
            if len(body) > 2 * 1024 * 1024:
                raise ValueError('response exceeds baseline size bound')
            value = json.loads(body)
            if not isinstance(value, dict):
                raise ValueError('unexpected response shape')
            return response.status, value
    except urllib.error.HTTPError as error:
        # Do not print or persist response bodies, cookies, OTPs or identifiers.
        code = error.code
        error.close()
        return code, {}


def login(base: str, identifier: str) -> tuple[str, dict]:
    jar = http.cookiejar.CookieJar()
    opener = urllib.request.build_opener(NoRedirect(), urllib.request.HTTPCookieProcessor(jar), urllib.request.ProxyHandler({}))
    status, challenge = read(opener, base, '/api/v1/auth/otp/challenges', {'identifier': identifier, 'channel': 'email', 'purpose': 'login'})
    code = challenge.get('development_code', '')
    if status != 202 or not isinstance(code, str) or re.fullmatch(r'\d{6}', code) is None:
        raise ValueError('real development OTP is unavailable; refusing synthetic authentication')
    status, _ = read(opener, base, '/api/v1/auth/otp/verify', {'challenge_id': challenge['challenge_id'], 'code': code, 'device_label': 'phase5-isolated-load'})
    if status != 200:
        raise ValueError('development authentication failed')
    status, me = read(opener, base, '/api/v1/me')
    if status != 200 or not jar:
        raise ValueError('authenticated session is unavailable')
    return '; '.join(cookie.name + '=' + cookie.value for cookie in jar), me


def valid_body(name: str, body: dict) -> bool:
    key = 'payments' if name == 'supplier_payments' else 'requests'
    items = body.get(key)
    return isinstance(items, list) and (name != 'buyer_requests' or len(items) > 0)


def run(base: str, requests: int, workers: int, p95_limit: float, p99_limit: float) -> dict:
    if os.environ.get('KREDIT_PHASE5_LOCAL') != '1' or os.environ.get('APP_ENV') != 'development':
        raise ValueError('isolated development acknowledgement is required')
    base = validate_origin(base)
    if not 20 <= requests <= 80 or not 1 <= workers <= 8 or not 0 < p95_limit <= p99_limit:
        raise ValueError('invalid workload or latency bounds')
    anonymous = urllib.request.build_opener(NoRedirect(), urllib.request.ProxyHandler({}))
    status, _ = read(anonymous, base, '/api/v1/ops/metrics/prometheus')
    if status not in (401, 403):
        raise ValueError('private metrics are not correctly protected')
    supplier_cookie, supplier = login(base, 'owner@abc-pharmaceuticals.test')
    buyer_cookie, _ = login(base, 'buyer@royal-pharmacy.test')
    organizations = supplier.get('organizations', [])
    if not organizations:
        raise ValueError('seeded supplier is missing')
    organization = str(uuid.UUID(organizations[0]['id']))
    paths = [('supplier_payments', '/api/v1/organizations/' + organization + '/payments', supplier_cookie),
             ('buyer_requests', '/api/v1/buyer/credit-requests', buyer_cookie)]
    status, _ = read(anonymous, base, paths[0][1])
    if status not in (401, 403):
        raise ValueError('supplier financial reads are not authentication-protected')

    def sample(index):
        name, path, cookie = paths[index % len(paths)]
        opener = urllib.request.build_opener(NoRedirect(), urllib.request.ProxyHandler({}))
        started = time.monotonic()
        try:
            status, body = read(opener, base, path, cookie=cookie)
            valid = status == 200 and valid_body(name, body)
        except (OSError, ValueError):
            status, valid = 0, False
        return name, (time.monotonic() - started) * 1000, status, valid

    started = time.monotonic()
    with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as pool:
        samples = list(pool.map(sample, range(requests)))
    routes = {}
    passed = True
    for name, _, _ in paths:
        subset = [row for row in samples if row[0] == name]
        durations = [row[1] for row in subset]
        failures = sum(not row[3] for row in subset)
        p95, p99 = percentile(durations, .95), percentile(durations, .99)
        ok = failures == 0 and p95 <= p95_limit and p99 <= p99_limit
        passed = passed and ok
        routes[name] = {'requests': len(subset), 'failures': failures, 'p95_ms': round(p95, 3), 'p99_ms': round(p99, 3), 'passed': ok}
    return {'evidence_type': 'isolated_ci_baseline', 'production_capacity_certified': False,
            'workers': workers, 'requests': requests, 'elapsed_seconds': round(time.monotonic() - started, 3),
            'p95_limit_ms': p95_limit, 'p99_limit_ms': p99_limit, 'routes': routes, 'passed': passed}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base-url', default='http://127.0.0.1:8080')
    parser.add_argument('--requests', type=int, default=64)
    parser.add_argument('--workers', type=int, default=4)
    parser.add_argument('--p95-ms', type=float, default=1000)
    parser.add_argument('--p99-ms', type=float, default=2000)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    try:
        report = run(args.base_url, args.requests, args.workers, args.p95_ms, args.p99_ms)
        fd = os.open(args.output, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, 'w', encoding='utf-8') as stream:
            json.dump(report, stream, indent=2)
            stream.write('\n')
        print(json.dumps(report, sort_keys=True))
        sys.exit(0 if report['passed'] else 1)
    except (OSError, ValueError, KeyError, TypeError):
        print('Phase 5 load baseline failed; inspect protected test logs. No credentials were written to evidence.', file=sys.stderr)
        sys.exit(1)
