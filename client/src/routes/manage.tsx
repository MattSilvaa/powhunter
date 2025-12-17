import React, { useState, useEffect } from 'react'
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
	Typography,
} from '@mui/material'
import { Delete, DeleteSweep, Logout } from '@mui/icons-material'
import { Link, useNavigate } from 'react-router'
import {
	useUserAlerts,
	useDeleteAlert,
	useDeleteAllAlerts,
} from '../shared/useManageAlerts.ts'
import { useAuth } from '../shared/useAuth.ts'

export default function ManageSubscriptionsPage() {
	const navigate = useNavigate()
	const { user, token, isAuthenticated, logout, isLoading: authLoading } = useAuth()
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
	} = useUserAlerts(token)

	const deleteAlertMutation = useDeleteAlert()
	const deleteAllMutation = useDeleteAllAlerts()

	// Redirect to login if not authenticated
	useEffect(() => {
		if (!authLoading && !isAuthenticated) {
			navigate('/login')
		}
	}, [isAuthenticated, authLoading, navigate])

	const handleDeleteClick = (resortUuid: string, resortName: string) => {
		setAlertToDelete({ resortUuid, resortName })
		setDeleteConfirmOpen(true)
	}

	const handleDeleteConfirm = () => {
		if (alertToDelete && token) {
			deleteAlertMutation.mutate(
				{
					token,
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
		if (token) {
			deleteAllMutation.mutate(token, {
				onSuccess: () => {
					setDeleteAllConfirmOpen(false)
				},
			})
		}
	}

	const handleLogout = async () => {
		await logout()
		navigate('/login')
	}

	if (authLoading) {
		return (
			<Container maxWidth="md" sx={{ py: 4, textAlign: 'center' }}>
				<Typography>Loading...</Typography>
			</Container>
		)
	}

	if (!isAuthenticated) {
		return null // Will redirect via useEffect
	}

	return (
		<Container maxWidth="md" sx={{ py: 4 }}>
			<Paper elevation={3} sx={{ p: 4 }}>
				<Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
					<Typography variant="h2" component="h1">
						Manage Your Subscriptions
					</Typography>
					<Button
						variant="outlined"
						startIcon={<Logout />}
						onClick={handleLogout}
						size="small"
					>
						Logout
					</Button>
				</Box>

				<Typography variant="body1" color="text.secondary" sx={{ mb: 3 }}>
					Welcome back, {user?.email}! Here you can view and manage your powder alert subscriptions.
				</Typography>

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

				{!isLoading && alerts.length === 0 && !error && (
					<Alert severity="info" sx={{ mb: 3 }}>
						You don't have any active subscriptions yet.
						<Button component={Link} to="/signup" sx={{ ml: 1 }}>
							Create one?
						</Button>
					</Alert>
				)}

				{isLoading && (
					<Box sx={{ textAlign: 'center', py: 4 }}>
						<Typography>Loading your subscriptions...</Typography>
					</Box>
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
													Created: {new Date(alert.created_at.Time).toLocaleDateString()}
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