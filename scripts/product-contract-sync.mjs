#!/usr/bin/env node
import {
  frontendCalls,
  serverRoutes,
  openapiRoutes,
} from "./api-operation-inventory.mjs";

const missingBackend = frontendCalls.filter(
  ({ method, path }) => !serverRoutes.has(`${method} ${path}`),
);

const missingOpenAPI = [...serverRoutes]
  .filter((route) => route.includes(" /api/v1/"))
  .filter((route) => !openapiRoutes.has(route));
const staleOpenAPI = [...openapiRoutes].filter(
  (route) => !serverRoutes.has(route),
);

for (const item of missingBackend)
  process.stderr.write(
    `${item.file}: ${item.method} ${item.path} has no matching backend route.\n`,
  );
for (const route of missingOpenAPI)
  process.stderr.write(`API contract is missing backend route ${route}.\n`);
for (const route of staleOpenAPI)
  process.stderr.write(
    `API contract declares missing backend route ${route}.\n`,
  );

const failures =
  missingBackend.length + missingOpenAPI.length + staleOpenAPI.length;
if (failures) {
  process.stderr.write(
    `Product contract sync failed with ${failures} mismatch(es).\n`,
  );
  process.exit(1);
}
process.stdout.write(
  `Product contract sync passed: ${frontendCalls.length} statically resolved frontend calls, ${serverRoutes.size} backend routes and ${openapiRoutes.size} API operations agree. Dynamic requests and response behavior are outside this check.\n`,
);
