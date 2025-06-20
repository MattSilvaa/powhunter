import { useMutation } from '@tanstack/react-query'
import { API_BASE_URL } from './types'

const CREATE_ALERT_RETRIES = 2

type AlertData = {
	email: string
	phone: string
	notificationDays: number
	minSnowAmount: number
	resortsUuids: string[]
}

type ErrorResponse = {
	error: string
	message: string
}

const getErrorMessage = (errorResponse: ErrorResponse): string => {
	switch (errorResponse.error) {
		case 'DUPLICATE_EMAIL':
			return 'This email address is already registered. Try using a different email or contact support if this is your account.'
		case 'MISSING_EMAIL':
			return 'Please enter a valid email address.'
		case 'MISSING_PHONE':
			return 'Please enter a phone number to receive SMS alerts.'
		case 'MISSING_RESORTS':
			return 'Please select at least one resort to receive alerts for.'
		case 'VALIDATION_ERROR':
			return 'Please check your information and try again.'
		case 'METHOD_NOT_ALLOWED':
			return 'Something went wrong. Please refresh the page and try again.'
		case 'INVALID_REQUEST':
			return 'Invalid information provided. Please check your entries and try again.'
		case 'INTERNAL_ERROR':
			return 'Something went wrong on our end. Please try again in a few moments.'
		default:
			return errorResponse.message || 'An unexpected error occurred. Please try again.'
	}
}

const createAlert = async (data: AlertData): Promise<void> => {
	const response = await fetch(`${API_BASE_URL}/alerts`, {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json',
		},
		credentials: 'include',
		body: JSON.stringify(data),
	})

	if (!response.ok) {
		console.log('Error response status:', response.status)
		console.log('Error response headers:', Object.fromEntries(response.headers.entries()))
		
		const responseText = await response.text()
		console.log('Raw error response:', responseText)
		
		let errorMessage = 'An unexpected error occurred. Please try again.'
		
		try {
			const errorData: ErrorResponse = JSON.parse(responseText)
			console.log('Parsed error data:', errorData)
			errorMessage = getErrorMessage(errorData)
			console.log('Final error message:', errorMessage)
		} catch (parseError) {
			console.warn('Failed to parse error response as JSON:', parseError)
			console.log('Falling back to status-based error messages')
			
			switch (response.status) {
				case 409:
					errorMessage = 'This email address is already registered. Try using a different email.'
					break
				case 400:
					errorMessage = 'Please check your information and try again.'
					break
				case 500:
				default:
					if (response.status >= 500) {
						errorMessage = 'Something went wrong on our end. Please try again in a few moments.'
					} else {
						errorMessage = `Request failed with status ${response.status}`
					}
					break
			}
		}
		
		throw new Error(errorMessage)
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
		retry: CREATE_ALERT_RETRIES,
	})

	return {
		createAlert: mutate,
		loading: isPending,
		error: isError ? error?.message || 'An error occurred' : null,
	}
}
