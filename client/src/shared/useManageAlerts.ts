import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError, apiRequest, retryOnlyTransport } from './apiClient.ts'
import { AlertSettings, UserAlert } from './types.ts'

const ALERT_RETRIES = 1

export const USER_ALERTS_QUERY_KEY = ['userAlerts']

// These endpoints identify the account from the session cookie. They used to
// take an email in the query string, which let anyone who guessed an address
// read or delete that person's alerts.
const fetchUserAlerts = async (): Promise<UserAlert[]> => {
	try {
		return await apiRequest<UserAlert[]>('/api/user/alerts')
	} catch (err) {
		// No subscriptions is a normal result, not a failure to show the user.
		if (err instanceof ApiError && err.status === 404) {
			return []
		}

		throw err
	}
}

const deleteAlert = (resortUuid: string): Promise<void> =>
	apiRequest<void>(
		`/api/user/alerts/delete?resort_uuid=${encodeURIComponent(resortUuid)}`,
		{ method: 'DELETE' }
	)

const deleteAllAlerts = (): Promise<void> =>
	apiRequest<void>('/api/user/alerts/delete-all', { method: 'DELETE' })

export function useUserAlerts(enabled: boolean) {
	return useQuery<UserAlert[]>({
		queryKey: USER_ALERTS_QUERY_KEY,
		queryFn: fetchUserAlerts,
		enabled,
		retry: retryOnlyTransport(ALERT_RETRIES),
	})
}

export function useDeleteAlert() {
	const queryClient = useQueryClient()

	return useMutation<void, Error, string>({
		mutationFn: deleteAlert,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: USER_ALERTS_QUERY_KEY })
		},
	})
}

export function useDeleteAllAlerts() {
	const queryClient = useQueryClient()

	return useMutation<void, Error, void>({
		mutationFn: deleteAllAlerts,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: USER_ALERTS_QUERY_KEY })
		},
	})
}

export type AddAlertsData = AlertSettings & {
	resortsUuids: string[]
}

export type UpdateAlertData = AlertSettings & {
	resortUuid: string
}

// Adding alerts while signed in sends no email or phone: the server takes both
// from the session's account.
const createUserAlerts = (data: AddAlertsData): Promise<void> =>
	apiRequest<void>('/api/user/alerts', { method: 'POST', body: data })

const updateAlert = ({
	resortUuid,
	...settings
}: UpdateAlertData): Promise<void> =>
	apiRequest<void>(
		`/api/user/alerts?resort_uuid=${encodeURIComponent(resortUuid)}`,
		{ method: 'PATCH', body: settings }
	)

export function useCreateUserAlerts() {
	const queryClient = useQueryClient()

	return useMutation<void, Error, AddAlertsData>({
		mutationFn: createUserAlerts,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: USER_ALERTS_QUERY_KEY })
		},
	})
}

export function useUpdateAlert() {
	const queryClient = useQueryClient()

	return useMutation<void, Error, UpdateAlertData>({
		mutationFn: updateAlert,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: USER_ALERTS_QUERY_KEY })
		},
	})
}
