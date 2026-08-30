# Wave 5 accessibility evidence

Evidence date: 29 August 2026

## Automated gate

`web/tests/accessibility.spec.ts` runs axe-core WCAG 2 A/AA, 2.1 AA, and 2.2
AA rules against sign-in, onboarding, credit creation, buyer acceptance, goods
release, goods receipt/trade-line drawdowns, payments, disputes, settings,
account recovery, privacy, and protected operations commands. Any serious or
critical violation fails the browser suite.

The interaction check also covers:

- skip-link navigation and programmatic focus at the content target;
- modal command-palette initial focus, native focus containment, Escape close,
  and focus restoration;
- reduced-motion media preference;
- 1280px-at-200%-zoom equivalent reflow and horizontal overflow;
- a 24 CSS-pixel automated minimum plus the product-wide 44px control target.

Current automated result: **15/15 passing; no serious or critical violations**.

## Manual assistive-technology and device review

These checks require real target hardware and a human accessibility reviewer.
They are release evidence, not something CI can truthfully simulate.

| Check | Owner | Status | Evidence required |
| --- | --- | --- | --- |
| VoiceOver + Safari on supported iPhone | Accessibility Reviewer | Pending external review | date, device/OS/browser, flow checklist, recording or signed notes |
| TalkBack + Chrome on supported Android | Accessibility Reviewer | Pending external review | date, device/OS/browser, flow checklist, recording or signed notes |
| Keyboard-only desktop critical journeys | Frontend Lead | Automated pass; manual sign-off pending | signed checklist covering order, visibility, escape, and restoration |
| 200% and 400% browser zoom | Frontend Lead | 200% automated pass; 400% manual review pending | screenshots and any approved exception |
| High-contrast/forced-colors mode | Accessibility Reviewer | Pending external review | Windows browser evidence and defect references |
| Payment/dispute terminology comprehension | Compliance Lead | Pending moderated review | participant notes and approved copy changes |

## Known limitations and disposition

- Native screen-reader pronunciation, virtual-cursor order, and mobile software
  keyboard behavior are not asserted by axe or Playwright. Owner:
  Accessibility Reviewer. Disposition: blocks `WCAG-AA` completion and release
  sign-off until the two target-device reviews above are attached.
- The current automated suite fails only on serious/critical axe findings as
  required. Moderate/minor findings remain visible in the raw Playwright trace
  and must be triaged during the manual review. Owner: Frontend Lead.
- Provider-hosted identity, mandate, settlement, and payment screens require
  separate accessibility evidence from each approved provider. Owner:
  Compliance Lead. Disposition: provider certification gate remains closed.
