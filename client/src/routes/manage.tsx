import React, { useState } from 'react'
import {
	Alert,
	Box,
	Button,
	Card,
	CardContent,
	Chip,
	Container,
	Dialog,
	DialogActions,
	DialogContent,
	DialogContentText,
	DialogTitle,
	IconButton,
	Paper,
	TextField,
	Typography,
} from '@mui/material'
import { Delete, DeleteSweep, Warning } from '@mui/icons-material'
import { Link } from 'react-router'
import {
	useUserAlerts,
	useDeleteAlert,
	useDeleteAllAlerts,
} from '../shared/useManageAlerts.ts'

export default function ManageSubscriptionsPage() {
	const [email, setEmail] = useState('')
	const [searchedEmail, setSearchedEmail] = useState('')
	const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false)
	const [deleteAllConfirmOpen, setDeleteAllConfirmOpen] = useState(false)
	const [alertToDelete, setAlertToDelete] = useState<{
		resortUuid: string
		resortName: string
	} | null>(null)

	const {
		data: alerts = [],
		isLoading,
		error,
	} = useUserAlerts(searchedEmail)

	const deleteAlertMutation = useDeleteAlert()
	const deleteAllMutation = useDeleteAllAlerts()

	const handleSearch = (e: React.FormEvent) => {
		e.preventDefault()
		if (email.trim()) {
			setSearchedEmail(email.trim())
		}
	}

	const handleDeleteClick = (resortUuid: string, resortName: string) => {
		setAlertToDelete({ resortUuid, resortName })
		setDeleteConfirmOpen(true)
	}

	const handleDeleteConfirm = () => {
		if (alertToDelete && searchedEmail) {
			deleteAlertMutation.mutate(
				{
					email: searchedEmail,
					resortUuid: alertToDelete.resortUuid,
				},
				{
					onSuccess: () => {
						setDeleteConfirmOpen(false)
						setAlertToDelete(null)
					},
				}
			)
		}
	}

	const handleDeleteAllConfirm = () => {
		if (searchedEmail) {
			deleteAllMutation.mutate(searchedEmail, {
				onSuccess: () => {
					setDeleteAllConfirmOpen(false)
				},
			})
		}
	}

	return (
		<Container maxWidth="md" sx={{ py: 4 }}>
			<Paper elevation={3} sx={{ p: 4 }}>
				<Typography variant="h2" component="h1" gutterBottom>
					Manage Your Subscriptions
				</Typography>
				<Typography variant="body1" color="text.secondary" sx={{ mb: 2 }}>
					Enter your email address to view and manage your powder alert
					subscriptions.
				</Typography>

				<Alert severity="info" icon={<Warning />} sx={{ mb: 4 }}>
					<Typography variant="body2">
						<strong>Note:</strong> We're working on adding email verification
						for enhanced security. For now, only use the email address you
						signed up with.
					</Typography>
				</Alert>

				<Box component="form" onSubmit={handleSearch} sx={{ mb: 4 }}>
					<Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
						<TextField
							fullWidth
							label="Email Address"
							type="email"
							value={email}
							onChange={(e) => setEmail(e.target.value)}
							required
						/>
						<Button
							type="submit"
							variant="contained"
							size="large"
							disabled={isLoading}
						>
							{isLoading ? 'Loading...' : 'Find Subscriptions'}
						</Button>
					</Box>
				</Box>

				{error && (
					<Alert severity="error" sx={{ mb: 3 }}>
						Failed to load subscriptions. Please try again.
					</Alert>
				)}

				{deleteAlertMutation.error && (
					<Alert severity="error" sx={{ mb: 3 }}>
						Failed to delete subscription. Please try again.
					</Alert>
				)}

				{deleteAllMutation.error && (
					<Alert severity="error" sx={{ mb: 3 }}>
						Failed to delete all subscriptions. Please try again.
					</Alert>
				)}

				{searchedEmail && alerts.length === 0 && !isLoading && !error && (
					<Alert severity="info" sx={{ mb: 3 }}>
						No active subscriptions found for {searchedEmail}.
						<Button component={Link} to="/signup" sx={{ ml: 1 }}>
							Create one?
						</Button>
					</Alert>
				)}

				{alerts.length > 0 && (
					<>
						<Box
							sx={{
								display: 'flex',
								justifyContent: 'space-between',
								alignItems: 'center',
								mb: 2,
							}}
						>
							<Typography variant="h5" component="h2">
								Active Subscriptions ({alerts.length})
							</Typography>
							<Button
								variant="outlined"
								color="error"
								startIcon={<DeleteSweep />}
								onClick={() => setDeleteAllConfirmOpen(true)}
								disabled={deleteAllMutation.isPending}
							>
								Delete All
							</Button>
						</Box>

						<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
							{alerts.map((alert) => (
								<Card key={alert.id} variant="outlined">
									<CardContent>
										<Box
											sx={{
												display: 'flex',
												justifyContent: 'space-between',
												alignItems: 'flex-start',
											}}
										>
											<Box sx={{ flex: 1 }}>
												<Typography variant="h6" component="h3">
													{alert.resort_name}
												</Typography>
												<Box sx={{ mt: 1, mb: 1 }}>
													<Chip
														label={`${alert.min_snow_amount}" minimum snow`}
														size="small"
														sx={{ mr: 1 }}
													/>
													<Chip
														label={`${alert.notification_days} days notice`}
														size="small"
													/>
												</Box>
												<Typography variant="body2" color="text.secondary">
													Created: {console.log(alert)}
													{new Date(alert.created_at.Time).toLocaleDateString()}
												</Typography>
											</Box>
											<IconButton
												color="error"
												onClick={() =>
													handleDeleteClick(
														alert.resort_uuid,
														alert.resort_name
													)
												}
												disabled={deleteAlertMutation.isPending}
											>
												<Delete />
											</IconButton>
										</Box>
									</CardContent>
								</Card>
							))}
						</Box>
					</>
				)}

				<Box sx={{ mt: 4, textAlign: 'center' }}>
					<Button component={Link} to="/" variant="text">
						← Back to Home
					</Button>
				</Box>
			</Paper>

			<Dialog
				open={deleteConfirmOpen}
				onClose={() => setDeleteConfirmOpen(false)}
			>
				<DialogTitle>Delete Subscription</DialogTitle>
				<DialogContent>
					<DialogContentText>
						Are you sure you want to delete your subscription for{' '}
						<strong>{alertToDelete?.resortName}</strong>? You will no longer
						receive powder alerts for this resort.
					</DialogContentText>
				</DialogContent>
				<DialogActions>
					<Button onClick={() => setDeleteConfirmOpen(false)}>Cancel</Button>
					<Button
						onClick={handleDeleteConfirm}
						color="error"
						variant="contained"
						disabled={deleteAlertMutation.isPending}
					>
						{deleteAlertMutation.isPending ? 'Deleting...' : 'Delete'}
					</Button>
				</DialogActions>
			</Dialog>

			<Dialog
				open={deleteAllConfirmOpen}
				onClose={() => setDeleteAllConfirmOpen(false)}
			>
				<DialogTitle>Delete All Subscriptions</DialogTitle>
				<DialogContent>
					<DialogContentText>
						Are you sure you want to delete ALL your powder alert subscriptions?
						This action cannot be undone and you will no longer receive any
						powder alerts.
					</DialogContentText>
				</DialogContent>
				<DialogActions>
					<Button onClick={() => setDeleteAllConfirmOpen(false)}>Cancel</Button>
					<Button
						onClick={handleDeleteAllConfirm}
						color="error"
						variant="contained"
						disabled={deleteAllMutation.isPending}
					>
						{deleteAllMutation.isPending ? 'Deleting...' : 'Delete All'}
					</Button>
				</DialogActions>
			</Dialog>
		</Container>
	)
}
