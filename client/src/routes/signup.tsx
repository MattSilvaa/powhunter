import React, { useState } from 'react'
import {
	Alert,
	Box,
	Button,
	Checkbox,
	Chip,
	Container,
	FormControl,
	FormHelperText,
	Grid,
	InputLabel,
	LinearProgress,
	ListItemText,
	MenuItem,
	Paper,
	Select,
	SelectChangeEvent,
	Slider,
	TextField,
	Typography,
} from '@mui/material'
import { useNavigate } from 'react-router'
import { useResorts } from '../shared/useResorts.ts'
import { Resort } from '../shared/types.ts'
import { useCreateAlert } from '../shared/useCreateAlert.ts'
import PageHeader from '../components/pageHeader.tsx'
import { tabularNums } from '../theme.ts'

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
									<FormControl fullWidth error={!!fieldErrors.resorts}>
										<InputLabel id="resorts-label">Resorts</InputLabel>
										{error && (
											<Alert severity="error" sx={{ mb: 1 }}>
												{error}
											</Alert>
										)}

										<Select
											required
											multiple
											labelId="resorts-label"
											name="resorts"
											value={formData.resorts}
											onChange={handleSelectChange}
											label="Resorts"
											MenuProps={{
												slotProps: { paper: { sx: { maxHeight: 320 } } },
											}}
											// Values are uuids so resorts sharing a name cannot
											// collide, but the field must still read as names.
											renderValue={(selected) => (
												<Box
													sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}
												>
													{selected.map((uuid) => (
														<Chip
															key={uuid}
															size="small"
															label={
																resorts.find((r) => r.uuid === uuid)?.name ??
																uuid
															}
															sx={{
																bgcolor: 'primary.light',
																color: 'primary.dark',
															}}
														/>
													))}
												</Box>
											)}
										>
											{resorts.map((resort: Resort) => (
												<MenuItem key={resort.uuid} value={resort.uuid} dense>
													<Checkbox
														size="small"
														checked={formData.resorts.includes(resort.uuid)}
														sx={{ py: 0.5, pl: 0 }}
													/>
													<ListItemText primary={resort.name} />
												</MenuItem>
											))}
										</Select>
										<FormHelperText>
											{fieldErrors.resorts || 'Pick as many as you like'}
										</FormHelperText>
									</FormControl>
								</Grid>

								<Grid size={12}>
									<SliderField
										label="Minimum snowfall"
										hint="Only alert me when at least this much is forecast"
										value={`${formData.minSnowAmount}″`}
									>
										<Slider
											aria-label="Minimum snow amount in inches"
											value={formData.minSnowAmount}
											onChange={(_, value) =>
												setFormData((prev) => ({
													...prev,
													minSnowAmount: value as number,
												}))
											}
											min={1}
											max={24}
											valueLabelDisplay="off"
										/>
									</SliderField>
								</Grid>

								<Grid size={12}>
									<SliderField
										label="Advance notice"
										hint="How far ahead of the snow you want to hear about it"
										value={`${formData.notificationDays} ${formData.notificationDays === 1 ? 'day' : 'days'}`}
									>
										<Slider
											aria-label="Days of advance notice"
											value={formData.notificationDays}
											onChange={(_, value) =>
												setFormData((prev) => ({
													...prev,
													notificationDays: value as number,
												}))
											}
											min={1}
											max={10}
											valueLabelDisplay="off"
										/>
									</SliderField>
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

type SliderFieldProps = {
	label: string
	hint: string
	value: string
	children: React.ReactNode
}

// Shows a slider's current value beside its label, like a forecast readout,
// instead of hiding it in a tooltip.
function SliderField({
	label,
	hint,
	value,
	children,
}: SliderFieldProps): React.ReactElement {
	return (
		<Box>
			<Box
				sx={{
					display: 'flex',
					justifyContent: 'space-between',
					alignItems: 'flex-start',
					gap: 2,
				}}
			>
				<Box>
					<Typography sx={{ fontWeight: 500 }}>{label}</Typography>
					<Typography variant="body2" color="text.secondary">
						{hint}
					</Typography>
				</Box>
				<Typography
					aria-hidden
					sx={{
						...tabularNums,
						fontSize: '1.5rem',
						fontWeight: 600,
						letterSpacing: '-0.02em',
						whiteSpace: 'nowrap',
					}}
				>
					{value}
				</Typography>
			</Box>
			<Box sx={{ px: 1, mt: 1 }}>{children}</Box>
		</Box>
	)
}
