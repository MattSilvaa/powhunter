import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { BASE_SERVER_URL, UserAlert } from './types.ts'
import { getAuthHeaders } from './useAuth.ts'

const fetchUserAlerts = async (token: string | null): Promise<UserAlert[]> => {
	const response = await fetch(
		`${BASE_SERVER_URL}/api/user/alerts`,
		{
			method: 'GET',
			headers: getAuthHeaders(token),
			credentials: 'include',
		}
	)

	if (!response.ok) {
		if (response.status === 404) {
			return []
		}
		throw new Error(`Failed to fetch alerts: ${response.status}`)
	}

	return response.json()
}

const deleteAlert = async ({
	token,
	resortUuid,
}: {
	token: string | null
	resortUuid: string
}): Promise<void> => {
	const response = await fetch(
		`${BASE_SERVER_URL}/api/user/alerts/delete?resort_uuid=${encodeURIComponent(resortUuid)}`,
		{
			method: 'DELETE',
			headers: getAuthHeaders(token),
			credentials: 'include',
		}
	)

	if (!response.ok) {
		throw new Error(`Failed to delete alert: ${response.status}`)
	}
}

const deleteAllAlerts = async (token: string | null): Promise<void> => {
	const response = await fetch(
		`${BASE_SERVER_URL}/api/user/alerts/delete-all`,
		{
			method: 'DELETE',
			headers: getAuthHeaders(token),
			credentials: 'include',
		}
	)

	if (!response.ok) {
		throw new Error(`Failed to delete all alerts: ${response.status}`)
	}
}

export function useUserAlerts(token: string | null) {
	return useQuery<UserAlert[]>({
		queryKey: ['userAlerts', token],
		queryFn: () => fetchUserAlerts(token),
		enabled: !!token,
		retry: 1,
	})
}

export function useDeleteAlert() {
	const queryClient = useQueryClient()

	return useMutation<void, Error, { token: string | null; resortUuid: string }>({
		mutationFn: deleteAlert,
		onSuccess: (_, { token }) => {
			queryClient.invalidateQueries({ queryKey: ['userAlerts', token] })
		},
	})
}

export function useDeleteAllAlerts() {
	const queryClient = useQueryClient()

	return useMutation<void, Error, string | null>({
		mutationFn: deleteAllAlerts,
		onSuccess: (_, token) => {
			queryClient.invalidateQueries({ queryKey: ['userAlerts', token] })
		},
	})
}
