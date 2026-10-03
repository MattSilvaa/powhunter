import React, { useState } from 'react'
import {
	Alert,
	Box,
	Button,
	CircularProgress,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	Typography,
} from '@mui/material'
import AlertSettingsFields from './alertSettingsFields.tsx'
import ResortSelect from './resortSelect.tsx'
import { useResorts } from '../shared/useResorts.ts'
import { useCreateUserAlerts } from '../shared/useManageAlerts.ts'
import { AlertSettings } from '../shared/types.ts'

type AddAlertsDialogProps = {
	// Resorts the user already has an alert for, which cannot be added twice.
	existingResortUuids: string[]
	onClose: () => void
}

const DEFAULT_SETTINGS: AlertSettings = {
	notificationDays: 3,
	minSnowAmount: 6,
}

export default function AddAlertsDialog({
	existingResortUuids,
	onClose,
}: AddAlertsDialogProps): React.ReactElement {
	const { resorts, loading, error: resortsError } = useResorts()
	const [selected, setSelected] = useState<string[]>([])
	const [settings, setSettings] = useState<AlertSettings>(DEFAULT_SETTINGS)
	const [selectionError, setSelectionError] = useState('')
	const { mutate, isPending, error } = useCreateUserAlerts()

	const available = resorts.filter(
		(resort) => !existingResortUuids.includes(resort.uuid)
	)
	const allFollowed = !loading && !resortsError && available.length === 0

	const handleSave = () => {
		if (selected.length === 0) {
			setSelectionError('Please select at least one resort')
			return
		}

		mutate({ resortsUuids: selected, ...settings }, { onSuccess: onClose })
	}

	return (
		<Dialog open onClose={onClose} fullWidth maxWidth="xs">
			<DialogTitle>Add resorts</DialogTitle>
			<DialogContent>
				{(error || resortsError) && (
					<Alert severity="error" sx={{ mb: 3 }}>
						{error?.message ?? resortsError}
					</Alert>
				)}

				{loading && (
					<Box sx={{ textAlign: 'center', py: 4 }}>
						<CircularProgress size={28} />
					</Box>
				)}

				{allFollowed && (
					<Typography color="text.secondary">
						You already have an alert for every resort we track.
					</Typography>
				)}

				{!loading && available.length > 0 && (
					<Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, pt: 1 }}>
						<ResortSelect
							id="add-resorts"
							resorts={available}
							value={selected}
							onChange={(next) => {
								setSelected(next)
								setSelectionError('')
							}}
							error={selectionError}
						/>
						<AlertSettingsFields value={settings} onChange={setSettings} />
					</Box>
				)}
			</DialogContent>
			<DialogActions sx={{ px: 3, pb: 2.5 }}>
				<Button onClick={onClose} sx={{ color: 'text.secondary' }}>
					{allFollowed ? 'Close' : 'Cancel'}
				</Button>
				{!allFollowed && (
					<Button
						onClick={handleSave}
						variant="contained"
						disabled={isPending || loading}
					>
						{isPending ? 'Adding…' : 'Add alerts'}
					</Button>
				)}
			</DialogActions>
		</Dialog>
	)
}
