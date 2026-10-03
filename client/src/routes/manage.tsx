import { useEffect, useState } from 'react'
import {
	Alert,
	Box,
	Button,
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
import { AcUnit, Add, DeleteOutline } from '@mui/icons-material'
import { Link, useNavigate } from 'react-router'
import {
	useUserAlerts,
	useDeleteAlert,
	useDeleteAllAlerts,
} from '../shared/useManageAlerts.ts'
import { useCurrentUser, useLogout } from '../shared/useAuth.ts'
import { UserAlert } from '../shared/types.ts'
import PageHeader from '../components/pageHeader.tsx'
import { tabularNums } from '../theme.ts'

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
		<Container maxWidth="sm" sx={{ py: { xs: 5, md: 8 } }}>
			<PageHeader
				title="Your alerts"
				subtitle={`Signed in as ${user.email}`}
				action={
					<Button
						variant="outlined"
						size="small"
						onClick={() =>
							logout(undefined, {
								onSuccess: () => navigate('/login', { replace: true }),
							})
						}
						disabled={loggingOut}
						sx={{ flexShrink: 0, mt: 0.5 }}
					>
						Sign out
					</Button>
				}
			/>

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
				<Box sx={{ textAlign: 'center', py: 6 }}>
					<CircularProgress size={28} />
				</Box>
			)}

			{!isLoading && alerts.length === 0 && !error && (
				<Paper variant="outlined" sx={{ py: 6, px: 3, textAlign: 'center' }}>
					<AcUnit sx={{ fontSize: 28, color: 'text.disabled', mb: 1.5 }} />
					<Typography variant="h6" component="h2">
						No alerts yet
					</Typography>
					<Typography color="text.secondary" sx={{ mt: 0.5, mb: 3 }}>
						Create one and we&apos;ll text you when it&apos;s about to dump.
					</Typography>
					<Button component={Link} to="/signup" variant="contained">
						Create an alert
					</Button>
				</Paper>
			)}

			{alerts.length > 0 && (
				<>
					<Box
						sx={{
							display: 'flex',
							justifyContent: 'space-between',
							alignItems: 'center',
							mb: 1.5,
						}}
					>
						<Typography
							variant="overline"
							component="h2"
							color="text.secondary"
							sx={tabularNums}
						>
							{alerts.length} active
						</Typography>
						<Button
							size="small"
							color="error"
							onClick={() => setDeleteAllConfirmOpen(true)}
							disabled={deleteAllMutation.isPending}
						>
							Delete all
						</Button>
					</Box>

					<Paper variant="outlined" sx={{ overflow: 'hidden' }}>
						{alerts.map((alert, i) => (
							<Box
								key={alert.id}
								sx={{
									display: 'flex',
									alignItems: 'center',
									gap: { xs: 2, sm: 2.5 },
									px: { xs: 2, sm: 3 },
									py: 2.25,
									borderTop: i === 0 ? 0 : 1,
									borderColor: 'divider',
								}}
							>
								<Typography
									aria-hidden
									sx={{
										...tabularNums,
										minWidth: 56,
										fontSize: '1.75rem',
										fontWeight: 600,
										letterSpacing: '-0.03em',
										lineHeight: 1,
									}}
								>
									{alert.min_snow_amount}&Prime;
								</Typography>
								<Box sx={{ flex: 1, minWidth: 0 }}>
									<Typography variant="h6" component="h3" noWrap>
										{alert.resort_name}
									</Typography>
									<Typography
										variant="body2"
										color="text.secondary"
										sx={tabularNums}
									>
										<span className="sr-only">
											{alert.min_snow_amount} inch minimum,{' '}
										</span>
										{alert.notification_days}{' '}
										{alert.notification_days === 1 ? 'day' : 'days'} notice ·
										Added {formatCreatedAt(alert.created_at)}
									</Typography>
								</Box>
								<IconButton
									aria-label={`Delete subscription for ${alert.resort_name}`}
									onClick={() =>
										handleDeleteClick(alert.resort_uuid, alert.resort_name)
									}
									disabled={deleteAlertMutation.isPending}
									sx={{
										color: 'text.secondary',
										'&:hover': {
											color: 'error.main',
											bgcolor: 'rgb(211 47 47 / 0.06)',
										},
									}}
								>
									<DeleteOutline fontSize="small" />
								</IconButton>
							</Box>
						))}
					</Paper>

					<Button
						component={Link}
						to="/signup"
						startIcon={<Add />}
						sx={{ mt: 2 }}
					>
						Add another resort
					</Button>
				</>
			)}

			<Dialog
				open={deleteConfirmOpen}
				onClose={() => setDeleteConfirmOpen(false)}
			>
				<DialogTitle>Delete this alert?</DialogTitle>
				<DialogContent>
					<DialogContentText>
						Are you sure you want to delete your subscription for{' '}
						<strong>{alertToDelete?.resortName}</strong>? You will no longer
						receive powder alerts for this resort.
					</DialogContentText>
				</DialogContent>
				<DialogActions sx={{ px: 3, pb: 2.5 }}>
					<Button
						onClick={() => setDeleteConfirmOpen(false)}
						sx={{ color: 'text.secondary' }}
					>
						Cancel
					</Button>
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
				<DialogTitle>Delete all alerts?</DialogTitle>
				<DialogContent>
					<DialogContentText>
						Are you sure you want to delete ALL your powder alert subscriptions?
						This action cannot be undone and you will no longer receive any
						powder alerts.
					</DialogContentText>
				</DialogContent>
				<DialogActions sx={{ px: 3, pb: 2.5 }}>
					<Button
						onClick={() => setDeleteAllConfirmOpen(false)}
						sx={{ color: 'text.secondary' }}
					>
						Cancel
					</Button>
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
