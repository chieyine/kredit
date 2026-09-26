/**
 * Simple Kredit switches some tools off (see src/lib/features.ts). Their tests
 * still exist and still pass when the tools are switched back on:
 *
 *   PUBLIC_KREDIT_FULL_WORKSPACE=1 pnpm test
 */
export const PARKED = process.env.PUBLIC_KREDIT_FULL_WORKSPACE !== '1';
export const PARKED_REASON = 'Switched off in simple Kredit; run with PUBLIC_KREDIT_FULL_WORKSPACE=1 to test it.';
