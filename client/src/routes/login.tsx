import React, { useEffect, useRef, useState } from 'react'
import {
	Alert,
	Box,
	Button,
	CircularProgress,
	Container,
	Link as MuiLink,
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
import PageHeader from '../components/pageHeader.tsx'

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

	// A sign-in link works exactly once, so it must be redeemed exactly once. In
	// development React mounts effects twice, and any remount would otherwise
	// spend the token a second time and report it as already used.
	const redeemedToken = useRef<string | null>(null)

	// Arriving with a token in the URL means the user clicked the emailed link,
	// so redeem it and send them on to their subscriptions.
	useEffect(() => {
		if (!token || redeemedToken.current === token) {
			return
		}

		redeemedToken.current = token

		void completeLogin(token).then((signedIn) => {
			if (signedIn) {
				navigate('/manage', { replace: true })
			}
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
			<Container maxWidth="xs" sx={{ py: { xs: 6, md: 10 } }}>
				<Paper variant="outlined" sx={{ p: 4, textAlign: 'center' }}>
					{completing && (
						<>
							<CircularProgress size={28} />
							<Typography color="text.secondary" sx={{ mt: 2 }}>
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
		<Container maxWidth="xs" sx={{ py: { xs: 6, md: 10 } }}>
			<PageHeader
				title="Sign in"
				subtitle="We'll email you a link to sign in. No password required."
			/>
			<Paper variant="outlined" sx={{ p: { xs: 2.5, sm: 3.5 } }}>
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
							autoComplete="email"
							value={email}
							onChange={(e) => {
								setEmail(e.target.value)
								setEmailError('')
							}}
							error={!!emailError}
							helperText={emailError}
							sx={{ mb: 2.5 }}
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
			</Paper>
			<Typography variant="body2" color="text.secondary" sx={{ mt: 3 }}>
				New here?{' '}
				<MuiLink component={Link} to="/signup" underline="hover">
					Create an alert
				</MuiLink>{' '}
				and you&apos;ll get an account with it.
			</Typography>
		</Container>
	)
}
