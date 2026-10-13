import { test, expect, describe, afterEach } from 'bun:test'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router'
import AlertLinkPage from './alertLink'

const fetchStub = globalThis as unknown as { fetch: unknown }
const originalFetch = globalThis.fetch
const nav = navigator as unknown as { share?: unknown }
const originalShare = nav.share

afterEach(() => {
	fetchStub.fetch = originalFetch
	nav.share = originalShare
})

const respondWith = (status: number, body: unknown) => {
	fetchStub.fetch = () =>
		Promise.resolve(new Response(JSON.stringify(body), { status }))
}

const renderPage = (code: string) =>
	render(
		<QueryClientProvider
			client={
				new QueryClient({ defaultOptions: { queries: { retry: false } } })
			}
		>
			<MemoryRouter initialEntries={[`/alert/${code}`]}>
				<Routes>
					<Route path="/alert/:code" element={<AlertLinkPage />} />
				</Routes>
			</MemoryRouter>
		</QueryClientProvider>
	)

const alta = {
	resortUuid: '11111111-1111-4111-8111-111111111111',
	resortName: 'Alta',
	snowAmount: 14,
	forecastDate: '2030-01-17',
	reportUrl: 'https://www.alta.com/conditions',
}

describe('AlertLinkPage', () => {
	test('shows the storm and links to the mountain report', async () => {
		respondWith(200, alta)
		renderPage('Ab12Cd34')

		expect(
			await screen.findByRole('heading', { level: 1, name: 'Alta' })
		).toBeTruthy()
		expect(screen.getByText('14″')).toBeTruthy()
		expect(
			screen
				.getByRole('link', { name: /view mountain report/i })
				.getAttribute('href')
		).toBe(alta.reportUrl)
	})

	test('shares a message with a signup link for the same resort', async () => {
		respondWith(200, alta)
		const shared: string[] = []
		nav.share = (data: { text: string }) => {
			shared.push(data.text)
			return Promise.resolve()
		}

		renderPage('Ab12Cd34')
		fireEvent.click(
			await screen.findByRole('button', { name: /text your ski crew/i })
		)

		await waitFor(() => expect(shared).toHaveLength(1))
		expect(shared[0]).toContain('Alta is expecting 14″ of snow')
		expect(shared[0]).toContain(
			'/signup?resort=11111111-1111-4111-8111-111111111111&ref=Ab12Cd34'
		)
	})

	test('explains an unknown link instead of showing an error page', async () => {
		respondWith(404, {
			error: 'ALERT_LINK_NOT_FOUND',
			message: 'That alert link does not exist',
		})
		renderPage('Zz99Yy88')

		expect(
			await screen.findByRole('heading', { name: /couldn.t find that alert/i })
		).toBeTruthy()
		expect(screen.getByText(/mistyped or out of date/i)).toBeTruthy()
	})
})
