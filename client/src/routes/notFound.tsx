import { Box, Button, Container, Paper, Typography } from '@mui/material'
import { Link } from 'react-router'

// Without a catch-all, an unmatched URL fell through to the root error
// boundary, which renders an unstyled page with no way back into the app.
export default function NotFound() {
	return (
		<Container maxWidth="sm" sx={{ py: 8 }}>
			<Paper elevation={3} sx={{ p: 4, textAlign: 'center' }}>
				<Typography variant="h2" component="h1" gutterBottom>
					Page not found
				</Typography>
				<Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
					We could not find that page. It may have moved, or the link may be
					out of date.
				</Typography>
				<Box sx={{ display: 'flex', gap: 2, justifyContent: 'center' }}>
					<Button component={Link} to="/" variant="contained">
						Back to home
					</Button>
					<Button component={Link} to="/signup" variant="outlined">
						Create an alert
					</Button>
				</Box>
			</Paper>
		</Container>
	)
}
