import React, { useEffect, useState } from 'react'
import {
	Alert,
	Box,
	Button,
	CircularProgress,
	Container,
	Paper,
	TextField,
	Typography,
} from '@mui/material'
import { Link, useNavigate, useSearchParams } from 'react-router'
import {
	useCompleteLogin,
	useCurrentUser,
	useRequestLoginLink,
} from '../shared/useAuth.ts'

export default function LoginPage() {
	const [searchParams] = useSearchParams()
	const navigate = useNavigate()
	const token = searchParams.get('token')

	const { user } = useCurrentUser()
	const {
		requestLink,
		loading: sending,
		sent,
		error: sendError,
	} = useRequestLoginLink()
	const {
		completeLogin,
		loading: completing,
		error: completeError,
	} = useCompleteLogin()

	const [email, setEmail] = useState('')
	const [emailError, setEmailError] = useState('')

	// Arriving with a token in the URL means the user clicked the emailed link,
	// so redeem it and send them on to their subscriptions.
	useEffect(() => {
		if (!token) {
			return
		}

		completeLogin(token, {
			onSuccess: () => {
				navigate('/manage', { replace: true })
			},
		})
	}, [token, completeLogin, navigate])

	useEffect(() => {
		if (user && !token) {
			navigate('/manage', { replace: true })
		}
	}, [user, token, navigate])

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault()

		const trimmed = email.trim()

		if (!trimmed) {
			setEmailError('Email is required')
			return
		}

		if (!/\S+@\S+\.\S+/.test(trimmed)) {
			setEmailError('Please enter a valid email address')
			return
		}

		setEmailError('')
		requestLink(trimmed)
	}

	if (token) {
		return (
			<Container maxWidth="sm" sx={{ py: 8 }}>
				<Paper elevation={3} sx={{ p: 4, textAlign: 'center' }}>
					{completing && (
						<>
							<CircularProgress />
							<Typography variant="body1" sx={{ mt: 2 }}>
								Signing you in…
							</Typography>
						</>
					)}
					{completeError && (
						<>
							<Alert severity="error" sx={{ mb: 3 }}>
								{completeError}
							</Alert>
							<Button component={Link} to="/login" variant="contained">
								Request a new link
							</Button>
						</>
					)}
				</Paper>
			</Container>
		)
	}

	return (
		<Container maxWidth="sm" sx={{ py: 8 }}>
			<Paper elevation={3} sx={{ p: 4 }}>
				<Typography variant="h2" component="h1" gutterBottom align="center">
					Sign in
				</Typography>
				<Typography
					variant="body1"
					color="text.secondary"
					align="center"
					sx={{ mb: 4 }}
				>
					We'll email you a link to sign in. No password required.
				</Typography>

				{sent ? (
					<Alert severity="success">
						If that address has an account, a sign-in link is on its way. The
						link works once and expires in 15 minutes.
					</Alert>
				) : (
					<Box component="form" onSubmit={handleSubmit} noValidate>
						{sendError && (
							<Alert severity="error" sx={{ mb: 3 }}>
								{sendError}
							</Alert>
						)}

						<TextField
							fullWidth
							label="Email Address"
							type="email"
							name="email"
							value={email}
							onChange={(e) => {
								setEmail(e.target.value)
								setEmailError('')
							}}
							error={!!emailError}
							helperText={emailError}
							sx={{ mb: 3 }}
						/>

						<Button
							type="submit"
							variant="contained"
							size="large"
							fullWidth
							disabled={sending}
						>
							{sending ? 'Sending…' : 'Email me a sign-in link'}
						</Button>
					</Box>
				)}

				<Box sx={{ mt: 4, textAlign: 'center' }}>
					<Button component={Link} to="/" variant="text">
						← Back to Home
					</Button>
				</Box>
			</Paper>
		</Container>
	)
}
