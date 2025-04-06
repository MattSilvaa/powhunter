import { useMutation } from '@tanstack/react-query'

const API_BASE_URL = 'http://localhost:8080/api'

type AlertData = {
	email: string;
	phone: string;
	notificationDays: number;
	minSnowAmount: number;
	resorts: string[];
  }


const createAlert = async (data: AlertData): Promise<void> => {
	const response = await fetch(`${API_BASE_URL}/createAlert`, {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json',
		},
		credentials: 'include',
		body: JSON.stringify(data)
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
	} = useMutation<void, Error, AlertData>({
		mutationFn: createAlert,
		retry: 3,
	})

	return {
        createAlert: mutate,
		loading: isPending,
		error: isError ? error?.message || 'An error occurred' : null,
	}
}
