import { Box, Button, Container, Typography } from '@mui/material'
import { Link } from 'react-router'
import { tabularNums } from '../theme.ts'

// Without a catch-all, an unmatched URL fell through to the root error
// boundary, which renders an unstyled page with no way back into the app.
export default function NotFound() {
	return (
		<Container maxWidth="sm" sx={{ py: { xs: 8, md: 12 } }}>
			<Typography
				variant="overline"
				component="p"
				color="primary"
				sx={tabularNums}
			>
				404
			</Typography>
			<Typography variant="h3" component="h1" sx={{ mt: 1, mb: 1.5 }}>
				Page not found
			</Typography>
			<Typography color="text.secondary" sx={{ mb: 4 }}>
				We could not find that page. It may have moved, or the link may be out
				of date.
			</Typography>
			<Box sx={{ display: 'flex', gap: 1.5, flexWrap: 'wrap' }}>
				<Button component={Link} to="/" variant="contained">
					Back to home
				</Button>
				<Button component={Link} to="/signup" variant="outlined">
					Create an alert
				</Button>
			</Box>
		</Container>
	)
}
