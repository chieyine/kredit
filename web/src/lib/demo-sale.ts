/** A labelled example, never evidence of a real customer, repayment or provider result. */
export const DEMO_SALE = Object.freeze({
  supplier: 'Kora Wholesale', customer: 'Adebayo Stores', reference: 'EXAMPLE-2048',
  principalKobo: 120000000, paidKobo: 40000000, dueDate: '2026-09-18', graceHours: 24
});
export const DEMO_BALANCE_KOBO = DEMO_SALE.principalKobo - DEMO_SALE.paidKobo;
export const DEMO_PAID_PERCENT = DEMO_SALE.paidKobo * 100 / DEMO_SALE.principalKobo;
