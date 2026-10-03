import { Link } from 'react-router'
import { Box, Container, Typography, Link as MuiLink } from '@mui/material'

const linkSx = {
	color: 'text.secondary',
	fontSize: '0.875rem',
	textDecoration: 'none',
	'&:hover': { color: 'text.primary' },
}

export default function Footer() {
	const currentYear = new Date().getFullYear()

	return (
		<Box
			component="footer"
			sx={{
				borderTop: 1,
				borderColor: 'divider',
				bgcolor: 'background.paper',
				py: 3,
				mt: 'auto',
			}}
		>
			<Container
				maxWidth="lg"
				sx={{
					display: 'flex',
					flexDirection: { xs: 'column', sm: 'row' },
					justifyContent: 'space-between',
					alignItems: 'center',
					gap: 1,
				}}
			>
				<Typography variant="body2" color="text.secondary">
					© {currentYear} Pow Hunter
				</Typography>
				<Box component="nav" sx={{ display: 'flex', gap: 3 }}>
					<MuiLink component={Link} to="/" sx={linkSx}>
						Home
					</MuiLink>
					<MuiLink component={Link} to="/contact" sx={linkSx}>
						Contact
					</MuiLink>
				</Box>
			</Container>
		</Box>
	)
}
