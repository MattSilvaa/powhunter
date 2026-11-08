import React from 'react'
import { Link } from 'react-router'
import { Box, Button, Link as MuiLink, Paper, Typography } from '@mui/material'
import { tabularNums } from '../theme.ts'
import {
	byDay,
	formatInches,
	LiveResort,
	resortRef,
	updatedAgo,
	weekdayInitial,
} from '../shared/snowReport.ts'

type SnowBoardProps = {
	resorts: LiveResort[]
	days: number
	updatedAt: string | null
	now: Date
}

// Bars are scaled to a fixed ceiling rather than the biggest day, so a dusting
// never looks like a storm when nothing else is falling.
const BAR_CEILING_IN = 12
const BAR_MAX_PX = 28

// Rank, resort, daily bars (sm and up), total, alerts link. Fixed widths, so
// the header and every row (each its own grid) line up.
const ROW_COLUMNS = {
	xs: '20px minmax(0, 1fr) 52px 56px',
	sm: '24px minmax(0, 1fr) 200px 60px 64px',
}

function DailyBars({ resort }: { resort: LiveResort }): React.ReactElement {
	return (
		<Box
			aria-hidden
			sx={{
				display: { xs: 'none', sm: 'grid' },
				gridTemplateColumns: `repeat(${resort.daily.length}, 1fr)`,
				gap: '4px',
				alignItems: 'end',
				height: BAR_MAX_PX,
			}}
		>
			{resort.daily.map((day) => {
				const inches = day.snow_in ?? 0
				const height =
					inches <= 0
						? 2
						: Math.max(3, Math.min(inches / BAR_CEILING_IN, 1) * BAR_MAX_PX)
				return (
					<Box
						key={day.date}
						title={`${day.date}: ${formatInches(inches)}`}
						sx={{
							height,
							borderRadius: '2px',
							bgcolor: inches >= 1 ? 'primary.main' : 'divider',
							opacity: day.snow_in === null ? 0.4 : 1,
						}}
					/>
				)
			})}
		</Box>
	)
}

function StormHeadline({
	resort,
	now,
}: {
	resort: LiveResort
	now: Date
}): React.ReactElement {
	const ref = resortRef(resort.name)
	return (
		<Box
			sx={{
				display: 'flex',
				flexWrap: 'wrap',
				alignItems: 'flex-end',
				justifyContent: 'space-between',
				gap: 2,
				p: { xs: 2.5, sm: 3 },
			}}
		>
			<Box sx={{ minWidth: 0 }}>
				<Typography variant="overline" component="h2" color="text.secondary">
					Biggest storm
				</Typography>
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
						{formatInches(resort.total_in)}
					</Typography>
					<Typography component="p" variant="h6">
						<MuiLink
							component={Link}
							to={`/resorts/${ref.slug}`}
							color="inherit"
							underline="hover"
						>
							{resort.name}
						</MuiLink>
						{resort.through && (
							<Box
								component="span"
								sx={{ color: 'text.secondary', fontWeight: 400 }}
							>
								{' '}
								by {byDay(resort.through, now)}
							</Box>
						)}
					</Typography>
				</Box>
			</Box>
			<Button
				component={Link}
				to={`/signup?resort=${ref.slug}`}
				variant="outlined"
			>
				Get {resort.name} alerts
			</Button>
		</Box>
	)
}

function BoardRow({
	resort,
	rank,
}: {
	resort: LiveResort
	rank: number
}): React.ReactElement {
	const ref = resortRef(resort.name)
	const dry = resort.total_in < 1
	return (
		<Box
			component="li"
			sx={{
				display: 'grid',
				gridTemplateColumns: ROW_COLUMNS,
				columnGap: { xs: 1.5, sm: 2 },
				alignItems: 'center',
				px: { xs: 2, sm: 3 },
				py: 1.25,
				borderTop: 1,
				borderColor: 'divider',
			}}
		>
			<Typography
				variant="body2"
				color="text.secondary"
				sx={{ ...tabularNums }}
			>
				{rank}
			</Typography>
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
				<Typography variant="body2" color="text.secondary" noWrap>
					{ref.region}
				</Typography>
			</Box>
			<DailyBars resort={resort} />
			<Typography
				sx={{
					...tabularNums,
					fontWeight: 650,
					fontSize: '1.125rem',
					textAlign: 'right',
					color: dry ? 'text.secondary' : 'text.primary',
				}}
			>
				{formatInches(resort.total_in)}
			</Typography>
			<Button
				component={Link}
				to={`/signup?resort=${ref.slug}`}
				size="small"
				aria-label={`Get ${resort.name} alerts`}
				sx={{ minWidth: 0, px: 1 }}
			>
				Alerts
			</Button>
		</Box>
	)
}

// The in-season board: every resort's snow over the coming days, most first,
// with the biggest storm called out on top.
export default function SnowBoard({
	resorts,
	days,
	updatedAt,
	now,
}: SnowBoardProps): React.ReactElement {
	const storm = resorts[0]?.total_in >= 1 ? resorts[0] : null
	const dayLabels = resorts[0]?.daily ?? []

	return (
		<Paper variant="outlined" component="section" aria-label="Snow report">
			{storm && <StormHeadline resort={storm} now={now} />}

			<Box
				sx={{
					display: 'grid',
					gridTemplateColumns: ROW_COLUMNS,
					columnGap: { xs: 1.5, sm: 2 },
					alignItems: 'end',
					px: { xs: 2, sm: 3 },
					pt: storm ? 1.5 : 2.5,
					pb: 1,
					borderTop: storm ? 1 : 0,
					borderColor: 'divider',
				}}
			>
				<Box />
				<Box sx={{ minWidth: 0 }}>
					<Typography
						variant="overline"
						component="h2"
						sx={{ lineHeight: 1.2 }}
					>
						Next {days} days
					</Typography>
					{updatedAt && (
						<Typography variant="caption" color="text.secondary" component="p">
							{updatedAgo(updatedAt, now)}
						</Typography>
					)}
				</Box>
				<Box
					aria-hidden
					sx={{
						display: { xs: 'none', sm: 'grid' },
						gridTemplateColumns: `repeat(${dayLabels.length}, 1fr)`,
						gap: '4px',
					}}
				>
					{dayLabels.map((day) => (
						<Typography
							key={day.date}
							variant="caption"
							color="text.secondary"
							sx={{ textAlign: 'center', lineHeight: 1 }}
						>
							{weekdayInitial(day.date)}
						</Typography>
					))}
				</Box>
				<Typography
					variant="caption"
					color="text.secondary"
					sx={{ textAlign: 'right', lineHeight: 1 }}
				>
					Total
				</Typography>
				<Box />
			</Box>

			<Box component="ol" sx={{ listStyle: 'none', m: 0, p: 0 }}>
				{resorts.map((resort, i) => (
					<BoardRow key={resort.name} resort={resort} rank={i + 1} />
				))}
			</Box>
		</Paper>
	)
}
