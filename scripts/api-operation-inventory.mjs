#!/usr/bin/env node

import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { spawnSync } from "node:child_process";
import { createRequire } from "node:module";
const require = createRequire(new URL("../web/package.json", import.meta.url));
const ts = require("typescript");

export const root = resolve(import.meta.dirname, "..");
export const serverSource = readFileSync(
  resolve(root, "internal/web/server.go"),
  "utf8",
);
const openapiSource = readFileSync(resolve(root, "api/openapi.yaml"), "utf8");
export let listing = spawnSync(
  "rg",
  [
    "--files",
    "web/src",
    "-g",
    "*.svelte",
    "-g",
    "*.ts",
    "-g",
    "!*.test.*",
    "-g",
    "!*.spec.*",
    "-g",
    "!web/src/lib/api/generated/**",
  ],
  {
    cwd: root,
    encoding: "utf8",
  },
);
if (listing.status !== 0) {
  listing = spawnSync(
    "git",
    [
      "ls-files",
      "--cached",
      "--others",
      "--exclude-standard",
      "web/src/**/*.svelte",
      "web/src/**/*.ts",
      ":(exclude)web/src/lib/api/generated/**",
      ":(exclude)**/*.test.*",
      ":(exclude)**/*.spec.*",
    ],
    {
      cwd: root,
      encoding: "utf8",
    },
  );
}
if (listing.status !== 0)
  throw new Error(listing.stderr || "Could not list frontend source files.");

export const normalize = (path) =>
  path
    .replace(/\$\{[^}]+\}|\{[^}]+\}/g, "{id}")
    .split("?")[0]
    .replace(/\/$/, "");
export const serverRoutes = new Set(
  [...serverSource.matchAll(/HandleFunc\("([A-Z]+) ([^"]+)"/g)].map(
    (match) => `${match[1]} ${normalize(match[2])}`,
  ),
);
for (const path of ["/api/v1/healthz", "/api/v1/readyz", "/api/v1/meta"])
  serverRoutes.add(`GET ${path}`);

function hasDynamicRouteChoice(path) {
  for (const match of path.matchAll(/\$\{([^}]+)\}/g)) {
    const expression = match[1].trim();
    if (
      !/(?:^|\.)(?:id|token|public_token)$/i.test(expression) &&
      !/ID$/.test(expression)
    )
      return true;
  }
  return false;
}

// Parse call arguments: decoder bodies may themselves contain a `method`
// field, which must never be mistaken for the request options.
export const frontendCalls = [];
for (const file of listing.stdout.trim().split("\n").filter(Boolean).filter((file) => existsSync(resolve(root, file)))) {
  const source = readFileSync(resolve(root, file), "utf8");
  const scripts = file.endsWith(".svelte")
    ? [...source.matchAll(/<script\b[^>]*>([\s\S]*?)<\/script>/g)].map(
        (match) => match[1],
      )
    : [source];
  for (const script of scripts) {
    const tree = ts.createSourceFile(
      file,
      script,
      ts.ScriptTarget.Latest,
      true,
      ts.ScriptKind.TS,
    );
    const intents = new Map();
    function captureIntent(name, expression) {
      if (
        !ts.isNewExpression(expression) ||
        expression.expression.getText(tree) !== "MutationIntent"
      )
        return;
      const argument = expression.arguments?.[1];
      const path =
        argument &&
        (ts.isStringLiteral(argument) ||
          ts.isNoSubstitutionTemplateLiteral(argument))
          ? argument.text
          : argument && ts.isTemplateExpression(argument)
            ? argument.getText(tree).slice(1, -1)
            : null;
      const values = intents.get(name) ?? new Set();
      values.add(path);
      intents.set(name, values);
    }
    function bindings(node) {
      if (
        ts.isVariableDeclaration(node) &&
        ts.isIdentifier(node.name) &&
        node.initializer
      )
        captureIntent(node.name.text, node.initializer);
      if (ts.isBinaryExpression(node) && ts.isIdentifier(node.left))
        captureIntent(node.left.text, node.right);
      ts.forEachChild(node, bindings);
    }
    bindings(tree);
    function visit(node) {
      if (
        ts.isCallExpression(node) &&
        ts.isIdentifier(node.expression) &&
        [
          "fetch",
          "checkedJSON",
          "readResource",
          "adminGet",
          "adminPost",
        ].includes(node.expression.text)
      ) {
        const name = node.expression.text;
        const argument = node.arguments[name === "readResource" ? 1 : 0];
        let path;
        if (
          argument &&
          (ts.isStringLiteral(argument) ||
            ts.isNoSubstitutionTemplateLiteral(argument))
        )
          path = argument.text;
        else if (argument && ts.isTemplateExpression(argument))
          path = argument.getText(tree).slice(1, -1);
        if (path?.startsWith("/api/v1/") && !hasDynamicRouteChoice(path)) {
          const options = ["fetch", "checkedJSON"].includes(name)
            ? node.arguments[name === "checkedJSON" ? 2 : 1]
            : undefined;
          let method = name === "adminPost" ? "POST" : "GET",
            known = true;
          if (name === "adminPost" && node.arguments[2]) {
            const arg = node.arguments[2];
            if (ts.isStringLiteral(arg)) method = arg.text.toUpperCase();
            else known = false;
          }
          if (options && options.kind !== ts.SyntaxKind.UndefinedKeyword) {
            if (!ts.isObjectLiteralExpression(options)) known = false;
            else
              for (const property of options.properties) {
                if (ts.isSpreadAssignment(property)) known = false;
                else if (
                  property.name &&
                  (ts.isIdentifier(property.name) ||
                    ts.isStringLiteral(property.name)) &&
                  property.name.text === "method"
                ) {
                  if (
                    ts.isPropertyAssignment(property) &&
                    ts.isStringLiteral(property.initializer)
                  ) {
                    method = property.initializer.text.toUpperCase();
                    known = true;
                  } else known = false;
                }
              }
          }
          if (known)
            frontendCalls.push({ file, method, path: normalize(path) });
        }
      }
      if (
        ts.isCallExpression(node) &&
        ts.isPropertyAccessExpression(node.expression) &&
        node.expression.name.text === "run" &&
        ts.isIdentifier(node.expression.expression)
      ) {
        const values = intents.get(node.expression.expression.text);
        const path = values?.size === 1 ? [...values][0] : null;
        const methodArg = node.arguments[2];
        const method = !methodArg
          ? "POST"
          : ts.isStringLiteral(methodArg)
            ? methodArg.text.toUpperCase()
            : null;
        if (
          path?.startsWith("/api/v1/") &&
          !hasDynamicRouteChoice(path) &&
          method
        )
          frontendCalls.push({ file, method, path: normalize(path) });
      }
      ts.forEachChild(node, visit);
    }
    visit(tree);
  }
}

export const openapiRoutes = new Set();
let currentPath = "";
for (const line of openapiSource.split("\n")) {
  const pathMatch = line.match(/^  (\/[^:]+):\s*$/);
  if (pathMatch) {
    currentPath = pathMatch[1];
    continue;
  }
  const methodMatch = line.match(/^    (get|post|put|patch|delete):\s*$/);
  if (currentPath && methodMatch)
    openapiRoutes.add(
      `${methodMatch[1].toUpperCase()} ${normalize(`/api/v1${currentPath}`)}`,
    );
}

// Path mentions are weaker mapping hints, never proof that an operation runs.
export const frontendSource = listing.stdout
  .trim()
  .split("\n")
  .filter(Boolean)
  .filter((file) => existsSync(resolve(root, file)))
  .map((file) => readFileSync(resolve(root, file), "utf8"))
  .join("\n");
export const routes = [
  ...serverSource.matchAll(/HandleFunc\("([A-Z]+) ([^"]+)"/g),
].map((match) => ({ method: match[1], path: normalize(match[2]) }));
export const knownOperations = new Set(
  frontendCalls.map((call) => `${call.method} ${call.path}`),
);
