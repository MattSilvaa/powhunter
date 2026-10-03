import React, { useState } from 'react'
import {
	Alert,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	TextField,
} from '@mui/material'
import { useUpdateProfile } from '../shared/useAuth.ts'

type PhoneDialogProps = {
	currentPhone?: string
	onClose: () => void
}

export default function PhoneDialog({
	currentPhone,
	onClose,
}: PhoneDialogProps): React.ReactElement {
	const [phone, setPhone] = useState(currentPhone ?? '')
	const [phoneError, setPhoneError] = useState('')
	const { updateProfile, loading, error } = useUpdateProfile()

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault()

		if (phone.replace(/\D/g, '').length < 10) {
			setPhoneError('Please enter a valid phone number')
			return
		}

		updateProfile(phone.trim(), { onSuccess: onClose })
	}

	return (
		<Dialog open onClose={onClose} fullWidth maxWidth="xs">
			<form onSubmit={handleSubmit} noValidate>
				<DialogTitle>
					{currentPhone ? 'Change phone number' : 'Add a phone number'}
				</DialogTitle>
				<DialogContent>
					{error && (
						<Alert severity="error" sx={{ mb: 2 }}>
							{error}
						</Alert>
					)}
					<TextField
						autoFocus
						fullWidth
						label="Phone Number"
						type="tel"
						autoComplete="tel"
						value={phone}
						onChange={(e) => {
							setPhone(e.target.value)
							setPhoneError('')
						}}
						error={!!phoneError}
						helperText={phoneError || 'Where we text your alerts'}
						sx={{ mt: 1 }}
					/>
				</DialogContent>
				<DialogActions sx={{ px: 3, pb: 2.5 }}>
					<Button onClick={onClose} sx={{ color: 'text.secondary' }}>
						Cancel
					</Button>
					<Button type="submit" variant="contained" disabled={loading}>
						{loading ? 'Saving…' : 'Save'}
					</Button>
				</DialogActions>
			</form>
		</Dialog>
	)
}
