#!/usr/bin/env node

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { createRequire } from 'node:module';
const require = createRequire(new URL('../web/package.json', import.meta.url));
const ts = require('typescript');

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

// Parse call arguments: decoder bodies may themselves contain a `method`
// field, which must never be mistaken for the request options.
const frontendCalls = [];
for (const file of listing.stdout.trim().split('\n').filter(Boolean)) {
 const source = readFileSync(resolve(root, file), 'utf8');
 const scripts = file.endsWith('.svelte') ? [...source.matchAll(/<script\b[^>]*>([\s\S]*?)<\/script>/g)].map(match => match[1]) : [source];
 for (const script of scripts) {
  const tree = ts.createSourceFile(file, script, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
  function visit(node) {
   if (ts.isCallExpression(node) && ts.isIdentifier(node.expression) && ['fetch','checkedJSON'].includes(node.expression.text)) {
    const argument = node.arguments[0];
    let path;
    if (argument && (ts.isStringLiteral(argument) || ts.isNoSubstitutionTemplateLiteral(argument))) path = argument.text;
    else if (argument && ts.isTemplateExpression(argument)) path = argument.getText(tree).slice(1,-1);
    if (path?.startsWith('/api/v1/') && !hasDynamicRouteChoice(path)) {
     const options = node.arguments[node.expression.text === 'checkedJSON' ? 2 : 1];
     let method = 'GET', known = true;
     if (options && options.kind !== ts.SyntaxKind.UndefinedKeyword) {
      if (!ts.isObjectLiteralExpression(options)) known = false;
      else for (const property of options.properties) {
       if (ts.isSpreadAssignment(property)) known = false;
       else if (property.name && (ts.isIdentifier(property.name) || ts.isStringLiteral(property.name)) && property.name.text === 'method') {
        if (ts.isPropertyAssignment(property) && ts.isStringLiteral(property.initializer)) { method = property.initializer.text.toUpperCase(); known = true; }
        else known = false;
       }
      }
     }
     if (known) frontendCalls.push({file,method,path:normalize(path)});
    }
   }
   ts.forEachChild(node,visit);
  }
  visit(tree);
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
