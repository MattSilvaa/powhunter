import React, { useEffect, useRef, useState } from 'react'
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
import { useNavigate, useSearchParams } from 'react-router'
import { useResorts } from '../shared/useResorts.ts'
import { useCreateAlert } from '../shared/useCreateAlert.ts'
import PageHeader from '../components/pageHeader.tsx'
import ResortSelect from '../components/resortSelect.tsx'
import AlertSettingsFields from '../components/alertSettingsFields.tsx'
import { resortUuidsFromLink } from '../shared/resortSlug.ts'
import { readAttribution } from '../shared/attribution.ts'
import { pageMeta } from '../shared/seo.ts'

export function meta() {
	return pageMeta({
		title: 'Create a powder alert | Pow Hunter',
		description:
			'Choose your resorts and how much fresh snow it takes. We text you when the forecast meets your number.',
		path: '/signup',
	})
}

export default function SignUpPage() {
	const navigate = useNavigate()
	// A friend arriving from a shared link lands here with resorts, and for a
	// shared alert its code, in the URL.
	const [searchParams] = useSearchParams()
	const sharedResorts = searchParams.get('resort') ?? ''
	const referralCode = searchParams.get('ref') ?? ''
	const prefilled = useRef(false)
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

	// Pick the shared resorts once the list has loaded, and only once: doing it
	// on every render would put them back each time the visitor unticked one.
	// A link names resorts by uuid (shared alerts) or by name, as in
	// /signup?resort=alta,snowbird.
	useEffect(() => {
		if (prefilled.current || !sharedResorts || resorts.length === 0) {
			return
		}

		prefilled.current = true
		const linked = resortUuidsFromLink(sharedResorts.split(','), resorts)
		if (linked.length > 0) {
			setFormData((prev) => ({ ...prev, resorts: linked }))
		}
	}, [resorts, sharedResorts])

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
				...(referralCode && { referralCode }),
				...readAttribution(),
			},
			{
				onSuccess: () => {
					// The success page invites friends to the same resorts.
					const resortNames = resorts
						.filter((r) => resortsUuids.includes(r.uuid))
						.map((r) => r.name)
					navigate('/success', { state: { resortNames } })
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
