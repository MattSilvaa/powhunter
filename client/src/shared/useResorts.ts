import { useQuery } from '@tanstack/react-query'
import { Resort } from './types.ts'

const API_BASE_URL = 'http://localhost:8080/api'

const fetchResorts = async (): Promise<Resort[]> => {
	const response = await fetch(`${API_BASE_URL}/resorts`, {
		method: 'GET',
		headers: {
			'Content-Type': 'application/json',
		},
		credentials: 'include', // For auth cookies if needed
	})

	if (!response.ok) {
		throw new Error(`Request failed: ${response.status}`)
	}

	return response.json()
}

export function useResorts() {
	const {
		data = [],
		isLoading,
		isError,
		error,
		refetch,
	} = useQuery<Resort[], Error>({
		queryKey: ['resorts'],
		queryFn: fetchResorts,
		staleTime: 5 * 60 * 1000,
		retry: 2,
	})

	return {
		resorts: data,
		loading: isLoading,
		error: isError ? error?.message || 'An error occurred' : null,
		refresh: refetch,
	}
}
