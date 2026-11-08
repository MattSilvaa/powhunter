import { afterEach, describe, expect, test } from 'bun:test'
import {
	fireEvent,
	render,
	screen,
	waitFor,
	within,
} from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Home from './home'
import { LiveResort, RecapResort, SnowReport } from '../shared/snowReport.ts'

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
		resort_count: 2,
		resorts: [
			{
				rank: 1,
				name: 'Palisades Tahoe',
				total_in: 18,
				through: '2026-01-18',
				daily: days([0, 6, 0, 12, 0, 0, 0, 0, 0, 0]),
			},
			{
				rank: 2,
				name: 'Alta',
				total_in: 4,
				through: '2026-01-16',
				daily: days([0, 4, 0, 0, 0, 0, 0, 0, 0, 0]),
			},
		],
	},
	recap: null,
}

const RECAP_ALTA: RecapResort = {
	rank: 1,
	name: 'Alta',
	total_in: 402,
	days_6in_plus: 21,
	biggest_storm_in: 41,
	biggest_storm_start: '2025-12-20',
	biggest_storm_end: '2025-12-22',
}

const RECAP: SnowReport = {
	mode: 'recap',
	updated_at: '2026-11-08T12:00:00Z',
	live: null,
	recap: {
		season: '2025-26',
		start: '2025-11-01',
		end: '2026-04-30',
		resort_count: 1,
		resorts: [RECAP_ALTA],
		biggest_storm: RECAP_ALTA,
	},
}

// A board with more resorts than the home page shows, so the rest are found
// through search. Alta sits at 14th.
const peak = (rank: number): LiveResort => ({
	rank,
	name: `Peak ${rank}`,
	total_in: 30 - rank,
	through: '2026-01-17',
	daily: days([0, 0, 30 - rank, 0, 0, 0, 0, 0, 0, 0]),
})

const LIVE_TOP_5: SnowReport = {
	...LIVE,
	live: { days: 10, resort_count: 28, resorts: [1, 2, 3, 4, 5].map(peak) },
}

const ALTA_14: LiveResort = { ...peak(14), name: 'Alta', total_in: 9 }

const respondWith = (status: number, body: unknown) => {
	fetchStub.fetch = () =>
		Promise.resolve(new Response(JSON.stringify(body), { status }))
}

// Types into a field the way a person does: focused first. MUI's
// Autocomplete clears text typed into an unfocused field.
const typeInto = (input: HTMLElement, text: string) => {
	input.focus()
	fireEvent.change(input, { target: { value: text } })
}

// Answers the report and the search separately, recording search queries.
const respondWithReportAndSearch = (
	report: SnowReport,
	search: unknown
): string[] => {
	const searches: string[] = []
	fetchStub.fetch = (url: string) => {
		const { pathname, searchParams } = new URL(url)
		if (pathname === '/api/snow-report/search') {
			searches.push(searchParams.get('q') ?? '')
			return Promise.resolve(new Response(JSON.stringify(search)))
		}
		return Promise.resolve(new Response(JSON.stringify(report)))
	}
	return searches
}

const renderHome = (
	client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
) =>
	render(
		<MemoryRouter>
			<QueryClientProvider client={client}>
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

	// React Query keeps the last report when a refetch fails, such as on
	// returning to a tab later. The board stays, with no error over it.
	test('keeps the board when a background refetch fails', async () => {
		const client = new QueryClient({
			defaultOptions: { queries: { retry: false } },
		})
		respondWith(200, LIVE)
		renderHome(client)
		await screen.findByRole('region', { name: 'Snow report' })

		respondWith(429, { error: 'RATE_LIMITED', message: 'slow down' })
		await client.refetchQueries({ queryKey: ['snow-report'] })
		await waitFor(() =>
			expect(client.getQueryState(['snow-report'])?.status).toBe('error')
		)
		// Let React render the error state before asserting it is not shown.
		await new Promise((resolve) => setTimeout(resolve, 50))

		expect(screen.getByRole('region', { name: 'Snow report' })).toBeTruthy()
		expect(screen.queryByText("The snow report didn't load")).toBeNull()
	})

	test('every resort fits, so there is no search box', async () => {
		respondWith(200, LIVE)
		renderHome()

		await screen.findByRole('region', { name: 'Snow report' })
		expect(screen.queryByLabelText('Find your resort')).toBeNull()
	})

	// The board shows the top 5 the API sent; the rest are a search away.
	test('finds a resort outside the top 5', async () => {
		const searches = respondWithReportAndSearch(LIVE_TOP_5, {
			mode: 'live',
			live: [ALTA_14],
			recap: [],
		})
		renderHome()

		const board = await screen.findByRole('region', { name: 'Snow report' })
		expect(within(board).getAllByRole('listitem')).toHaveLength(5)

		typeInto(within(board).getByLabelText('Find your resort'), 'alt')
		fireEvent.click(await screen.findByRole('option', { name: 'Alta' }))

		// Showing the picked name in the box is not a new search.
		await new Promise((resolve) => setTimeout(resolve, 400))
		expect(searches).toEqual(['alt'])
		const rows = within(board).getAllByRole('listitem')
		expect(rows).toHaveLength(6)
		const alta = rows[5]
		expect(alta.textContent).toContain('14')
		expect(within(alta).getByText('9″')).toBeTruthy()
		expect(
			within(alta).getByRole('link', { name: 'Alta' }).getAttribute('href')
		).toBe('/resorts/alta')
		expect(
			within(alta)
				.getByRole('link', { name: 'Get Alta alerts' })
				.getAttribute('href')
		).toBe('/signup?resort=alta')
	})

	test('finds a resort in the season recap', async () => {
		const vail: RecapResort = {
			...RECAP_ALTA,
			rank: 20,
			name: 'Vail',
			total_in: 188,
		}
		respondWithReportAndSearch(
			{ ...RECAP, recap: { ...RECAP.recap!, resort_count: 28 } },
			{ mode: 'recap', live: [], recap: [vail] }
		)
		renderHome()

		const recap = await screen.findByRole('region', { name: 'Season recap' })
		typeInto(within(recap).getByLabelText('Find your resort'), 'vail')
		fireEvent.click(await screen.findByRole('option', { name: 'Vail' }))

		expect(within(recap).getByText('188″')).toBeTruthy()
	})

	// While typing settles, the menu must not claim there are no matches.
	test('says it is searching while the search is pending', async () => {
		respondWithReportAndSearch(LIVE_TOP_5, {
			mode: 'live',
			live: [ALTA_14],
			recap: [],
		})
		renderHome()

		const board = await screen.findByRole('region', { name: 'Snow report' })
		typeInto(within(board).getByLabelText('Find your resort'), 'al')

		expect(screen.getByText('Searching…')).toBeTruthy()
		expect(screen.queryByText('No resort matches')).toBeNull()
		expect(await screen.findByRole('option', { name: 'Alta' })).toBeTruthy()
	})
})
