import { test, expect } from '@playwright/test';
import { parseNaira, formatKobo, sumKobo, verbalizeNaira } from '../src/lib/money';

test('naira input preserves exact kobo and rejects ambiguous amounts', () => {
 for (const [input, expected] of [['0',0],['0.29',29],['1,234.50',123450],['90071992547409.91',Number.MAX_SAFE_INTEGER],['0.001',-1],['1e3',-1],['12,34',-1],['Infinity',-1],['90071992547409.92',-1],['',-1]] as const) {
  expect(parseNaira(input), input).toBe(expected);
 }
});

test('financial display preserves large integer amounts and never turns invalid data into zero',()=>{
 expect(formatKobo('9223372036854775807')).toBe('₦92,233,720,368,547,758.07');
 expect(formatKobo(9007199254740992)).toBe('Amount unavailable');
 expect(formatKobo('invalid')).toBe('Amount unavailable');
 expect(formatKobo(sumKobo([Number.MAX_SAFE_INTEGER,1]))).toBe('₦90,071,992,547,409.92');
 expect(formatKobo(-29)).toBe('-₦0.29');
});

test('verbalizeNaira produces accurate verbal currency description', () => {
 expect(verbalizeNaira(100_000_00n)).toBe('One hundred thousand Naira');
 expect(verbalizeNaira(1_500_000_00n)).toBe('One million and five hundred thousand Naira');
 expect(verbalizeNaira(250_50n)).toBe('Two hundred and fifty Naira and 50 kobo');
 expect(verbalizeNaira(100_000_000_000_000n)).toBe('One trillion Naira');
 expect(verbalizeNaira(0)).toBe('');
});

