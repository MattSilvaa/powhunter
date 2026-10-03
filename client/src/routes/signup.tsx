import React, { useState } from 'react'
import {
	Alert,
	Box,
	Button,
	Container,
	Grid,
	LinearProgress,
	Paper,
	TextField,
} from '@mui/material'
import { useNavigate } from 'react-router'
import { useResorts } from '../shared/useResorts.ts'
import { useCreateAlert } from '../shared/useCreateAlert.ts'
import PageHeader from '../components/pageHeader.tsx'
import ResortSelect from '../components/resortSelect.tsx'
import AlertSettingsFields from '../components/alertSettingsFields.tsx'

export default function SignUpPage() {
	const navigate = useNavigate()
	const [formData, setFormData] = useState({
		email: '',
		phone: '',
		notificationDays: 3,
		minSnowAmount: 6,
		resorts: [] as string[],
	})
	const [dismissedError, setDismissedError] = useState(false)
	const [fieldErrors, setFieldErrors] = useState({
		email: '',
		phone: '',
		resorts: '',
	})
	const {
		createAlert,
		loading: isCreateAlertLoading,
		error: createAlertError,
	} = useCreateAlert()

	const { resorts = [], loading, error } = useResorts()

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

	const handleResortsChange = (selected: string[]) => {
		setFormData((prev) => ({ ...prev, resorts: selected }))

		// Clear resorts error when user selects resorts
		if (fieldErrors.resorts) {
			setFieldErrors((prev) => ({ ...prev, resorts: '' }))
		}
	}

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault()

		setDismissedError(false)

		// Clear previous field errors
		setFieldErrors({
			email: '',
			phone: '',
			resorts: '',
		})

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
		} else if (formData.phone.replace(/\D/g, '').length < 10) {
			errors.phone = 'Please enter a valid phone number'
		}

		if (formData.resorts.length === 0) {
			errors.resorts = 'Please select at least one resort'
		}

		// If there are validation errors, show them and don't submit
		if (errors.email || errors.phone || errors.resorts) {
			setFieldErrors(errors)
			return
		}

		const resortsUuids = formData.resorts.filter((uuid) =>
			resorts.some((r) => r.uuid === uuid)
		)

		if (resortsUuids.length !== formData.resorts.length) {
			setFieldErrors((prev) => ({
				...prev,
				resorts:
					'Some selected resorts are no longer available. Please reselect them.',
			}))
			return
		}

		createAlert(
			{
				email: formData.email.trim(),
				phone: formData.phone.trim(),
				minSnowAmount: formData.minSnowAmount,
				notificationDays: formData.notificationDays,
				resortsUuids,
			},
			{
				onSuccess: () => {
					navigate('/success')
				},
				onError: (error) => {
					console.error('Failed to create alert:', error)
				},
			}
		)
	}

	return (
		<Container maxWidth="sm" sx={{ py: { xs: 5, md: 8 } }}>
			<PageHeader
				title="Create an alert"
				subtitle="Tell us where you ride and what counts as a powder day. We'll text you when the forecast delivers."
			/>

			<Paper variant="outlined" sx={{ p: { xs: 2.5, sm: 4 } }}>
				{loading ? (
					<LinearProgress />
				) : (
					<>
						{createAlertError && !dismissedError && (
							<Alert
								severity="error"
								role="alert"
								sx={{ mb: 3 }}
								onClose={() => setDismissedError(true)}
							>
								<strong>Oops!</strong> {createAlertError}
							</Alert>
						)}

						<Box component="form" onSubmit={handleSubmit} noValidate>
							<Grid container spacing={3}>
								<Grid size={{ xs: 12, sm: 6 }}>
									<TextField
										required
										fullWidth
										label="Email"
										name="email"
										type="email"
										autoComplete="email"
										value={formData.email}
										onChange={handleChange}
										error={!!fieldErrors.email}
										helperText={
											fieldErrors.email || 'Used to sign in and manage alerts'
										}
									/>
								</Grid>
								<Grid size={{ xs: 12, sm: 6 }}>
									<TextField
										required
										fullWidth
										label="Phone Number"
										name="phone"
										type="tel"
										autoComplete="tel"
										value={formData.phone}
										onChange={handleChange}
										error={!!fieldErrors.phone}
										helperText={
											fieldErrors.phone || 'Where we text your alerts'
										}
									/>
								</Grid>

								<Grid size={12}>
									{error && (
										<Alert severity="error" sx={{ mb: 1 }}>
											{error}
										</Alert>
									)}
									<ResortSelect
										id="resorts"
										resorts={resorts}
										value={formData.resorts}
										onChange={handleResortsChange}
										error={fieldErrors.resorts}
									/>
								</Grid>

								<Grid size={12}>
									<AlertSettingsFields
										value={{
											minSnowAmount: formData.minSnowAmount,
											notificationDays: formData.notificationDays,
										}}
										onChange={(settings) =>
											setFormData((prev) => ({ ...prev, ...settings }))
										}
									/>
								</Grid>

								<Grid size={12}>
									<Button
										type="submit"
										variant="contained"
										size="large"
										fullWidth
										disabled={isCreateAlertLoading}
									>
										{isCreateAlertLoading
											? 'Creating Alert...'
											: 'Create Alert'}
									</Button>
								</Grid>
							</Grid>
						</Box>
					</>
				)}
			</Paper>
		</Container>
	)
}
