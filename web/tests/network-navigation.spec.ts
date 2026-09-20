import { expect, test } from '@playwright/test';
import { workspaceHref } from '../src/lib/workspace-navigation';
import { hostedAuthorizationURL } from '../src/lib/financial-copy';
import { safeNext } from '../src/lib/api/reliable';

test('business context stays attached without contaminating a different explicit workspace',()=>{
 const current=new URL('https://kredit.test/workspace/purchases?organization=org-a&business_id=buyer-a');
 expect(workspaceHref('/workspace/purchases/orders',current)).toBe('/workspace/purchases/orders?organization=org-a&business_id=buyer-a');
 expect(workspaceHref('/workspace/sales',current)).toBe('/workspace/sales?organization=org-a');
 expect(workspaceHref('/workspace/purchases?organization=org-b',current)).toBe('/workspace/purchases?organization=org-b');
 expect(workspaceHref('/workspace/purchases?business_id=buyer-b',current)).toBe('/workspace/purchases?business_id=buyer-b');
 expect(workspaceHref('/personal/purchases',current)).toBe('/personal/purchases');
});
test('bank authorization accepts only the current internal route and approved HTTPS hosts',()=>{
 const reference='a'.repeat(32);
 expect(hostedAuthorizationURL(`/workspace/purchases/bank-authorization/${reference}`,'native')).toBe(`/workspace/purchases/bank-authorization/${reference}`);
 expect(hostedAuthorizationURL(`/buyer/bank-authorization/${reference}`,'native')).toBeNull();
 expect(hostedAuthorizationURL('https://checkout.paystack.com/authorization','paystack')).toBeTruthy();
 expect(hostedAuthorizationURL('https://checkout.paystack.com.evil.test/authorization','paystack')).toBeNull();
});
test('one account entry retains safe invitation destinations and rejects external redirects',()=>{
 expect(safeNext(null,'https://kredit.test')).toBe('/start');
 expect(safeNext('/personal/purchases/sale-1','https://kredit.test')).toBe('/personal/purchases/sale-1');
 expect(safeNext('https://evil.test/','https://kredit.test')).toBe('/start');
});
