import React from 'react'
import { Link } from 'react-router'
import { Box, Container, Typography, Link as MuiLink } from '@mui/material'

export default function Footer() {
	const currentYear = new Date().getFullYear()

	return (
		<Box
			component="footer"
			sx={{
				bgcolor: 'primary.main',
				color: 'white',
				py: 2,
				mt: 'auto',
			}}
		>
			<Container maxWidth="lg">
				<Box
					sx={{
						display: 'flex',
						flexDirection: { xs: 'column', sm: 'row' },
						justifyContent: 'space-between',
						alignItems: 'center',
						gap: { xs: 1, sm: 2 },
					}}
				>
					<Typography
						variant="body2"
						sx={{ opacity: 0.8, fontSize: '0.875rem' }}
					>
						© {currentYear} Pow Hunter
					</Typography>
					<Box sx={{ display: 'flex', gap: 2 }}>
						<MuiLink
							component={Link}
							to="/"
							sx={{
								color: 'white',
								fontSize: '0.875rem',
								textDecoration: 'none',
								opacity: 0.8,
								'&:hover': {
									opacity: 1,
									textDecoration: 'underline',
								},
							}}
						>
							Home
						</MuiLink>
						<MuiLink
							component={Link}
							to="/contact"
							sx={{
								color: 'white',
								fontSize: '0.875rem',
								textDecoration: 'none',
								opacity: 0.8,
								'&:hover': {
									opacity: 1,
									textDecoration: 'underline',
								},
							}}
						>
							Contact
						</MuiLink>
					</Box>
				</Box>
			</Container>
		</Box>
	)
}
