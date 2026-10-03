import { test, expect, describe, afterEach } from 'bun:test'
import React from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { useCreateUserAlerts, useUpdateAlert } from './useManageAlerts.ts'
import { AUTH_QUERY_KEY, AuthUser, useUpdateProfile } from './useAuth.ts'

const fetchStub = globalThis as unknown as { fetch: unknown }
const originalFetch = globalThis.fetch

afterEach(() => {
	fetchStub.fetch = originalFetch
})

type Call = { url: string; init: RequestInit }

const captureRequests = (body: string): Call[] => {
	const calls: Call[] = []

	fetchStub.fetch = (url: string, init: RequestInit) => {
		calls.push({ url, init })
		return Promise.resolve(new Response(body, { status: 200 }))
	}

	return calls
}

const renderWithClient = <T>(hook: () => T, client = new QueryClient()) =>
	renderHook(hook, {
		wrapper: ({ children }) =>
			React.createElement(QueryClientProvider, { client }, children),
	})

describe('account management requests', () => {
	// The session identifies the account, so adding alerts must not send an
	// email or phone the server would have to trust.
	test('adds alerts without sending contact details', async () => {
		const calls = captureRequests('')
		const { result } = renderWithClient(() => useCreateUserAlerts())

		result.current.mutate({
			resortsUuids: ['r1'],
			notificationDays: 3,
			minSnowAmount: 6,
		})
		await waitFor(() => expect(result.current.isSuccess).toBe(true))

		expect(new URL(calls[0].url).pathname).toBe('/api/user/alerts')
		expect(calls[0].init.method).toBe('POST')
		expect(JSON.parse(calls[0].init.body as string)).toEqual({
			resortsUuids: ['r1'],
			notificationDays: 3,
			minSnowAmount: 6,
		})
	})

	test('patches one alert by resort', async () => {
		const calls = captureRequests('{}')
		const { result } = renderWithClient(() => useUpdateAlert())

		result.current.mutate({
			resortUuid: 'r1',
			notificationDays: 5,
			minSnowAmount: 8,
		})
		await waitFor(() => expect(result.current.isSuccess).toBe(true))

		const url = new URL(calls[0].url)
		expect(url.pathname).toBe('/api/user/alerts')
		expect(url.searchParams.get('resort_uuid')).toBe('r1')
		expect(calls[0].init.method).toBe('PATCH')
		expect(JSON.parse(calls[0].init.body as string)).toEqual({
			notificationDays: 5,
			minSnowAmount: 8,
		})
	})

	test('stores the updated account after changing the phone', async () => {
		const updated: AuthUser = {
			email: 'a@b.com',
			phone: '+15551234567',
			emailVerified: true,
		}
		const calls = captureRequests(JSON.stringify(updated))
		const client = new QueryClient()
		const { result } = renderWithClient(() => useUpdateProfile(), client)

		result.current.updateProfile('555 123 4567')
		await waitFor(() =>
			expect(client.getQueryData<AuthUser>(AUTH_QUERY_KEY)).toEqual(updated)
		)

		expect(new URL(calls[0].url).pathname).toBe('/api/user/profile')
		expect(calls[0].init.method).toBe('PATCH')
	})
})
