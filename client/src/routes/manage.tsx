import { useEffect, useState } from 'react'
import {
	Alert,
	Box,
	Button,
	Card,
	CardContent,
	Chip,
	CircularProgress,
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
import { Delete, DeleteSweep } from '@mui/icons-material'
import { Link, useNavigate } from 'react-router'
import {
	useUserAlerts,
	useDeleteAlert,
	useDeleteAllAlerts,
} from '../shared/useManageAlerts.ts'
import { useCurrentUser, useLogout } from '../shared/useAuth.ts'
import { UserAlert } from '../shared/types.ts'

// created_at arrives from Go as a nullable timestamp, so guard both the null
// case and an unparseable value rather than rendering "Invalid Date".
function formatCreatedAt(createdAt: UserAlert['created_at']): string {
	if (!createdAt?.Valid) {
		return 'Unknown'
	}

	const parsed = new Date(createdAt.Time)
	if (Number.isNaN(parsed.getTime())) {
		return 'Unknown'
	}

	return parsed.toLocaleDateString()
}

export default function ManageSubscriptionsPage() {
	const navigate = useNavigate()
	const { user, loading: userLoading } = useCurrentUser()
	const { logout, loading: loggingOut } = useLogout()

	const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false)
	const [deleteAllConfirmOpen, setDeleteAllConfirmOpen] = useState(false)
	const [alertToDelete, setAlertToDelete] = useState<{
		resortUuid: string
		resortName: string
	} | null>(null)

	// This page only ever shows the signed-in user's own subscriptions, so a
	// signed-out visitor goes to the sign-in screen rather than a blank list.
	useEffect(() => {
		if (!userLoading && !user) {
			navigate('/login', { replace: true })
		}
	}, [userLoading, user, navigate])

	const { data: alerts = [], isLoading, error } = useUserAlerts(!!user)

	const deleteAlertMutation = useDeleteAlert()
	const deleteAllMutation = useDeleteAllAlerts()

	const handleDeleteClick = (resortUuid: string, resortName: string) => {
		setAlertToDelete({ resortUuid, resortName })
		setDeleteConfirmOpen(true)
	}

	const handleDeleteConfirm = () => {
		if (!alertToDelete) {
			return
		}

		deleteAlertMutation.mutate(alertToDelete.resortUuid, {
			onSuccess: () => {
				setDeleteConfirmOpen(false)
				setAlertToDelete(null)
			},
		})
	}

	const handleDeleteAllConfirm = () => {
		deleteAllMutation.mutate(undefined, {
			onSuccess: () => {
				setDeleteAllConfirmOpen(false)
			},
		})
	}

	if (userLoading || !user) {
		return (
			<Container maxWidth="md" sx={{ py: 8, textAlign: 'center' }}>
				<CircularProgress />
			</Container>
		)
	}

	return (
		<Container maxWidth="md" sx={{ py: 4 }}>
			<Paper elevation={3} sx={{ p: 4 }}>
				<Box
					sx={{
						display: 'flex',
						justifyContent: 'space-between',
						alignItems: 'flex-start',
						gap: 2,
						mb: 2,
					}}
				>
					<Box>
						<Typography variant="h2" component="h1" gutterBottom>
							Manage Your Subscriptions
						</Typography>
						<Typography variant="body1" color="text.secondary">
							Signed in as {user.email}
						</Typography>
					</Box>
					<Button
						variant="text"
						onClick={() =>
							logout(undefined, {
								onSuccess: () => navigate('/login', { replace: true }),
							})
						}
						disabled={loggingOut}
					>
						Sign out
					</Button>
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

				{isLoading && (
					<Box sx={{ textAlign: 'center', py: 4 }}>
						<CircularProgress />
					</Box>
				)}

				{!isLoading && alerts.length === 0 && !error && (
					<Alert severity="info" sx={{ mb: 3 }}>
						No active subscriptions yet.
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
													Created: {formatCreatedAt(alert.created_at)}
												</Typography>
											</Box>
											<IconButton
												color="error"
												aria-label={`Delete subscription for ${alert.resort_name}`}
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
