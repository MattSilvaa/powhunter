import { Link } from 'react-router'
import {
	Box,
	Button,
	Chip,
	Container,
	Grid,
	Paper,
	Typography,
} from '@mui/material'
import { ArrowForward, Check } from '@mui/icons-material'
import { tabularNums } from '../theme.ts'

// Illustrative forecast for the hero card. It is labelled as an example so it
// never reads as a live report.
const EXAMPLE_FORECAST = [
	{ day: 'Thu', inches: 2, inWindow: true },
	{ day: 'Fri', inches: 9, inWindow: true },
	{ day: 'Sat', inches: 3, inWindow: true },
	{ day: 'Sun', inches: 0, inWindow: false },
	{ day: 'Mon', inches: 1, inWindow: false },
]
const EXAMPLE_TOTAL = EXAMPLE_FORECAST.filter((d) => d.inWindow).reduce(
	(sum, d) => sum + d.inches,
	0
)
const EXAMPLE_THRESHOLD = 12
const MAX_DAILY = Math.max(...EXAMPLE_FORECAST.map((d) => d.inches))

const STEPS = [
	{
		title: 'Pick your resorts',
		body: 'Follow as many mountains as you like. Each alert is managed on its own, so you can tune them per resort.',
	},
	{
		title: 'Set your number',
		body: 'Choose the minimum snowfall worth chasing, from 1″ to 24″, and how many days of notice you want.',
	},
	{
		title: 'Get a text',
		body: 'When the forecast meets your number, we send an SMS up to 10 days ahead, so you can plan the trip.',
	},
]

function ExampleForecast() {
	return (
		<Paper variant="outlined" sx={{ p: { xs: 2.5, sm: 3 } }}>
			<Box
				sx={{
					display: 'flex',
					justifyContent: 'space-between',
					alignItems: 'center',
					mb: 2,
				}}
			>
				<Typography variant="h6" component="p">
					Alta, UT
				</Typography>
				<Chip label="Example" size="small" variant="outlined" />
			</Box>

			<Box sx={{ display: 'flex', alignItems: 'baseline', gap: 1.5 }}>
				<Typography
					component="p"
					sx={{
						...tabularNums,
						fontSize: { xs: '3.5rem', sm: '4rem' },
						fontWeight: 600,
						lineHeight: 1,
						letterSpacing: '-0.04em',
					}}
				>
					{EXAMPLE_TOTAL}&Prime;
				</Typography>
				<Typography color="text.secondary">in the next 3 days</Typography>
			</Box>

			<Box
				aria-hidden
				sx={{
					display: 'grid',
					gridTemplateColumns: `repeat(${EXAMPLE_FORECAST.length}, 1fr)`,
					gap: 1.5,
					alignItems: 'end',
					height: 120,
					mt: 3,
				}}
			>
				{EXAMPLE_FORECAST.map(({ day, inches, inWindow }) => (
					<Box
						key={day}
						sx={{
							display: 'flex',
							flexDirection: 'column',
							alignItems: 'center',
							gap: 0.75,
							height: '100%',
							justifyContent: 'flex-end',
						}}
					>
						<Typography
							variant="caption"
							sx={{ ...tabularNums, color: 'text.secondary', fontWeight: 500 }}
						>
							{inches}&Prime;
						</Typography>
						<Box
							sx={{
								width: '100%',
								maxWidth: 36,
								height: `${Math.max((inches / MAX_DAILY) * 72, 3)}px`,
								borderRadius: '4px 4px 2px 2px',
								bgcolor: inWindow ? 'primary.main' : 'divider',
							}}
						/>
						<Typography variant="caption" color="text.secondary">
							{day}
						</Typography>
					</Box>
				))}
			</Box>

			<Box
				sx={{
					display: 'flex',
					alignItems: 'center',
					gap: 1.25,
					mt: 3,
					pt: 2,
					borderTop: 1,
					borderColor: 'divider',
				}}
			>
				<Box
					sx={{
						display: 'grid',
						placeItems: 'center',
						width: 24,
						height: 24,
						borderRadius: '50%',
						bgcolor: 'primary.light',
						color: 'primary.main',
						flexShrink: 0,
					}}
				>
					<Check sx={{ fontSize: 16 }} />
				</Box>
				<Typography variant="body2" color="text.secondary">
					Over your{' '}
					<Box
						component="span"
						sx={{ ...tabularNums, color: 'text.primary', fontWeight: 600 }}
					>
						{EXAMPLE_THRESHOLD}&Prime;
					</Box>{' '}
					threshold. Text sent.
				</Typography>
			</Box>
		</Paper>
	)
}

export default function Home() {
	return (
		<Box>
			<Container maxWidth="lg" sx={{ py: { xs: 7, md: 12 } }}>
				<Grid
					container
					spacing={{ xs: 6, md: 8 }}
					sx={{ alignItems: 'center' }}
				>
					<Grid size={{ xs: 12, md: 7 }}>
						<Typography variant="overline" component="p" color="primary">
							Snow alerts by text
						</Typography>
						<Typography variant="h1" sx={{ mt: 1.5, mb: 2.5 }}>
							Never miss a powder day.
						</Typography>
						<Typography
							variant="subtitle1"
							color="text.secondary"
							sx={{ maxWidth: 480, mb: 4 }}
						>
							Pick your resorts and how much fresh snow it takes to get you out
							the door. Pow Hunter texts you when the forecast says it&apos;s
							coming.
						</Typography>
						<Box sx={{ display: 'flex', gap: 1.5, flexWrap: 'wrap' }}>
							<Button
								component={Link}
								to="/signup"
								variant="contained"
								size="large"
							>
								Sign up for alerts
							</Button>
							<Button
								component={Link}
								to="/manage"
								size="large"
								endIcon={<ArrowForward sx={{ fontSize: 18 }} />}
								sx={{ color: 'text.primary' }}
							>
								Manage subscriptions
							</Button>
						</Box>
					</Grid>
					<Grid size={{ xs: 12, md: 5 }}>
						<Box sx={{ maxWidth: 440, ml: { md: 'auto' } }}>
							<ExampleForecast />
						</Box>
					</Grid>
				</Grid>
			</Container>

			<Box
				sx={{
					borderTop: 1,
					borderColor: 'divider',
					bgcolor: 'background.paper',
				}}
			>
				<Container maxWidth="lg" sx={{ py: { xs: 7, md: 10 } }}>
					<Typography variant="h2" sx={{ mb: { xs: 4, md: 6 } }}>
						How it works
					</Typography>
					<Grid container spacing={{ xs: 4, md: 6 }}>
						{STEPS.map((step, i) => (
							<Grid key={step.title} size={{ xs: 12, md: 4 }}>
								<Box sx={{ borderTop: 1, borderColor: 'divider', pt: 2.5 }}>
									<Typography
										variant="body2"
										sx={{
											...tabularNums,
											color: 'primary.main',
											fontWeight: 600,
											mb: 1,
										}}
									>
										{String(i + 1).padStart(2, '0')}
									</Typography>
									<Typography variant="h6" component="h3" sx={{ mb: 1 }}>
										{step.title}
									</Typography>
									<Typography color="text.secondary">{step.body}</Typography>
								</Box>
							</Grid>
						))}
					</Grid>
				</Container>
			</Box>
		</Box>
	)
}
