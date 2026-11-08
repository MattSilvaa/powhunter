import React from 'react'
import { Link } from 'react-router'
import { Box, Link as MuiLink, Paper, Typography } from '@mui/material'
import { tabularNums } from '../theme.ts'
import {
	dateRange,
	formatInches,
	RecapResort,
	resortRef,
	seasonLabel,
	shortDate,
	SnowReport,
} from '../shared/snowReport.ts'

type Recap = NonNullable<SnowReport['recap']>

// Resort, total, 6″+ days and biggest storm from sm up. On a phone the last
// two move under the resort name.
const ROW_COLUMNS = {
	xs: 'minmax(0, 1fr) 64px',
	sm: 'minmax(0, 1fr) 72px 72px 156px',
}

function stormText(resort: RecapResort): string {
	if (!resort.biggest_storm_start || !resort.biggest_storm_end) return '—'

	return `${formatInches(resort.biggest_storm_in)} · ${dateRange(
		resort.biggest_storm_start,
		resort.biggest_storm_end
	)}`
}

const figure = {
	...tabularNums,
	textAlign: 'right',
} as const

function RecapRow({ resort }: { resort: RecapResort }): React.ReactElement {
	const ref = resortRef(resort.name)
	return (
		<Box
			component="li"
			sx={{
				display: 'grid',
				gridTemplateColumns: ROW_COLUMNS,
				columnGap: 2,
				alignItems: 'center',
				px: { xs: 2, sm: 3 },
				py: 1.25,
				borderTop: 1,
				borderColor: 'divider',
			}}
		>
			<Box sx={{ minWidth: 0 }}>
				<MuiLink
					component={Link}
					to={`/resorts/${ref.slug}`}
					color="text.primary"
					underline="hover"
					sx={{ fontWeight: 600, display: 'block' }}
					noWrap
				>
					{resort.name}
				</MuiLink>
				<Typography
					variant="body2"
					color="text.secondary"
					noWrap
					sx={{ display: { xs: 'none', sm: 'block' } }}
				>
					{ref.region}
				</Typography>
				<Typography
					variant="body2"
					color="text.secondary"
					sx={{ ...tabularNums, display: { xs: 'block', sm: 'none' } }}
				>
					{resort.days_6in_plus} days 6″+ ·{' '}
					<Box component="span" sx={{ whiteSpace: 'nowrap' }}>
						biggest {stormText(resort)}
					</Box>
				</Typography>
			</Box>
			<Typography sx={{ ...figure, fontWeight: 650, fontSize: '1.125rem' }}>
				{formatInches(resort.total_in)}
			</Typography>
			<Typography sx={{ ...figure, display: { xs: 'none', sm: 'block' } }}>
				{resort.days_6in_plus}
			</Typography>
			<Typography
				sx={{ ...figure, display: { xs: 'none', sm: 'block' } }}
				noWrap
			>
				{stormText(resort)}
			</Typography>
		</Box>
	)
}

// The off-season board: last season, resort by resort.
export default function SeasonRecap({
	recap,
}: {
	recap: Recap
}): React.ReactElement {
	const biggest = recap.resorts.reduce<RecapResort | null>(
		(best, resort) =>
			!best || resort.biggest_storm_in > best.biggest_storm_in ? resort : best,
		null
	)

	return (
		<Paper variant="outlined" component="section" aria-label="Season recap">
			<Box sx={{ p: { xs: 2.5, sm: 3 } }}>
				<Typography variant="overline" component="h2" color="text.secondary">
					{seasonLabel(recap.season)} season · {shortDate(recap.start)} –{' '}
					{shortDate(recap.end)}
				</Typography>
				{biggest && biggest.biggest_storm_in > 0 && (
					<Box
						sx={{
							display: 'flex',
							alignItems: 'baseline',
							flexWrap: 'wrap',
							columnGap: 1.5,
						}}
					>
						<Typography
							component="p"
							sx={{
								...tabularNums,
								fontSize: { xs: '3rem', sm: '3.5rem' },
								fontWeight: 650,
								lineHeight: 1,
								letterSpacing: '-0.04em',
							}}
						>
							{formatInches(biggest.biggest_storm_in)}
						</Typography>
						<Typography component="p" variant="h6">
							{biggest.name}
							<Box
								component="span"
								sx={{ color: 'text.secondary', fontWeight: 400 }}
							>
								{' '}
								in 3 days,{' '}
								{biggest.biggest_storm_start &&
									biggest.biggest_storm_end &&
									dateRange(
										biggest.biggest_storm_start,
										biggest.biggest_storm_end
									)}
							</Box>
						</Typography>
					</Box>
				)}
			</Box>

			<Box
				aria-hidden
				sx={{
					display: { xs: 'none', sm: 'grid' },
					gridTemplateColumns: ROW_COLUMNS,
					columnGap: 2,
					px: 3,
					pb: 1,
				}}
			>
				{['Resort', 'Total', '6″+ days', 'Biggest storm'].map((label, i) => (
					<Typography
						key={label}
						variant="caption"
						color="text.secondary"
						sx={{ textAlign: i === 0 ? 'left' : 'right' }}
					>
						{label}
					</Typography>
				))}
			</Box>

			<Box component="ol" sx={{ listStyle: 'none', m: 0, p: 0 }}>
				{recap.resorts.map((resort) => (
					<RecapRow key={resort.name} resort={resort} />
				))}
			</Box>

			<Typography
				variant="body2"
				color="text.secondary"
				sx={{
					px: { xs: 2, sm: 3 },
					py: 2,
					borderTop: 1,
					borderColor: 'divider',
				}}
			>
				Modeled snowfall from Open-Meteo&apos;s historical weather archive. It
				usually runs lower than what resorts report. Biggest storm is the most
				snow over 3 days.
			</Typography>
		</Paper>
	)
}
