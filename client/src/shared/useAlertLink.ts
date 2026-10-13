import { useQuery } from '@tanstack/react-query'
import { apiRequest, retryOnlyTransport } from './apiClient.ts'

const ALERT_LINK_RETRIES = 2

// What the server shows for an alert link: the storm, never the person it was
// sent to.
export type AlertLinkDetails = {
	resortUuid: string
	resortName: string
	snowAmount: number
	forecastDate: string
	reportUrl: string
}

const fetchAlertLink = (code: string): Promise<AlertLinkDetails> =>
	apiRequest<AlertLinkDetails>(`/api/alert-links/${encodeURIComponent(code)}`)

export function useAlertLink(code: string) {
	const { data, isLoading, error } = useQuery<AlertLinkDetails>({
		queryKey: ['alert-link', code],
		queryFn: () => fetchAlertLink(code),
		retry: retryOnlyTransport(ALERT_LINK_RETRIES),
		enabled: code !== '',
	})

	return { alert: data, loading: isLoading, error }
}

const DAY_MS = 24 * 60 * 60 * 1000

// The forecast date is a calendar date ("2026-10-16"). Parsing it with
// new Date() would read it as UTC midnight and show the previous day to anyone
// west of UTC, so it is built from its parts in local time instead.
const localDate = (isoDate: string): Date => {
	const [year, month, day] = isoDate.split('-').map(Number)
	return new Date(year, month - 1, day)
}

// forecastDay phrases a forecast date the way a skier would say it: "today",
// "tomorrow", or "Thursday, Oct 16".
export const forecastDay = (
	isoDate: string,
	now: Date = new Date()
): string => {
	const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
	const target = localDate(isoDate)
	const days = Math.round((target.getTime() - today.getTime()) / DAY_MS)

	if (days === 0) {
		return 'today'
	}

	if (days === 1) {
		return 'tomorrow'
	}

	return target.toLocaleDateString('en-US', {
		weekday: 'long',
		month: 'short',
		day: 'numeric',
	})
}

// formatInches drops a trailing ".0" so 14 reads "14″" and 8.5 reads "8.5″".
export const formatInches = (amount: number): string =>
	`${Number(amount.toFixed(1))}″`

// signupPath opens signup with the resort already picked, carrying the alert's
// code so the signup can be counted as coming from a share.
export const signupPath = (alert: AlertLinkDetails, code: string): string => {
	const params = new URLSearchParams({ resort: alert.resortUuid, ref: code })
	return `/signup?${params.toString()}`
}

// shareMessage is what lands in the group chat.
export const shareMessage = (
	alert: AlertLinkDetails,
	link: string,
	now: Date = new Date()
): string => {
	const day = forecastDay(alert.forecastDate, now)
	const when = day === 'today' || day === 'tomorrow' ? day : `on ${day}`

	return (
		`${alert.resortName} is expecting ${formatInches(alert.snowAmount)} of snow ${when}. ` +
		`Get free powder alerts: ${link}`
	)
}
