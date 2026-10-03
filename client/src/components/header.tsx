import { Link } from 'react-router'
import { Box, Button, Container, Typography } from '@mui/material'
import { AcUnit as SnowflakeIcon } from '@mui/icons-material'

export default function Header() {
	return (
		<Box
			component="header"
			sx={{
				position: 'sticky',
				top: 0,
				zIndex: 'appBar',
				bgcolor: 'rgb(255 255 255 / 0.85)',
				backdropFilter: 'saturate(180%) blur(10px)',
				borderBottom: 1,
				borderColor: 'divider',
			}}
		>
			<Container
				maxWidth="lg"
				sx={{
					height: 64,
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'space-between',
				}}
			>
				<Box
					component={Link}
					to="/"
					aria-label="Pow Hunter home"
					sx={{
						display: 'flex',
						alignItems: 'center',
						gap: 1,
						color: 'text.primary',
						textDecoration: 'none',
					}}
				>
					<SnowflakeIcon sx={{ fontSize: 22, color: 'primary.main' }} />
					<Typography
						component="span"
						sx={{
							fontWeight: 650,
							fontSize: '1.0625rem',
							letterSpacing: '-0.02em',
						}}
					>
						Pow Hunter
					</Typography>
				</Box>
				<Box
					component="nav"
					sx={{ display: 'flex', alignItems: 'center', gap: 1 }}
				>
					<Button
						component={Link}
						to="/manage"
						sx={{
							display: { xs: 'none', sm: 'inline-flex' },
							color: 'text.secondary',
							'&:hover': { color: 'text.primary', bgcolor: 'transparent' },
						}}
					>
						Manage alerts
					</Button>
					<Button
						component={Link}
						to="/signup"
						variant="contained"
						size="small"
					>
						Create alert
					</Button>
				</Box>
			</Container>
		</Box>
	)
}
