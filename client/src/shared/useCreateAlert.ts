import { useMutation } from '@tanstack/react-query'
import { apiRequest, retryOnlyTransport } from './apiClient.ts'

// A create is not idempotent. Retrying a request the server already committed
// produces a duplicate-alert conflict, so only transport failures are retried.
const CREATE_ALERT_RETRIES = 2

type AlertData = {
	email: string
	phone: string
	notificationDays: number
	minSnowAmount: number
	resortsUuids: string[]
}

const createAlert = (data: AlertData): Promise<void> =>
	apiRequest<void>('/api/alerts', { method: 'POST', body: data })

export function useCreateAlert() {
	const { mutate, isPending, isError, error } = useMutation<
		void,
		Error,
		AlertData
	>({
		mutationFn: createAlert,
		retry: retryOnlyTransport(CREATE_ALERT_RETRIES),
	})

	return {
		createAlert: mutate,
		loading: isPending,
		error: isError ? error?.message || 'An error occurred' : null,
	}
}
