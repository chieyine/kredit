#!/usr/bin/env node

import { createHash } from 'node:crypto';
import { existsSync, readFileSync } from 'node:fs';
import { dirname, extname, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';

const root = resolve(import.meta.dirname, '..');
const excluded = [
	'!.git/**', '!node_modules/**', '!web/node_modules/**', '!web/build/**',
	'!web/.svelte-kit/**', '!web/test-results/**', '!web/playwright-report/**',
	'!.pnpm-store/**', '!.tmp/**'
];
const listing = spawnSync('rg', ['--files', '--hidden', ...excluded.flatMap((pattern) => ['-g', pattern])], {
	cwd: root,
	encoding: 'utf8'
});
if (listing.status !== 0) {
	process.stderr.write(listing.stderr || 'Could not list repository files.\n');
	process.exit(1);
}

const files = listing.stdout.trim().split('\n').filter(Boolean).sort();
const routes = new Set(files.filter((file) => /^web\/src\/routes\/.*\/\+page\.svelte$|^web\/src\/routes\/\+page\.svelte$/.test(file)).map((file) => {
	const path = file.slice('web/src/routes'.length).replace(/\/\+page\.svelte$/, '') || '/';
	return path;
}));
const binaryExtensions = new Set(['.jpg', '.jpeg', '.png', '.webp', '.gif', '.ico', '.woff', '.woff2']);
const failures = [];
const counts = new Map();
const repositoryHash = createHash('sha256');
const decoder = new TextDecoder('utf-8', { fatal: true });

function fail(file, message) {
	failures.push(`${file}: ${message}`);
}

function checkMarkdownLinks(file, text) {
	const directory = dirname(resolve(root, file));
	for (const match of text.matchAll(/\[[^\]]*\]\(([^)]+)\)/g)) {
		let target = match[1].trim().replace(/^<|>$/g, '').split(/\s+["']/)[0];
		if (!target || /^(?:[a-z]+:|#|\/)/i.test(target)) continue;
		target = decodeURIComponent(target.split('#')[0]);
		if (target && !existsSync(resolve(directory, target))) fail(file, `broken relative link: ${target}`);
	}
}

function checkStaticRouteLinks(file, text) {
	for (const match of text.matchAll(/href="(\/[^"{]*)"/g)) {
		const target = match[1].split(/[?#]/)[0].replace(/\/$/, '') || '/';
		if (target.startsWith('/api/') || target.startsWith('/_app/')) continue;
		if (routes.has(target) || existsSync(resolve(root, 'web/static', target.slice(1)))) continue;
		fail(file, `static link has no matching route or asset: ${target}`);
	}
}

for (const file of files) {
	const absolute = resolve(root, file);
	const bytes = readFileSync(absolute);
	const extension = extname(file).toLowerCase() || '[none]';
	counts.set(extension, (counts.get(extension) ?? 0) + 1);
	repositoryHash.update(file).update('\0').update(bytes).update('\0');
	if (bytes.length === 0) {
		fail(file, 'file is empty');
		continue;
	}
	if (binaryExtensions.has(extension)) continue;

	let text;
	try {
		text = decoder.decode(bytes);
	} catch {
		fail(file, 'text file is not valid UTF-8');
		continue;
	}
	if (text.includes('\0')) fail(file, 'text file contains a NUL byte');
	if (text.includes('\r')) fail(file, 'text file contains CRLF or stray carriage returns');
	if (!text.endsWith('\n')) fail(file, 'text file must end with a newline');
	const retiredName = ['ti', 'chara'].join('');
	if (new RegExp(retiredName, 'i').test(text)) fail(file, 'old product name remains');

	if (extension === '.json' || extension === '.webmanifest') {
		try { JSON.parse(text); } catch (error) { fail(file, `invalid JSON: ${error.message}`); }
	}
	if (extension === '.svg' && !/^<svg\b/.test(text.trimStart())) fail(file, 'SVG root element is missing');
	if (extension === '.xml' && !/<urlset\b/.test(text)) fail(file, 'sitemap root element is missing');
	if (extension === '.md') checkMarkdownLinks(file, text);
	if (extension === '.svelte') checkStaticRouteLinks(file, text);
}

const requiredBrandEvidence = new Map([
	['web/src/routes/+layout.svelte', "const SITE_URL = 'https://kredit.com.ng'"],
	['web/static/manifest.webmanifest', '"short_name": "Kredit"'],
	['api/openapi.yaml', 'title: Kredit API'],
	['go.mod', 'module kredit'],
	['internal/web/auth_handlers.go', 'kredit_session']
]);
for (const [file, evidence] of requiredBrandEvidence) {
	if (!readFileSync(resolve(root, file), 'utf8').includes(evidence)) fail(file, `missing required evidence: ${evidence}`);
}

for (const failure of failures) process.stderr.write(`${failure}\n`);
const summary = [...counts.entries()].sort((a, b) => b[1] - a[1]).map(([type, count]) => `${type}:${count}`).join(', ');
process.stdout.write(`Repository audit read ${files.length} owned files (${summary}).\n`);
process.stdout.write(`Aggregate source fingerprint: ${repositoryHash.digest('hex')}\n`);
if (failures.length > 0) {
	process.stderr.write(`Repository audit failed with ${failures.length} issue(s).\n`);
	process.exit(1);
}
process.stdout.write('Repository file integrity and brand consistency passed.\n');
