import _React from 'react'
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
	Cloud as CloudIcon,
	Favorite as FavoriteIcon,
	Notifications as NotificationsIcon,
} from '@mui/icons-material'

export default function Home() {
	const theme = useTheme()

	return (
		<Box sx={{ minHeight: '100vh' }}>
			<Box
				sx={{
					background:
						`linear-gradient(135deg, ${theme.palette.primary.main} 0%, ${theme.palette.primary.dark} 100%)`,
					color: 'white',
					py: 8,
					textAlign: 'center',
				}}
			>
				<Container maxWidth='md'>
					<Typography
						variant='h1'
						component='h1'
						sx={{
							fontSize: { xs: '2.5rem', md: '3.5rem' },
							fontWeight: 800,
							mb: 2,
						}}
					>
						Pow Hunter
					</Typography>
					<Typography variant='h5' sx={{ mb: 4, opacity: 0.9 }}>
						Never miss a powder day at your favorite resort
					</Typography>
					<Box sx={{ display: 'flex', gap: 2, justifyContent: 'center' }}>
						<Button
							component={Link}
							to='/signup'
							variant='outlined'
							size='large'
							sx={{
								borderColor: 'white',
								color: 'white',
								'&:hover': {
									borderColor: 'white',
									bgcolor: 'rgba(255,255,255,0.1)',
								},
							}}
						>
							Sign Up for Alerts
						</Button>
					</Box>
				</Container>
			</Box>

			<Container maxWidth='lg' sx={{ py: 8 }}>
				<Typography variant='h2' component='h2' align='center' sx={{ mb: 6 }}>
					Why Choose Pow Hunter?
				</Typography>
				<Grid container spacing={4}>
					<Grid size={{ xs: 12, md: 4 }}>
						<Card sx={{ height: '100%', textAlign: 'center' }}>
							<CardContent>
								<FavoriteIcon
									sx={{ fontSize: 48, color: 'primary.main', mb: 2 }}
								/>
								<Typography variant='h5' component='h3' gutterBottom>
									Save Your Favorites
								</Typography>
								<Typography color='text.secondary'>
									Keep track of your favorite ski resorts in one place
								</Typography>
							</CardContent>
						</Card>
					</Grid>
					<Grid size={{ xs: 12, md: 4 }}>
						<Card sx={{ height: '100%', textAlign: 'center' }}>
							<CardContent>
								<CloudIcon
									sx={{ fontSize: 48, color: 'primary.main', mb: 2 }}
								/>
								<Typography variant='h5' component='h3' gutterBottom>
									Weather Forecasts
								</Typography>
								<Typography color='text.secondary'>
									Get detailed snow and weather forecasts for your resorts
								</Typography>
							</CardContent>
						</Card>
					</Grid>
					<Grid size={{ xs: 12, md: 4 }}>
						<Card sx={{ height: '100%', textAlign: 'center' }}>
							<CardContent>
								<NotificationsIcon
									sx={{ fontSize: 48, color: 'primary.main', mb: 2 }}
								/>
								<Typography variant='h5' component='h3' gutterBottom>
									Text Alerts
								</Typography>
								<Typography color='text.secondary'>
									Receive notifications when fresh powder is on the way
								</Typography>
							</CardContent>
						</Card>
					</Grid>
				</Grid>
			</Container>
		</Box>
	)
}
