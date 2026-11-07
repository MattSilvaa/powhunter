import { test, expect, describe, afterEach } from 'bun:test'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import ResortHistoryPage from './resortHistory'
import { SnowHistory, UserAlert } from '../shared/types.ts'

const RESORT_UUID = '11111111-1111-4111-8111-111111111111'

const history: SnowHistory = {
	resortUuid: RESORT_UUID,
	resortName: 'Vail',
	threshold: 6,
	averageSeasonInches: 312.4,
	averageDaysAtOrAbove: 9.2,
	monthlyAverages: [
		{ month: 12, inches: 60 },
		{ month: 1, inches: 72.5 },
	],
	seasons: [
		{
			label: '2025–26',
			startYear: 2025,
			partial: true,
			totalInches: 40,
			daysAtOrAbove: 2,
		},
		{
			label: '2024–25',
			startYear: 2024,
			partial: false,
			totalInches: 330,
			daysAtOrAbove: 11,
		},
	],
}

const alert = {
	id: 1,
	resort_uuid: RESORT_UUID,
	resort_name: 'Vail',
	min_snow_amount: 6,
} as UserAlert

const originalFetch = global.fetch

// stubApi answers each endpoint the page calls, and records the history URL.
const stubApi = (signedIn: boolean): string[] => {
	const historyUrls: string[] = []

	global.fetch = (async (input: RequestInfo | URL) => {
		const url = input.toString()
		const json = (status: number, body: unknown): Response =>
			new Response(JSON.stringify(body), { status })

		if (url.endsWith('/api/auth/me')) {
			return signedIn
				? json(200, { email: 'a@b.co', phone: null })
				: json(401, { error: 'UNAUTHENTICATED' })
		}

		if (url.endsWith('/api/user/alerts')) {
			return json(200, [alert])
		}

		historyUrls.push(url)
		return json(200, signedIn ? history : { ...history, threshold: null })
	}) as typeof fetch

	return historyUrls
}

const renderPage = () =>
	render(
		<QueryClientProvider
			client={
				new QueryClient({ defaultOptions: { queries: { retry: false } } })
			}
		>
			<MemoryRouter initialEntries={[`/resorts/${RESORT_UUID}/history`]}>
				<Routes>
					<Route
						path="/resorts/:resortUuid/history"
						element={<ResortHistoryPage />}
					/>
				</Routes>
			</MemoryRouter>
		</QueryClientProvider>
	)

describe('ResortHistoryPage', () => {
	afterEach(() => {
		global.fetch = originalFetch
	})

	test('shows seasons and monthly averages', async () => {
		stubApi(false)
		renderPage()

		expect(
			await screen.findByRole('heading', { level: 1, name: 'Vail' })
		).toBeTruthy()
		expect(screen.getByText('in an average season')).toBeTruthy()
		expect(screen.getByText('Jan')).toBeTruthy()
		expect(screen.getByText('73″')).toBeTruthy()
		expect(screen.getByText('so far')).toBeTruthy()
		expect(screen.queryByText(/your alert threshold/)).toBeNull()
	})

	test("uses the signed-in user's alert threshold", async () => {
		const historyUrls = stubApi(true)
		renderPage()

		expect(await screen.findByText(/your alert threshold/)).toBeTruthy()
		expect(screen.getByText('11 days ≥ 6″')).toBeTruthy()
		expect(historyUrls).toEqual([
			expect.stringContaining(
				`/api/resorts/${RESORT_UUID}/snow-history?threshold=6`
			),
		])
	})
})
