// These limits protect the Node proxy before any upstream request exists.
// Never trust Content-Length, and never keep one retained object per chunk.
export class ProxyBodyError extends Error {
	readonly status: 400 | 408 | 413;

	constructor(status: 400 | 408 | 413, message: string) {
		super(message);
		this.name = 'ProxyBodyError';
		this.status = status;
	}
}

function readWithSignal(reader: ReadableStreamDefaultReader<Uint8Array>, signal: AbortSignal): Promise<ReadableStreamReadResult<Uint8Array>> {
	return new Promise((resolve, reject) => {
		const cleanup = () => signal.removeEventListener('abort', aborted);
		const aborted = () => {
			cleanup();
			reject(signal.reason ?? new DOMException('Request aborted', 'AbortError'));
		};
		if (signal.aborted) {
			aborted();
			return;
		}
		signal.addEventListener('abort', aborted, { once: true });
		// One listener per pending read, removed on every settlement. Repeatedly
		// racing a shared never-settled abort promise would retain chunk callbacks.
		reader.read().then(
			value => { cleanup(); resolve(value); },
			error => { cleanup(); reject(error); }
		);
	});
}

export async function readProxyBody(
	stream: ReadableStream<Uint8Array>,
	maxBytes: number,
	requestSignal: AbortSignal,
	timeoutMs = 60_000
): Promise<Uint8Array<ArrayBuffer>> {
	if (!Number.isSafeInteger(maxBytes) || maxBytes < 0 || !Number.isSafeInteger(timeoutMs) || timeoutMs <= 0 || timeoutMs > 2_147_483_647) {
		throw new RangeError('Invalid proxy body bounds');
	}
	const deadline = new AbortController();
	const expired = () => new ProxyBodyError(408, 'The request body took too long to arrive.');
	const expiresAt = performance.now() + timeoutMs;
	const signal = AbortSignal.any([requestSignal, deadline.signal]);
	const timer = setTimeout(() => deadline.abort(expired()), timeoutMs);
	let reader: ReadableStreamDefaultReader<Uint8Array> | undefined;
	let complete = false;
	let buffer: Uint8Array<ArrayBuffer> = new Uint8Array(0);
	let size = 0;
	let emptyChunks = 0;
	try {
		reader = stream.getReader();
		while (true) {
			signal.throwIfAborted();
			// Also enforce elapsed time when immediately-ready chunks keep the
			// microtask queue busy and delay the timer callback itself.
			if (performance.now() >= expiresAt) throw expired();
			const { value, done } = await readWithSignal(reader, signal);
			if (done) {
				complete = true;
				return buffer.subarray(0, size);
			}
			if (!(value instanceof Uint8Array)) throw new ProxyBodyError(400, 'Invalid request body.');
			if (value.byteLength === 0) {
				if (++emptyChunks > 32) throw new ProxyBodyError(400, 'The request body is not making progress.');
				continue;
			}
			emptyChunks = 0;
			if (value.byteLength > maxBytes - size) throw new ProxyBodyError(413, 'The request body exceeds the allowed size.');
			const needed = size + value.byteLength;
			if (needed > buffer.byteLength) {
				const capacity = Math.min(maxBytes, Math.max(needed, 16_384, buffer.byteLength * 2));
				const grown = new Uint8Array(capacity);
				grown.set(buffer.subarray(0, size));
				buffer = grown;
			}
			buffer.set(value, size);
			size = needed;
		}
	} finally {
		clearTimeout(timer);
		if (reader) {
			if (!complete) {
				// A misbehaving producer may never settle cancel(). Preserve the
				// size/deadline/read error without waiting indefinitely for cleanup.
				void reader.cancel().catch(() => {});
			}
			reader.releaseLock();
		}
	}
}

// RFC 9110 section 7.6.1: connection-specific fields are not end-to-end data.
export function proxyHeaders(source: Headers): Headers {
	const headers = new Headers(source);
	for (const name of (headers.get('connection') ?? '').split(',')) {
		const token = name.trim();
		if (/^[!#$%&'*+.^_`|~0-9A-Za-z-]+$/.test(token)) headers.delete(token);
	}
	for (const name of ['connection', 'proxy-connection', 'keep-alive', 'proxy-authenticate', 'proxy-authorization', 'te', 'trailer', 'transfer-encoding', 'upgrade']) {
		headers.delete(name);
	}
	return headers;
}
