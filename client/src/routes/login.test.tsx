import { afterEach, describe, expect, test } from 'bun:test'
import { render, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import LoginPage from './login'

const fetchStub = globalThis as unknown as { fetch: unknown }
const originalFetch = globalThis.fetch

afterEach(() => {
	fetchStub.fetch = originalFetch
})

describe('LoginPage', () => {
	// A sign-in token is a credential. Left in the URL after a failed sign-in, it
	// sits in browser history for anyone with the device to find.
	test('removes the token from the address bar', async () => {
		fetchStub.fetch = () =>
			Promise.resolve(
				new Response(
					JSON.stringify({ error: 'INVALID_TOKEN', message: 'expired' }),
					{ status: 401 }
				)
			)
		;(
			window as unknown as { happyDOM: { setURL: (url: string) => void } }
		).happyDOM.setURL('http://localhost:3000/login?token=secret-token')
		expect(window.location.search).toBe('?token=secret-token')

		render(
			<QueryClientProvider client={new QueryClient()}>
				<MemoryRouter initialEntries={['/login?token=secret-token']}>
					<LoginPage />
				</MemoryRouter>
			</QueryClientProvider>
		)

		await waitFor(() => {
			expect(window.location.search).toBe('')
		})
		expect(window.location.pathname).toBe('/login')
	})
})
