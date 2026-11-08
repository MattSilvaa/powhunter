import { useQuery, UseQueryResult } from '@tanstack/react-query'
import { apiRequest, retryOnlyTransport } from './apiClient.ts'
import { CATALOG } from './catalog.ts'
import { resortSlug } from './resortSlug.ts'

// The shape of GET /api/snow-report. Dates are UTC calendar days
// ("2026-01-15"); only the block for the current mode is filled in.
export type SnowReportDay = {
	date: string
	snow_in: number | null
}

export type LiveResort = {
	name: string
	total_in: number
	through: string | null
	daily: SnowReportDay[]
}

export type RecapResort = {
	name: string
	total_in: number
	days_6in_plus: number
	biggest_storm_in: number
	biggest_storm_start: string | null
	biggest_storm_end: string | null
}

export type SnowReport = {
	mode: 'live' | 'recap'
	updated_at: string | null
	live: { days: number; resorts: LiveResort[] } | null
	recap: {
		season: string
		start: string
		end: string
		resorts: RecapResort[]
	} | null
}

const SNOW_REPORT_RETRIES = 2
// The server rebuilds the report at most every five minutes and the forecast
// itself changes twice a day, so refetching more often gains nothing.
const SNOW_REPORT_STALE_TIME_MS = 5 * 60 * 1000

export function useSnowReport(): UseQueryResult<SnowReport> {
	return useQuery<SnowReport>({
		queryKey: ['snow-report'],
		queryFn: () => apiRequest<SnowReport>('/api/snow-report'),
		staleTime: SNOW_REPORT_STALE_TIME_MS,
		retry: retryOnlyTransport(SNOW_REPORT_RETRIES),
	})
}

const MONTHS = [
	'Jan',
	'Feb',
	'Mar',
	'Apr',
	'May',
	'Jun',
	'Jul',
	'Aug',
	'Sep',
	'Oct',
	'Nov',
	'Dec',
]
const WEEKDAYS = [
	'Sunday',
	'Monday',
	'Tuesday',
	'Wednesday',
	'Thursday',
	'Friday',
	'Saturday',
]
const MS_PER_DAY = 24 * 60 * 60 * 1000

// Parses "2026-01-15" as that day in UTC, the way the server writes it.
function parseDay(date: string): Date {
	return new Date(`${date}T00:00:00Z`)
}

function startOfUtcDay(now: Date): number {
	return Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate())
}

// Snow amounts read like a weather readout: whole inches from 10″ up, one
// decimal below, and never a trailing ".0".
export function formatInches(inches: number): string {
	const rounded =
		inches >= 10 ? Math.round(inches) : Math.round(inches * 10) / 10
	return `${rounded}″`
}

export function shortDate(date: string): string {
	const day = parseDay(date)
	return `${MONTHS[day.getUTCMonth()]} ${day.getUTCDate()}`
}

export function weekdayInitial(date: string): string {
	return WEEKDAYS[parseDay(date).getUTCDay()][0]
}

// "Jan 3–5", "Dec 30–Jan 1", or "Jan 3" for a one-day storm.
export function dateRange(start: string, end: string): string {
	if (start === end) return shortDate(start)

	const from = parseDay(start)
	const to = parseDay(end)
	if (from.getUTCMonth() === to.getUTCMonth()) {
		return `${shortDate(start)}–${to.getUTCDate()}`
	}

	return `${shortDate(start)}–${shortDate(end)}`
}

// "today", "tomorrow", "Thursday", or "Jan 24" once a weekday name would be
// ambiguous inside a ten-day window.
export function byDay(date: string, now: Date): string {
	const days = Math.round(
		(parseDay(date).getTime() - startOfUtcDay(now)) / MS_PER_DAY
	)
	if (days <= 0) return 'today'
	if (days === 1) return 'tomorrow'
	if (days < 7) return WEEKDAYS[parseDay(date).getUTCDay()]

	return shortDate(date)
}

// "2025-26" reads as "2025–26".
export function seasonLabel(season: string): string {
	return season.replace('-', '–')
}

export function updatedAgo(iso: string, now: Date): string {
	const minutes = Math.max(
		0,
		Math.round((now.getTime() - new Date(iso).getTime()) / 60000)
	)
	if (minutes < 60) return 'Updated just now'

	const hours = Math.round(minutes / 60)
	if (hours < 48) return `Updated ${hours}h ago`

	return `Updated ${Math.round(hours / 24)} days ago`
}

export type ResortRef = { slug: string; region: string }

// The report names resorts as the database does; the catalog adds the slug
// for links and the region. A resort the catalog lacks still gets a slug.
export function resortRef(name: string): ResortRef {
	const entry = CATALOG.find((resort) => resort.name === name)
	return {
		slug: entry?.slug ?? resortSlug(name),
		region: entry?.region ?? '',
	}
}
