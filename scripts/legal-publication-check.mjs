import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { legalPublication as publication } from "../web/src/lib/server/legal-publication.ts";

const backend = readFileSync(
  new URL("../internal/legalpublication/versions.go", import.meta.url),
  "utf8",
);
for (const [field, constant] of [
  ["termsVersion", "TermsVersion"],
  ["privacyVersion", "PrivacyVersion"],
]) {
  const version = backend.match(
    new RegExp(`const ${constant} = "([^"]+)"`),
  )?.[1];
  assert.ok(version, `Missing backend ${constant}`);
  assert.equal(publication[field], version, `${field} must match onboarding`);
}
assert.equal(
  publication.active,
  true,
  "The published documents must be active",
);
for (const field of [
  "entityName",
  "registrationNumber",
  "serviceAddress",
  "complaintsVersion",
]) {
  assert.ok(publication[field].trim(), `Missing published ${field}`);
}
for (const field of ["legalEmail", "privacyEmail"]) {
  assert.match(
    publication[field],
    /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
    `Invalid published ${field}`,
  );
}
assert.match(publication.effectiveDate, /^\d{4}-\d{2}-\d{2}$/);
assert.equal(
  new Date(publication.effectiveDate).toISOString().slice(0, 10),
  publication.effectiveDate,
);
console.log(
  "Published legal metadata matches onboarding. Legal approval remains a separate release gate.",
);
