import { afterEach, describe, expect, test } from 'bun:test'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Home from './home'
import { SnowReport } from '../shared/snowReport.ts'

const fetchStub = globalThis as unknown as { fetch: unknown }
const originalFetch = globalThis.fetch

afterEach(() => {
	fetchStub.fetch = originalFetch
})

// A Thursday, so "by Sunday" below is three days out.
const now = new Date('2026-01-15T15:30:00Z')

const days = (snow: number[]) =>
	snow.map((inches, i) => ({
		date: `2026-01-${String(15 + i).padStart(2, '0')}`,
		snow_in: inches,
	}))

const LIVE: SnowReport = {
	mode: 'live',
	updated_at: '2026-01-15T12:00:00Z',
	live: {
		days: 10,
		resorts: [
			{
				name: 'Palisades Tahoe',
				total_in: 18,
				through: '2026-01-18',
				daily: days([0, 6, 0, 12, 0, 0, 0, 0, 0, 0]),
			},
			{
				name: 'Alta',
				total_in: 4,
				through: '2026-01-16',
				daily: days([0, 4, 0, 0, 0, 0, 0, 0, 0, 0]),
			},
		],
	},
	recap: null,
}

const RECAP: SnowReport = {
	mode: 'recap',
	updated_at: '2026-11-08T12:00:00Z',
	live: null,
	recap: {
		season: '2025-26',
		start: '2025-11-01',
		end: '2026-04-30',
		resorts: [
			{
				name: 'Alta',
				total_in: 402,
				days_6in_plus: 21,
				biggest_storm_in: 41,
				biggest_storm_start: '2025-12-20',
				biggest_storm_end: '2025-12-22',
			},
		],
	},
}

const respondWith = (status: number, body: unknown) => {
	fetchStub.fetch = () =>
		Promise.resolve(new Response(JSON.stringify(body), { status }))
}

const renderHome = () =>
	render(
		<MemoryRouter>
			<QueryClientProvider
				client={
					new QueryClient({ defaultOptions: { queries: { retry: false } } })
				}
			>
				<Home now={now} />
			</QueryClientProvider>
		</MemoryRouter>
	)

describe('Home', () => {
	test('explains the product and links to signup before any data loads', () => {
		respondWith(200, LIVE)
		renderHome()

		expect(
			screen.getByRole('heading', { level: 1, name: /snow alerts by text/i })
		).toBeTruthy()
		expect(
			screen.getByRole('link', { name: 'Get snow alerts' }).getAttribute('href')
		).toBe('/signup')
		expect(screen.getByLabelText('Loading snow report')).toBeTruthy()
	})

	test('in season, ranks resorts by snow coming and calls out the biggest storm', async () => {
		respondWith(200, LIVE)
		renderHome()

		const board = await screen.findByRole('region', { name: 'Snow report' })
		expect(within(board).getByText('by Sunday')).toBeTruthy()

		const rows = within(board).getAllByRole('listitem')
		expect(rows.map((row) => row.textContent)).toEqual([
			expect.stringContaining('Palisades Tahoe'),
			expect.stringContaining('Alta'),
		])
		expect(within(rows[0]).getByText('18″')).toBeTruthy()

		expect(
			within(rows[1]).getByRole('link', { name: 'Alta' }).getAttribute('href')
		).toBe('/resorts/alta')
		expect(
			within(rows[1])
				.getByRole('link', { name: 'Get Alta alerts' })
				.getAttribute('href')
		).toBe('/signup?resort=alta')
		// Both the storm callout and its row offer alerts for the resort.
		const stormLinks = within(board).getAllByRole('link', {
			name: 'Get Palisades Tahoe alerts',
		})
		expect(stormLinks.map((link) => link.getAttribute('href'))).toEqual([
			'/signup?resort=palisades-tahoe',
			'/signup?resort=palisades-tahoe',
		])
	})

	test('off season, shows last season and asks for the first storm', async () => {
		respondWith(200, RECAP)
		renderHome()

		const recap = await screen.findByRole('region', { name: 'Season recap' })
		expect(within(recap).getByText(/2025–26 season/)).toBeTruthy()
		expect(within(recap).getByText('402″')).toBeTruthy()
		expect(within(recap).getByText('21')).toBeTruthy()
		expect(
			within(recap).getAllByText('41″ · Dec 20–22').length
		).toBeGreaterThan(0)
		expect(within(recap).getByText(/modeled snowfall/i)).toBeTruthy()

		expect(
			screen
				.getByRole('link', {
					name: 'Get a text when the first storm of the season shows up',
				})
				.getAttribute('href')
		).toBe('/signup')
	})

	// A rate-limited request is not retried, so the error shows at once.
	test('still offers signup when the report fails to load', async () => {
		respondWith(429, { error: 'RATE_LIMITED', message: 'slow down' })
		renderHome()

		expect(await screen.findByText("The snow report didn't load")).toBeTruthy()
		expect(screen.getByRole('button', { name: 'Try again' })).toBeTruthy()
		expect(
			screen.getByRole('link', { name: 'Get snow alerts' }).getAttribute('href')
		).toBe('/signup')
	})
})
