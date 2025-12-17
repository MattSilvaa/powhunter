import React, { useState } from 'react'
import {
	Alert,
	Box,
	Button,
	CircularProgress,
	Container,
	FormControl,
	Grid,
	InputLabel,
	LinearProgress,
	MenuItem,
	Paper,
	Select,
	SelectChangeEvent,
	Slider,
	TextField,
	Typography,
} from '@mui/material'
import { useNavigate, Link } from 'react-router'
import { useResorts } from '../shared/useResorts.ts'
import { Resort } from '../shared/types.ts'

export default function SignUpPage() {
	const navigate = useNavigate()
	const [step, setStep] = useState<'form' | 'email-sent'>('form')
	const [formData, setFormData] = useState({
		email: '',
		phone: '',
		notificationDays: 3,
		minSnowAmount: 6,
		resorts: [] as string[],
	})
	const [fieldErrors, setFieldErrors] = useState({
		email: '',
		phone: '',
		resorts: '',
	})
	const [isLoading, setIsLoading] = useState(false)
	const [error, setError] = useState('')

	const { resorts = [], loading, error: resortsError } = useResorts()

	const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
		const { name, value } = e.target
		setFormData((prev) => ({
			...prev,
			[name]: value,
		}))

		// Clear field-specific error when user starts typing
		if (fieldErrors[name as keyof typeof fieldErrors]) {
			setFieldErrors((prev) => ({
				...prev,
				[name]: '',
			}))
		}
	}

	const handleSelectChange = (e: SelectChangeEvent<string[]>) => {
		const { name, value } = e.target
		setFormData((prev) => ({
			...prev,
			[name]: value,
		}))

		// Clear resorts error when user selects resorts
		if (name === 'resorts' && fieldErrors.resorts) {
			setFieldErrors((prev) => ({
				...prev,
				resorts: '',
			}))
		}
	}

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault()

		// Clear previous field errors
		setFieldErrors({
			email: '',
			phone: '',
			resorts: '',
		})
		setError('')

		// Validate form fields
		const errors = {
			email: '',
			phone: '',
			resorts: '',
		}

		if (!formData.email.trim()) {
			errors.email = 'Email is required'
		} else if (!/\S+@\S+\.\S+/.test(formData.email)) {
			errors.email = 'Please enter a valid email address'
		}

		if (!formData.phone.trim()) {
			errors.phone = 'Phone number is required'
		}

		if (formData.resorts.length === 0) {
			errors.resorts = 'Please select at least one resort'
		}

		// If there are validation errors, show them and don't submit
		if (errors.email || errors.phone || errors.resorts) {
			setFieldErrors(errors)
			return
		}

		setIsLoading(true)

		try {
			// Store the form data in localStorage to use after email verification
			localStorage.setItem('signup_data', JSON.stringify({
				phone: formData.phone.trim(),
				notificationDays: formData.notificationDays,
				minSnowAmount: formData.minSnowAmount,
				resorts: formData.resorts,
			}))

			// Send magic link for signup
			const response = await fetch('/api/auth/magic-link', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({
					email: formData.email.trim(),
					purpose: 'signup'
				}),
			})

			const data = await response.json()

			if (response.ok) {
				setStep('email-sent')
			} else {
				setError(data.message || 'Failed to send verification email')
			}
		} catch (err) {
			setError('Network error. Please try again.')
		} finally {
			setIsLoading(false)
		}
	}

	if (step === 'email-sent') {
		return (
			<Container maxWidth="md" sx={{ py: 4 }}>
				<Paper elevation={3} sx={{ p: 4, textAlign: 'center' }}>
					<Typography variant="h4" component="h1" gutterBottom color="primary">
						Check Your Email!
					</Typography>
					
					<Alert severity="success" sx={{ mb: 3, textAlign: 'left' }}>
						We've sent a verification link to <strong>{formData.email}</strong>
					</Alert>

					<Typography variant="body1" sx={{ mb: 3 }}>
						Click the link in your email to verify your account and complete your powder alert setup.
					</Typography>

					<Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
						The email should arrive within a few minutes. Don't forget to check your spam folder!
					</Typography>

					<Box sx={{ mt: 4 }}>
						<Button 
							variant="outlined" 
							onClick={() => setStep('form')}
							sx={{ mr: 2 }}
						>
							← Back to Form
						</Button>
						<Button component={Link} to="/" variant="text">
							Return Home
						</Button>
					</Box>
				</Paper>
			</Container>
		)
	}

	if (loading) {
		return (
			<Container maxWidth="md" sx={{ py: 4 }}>
				<LinearProgress />
				<Typography variant="h6" sx={{ mt: 2, textAlign: 'center' }}>
					Loading resorts...
				</Typography>
			</Container>
		)
	}

	if (resortsError) {
		return (
			<Container maxWidth="md" sx={{ py: 4 }}>
				<Alert severity="error">
					Failed to load resort data. Please refresh the page to try again.
				</Alert>
			</Container>
		)
	}

	return (
		<Container maxWidth="md" sx={{ py: 4 }}>
			<Paper elevation={3} sx={{ p: 4 }}>
				<Typography variant="h2" component="h1" gutterBottom>
					Get Powder Alerts
				</Typography>
				<Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
					Never miss a powder day again! Set up custom alerts for your favorite resorts.
				</Typography>

				{error && (
					<Alert severity="error" sx={{ mb: 3 }}>
						{error}
					</Alert>
				)}

				<form onSubmit={handleSubmit}>
					<Grid container spacing={3}>
						<Grid item xs={12} md={6}>
							<TextField
								fullWidth
								label="Email Address"
								name="email"
								type="email"
								value={formData.email}
								onChange={handleChange}
								required
								error={!!fieldErrors.email}
								helperText={fieldErrors.email}
								disabled={isLoading}
							/>
						</Grid>

						<Grid item xs={12} md={6}>
							<TextField
								fullWidth
								label="Phone Number"
								name="phone"
								type="tel"
								value={formData.phone}
								onChange={handleChange}
								required
								error={!!fieldErrors.phone}
								helperText={fieldErrors.phone || "We'll send SMS alerts to this number"}
								disabled={isLoading}
							/>
						</Grid>

						<Grid item xs={12}>
							<FormControl fullWidth error={!!fieldErrors.resorts} disabled={isLoading}>
								<InputLabel id="resorts-label">Select Resorts *</InputLabel>
								<Select
									labelId="resorts-label"
									name="resorts"
									multiple
									value={formData.resorts}
									onChange={handleSelectChange}
									label="Select Resorts *"
								>
									{resorts.map((resort) => (
										<MenuItem key={resort.uuid} value={resort.name}>
											{resort.name}
										</MenuItem>
									))}
								</Select>
								{fieldErrors.resorts && (
									<Typography variant="caption" color="error" sx={{ mt: 1 }}>
										{fieldErrors.resorts}
									</Typography>
								)}
							</FormControl>
						</Grid>

						<Grid item xs={12} md={6}>
							<Typography id="min-snow-label" gutterBottom>
								Minimum Snow Amount: {formData.minSnowAmount}"
							</Typography>
							<Slider
								name="minSnowAmount"
								value={formData.minSnowAmount}
								onChange={(_, value) =>
									setFormData((prev) => ({
										...prev,
										minSnowAmount: value as number,
									}))
								}
								min={1}
								max={24}
								step={1}
								valueLabelDisplay="auto"
								aria-labelledby="min-snow-label"
								disabled={isLoading}
							/>
						</Grid>

						<Grid item xs={12} md={6}>
							<Typography id="notification-days-label" gutterBottom>
								Days in Advance: {formData.notificationDays}
							</Typography>
							<Slider
								name="notificationDays"
								value={formData.notificationDays}
								onChange={(_, value) =>
									setFormData((prev) => ({
										...prev,
										notificationDays: value as number,
									}))
								}
								min={1}
								max={7}
								step={1}
								valueLabelDisplay="auto"
								aria-labelledby="notification-days-label"
								disabled={isLoading}
							/>
						</Grid>

						<Grid item xs={12}>
							<Box sx={{ display: 'flex', gap: 2, justifyContent: 'center' }}>
								<Button
									type="submit"
									variant="contained"
									size="large"
									disabled={isLoading}
									sx={{ minWidth: 200 }}
								>
									{isLoading ? (
										<>
											<CircularProgress size={20} sx={{ mr: 1 }} />
											Sending Verification Email...
										</>
									) : (
										'Get Started'
									)}
								</Button>
								<Button component={Link} to="/" variant="outlined" size="large">
									Cancel
								</Button>
							</Box>
						</Grid>
					</Grid>
				</form>

				<Box sx={{ mt: 4, textAlign: 'center' }}>
					<Typography variant="body2" color="text.secondary">
						Already have an account?{' '}
						<Button component={Link} to="/login" variant="text">
							Sign In
						</Button>
					</Typography>
				</Box>
			</Paper>
		</Container>
	)
}