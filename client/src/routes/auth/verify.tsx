import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Box, Typography, CircularProgress, Alert, Button } from '@mui/material'
import { useAuth } from '../../shared/useAuth.ts'

export default function VerifyAuth() {
	const [searchParams] = useSearchParams()
	const navigate = useNavigate()
	const { login } = useAuth()
	const [status, setStatus] = useState<'verifying' | 'creating-alerts' | 'success' | 'error'>('verifying')
	const [error, setError] = useState('')

	useEffect(() => {
		const token = searchParams.get('token')
		const purpose = searchParams.get('purpose')

		if (!token) {
			setStatus('error')
			setError('Invalid verification link')
			return
		}

		verifyToken(token, purpose)
	}, [searchParams])

	const verifyToken = async (token: string, purpose: string | null) => {
		try {
			const response = await fetch('/api/auth/verify', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({ token }),
			})

			const data = await response.json()

			if (response.ok) {
				// Update auth context
				login(data.token, data.user)
				
				// If this is a signup flow, create the alerts
				if (purpose === 'signup') {
					const signupData = localStorage.getItem('signup_data')
					if (signupData) {
						setStatus('creating-alerts')
						await createSignupAlerts(JSON.parse(signupData), data.token)
						localStorage.removeItem('signup_data')
					}
				}
				
				setStatus('success')
				
				// Redirect after success
				setTimeout(() => {
					if (purpose === 'signup') {
						navigate('/success')
					} else {
						navigate('/manage')
					}
				}, 2000)
			} else {
				setStatus('error')
				setError(data.message || 'Failed to verify magic link')
			}
		} catch (err) {
			setStatus('error')
			setError('Network error. Please try again.')
		}
	}

	const createSignupAlerts = async (signupData: any, authToken: string) => {
		try {
			// Map resort names to UUIDs (we'll need to fetch resorts)
			const resortsResponse = await fetch('/api/resorts')
			const resorts = await resortsResponse.json()
			
			const resortsUuids = signupData.resorts
				.map((resortName: string) => {
					const resort = resorts.find((r: any) => r.name === resortName)
					return resort?.uuid
				})
				.filter((uuid: string) => !!uuid)

			const alertData = {
				phone: signupData.phone,
				notificationDays: signupData.notificationDays,
				minSnowAmount: signupData.minSnowAmount,
				resortsUuids,
			}

			const response = await fetch('/api/alerts', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'Authorization': `Bearer ${authToken}`,
				},
				body: JSON.stringify(alertData),
			})

			if (!response.ok) {
				throw new Error('Failed to create alerts')
			}
		} catch (err) {
			console.error('Failed to create signup alerts:', err)
			// Don't fail the verification process if alert creation fails
		}
	}

	if (status === 'verifying') {
		return (
			<Box sx={{ 
				maxWidth: 400, 
				mx: 'auto', 
				mt: 8, 
				p: 3,
				textAlign: 'center'
			}}>
				<CircularProgress size={40} sx={{ mb: 2 }} />
				<Typography variant="h6" gutterBottom>
					Verifying your magic link...
				</Typography>
				<Typography variant="body2" color="text.secondary">
					Please wait while we sign you in.
				</Typography>
			</Box>
		)
	}

	if (status === 'creating-alerts') {
		return (
			<Box sx={{ 
				maxWidth: 400, 
				mx: 'auto', 
				mt: 8, 
				p: 3,
				textAlign: 'center'
			}}>
				<CircularProgress size={40} sx={{ mb: 2 }} />
				<Typography variant="h6" gutterBottom>
					Setting up your powder alerts...
				</Typography>
				<Typography variant="body2" color="text.secondary">
					Almost done! We're creating your custom alerts.
				</Typography>
			</Box>
		)
	}

	if (status === 'success') {
		return (
			<Box sx={{ 
				maxWidth: 400, 
				mx: 'auto', 
				mt: 8, 
				p: 3,
				textAlign: 'center'
			}}>
				<Alert severity="success" sx={{ mb: 2 }}>
					Successfully signed in!
				</Alert>
				<Typography variant="body1" gutterBottom>
					Redirecting you to your dashboard...
				</Typography>
			</Box>
		)
	}

	return (
		<Box sx={{ 
			maxWidth: 400, 
			mx: 'auto', 
			mt: 8, 
			p: 3,
			textAlign: 'center'
		}}>
			<Alert severity="error" sx={{ mb: 2 }}>
				{error}
			</Alert>
			<Typography variant="body1" gutterBottom>
				The magic link may have expired or been used already.
			</Typography>
			<Button 
				variant="contained" 
				onClick={() => navigate('/login')}
				sx={{ mt: 2 }}
			>
				Get New Magic Link
			</Button>
		</Box>
	)
}