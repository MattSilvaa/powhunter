import React, { useState } from 'react'
import {
	Alert,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
} from '@mui/material'
import AlertSettingsFields from './alertSettingsFields.tsx'
import { useUpdateAlert } from '../shared/useManageAlerts.ts'
import { AlertSettings, UserAlert } from '../shared/types.ts'

type EditAlertDialogProps = {
	alert: UserAlert
	onClose: () => void
}

// Mounted only while open, so each edit starts from the alert's saved values.
export default function EditAlertDialog({
	alert,
	onClose,
}: EditAlertDialogProps): React.ReactElement {
	const [settings, setSettings] = useState<AlertSettings>({
		notificationDays: alert.notification_days,
		minSnowAmount: alert.min_snow_amount,
	})
	const { mutate, isPending, error } = useUpdateAlert()

	const handleSave = () => {
		mutate(
			{ resortUuid: alert.resort_uuid, ...settings },
			{ onSuccess: onClose }
		)
	}

	return (
		<Dialog open onClose={onClose} fullWidth maxWidth="xs">
			<DialogTitle>Edit {alert.resort_name}</DialogTitle>
			<DialogContent>
				{error && (
					<Alert severity="error" sx={{ mb: 3 }}>
						{error.message}
					</Alert>
				)}
				<AlertSettingsFields value={settings} onChange={setSettings} />
			</DialogContent>
			<DialogActions sx={{ px: 3, pb: 2.5 }}>
				<Button onClick={onClose} sx={{ color: 'text.secondary' }}>
					Cancel
				</Button>
				<Button onClick={handleSave} variant="contained" disabled={isPending}>
					{isPending ? 'Saving…' : 'Save'}
				</Button>
			</DialogActions>
		</Dialog>
	)
}
