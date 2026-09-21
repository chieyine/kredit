import { expect, test } from '@playwright/test';
import { ProxyBodyError, proxyHeaders, readProxyBody } from '../src/lib/server/proxy-body';

const signal = () => new AbortController().signal;

function bytes(chunks: Uint8Array[]): ReadableStream<Uint8Array> {
	return new ReadableStream({
		start(controller) {
			for (const chunk of chunks) controller.enqueue(chunk);
			controller.close();
		}
	});
}

test('proxy preserves exact body bytes and typed-array view boundaries', async () => {
	const backing = new Uint8Array([99, 1, 2, 99]);
	const stream = bytes([backing.subarray(1, 3), new Uint8Array([3, 4])]);
	const result = await readProxyBody(stream, 4, signal());
	expect(Array.from(result)).toEqual([1, 2, 3, 4]);
	expect(result.byteLength).toBe(4);
	expect(stream.locked).toBe(false);
	expect((await readProxyBody(bytes([]), 0, signal())).byteLength).toBe(0);
});

test('proxy accepts many small chunks without retaining per-chunk buffers', async () => {
	let remaining = 65_536;
	const stream = new ReadableStream<Uint8Array>({
		pull(controller) {
			if (remaining-- > 0) controller.enqueue(new Uint8Array([7]));
			else controller.close();
		}
	});
	const result = await readProxyBody(stream, 65_536, signal());
	expect(result.length).toBe(65_536);
	expect(result.every(value => value === 7)).toBe(true);
	expect(result.buffer.byteLength).toBeLessThanOrEqual(65_536);
	expect(stream.locked).toBe(false);
});

test('oversized body fails without waiting for a stuck cancellation hook', async () => {
	let cancelled = false;
	const stream = new ReadableStream<Uint8Array>({
		start(controller) { controller.enqueue(new Uint8Array([1, 2, 3])); },
		cancel() {
			cancelled = true;
			return new Promise<void>(() => {});
		}
	});
	await expect(readProxyBody(stream, 2, signal())).rejects.toMatchObject({ status: 413 });
	expect(cancelled).toBe(true);
	expect(stream.locked).toBe(false);
});

test('a stalled body is rejected by the input deadline', async () => {
	let cancelled = false;
	const stream = new ReadableStream<Uint8Array>({ cancel() { cancelled = true; } });
	await expect(readProxyBody(stream, 8, signal(), 20)).rejects.toMatchObject({ status: 408 });
	expect(cancelled).toBe(true);
	expect(stream.locked).toBe(false);
});

test('client cancellation interrupts an already pending read', async () => {
	const controller = new AbortController();
	const reason = new Error('synthetic client cancellation');
	let cancelled = false;
	const stream = new ReadableStream<Uint8Array>({ cancel() { cancelled = true; } });
	const operation = readProxyBody(stream, 8, controller.signal);
	controller.abort(reason);
	await expect(operation).rejects.toBe(reason);
	expect(cancelled).toBe(true);
	expect(stream.locked).toBe(false);
});

test('pre-cancelled requests do not consume a body', async () => {
	const controller = new AbortController();
	const reason = new Error('synthetic pre-cancellation');
	controller.abort(reason);
	const stream = new ReadableStream<Uint8Array>();
	await expect(readProxyBody(stream, 8, controller.signal)).rejects.toBe(reason);
	expect(stream.locked).toBe(false);
});

test('body read errors remain errors rather than truncated requests', async () => {
	const reason = new Error('synthetic read failure');
	const stream = new ReadableStream<Uint8Array>({ start(controller) { controller.error(reason); } });
	await expect(readProxyBody(stream, 8, signal())).rejects.toBe(reason);
	expect(stream.locked).toBe(false);
});

test('empty chunks cannot keep a request alive without making progress', async () => {
	const stream = new ReadableStream<Uint8Array>({ pull(controller) { controller.enqueue(new Uint8Array(0)); } });
	await expect(readProxyBody(stream, 8, signal())).rejects.toMatchObject({ status: 400 });
	expect(stream.locked).toBe(false);
});

test('invalid bounds fail before acquiring the stream', async () => {
	for (const limit of [-1, 0.5, Number.NaN, Number.POSITIVE_INFINITY]) {
		const stream = bytes([]);
		await expect(readProxyBody(stream, limit, signal())).rejects.toBeInstanceOf(RangeError);
		expect(stream.locked).toBe(false);
	}
	await expect(readProxyBody(bytes([]), 1, signal(), 0)).rejects.toBeInstanceOf(RangeError);
});

test('an existing reader is not replaced or cancelled', async () => {
	const stream = bytes([new Uint8Array([1])]);
	const existing = stream.getReader();
	await expect(readProxyBody(stream, 1, signal())).rejects.toBeInstanceOf(TypeError);
	expect(stream.locked).toBe(true);
	expect((await existing.read()).value).toEqual(new Uint8Array([1]));
	existing.releaseLock();
});

test('connection-specific headers are removed without changing end-to-end controls', () => {
	const original = new Headers({
		Connection: 'keep-alive, X-Local-Only, TE',
		'X-Local-Only': 'hop value',
		'Keep-Alive': 'timeout=5',
		'Proxy-Connection': 'keep-alive',
		'Proxy-Authorization': 'synthetic proxy credentials',
		TE: 'trailers',
		Trailer: 'x-checksum',
		'Transfer-Encoding': 'chunked',
		Upgrade: 'websocket',
		Authorization: 'synthetic application credentials',
		Cookie: 'kredit_session=synthetic',
		'X-CSRF-Token': 'synthetic csrf',
		'Idempotency-Key': 'synthetic retry identity',
		'Content-Type': 'application/json'
	});
	const result = proxyHeaders(original);
	for (const name of ['connection', 'x-local-only', 'keep-alive', 'proxy-connection', 'proxy-authorization', 'te', 'trailer', 'transfer-encoding', 'upgrade']) {
		expect(result.has(name), name).toBe(false);
	}
	for (const name of ['authorization', 'cookie', 'x-csrf-token', 'idempotency-key', 'content-type']) {
		expect(result.get(name), name).toBe(original.get(name));
	}
	expect(original.get('x-local-only')).toBe('hop value');
	expect(new ProxyBodyError(413, 'synthetic').status).toBe(413);
});

test('unauthenticated private redirects carry the same security headers', async ({ request }) => {
	const response = await request.get('/workspace/today', { maxRedirects: 0 });
	expect(response.status()).toBe(303);
	expect(response.headers()['cache-control']).toBe('private, no-store');
	expect(response.headers()['x-content-type-options']).toBe('nosniff');
	expect(response.headers()['x-frame-options']).toBe('DENY');
	expect(response.headers()['location']).toContain('/signin?next=');
});
