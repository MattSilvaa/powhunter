import { useCallback, useState } from 'react'
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

// useCompleteLogin redeems an emailed sign-in token. It resolves to the
// signed-in user, or null with error set.
//
// This is deliberately not a useMutation. The redeem runs from a mount effect,
// and in development React Router renders under StrictMode, which mounts,
// unmounts and remounts the page. The simulated unmount detaches a mutation
// observer from the in-flight mutation and the remount never reattaches it, so
// the request succeeded while the page sat on "Signing you in…" forever and
// its onSuccess navigation never ran. A plain promise has no observer to lose.
export function useCompleteLogin() {
	const queryClient = useQueryClient()
	const [loading, setLoading] = useState(false)
	const [error, setError] = useState<string | null>(null)

	const completeLogin = useCallback(
		async (token: string): Promise<AuthUser | null> => {
			setLoading(true)
			setError(null)

			try {
				await apiRequest<void>('/api/auth/callback', {
					method: 'POST',
					body: { token },
				})

				// Redeeming the token is only half the job: the session lives in a
				// cookie the browser may decline to keep, and a 200 here says nothing
				// about that. Reading the session back proves the sign-in actually
				// took, and leaves the user in the cache so the page we navigate to
				// sees them immediately rather than reading a stale null and bouncing
				// back to sign-in.
				const user = await queryClient.fetchQuery({
					queryKey: AUTH_QUERY_KEY,
					queryFn: fetchCurrentUser,
					staleTime: 0,
				})

				if (!user) {
					setError(
						'Signed in, but your browser did not keep the session. Check that cookies are enabled and try the link again.'
					)
					return null
				}

				return user
			} catch (err) {
				setError(
					err instanceof Error && err.message
						? err.message
						: 'An error occurred'
				)
				return null
			} finally {
				setLoading(false)
			}
		},
		[queryClient]
	)

	return { completeLogin, loading, error }
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

// useUpdateProfile changes the phone number alerts are sent to. The server
// answers with the updated account, which replaces the cached one directly.
export function useUpdateProfile() {
	const queryClient = useQueryClient()

	const { mutate, isPending, isError, error, reset } = useMutation<
		AuthUser,
		Error,
		string
	>({
		mutationFn: (phone: string) =>
			apiRequest<AuthUser>('/api/user/profile', {
				method: 'PATCH',
				body: { phone },
			}),
		onSuccess: (user) => {
			queryClient.setQueryData(AUTH_QUERY_KEY, user)
		},
	})

	return {
		updateProfile: mutate,
		loading: isPending,
		error: isError ? error?.message || 'An error occurred' : null,
		reset,
	}
}
