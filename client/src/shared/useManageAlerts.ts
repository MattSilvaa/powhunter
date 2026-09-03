import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError, apiRequest, retryOnlyTransport } from './apiClient.ts'
import { UserAlert } from './types.ts'

const ALERT_RETRIES = 1

const fetchUserAlerts = async (email: string): Promise<UserAlert[]> => {
	try {
		return await apiRequest<UserAlert[]>(
			`/api/user/alerts?email=${encodeURIComponent(email)}`
		)
	} catch (err) {
		// No subscriptions is a normal result, not a failure to show the user.
		if (err instanceof ApiError && err.status === 404) {
			return []
		}

		throw err
	}
}

const deleteAlert = ({
	email,
	resortUuid,
}: {
	email: string
	resortUuid: string
}): Promise<void> =>
	apiRequest<void>(
		`/api/user/alerts/delete?email=${encodeURIComponent(
			email
		)}&resort_uuid=${encodeURIComponent(resortUuid)}`,
		{ method: 'DELETE' }
	)

const deleteAllAlerts = (email: string): Promise<void> =>
	apiRequest<void>(
		`/api/user/alerts/delete-all?email=${encodeURIComponent(email)}`,
		{ method: 'DELETE' }
	)

export function useUserAlerts(email: string) {
	return useQuery<UserAlert[]>({
		queryKey: ['userAlerts', email],
		queryFn: () => fetchUserAlerts(email),
		enabled: !!email,
		retry: retryOnlyTransport(ALERT_RETRIES),
	})
}

export function useDeleteAlert() {
	const queryClient = useQueryClient()

	return useMutation<void, Error, { email: string; resortUuid: string }>({
		mutationFn: deleteAlert,
		onSuccess: (_, { email }) => {
			queryClient.invalidateQueries({ queryKey: ['userAlerts', email] })
		},
	})
}

export function useDeleteAllAlerts() {
	const queryClient = useQueryClient()

	return useMutation<void, Error, string>({
		mutationFn: deleteAllAlerts,
		onSuccess: (_, email) => {
			queryClient.invalidateQueries({ queryKey: ['userAlerts', email] })
		},
	})
}
