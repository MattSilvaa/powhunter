import React from 'react'
import {
	Alert,
	Box,
	CircularProgress,
	Container,
	Paper,
	Typography,
} from '@mui/material'
import { useParams } from 'react-router'
import { useCurrentUser } from '../shared/useAuth.ts'
import { useUserAlerts } from '../shared/useManageAlerts.ts'
import { useSnowHistory } from '../shared/useSnowHistory.ts'
import { MonthlySnowfall, SeasonSnowfall } from '../shared/types.ts'
import PageHeader from '../components/pageHeader.tsx'
import { tabularNums } from '../theme.ts'

const monthName = (month: number): string =>
	new Date(2000, month - 1, 1).toLocaleString('en-US', { month: 'short' })

const inches = (value: number): string => `${Math.round(value)}″`

function Readout({
	value,
	label,
}: {
	value: string
	label: string
}): React.ReactElement {
	return (
		<Box sx={{ flex: 1, minWidth: 0 }}>
			<Typography
				aria-hidden
				sx={{
					...tabularNums,
					fontSize: '2.25rem',
					fontWeight: 600,
					letterSpacing: '-0.03em',
					lineHeight: 1,
				}}
			>
				{value}
			</Typography>
			<Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
				<span className="sr-only">{value} </span>
				{label}
			</Typography>
		</Box>
	)
}

function MonthBars({
	months,
}: {
	months: MonthlySnowfall[]
}): React.ReactElement {
	const max = Math.max(...months.map((month) => month.inches), 1)

	return (
		<Paper variant="outlined" sx={{ px: { xs: 2, sm: 3 }, py: 2.5 }}>
			<Box component="ul" sx={{ listStyle: 'none', m: 0, p: 0 }}>
				{months.map((month) => (
					<Box
						component="li"
						key={month.month}
						sx={{
							display: 'flex',
							alignItems: 'center',
							gap: 1.5,
							py: 0.5,
						}}
					>
						<Typography
							variant="body2"
							color="text.secondary"
							sx={{ width: 36, flexShrink: 0 }}
						>
							{monthName(month.month)}
						</Typography>
						<Box sx={{ flex: 1, height: 14 }}>
							<Box
								sx={{
									width: `${(month.inches / max) * 100}%`,
									minWidth: month.inches > 0 ? 4 : 0,
									height: '100%',
									bgcolor: 'primary.main',
									borderRadius: '0 4px 4px 0',
								}}
							/>
						</Box>
						<Typography
							variant="body2"
							sx={{
								...tabularNums,
								width: 40,
								flexShrink: 0,
								textAlign: 'right',
								fontWeight: 600,
							}}
						>
							{inches(month.inches)}
						</Typography>
					</Box>
				))}
			</Box>
		</Paper>
	)
}

function SeasonRows({
	seasons,
	threshold,
}: {
	seasons: SeasonSnowfall[]
	threshold: number | null
}): React.ReactElement {
	return (
		<Paper variant="outlined" sx={{ overflow: 'hidden' }}>
			{seasons.map((season, i) => (
				<Box
					key={season.startYear}
					sx={{
						display: 'flex',
						alignItems: 'baseline',
						gap: 2,
						px: { xs: 2, sm: 3 },
						py: 1.75,
						borderTop: i === 0 ? 0 : 1,
						borderColor: 'divider',
					}}
				>
					<Typography sx={{ ...tabularNums, flex: 1, fontWeight: 600 }}>
						{season.label}
						{season.partial && (
							<Typography
								component="span"
								variant="body2"
								color="text.secondary"
								sx={{ ml: 1 }}
							>
								so far
							</Typography>
						)}
					</Typography>
					{threshold !== null && season.daysAtOrAbove !== null && (
						<Typography variant="body2" color="text.secondary" sx={tabularNums}>
							{season.daysAtOrAbove}{' '}
							{season.daysAtOrAbove === 1 ? 'day' : 'days'} ≥ {threshold}″
						</Typography>
					)}
					<Typography
						sx={{
							...tabularNums,
							width: 56,
							textAlign: 'right',
							fontWeight: 600,
						}}
					>
						{inches(season.totalInches)}
					</Typography>
				</Box>
			))}
		</Paper>
	)
}

const sectionLabel = { display: 'block', mt: 5, mb: 1.5 }

export default function ResortHistoryPage(): React.ReactElement {
	const { resortUuid = '' } = useParams()
	const { user, loading: userLoading } = useCurrentUser()
	const { data: alerts, isLoading: alertsLoading } = useUserAlerts(!!user)

	// A signed-in user's own threshold for this resort turns the history into an
	// answer to "how often would my alert have fired?".
	const alert = alerts?.find((item) => item.resort_uuid === resortUuid)
	const threshold = alert ? alert.min_snow_amount : null
	const ready = !userLoading && (!user || !alertsLoading)

	const { data, isLoading, error } = useSnowHistory(
		resortUuid,
		threshold,
		ready
	)

	return (
		<Container maxWidth="sm" sx={{ py: { xs: 5, md: 8 } }}>
			<PageHeader
				title={data?.resortName ?? 'Snowfall history'}
				subtitle="Snowfall over the last five seasons"
			/>

			{error && (
				<Alert severity="error" sx={{ mb: 3 }}>
					{error.message}
				</Alert>
			)}

			{(!ready || isLoading) && (
				<Box sx={{ textAlign: 'center', py: 6 }}>
					<CircularProgress size={28} />
				</Box>
			)}

			{data && (
				<>
					<Paper
						variant="outlined"
						sx={{
							display: 'flex',
							gap: 3,
							px: { xs: 2, sm: 3 },
							py: 2.5,
						}}
					>
						<Readout
							value={inches(data.averageSeasonInches)}
							label="in an average season"
						/>
						{threshold !== null && data.averageDaysAtOrAbove !== null && (
							<Readout
								value={`${data.averageDaysAtOrAbove}`}
								label={`days a season with ${threshold}″ or more, your alert threshold`}
							/>
						)}
					</Paper>

					<Typography
						variant="overline"
						component="h2"
						color="text.secondary"
						sx={sectionLabel}
					>
						Average by month
					</Typography>
					<MonthBars months={data.monthlyAverages} />

					<Typography
						variant="overline"
						component="h2"
						color="text.secondary"
						sx={sectionLabel}
					>
						By season
					</Typography>
					<SeasonRows seasons={data.seasons} threshold={threshold} />

					<Typography variant="body2" color="text.secondary" sx={{ mt: 3 }}>
						Modelled snowfall from Open-Meteo&apos;s historical weather archive,
						October through May. Totals at the resort itself may differ.
					</Typography>
				</>
			)}
		</Container>
	)
}
