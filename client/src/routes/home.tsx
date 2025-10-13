import { Link } from 'react-router'
import {
	Box,
	Button,
	Card,
	CardContent,
	Container,
	Grid,
	Typography,
	useTheme,
} from '@mui/material'
import {
	Favorite as FavoriteIcon,
	Notifications as NotificationsIcon,
	Rocket as RocketIcon,
} from '@mui/icons-material'
import React from 'react'

export default function Home() {
	const theme = useTheme()

	return (
		<Box sx={{ minHeight: '100vh' }}>
			<Box
				sx={{
					background: `linear-gradient(135deg, ${theme.palette.primary.main} 0%, ${theme.palette.primary.light} 50%, ${theme.palette.secondary.main} 100%)`,
					color: 'white',
					py: 12,
					textAlign: 'center',
					position: 'relative',
					overflow: 'hidden',
					'&::before': {
						content: '""',
						position: 'absolute',
						top: 0,
						left: 0,
						right: 0,
						bottom: 0,
						background:
							'radial-gradient(circle at 30% 20%, rgba(255,255,255,0.1) 0%, transparent 50%)',
					},
				}}
			>
				<Container maxWidth="md" sx={{ position: 'relative', zIndex: 1 }}>
					<Typography
						variant="h1"
						component="h1"
						sx={{
							fontSize: { xs: '2.5rem', md: '4rem' },
							fontWeight: 800,
							mb: 3,
							background:
								'linear-gradient(135deg, #ffffff 0%, rgba(255,255,255,0.8) 100%)',
							WebkitBackgroundClip: 'text',
							WebkitTextFillColor: 'transparent',
							backgroundClip: 'text',
						}}
					>
						Pow Hunter
					</Typography>
					<Typography
						variant="h5"
						sx={{
							mb: 6,
							opacity: 0.95,
							fontSize: { xs: '1.25rem', md: '1.5rem' },
							fontWeight: 400,
							maxWidth: '600px',
							mx: 'auto',
						}}
					>
						Never miss a powder day at your favorite resort
					</Typography>
					<Box
						sx={{
							display: 'flex',
							gap: 3,
							justifyContent: 'center',
							flexWrap: 'wrap',
						}}
					>
						<Button
							component={Link}
							to="/signup"
							variant="contained"
							size="large"
							sx={{
								bgcolor: 'rgba(255,255,255,0.15)',
								color: 'white',
								border: '1px solid rgba(255,255,255,0.2)',
								backdropFilter: 'blur(10px)',
								px: 4,
								py: 1.5,
								fontSize: '1.1rem',
								'&:hover': {
									bgcolor: 'rgba(255,255,255,0.25)',
									transform: 'translateY(-2px)',
								},
							}}
						>
							Sign Up for Alerts
						</Button>
						<Button
							component={Link}
							to="/manage"
							variant="outlined"
							size="large"
							sx={{
								borderColor: 'rgba(255,255,255,0.4)',
								color: 'white',
								px: 4,
								py: 1.5,
								fontSize: '1.1rem',
								'&:hover': {
									borderColor: 'rgba(255,255,255,0.6)',
									bgcolor: 'rgba(255,255,255,0.1)',
									transform: 'translateY(-2px)',
								},
							}}
						>
							Manage Subscriptions
						</Button>
					</Box>
				</Container>
			</Box>

			<Container maxWidth="lg" sx={{ py: 12 }}>
				<Typography
					variant="h2"
					component="h2"
					align="center"
					sx={{
						mb: 8,
						fontSize: { xs: '2rem', md: '2.5rem' },
						fontWeight: 700,
						color: 'text.primary',
					}}
				>
					Why Choose Pow Hunter?
				</Typography>
				<Grid container spacing={6}>
					<Grid size={{ xs: 12, md: 4 }}>
						<Card sx={{ height: '100%', textAlign: 'center', p: 1 }}>
							<CardContent sx={{ p: 4 }}>
								<NotificationsIcon
									sx={{
										fontSize: 56,
										mb: 3,
										p: 1.5,
										borderRadius: 2,
										bgcolor: 'secondary.light',
										color: 'white',
									}}
								/>
								<Typography
									variant="h5"
									component="h3"
									gutterBottom
									sx={{ mb: 2 }}
								>
									Smart SMS Alerts
								</Typography>
								<Typography color="text.secondary" sx={{ lineHeight: 1.7 }}>
									Set your minimum snow amount and get SMS alerts 1-10 days in
									advance when powder is coming to your resorts
								</Typography>
							</CardContent>
						</Card>
					</Grid>
					<Grid size={{ xs: 12, md: 4 }}>
						<Card sx={{ height: '100%', textAlign: 'center', p: 1 }}>
							<CardContent sx={{ p: 4 }}>
								<FavoriteIcon
									sx={{
										fontSize: 56,
										color: 'white',
										mb: 3,
										p: 1.5,
										borderRadius: 2,
										bgcolor: 'primary.main',
									}}
								/>
								<Typography
									variant="h5"
									component="h3"
									gutterBottom
									sx={{ mb: 2 }}
								>
									Multiple Resort Tracking
								</Typography>
								<Typography color="text.secondary" sx={{ lineHeight: 1.7 }}>
									Subscribe to alerts for multiple resorts and manage each
									subscription independently with custom settings
								</Typography>
							</CardContent>
						</Card>
					</Grid>
					<Grid size={{ xs: 12, md: 4 }}>
						<Card sx={{ height: '100%', textAlign: 'center', p: 1 }}>
							<CardContent sx={{ p: 4 }}>
								<RocketIcon
									sx={{
										fontSize: 56,
										color: 'white',
										mb: 3,
										p: 1.5,
										borderRadius: 2,
										bgcolor: '#7c3aed',
									}}
								/>
								<Typography
									variant="h5"
									component="h3"
									gutterBottom
									sx={{ mb: 2 }}
								>
									More Features Coming
								</Typography>
								<Typography color="text.secondary" sx={{ lineHeight: 1.7 }}>
									Account management, email alerts, detailed weather data,
									historical snow reports, and advanced filtering options coming
									soon
								</Typography>
							</CardContent>
						</Card>
					</Grid>
				</Grid>
			</Container>
		</Box>
	)
}
