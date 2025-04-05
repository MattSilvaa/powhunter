import { useMutation } from '@tanstack/react-query'
import { Resort } from './types.ts'

const API_BASE_URL = 'http://localhost:8080/api'

const createAlert = async (): Promise<void> => {
	const response = await fetch(`${API_BASE_URL}/createAlert`, {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json',
		},
		credentials: 'include',
	})

	if (!response.ok) {
		throw new Error(`Request failed: ${response.status}`)
	}
}

export function useCreateAlert() {
	const {
        mutate,
		isPending,
		isError,
		error,
	} = useMutation<void, Error>({
		mutationFn: createAlert,
		retry: 3,
	})

	return {
        createAlert: mutate,
		loading: isPending,
		error: isError ? error?.message || 'An error occurred' : null,
	}
}
