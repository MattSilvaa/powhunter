import { test, expect, describe, afterEach } from 'bun:test'
import { ApiError, apiRequest, retryOnlyTransport } from './apiClient.ts'

const fetchStub = globalThis as unknown as { fetch: unknown }
const originalFetch = globalThis.fetch

afterEach(() => {
	fetchStub.fetch = originalFetch
})

const respondWith = (body: string, status: number) => {
	fetchStub.fetch = () => Promise.resolve(new Response(body, { status }))
}

describe('apiRequest', () => {
	test('parses a JSON response', async () => {
		respondWith(JSON.stringify([{ name: 'Vail' }]), 200)

		const result = await apiRequest<{ name: string }[]>('/api/resorts')

		expect(result).toEqual([{ name: 'Vail' }])
	})

	test('maps a server error code to a readable message', async () => {
		respondWith(
			JSON.stringify({ error: 'DUPLICATE_ALERT', message: 'raw' }),
			409
		)

		const error = (await apiRequest('/api/alerts', { method: 'POST' }).catch(
			(err: unknown) => err
		)) as ApiError

		expect(error).toBeInstanceOf(ApiError)
		expect(error.status).toBe(409)
		expect(error.message).toContain('already have an alert')
	})

	// The reverse proxy can return an HTML error page; parsing it blindly used to
	// surface a SyntaxError to the user instead of the real failure.
	test('falls back to a status message when the body is not JSON', async () => {
		respondWith('<html><body>502 Bad Gateway</body></html>', 502)

		const error = (await apiRequest('/api/resorts').catch(
			(err: unknown) => err
		)) as ApiError

		expect(error).toBeInstanceOf(ApiError)
		expect(error.message).toContain('our end')
	})

	test('reports a rate limited request clearly', async () => {
		respondWith(JSON.stringify({ error: 'RATE_LIMITED', message: 'slow' }), 429)

		const error = (await apiRequest('/api/contact', { method: 'POST' }).catch(
			(err: unknown) => err
		)) as ApiError

		expect(error.message).toContain('Too many requests')
	})

	test('turns a network failure into a readable error', async () => {
		fetchStub.fetch = () => Promise.reject(new TypeError('failed to fetch'))

		const error = (await apiRequest('/api/resorts').catch(
			(err: unknown) => err
		)) as ApiError

		expect(error.status).toBe(0)
		expect(error.isRetryable).toBe(true)
	})
})

describe('retryOnlyTransport', () => {
	test('does not retry a request the server rejected on its merits', () => {
		const shouldRetry = retryOnlyTransport(2)

		expect(shouldRetry(0, new ApiError('conflict', 409, 'DUPLICATE_ALERT'))).toBe(
			false
		)
		expect(shouldRetry(0, new ApiError('bad', 400, 'VALIDATION_ERROR'))).toBe(
			false
		)
	})

	test('retries transport and server failures up to the limit', () => {
		const shouldRetry = retryOnlyTransport(2)

		expect(shouldRetry(0, new ApiError('offline', 0, 'NETWORK'))).toBe(true)
		expect(shouldRetry(1, new ApiError('boom', 500, 'INTERNAL_ERROR'))).toBe(true)
		expect(shouldRetry(2, new ApiError('boom', 500, 'INTERNAL_ERROR'))).toBe(false)
	})
})
