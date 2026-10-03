import { Box, Button, Container, Paper, Typography } from '@mui/material'
import { Check } from '@mui/icons-material'
import { Link } from 'react-router'

export default function SuccessPage() {
	return (
		<Container maxWidth="sm" sx={{ py: { xs: 6, md: 10 } }}>
			<Paper
				variant="outlined"
				sx={{ p: { xs: 3, sm: 5 }, textAlign: 'center' }}
			>
				<Box
					sx={{
						display: 'grid',
						placeItems: 'center',
						width: 48,
						height: 48,
						mx: 'auto',
						mb: 3,
						borderRadius: '50%',
						bgcolor: 'primary.light',
						color: 'primary.main',
					}}
				>
					<Check />
				</Box>

				<Typography variant="h3" component="h1" gutterBottom>
					{"You're all set"}
				</Typography>
				<Typography
					color="text.secondary"
					sx={{ mb: 4, maxWidth: 420, mx: 'auto' }}
				>
					{"Your powder alert is live. We'll text you when fresh snow is in " +
						'the forecast at your resorts.'}
				</Typography>

				<Box
					sx={{
						display: 'flex',
						gap: 1.5,
						justifyContent: 'center',
						flexWrap: 'wrap',
					}}
				>
					<Button component={Link} to="/manage" variant="outlined" size="large">
						Manage Subscriptions
					</Button>
					<Button component={Link} to="/" variant="contained" size="large">
						Back to Home
					</Button>
				</Box>
			</Paper>
		</Container>
	)
}
