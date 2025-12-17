import { createContext, useContext, useState, useEffect, ReactNode, createElement } from 'react'

interface User {
	id: number
	uuid: string
	email: string
	phone?: string
	created_at: string
	email_verified: boolean
	verified_at?: string
}

interface AuthContextType {
	user: User | null
	token: string | null
	isAuthenticated: boolean
	login: (token: string, user: User) => void
	logout: () => void
	isLoading: boolean
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function useAuth() {
	const context = useContext(AuthContext)
	if (context === undefined) {
		throw new Error('useAuth must be used within an AuthProvider')
	}
	return context
}

export function AuthProvider({ children }: { children: ReactNode }) {
	const [user, setUser] = useState<User | null>(null)
	const [token, setToken] = useState<string | null>(null)
	const [isLoading, setIsLoading] = useState(true)

	useEffect(() => {
		// Check for existing auth on mount
		const storedToken = localStorage.getItem('auth_token')
		const storedUser = localStorage.getItem('user')

		if (storedToken && storedUser) {
			try {
				const parsedUser = JSON.parse(storedUser)
				setToken(storedToken)
				setUser(parsedUser)
			} catch (error) {
				// Invalid stored data, clear it
				localStorage.removeItem('auth_token')
				localStorage.removeItem('user')
			}
		}

		setIsLoading(false)
	}, [])

	const login = (newToken: string, newUser: User) => {
		localStorage.setItem('auth_token', newToken)
		localStorage.setItem('user', JSON.stringify(newUser))
		setToken(newToken)
		setUser(newUser)
	}

	const logout = async () => {
		// Call logout API to invalidate session
		if (token) {
			try {
				await fetch('/api/auth/logout', {
					method: 'POST',
					headers: {
						'Authorization': `Bearer ${token}`,
					},
				})
			} catch (error) {
				// Logout API call failed, but we'll still clear local state
				console.error('Logout API call failed:', error)
			}
		}

		// Clear local storage and state
		localStorage.removeItem('auth_token')
		localStorage.removeItem('user')
		setToken(null)
		setUser(null)
	}

	const value: AuthContextType = {
		user,
		token,
		isAuthenticated: !!token && !!user,
		login,
		logout,
		isLoading,
	}

	return createElement(AuthContext.Provider, { value }, children)
}

// Helper function to get auth headers for API calls
export function getAuthHeaders(token: string | null): Record<string, string> {
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
	}

	if (token) {
		headers['Authorization'] = `Bearer ${token}`
	}

	return headers
}