import { BASE_SERVER_URL } from './types.ts'

// Requests that never resolve leave the UI stuck on a spinner with no way out,
// so every call carries a deadline.
const REQUEST_TIMEOUT_MS = 15000

export type ApiErrorResponse = {
	error: string
	message: string
}

export class ApiError extends Error {
	readonly status: number
	readonly code: string

	constructor(message: string, status: number, code: string) {
		super(message)
		this.name = 'ApiError'
		this.status = status
		this.code = code
	}

	// A 4xx is the caller's fault and will fail again identically, so retrying
	// only delays the error the user needs to see.
	get isRetryable(): boolean {
		return this.status === 0 || this.status >= 500
	}
}

const ERROR_MESSAGES: Record<string, string> = {
	DUPLICATE_ALERT:
		'You already have an alert for this resort. Try selecting a different resort or managing your existing alerts.',
	DUPLICATE_ENTRY: 'This alert already exists.',
	MISSING_EMAIL: 'Please enter a valid email address.',
	INVALID_EMAIL: 'Please enter a valid email address.',
	MISSING_PHONE: 'Please enter a valid phone number to receive SMS alerts.',
	MISSING_RESORTS: 'Please select at least one resort to receive alerts for.',
	MISSING_NAME: 'Please enter your name.',
	MISSING_MESSAGE: 'Please enter a message.',
	VALIDATION_ERROR: 'Please check your information and try again.',
	MISSING_REQUIRED_FIELD: 'Please fill in all required fields.',
	METHOD_NOT_ALLOWED:
		'Something went wrong. Please refresh the page and try again.',
	INVALID_REQUEST:
		'Invalid information provided. Please check your entries and try again.',
	REQUEST_TOO_LARGE: 'That message is too long. Please shorten it.',
	RATE_LIMITED: 'Too many requests. Please wait a moment and try again.',
	INTERNAL_ERROR:
		'Something went wrong on our end. Please try again in a few moments.',
}

const statusFallback = (status: number): string => {
	if (status === 409) {
		return 'You already have an alert for this resort. Try selecting a different resort.'
	}

	if (status === 429) {
		return 'Too many requests. Please wait a moment and try again.'
	}

	if (status >= 500) {
		return 'Something went wrong on our end. Please try again in a few moments.'
	}

	if (status >= 400) {
		return 'Please check your information and try again.'
	}

	return 'An unexpected error occurred. Please try again.'
}

const messageFor = (body: string, status: number): string => {
	// The proxy can return an HTML error page, so never assume the body is JSON.
	try {
		const parsed = JSON.parse(body) as Partial<ApiErrorResponse>
		if (parsed.error && ERROR_MESSAGES[parsed.error]) {
			return ERROR_MESSAGES[parsed.error]
		}

		return parsed.message || statusFallback(status)
	} catch {
		return statusFallback(status)
	}
}

const codeFor = (body: string): string => {
	try {
		const parsed = JSON.parse(body) as Partial<ApiErrorResponse>
		return parsed.error || 'UNKNOWN'
	} catch {
		return 'UNKNOWN'
	}
}

type RequestOptions = {
	method?: string
	body?: unknown
	signal?: AbortSignal
}

export async function apiRequest<T>(
	path: string,
	options: RequestOptions = {}
): Promise<T> {
	const { method = 'GET', body, signal } = options

	let response: Response

	try {
		response = await fetch(`${BASE_SERVER_URL}${path}`, {
			method,
			headers: { 'Content-Type': 'application/json' },
			body: body === undefined ? undefined : JSON.stringify(body),
			credentials: 'include',
			signal: signal ?? AbortSignal.timeout(REQUEST_TIMEOUT_MS),
		})
	} catch (err) {
		// Status 0 marks a transport failure, which is worth retrying.
		if (err instanceof DOMException && err.name === 'TimeoutError') {
			throw new ApiError('The request timed out. Please try again.', 0, 'TIMEOUT')
		}

		throw new ApiError(
			'Could not reach the server. Please check your connection.',
			0,
			'NETWORK'
		)
	}

	if (!response.ok) {
		const text = await response.text().catch(() => '')
		throw new ApiError(
			messageFor(text, response.status),
			response.status,
			codeFor(text)
		)
	}

	if (response.status === 204) {
		return undefined as T
	}

	const text = await response.text()
	if (!text) {
		return undefined as T
	}

	try {
		return JSON.parse(text) as T
	} catch {
		throw new ApiError(
			'The server returned an unexpected response.',
			response.status,
			'INVALID_RESPONSE'
		)
	}
}

// retryOnlyTransport keeps React Query from re-sending a request the server has
// already rejected on its merits.
export const retryOnlyTransport =
	(max: number) =>
	(failureCount: number, error: Error): boolean => {
		if (error instanceof ApiError && !error.isRetryable) {
			return false
		}

		return failureCount < max
	}
