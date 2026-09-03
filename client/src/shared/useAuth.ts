import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError, apiRequest } from './apiClient.ts'

export type AuthUser = {
	email: string
	phone?: string
	emailVerified: boolean
}

export const AUTH_QUERY_KEY = ['auth', 'me']

const fetchCurrentUser = async (): Promise<AuthUser | null> => {
	try {
		return await apiRequest<AuthUser>('/api/auth/me')
	} catch (err) {
		// Signed out is an ordinary state, not an error to surface.
		if (err instanceof ApiError && err.status === 401) {
			return null
		}

		throw err
	}
}

// useCurrentUser reports who is signed in, or null. The session lives in an
// HttpOnly cookie, so the server is the only thing that can answer this.
export function useCurrentUser() {
	const { data, isLoading, isError } = useQuery<AuthUser | null>({
		queryKey: AUTH_QUERY_KEY,
		queryFn: fetchCurrentUser,
		// A 401 is a definitive answer, and retryOnlyTransport already filters it
		// out; keep the signed-out check snappy.
		retry: false,
		staleTime: 5 * 60 * 1000,
	})

	return {
		user: data ?? null,
		loading: isLoading,
		error: isError,
	}
}

export function useRequestLoginLink() {
	const { mutate, isPending, isSuccess, isError, error, reset } = useMutation<
		void,
		Error,
		string
	>({
		mutationFn: (email: string) =>
			apiRequest<void>('/api/auth/request-link', {
				method: 'POST',
				body: { email },
			}),
	})

	return {
		requestLink: mutate,
		loading: isPending,
		sent: isSuccess,
		error: isError ? error?.message || 'An error occurred' : null,
		reset,
	}
}

export function useCompleteLogin() {
	const queryClient = useQueryClient()

	const { mutate, isPending, isError, error } = useMutation<
		void,
		Error,
		string
	>({
		mutationFn: (token: string) =>
			apiRequest<void>('/api/auth/callback', {
				method: 'POST',
				body: { token },
			}),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: AUTH_QUERY_KEY })
		},
	})

	return {
		completeLogin: mutate,
		loading: isPending,
		error: isError ? error?.message || 'An error occurred' : null,
	}
}

export function useLogout() {
	const queryClient = useQueryClient()

	const { mutate, isPending } = useMutation<void, Error, void>({
		mutationFn: () => apiRequest<void>('/api/auth/logout', { method: 'POST' }),
		onSuccess: () => {
			// Drop every cached answer: what is readable depends on who is signed in.
			queryClient.clear()
		},
	})

	return {
		logout: mutate,
		loading: isPending,
	}
}
