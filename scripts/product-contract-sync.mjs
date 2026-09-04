#!/usr/bin/env node

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { spawnSync } from 'node:child_process';

const root = resolve(import.meta.dirname, '..');
const serverSource = readFileSync(resolve(root, 'internal/web/server.go'), 'utf8');
const openapiSource = readFileSync(resolve(root, 'api/openapi.yaml'), 'utf8');
let listing = spawnSync('rg', ['--files', 'web/src', '-g', '*.svelte', '-g', '*.ts', '-g', '!web/src/lib/api/generated/**'], {
	cwd: root,
	encoding: 'utf8'
});
if (listing.status !== 0) {
	listing = spawnSync('git', ['ls-files', 'web/src/**/*.svelte', 'web/src/**/*.ts', ':(exclude)web/src/lib/api/generated/**'], {
		cwd: root,
		encoding: 'utf8'
	});
}
if (listing.status !== 0) throw new Error(listing.stderr || 'Could not list frontend source files.');

const normalize = (path) => path
	.replace(/\$\{[^}]+\}|\{[^}]+\}/g, '{id}')
	.split('?')[0]
	.replace(/\/$/, '');
const serverRoutes = new Set(
	[...serverSource.matchAll(/HandleFunc\("([A-Z]+) ([^"]+)"/g)]
		.map((match) => `${match[1]} ${normalize(match[2])}`)
);
for (const path of ['/api/v1/healthz', '/api/v1/readyz', '/api/v1/meta']) serverRoutes.add(`GET ${path}`);

function hasDynamicRouteChoice(path) {
	for (const match of path.matchAll(/\$\{([^}]+)\}/g)) {
		const expression = match[1].trim();
		if (!/(?:^|\.)(?:id|token|public_token)$/i.test(expression) && !/ID$/.test(expression)) return true;
	}
	return false;
}

function readFetchCall(source, start) {
	let depth = 1;
	let quote = '';
	let escaped = false;
	for (let index = start; index < source.length; index++) {
		const character = source[index];
		if (quote) {
			if (escaped) escaped = false;
			else if (character === '\\') escaped = true;
			else if (character === quote) quote = '';
			continue;
		}
		if (character === '"' || character === "'" || character === '`') { quote = character; continue; }
		if (character === '(') depth++;
		if (character === ')' && --depth === 0) return source.slice(start, index);
	}
	return '';
}

const frontendCalls = [];
for (const file of listing.stdout.trim().split('\n').filter(Boolean)) {
	const source = readFileSync(resolve(root, file), 'utf8');
	for (const match of source.matchAll(/\bfetch\(/g)) {
		const call = readFetchCall(source, match.index + match[0].length);
		const pathMatch = call.match(/^\s*([`'"])(\/api\/v1\/[\s\S]*?)\1/);
		if (!pathMatch) continue;
		if (hasDynamicRouteChoice(pathMatch[2])) continue;
		const method = call.match(/\bmethod\s*:\s*['"]([A-Z]+)['"]/)?.[1] ?? 'GET';
		frontendCalls.push({ file, method, path: normalize(pathMatch[2]) });
	}
}

const missingBackend = frontendCalls.filter(({ method, path }) => !serverRoutes.has(`${method} ${path}`));

const openapiRoutes = new Set();
let currentPath = '';
for (const line of openapiSource.split('\n')) {
	const pathMatch = line.match(/^  (\/[^:]+):\s*$/);
	if (pathMatch) { currentPath = pathMatch[1]; continue; }
	const methodMatch = line.match(/^    (get|post|put|patch|delete):\s*$/);
	if (currentPath && methodMatch) openapiRoutes.add(`${methodMatch[1].toUpperCase()} ${normalize(`/api/v1${currentPath}`)}`);
}
const missingOpenAPI = [...serverRoutes]
	.filter((route) => route.includes(' /api/v1/'))
	.filter((route) => !openapiRoutes.has(route));
const staleOpenAPI = [...openapiRoutes].filter((route) => !serverRoutes.has(route));

for (const item of missingBackend) process.stderr.write(`${item.file}: ${item.method} ${item.path} has no matching backend route.\n`);
for (const route of missingOpenAPI) process.stderr.write(`API contract is missing backend route ${route}.\n`);
for (const route of staleOpenAPI) process.stderr.write(`API contract declares missing backend route ${route}.\n`);

const failures = missingBackend.length + missingOpenAPI.length + staleOpenAPI.length;
if (failures) {
	process.stderr.write(`Product contract sync failed with ${failures} mismatch(es).\n`);
	process.exit(1);
}
process.stdout.write(`Product contract sync passed: ${frontendCalls.length} explicit frontend calls, ${serverRoutes.size} backend routes and ${openapiRoutes.size} API operations agree.\n`);
