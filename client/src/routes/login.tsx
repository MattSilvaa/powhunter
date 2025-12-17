import { useState } from 'react'
import { Box, Button, TextField, Typography, Alert, CircularProgress } from '@mui/material'

export default function Login() {
	const [email, setEmail] = useState('')
	const [isLoading, setIsLoading] = useState(false)
	const [message, setMessage] = useState('')
	const [error, setError] = useState('')

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault()
		if (!email) return

		setIsLoading(true)
		setError('')
		setMessage('')

		try {
			const response = await fetch('/api/auth/magic-link', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({
					email,
					purpose: 'login'
				}),
			})

			const data = await response.json()

			if (response.ok) {
				setMessage('Magic link sent! Check your email and click the link to sign in.')
			} else {
				setError(data.message || 'Failed to send magic link')
			}
		} catch (err) {
			setError('Network error. Please try again.')
		} finally {
			setIsLoading(false)
		}
	}

	return (
		<Box sx={{ 
			maxWidth: 400, 
			mx: 'auto', 
			mt: 4, 
			p: 3,
			border: 1,
			borderColor: 'grey.300',
			borderRadius: 2
		}}>
			<Typography variant="h4" component="h1" gutterBottom textAlign="center">
				Sign In
			</Typography>
			
			{message && (
				<Alert severity="success" sx={{ mb: 2 }}>
					{message}
				</Alert>
			)}
			
			{error && (
				<Alert severity="error" sx={{ mb: 2 }}>
					{error}
				</Alert>
			)}

			<form onSubmit={handleSubmit}>
				<TextField
					fullWidth
					label="Email"
					type="email"
					value={email}
					onChange={(e) => setEmail(e.target.value)}
					required
					margin="normal"
					disabled={isLoading}
				/>
				
				<Button
					type="submit"
					fullWidth
					variant="contained"
					sx={{ mt: 3, mb: 2 }}
					disabled={isLoading || !email}
				>
					{isLoading ? (
						<>
							<CircularProgress size={20} sx={{ mr: 1 }} />
							Sending Magic Link...
						</>
					) : (
						'Send Magic Link'
					)}
				</Button>
			</form>

			<Typography variant="body2" color="text.secondary" textAlign="center" sx={{ mt: 2 }}>
				Don't have an account? The magic link will create one for you automatically.
			</Typography>
		</Box>
	)
}