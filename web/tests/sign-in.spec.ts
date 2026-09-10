import { expect, test } from '@playwright/test';

/**
 * Sign-in is the one screen every person meets, and until now no test drove it.
 * That is how a six-digit code field shipped with pattern="[0-9]{6}" written as
 * a plain attribute: Svelte reads `{` in an attribute value as the start of an
 * expression, so the compiled attribute was `[0-9]6` and the browser rejected
 * every real code as invalid. Nothing in the suite ever typed six digits.
 *
 * These tests drive the form the way a person does. They stub the two auth
 * endpoints so the flow can be exercised without SMS or email — which is also
 * what makes them run on a machine with no delivery provider configured.
 */

const CHALLENGE = '/api/v1/auth/otp/challenges';
const VERIFY = '/api/v1/auth/otp/verify';

async function stubChallenge(page: import('@playwright/test').Page, minutes = 10) {
	await page.route(`**${CHALLENGE}`, async (route) => {
		expect(route.request().postDataJSON()).toMatchObject({ channel: 'phone', identifier: '+2348012345678', purpose: 'login' });
		await route.fulfill({
			status: 202,
			contentType: 'application/json',
			body: JSON.stringify({
				challenge_id: 'challenge-1',
				expires_at: new Date(Date.now() + minutes * 60_000).toISOString()
			})
		});
	});
}

test('a six-digit code is accepted by the field that asks for six digits', async ({ page }) => {
	await stubChallenge(page);
	let submitted: { challenge_id?: string; code?: string } = {};
	await page.route(`**${VERIFY}`, async (route) => {
		submitted = JSON.parse(route.request().postData() ?? '{}');
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({ user: { id: 'user-1' } })
		});
	});
	await page.route('**/api/v1/me', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({ user: { id: 'user-1' } })
		});
	});

	await page.goto('/app');
	await page.getByRole('radio', { name: /Phone/ }).check();
	await page.getByLabel('Phone number').fill('08012345678');
	await page.getByRole('button', { name: 'Send me a code' }).click();

	const codeField = page.getByLabel('Six-digit code');
	await expect(codeField).toBeVisible();
	await codeField.fill('123456');

	// The bug lived here: with pattern="[0-9]6" the browser marks a real code
	// invalid, the form never submits, and the person is stuck with no message.
	await expect(codeField).toHaveJSProperty('validity.valid', true);
	await expect(page.getByRole('button', { name: 'Open my account' })).toBeEnabled();

	await page.getByRole('button', { name: 'Open my account' }).click();
	await expect.poll(() => submitted.code).toBe('123456');
	expect(submitted.challenge_id).toBe('challenge-1');
});

test('a code the field should refuse is refused, and the button stays closed', async ({ page }) => {
	await stubChallenge(page);
	await page.goto('/app');
	await page.getByRole('radio', { name: /Phone/ }).check();
	await page.getByLabel('Phone number').fill('08012345678');
	await page.getByRole('button', { name: 'Send me a code' }).click();

	const codeField = page.getByLabel('Six-digit code');
	await codeField.fill('12345');
	await expect(codeField).toHaveJSProperty('validity.valid', false);
	await expect(page.getByRole('button', { name: 'Open my account' })).toBeDisabled();
});

test('a wrong code is explained in words a person can act on', async ({ page }) => {
	await stubChallenge(page);
	await page.route(`**${VERIFY}`, async (route) => {
		await route.fulfill({
			status: 401,
			contentType: 'application/problem+json',
			body: JSON.stringify({ detail: 'Authentication failed.' })
		});
	});

	await page.goto('/app');
	await page.getByRole('radio', { name: /Phone/ }).check();
	await page.getByLabel('Phone number').fill('08012345678');
	await page.getByRole('button', { name: 'Send me a code' }).click();
	await page.getByLabel('Six-digit code').fill('000000');
	await page.getByRole('button', { name: 'Open my account' }).click();

	const message = page.getByRole('alert');
	await expect(message).toContainText('That code is incorrect or has expired');
	// It has to say what to do next, not only that something went wrong.
	await expect(message).toContainText('request a new code');
	await expect(page).toHaveURL(/\/app$/);
});

test('an expired code closes the field and offers a new one', async ({ page }) => {
	await page.route(`**${CHALLENGE}`, async (route) => {
		expect(route.request().postDataJSON()).toMatchObject({ channel: 'phone', identifier: '+2348012345678', purpose: 'login' });
		await route.fulfill({
			status: 202,
			contentType: 'application/json',
			body: JSON.stringify({
				challenge_id: 'challenge-2',
				// Just far enough ahead to pass the client's sanity check, then gone.
				expires_at: new Date(Date.now() + 1_500).toISOString()
			})
		});
	});
	await page.goto('/app');
	await page.getByRole('radio', { name: /Phone/ }).check();
	await page.getByLabel('Phone number').fill('08012345678');
	await page.getByRole('button', { name: 'Send me a code' }).click();
	await expect(page.getByLabel('Six-digit code')).toBeVisible();

	await expect(page.getByText('This code has expired. Request a new one below.')).toBeVisible({
		timeout: 15_000
	});
	await expect(page.getByLabel('Six-digit code')).toBeDisabled();
	await expect(page.getByRole('button', { name: 'Open my account' })).toBeDisabled();
});

test('the sign-in page says a code will never be asked for by support', async ({ page }) => {
	await page.goto('/app');
	await expect(
		page.getByText('Kredit support will never ask you to share your sign-in code.')
	).toBeVisible();
});
