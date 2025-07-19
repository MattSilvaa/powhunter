import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { BASE_SERVER_URL, UserAlert } from './types.ts'

const fetchUserAlerts = async (email: string): Promise<UserAlert[]> => {
	const response = await fetch(
		`${BASE_SERVER_URL}/api/user/alerts?email=${encodeURIComponent(email)}`,
		{
			method: 'GET',
			headers: {
				'Content-Type': 'application/json',
			},
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
	email,
	resortUuid,
}: {
	email: string
	resortUuid: string
}): Promise<void> => {
	const response = await fetch(
		`${BASE_SERVER_URL}/api/user/alerts/delete?email=${encodeURIComponent(
			email
		)}&resort_uuid=${encodeURIComponent(resortUuid)}`,
		{
			method: 'DELETE',
			headers: {
				'Content-Type': 'application/json',
			},
			credentials: 'include',
		}
	)

	if (!response.ok) {
		throw new Error(`Failed to delete alert: ${response.status}`)
	}
}

const deleteAllAlerts = async (email: string): Promise<void> => {
	const response = await fetch(
		`${BASE_SERVER_URL}/api/user/alerts/delete-all?email=${encodeURIComponent(
			email
		)}`,
		{
			method: 'DELETE',
			headers: {
				'Content-Type': 'application/json',
			},
			credentials: 'include',
		}
	)

	if (!response.ok) {
		throw new Error(`Failed to delete all alerts: ${response.status}`)
	}
}

export function useUserAlerts(email: string) {
	return useQuery<UserAlert[]>({
		queryKey: ['userAlerts', email],
		queryFn: () => fetchUserAlerts(email),
		enabled: !!email,
		retry: 1,
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