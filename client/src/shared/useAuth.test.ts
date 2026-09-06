import { test, expect, describe, afterEach } from 'bun:test'
import React from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { ApiError, apiRequest, retryOnlyTransport } from './apiClient.ts'
import { useCompleteLogin } from './useAuth.ts'

const fetchStub = globalThis as unknown as { fetch: unknown }
const originalFetch = globalThis.fetch

afterEach(() => {
	fetchStub.fetch = originalFetch
})

type Call = { url: string; init: RequestInit }

const captureRequests = (body: string, status: number): Call[] => {
	const calls: Call[] = []

	fetchStub.fetch = (url: string, init: RequestInit) => {
		calls.push({ url, init })
		return Promise.resolve(new Response(body, { status }))
	}

	return calls
}

describe('authenticated requests', () => {
	// The session lives in an HttpOnly cookie, so a request that omits
	// credentials silently reads as signed out.
	test('sends cookies with every call', async () => {
		const calls = captureRequests(JSON.stringify({ email: 'a@b.com' }), 200)

		await apiRequest('/api/auth/me')

		expect(calls).toHaveLength(1)
		expect(calls[0].init.credentials).toBe('include')
	})

	test('surfaces a 401 as an UNAUTHENTICATED ApiError', async () => {
		captureRequests(
			JSON.stringify({ error: 'UNAUTHENTICATED', message: 'nope' }),
			401
		)

		const error = (await apiRequest('/api/user/alerts').catch(
			(err: unknown) => err
		)) as ApiError

		expect(error).toBeInstanceOf(ApiError)
		expect(error.status).toBe(401)
		expect(error.code).toBe('UNAUTHENTICATED')
		expect(error.message).toContain('sign in')
	})

	// Retrying a 401 cannot succeed: the cookie will not have appeared, and each
	// attempt just delays showing the sign-in screen.
	test('does not retry an unauthenticated response', () => {
		const shouldRetry = retryOnlyTransport(3)
		const unauthorized = new ApiError('nope', 401, 'UNAUTHENTICATED')

		expect(shouldRetry(0, unauthorized)).toBe(false)
	})

	test('reports a spent sign-in link in plain language', async () => {
		captureRequests(
			JSON.stringify({ error: 'INVALID_TOKEN', message: 'raw' }),
			401
		)

		const error = (await apiRequest('/api/auth/callback', {
			method: 'POST',
			body: { token: 'used' },
		}).catch((err: unknown) => err)) as ApiError

		expect(error.message).toContain('expired or has already been used')
	})

	test('posts the token when completing a login', async () => {
		const calls = captureRequests('', 200)

		await apiRequest('/api/auth/callback', {
			method: 'POST',
			body: { token: 'abc123' },
		})

		expect(calls[0].init.method).toBe('POST')
		expect(calls[0].init.body).toBe(JSON.stringify({ token: 'abc123' }))
	})
})

describe('completing a sign-in link', () => {
	const renderCompleteLogin = () => {
		const client = new QueryClient({
			defaultOptions: { queries: { retry: false } },
		})

		return renderHook(() => useCompleteLogin(), {
			wrapper: ({ children }) =>
				React.createElement(QueryClientProvider, { client }, children),
		})
	}

	// The callback returning 200 does not mean the browser kept the session
	// cookie. Navigating on that alone lands on a page that reads "signed out"
	// and bounces straight back to the sign-in form, with nothing explaining why.
	test('resolves only once the session reads back as signed in', async () => {
		const calls = captureRequests('', 200)
		fetchStub.fetch = (url: string, init: RequestInit) => {
			calls.push({ url, init })

			if (url.endsWith('/api/auth/me')) {
				return Promise.resolve(
					new Response(JSON.stringify({ email: 'a@b.com' }), { status: 200 })
				)
			}

			return Promise.resolve(new Response('', { status: 200 }))
		}

		const { result } = renderCompleteLogin()
		result.current.completeLogin('token-123')

		await waitFor(() => expect(result.current.loading).toBe(false))

		expect(result.current.error).toBeNull()
		expect(calls.map((call) => new URL(call.url).pathname)).toEqual([
			'/api/auth/callback',
			'/api/auth/me',
		])
	})

	test('reports a session the browser did not keep', async () => {
		fetchStub.fetch = (url: string) =>
			Promise.resolve(
				url.endsWith('/api/auth/me')
					? new Response(JSON.stringify({ error: 'UNAUTHENTICATED' }), {
							status: 401,
						})
					: new Response('', { status: 200 })
			)

		const { result } = renderCompleteLogin()
		result.current.completeLogin('token-123')

		await waitFor(() => expect(result.current.error).not.toBeNull())
		expect(result.current.error).toContain('did not keep the session')
	})
})
