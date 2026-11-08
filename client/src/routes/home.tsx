import React from 'react'
import { Link } from 'react-router'
import {
	Box,
	Button,
	Container,
	Paper,
	Skeleton,
	Typography,
} from '@mui/material'
import SnowBoard from '../components/snowBoard.tsx'
import SeasonRecap from '../components/seasonRecap.tsx'
import { pageMeta } from '../shared/seo.ts'
import { SnowReport, useSnowReport } from '../shared/snowReport.ts'

// TODO(founder): a few sentences in your own words: who you are, why you
// built this, where you ski. It shows under the snow report once this is not
// empty. Leave it empty rather than filling it with invented copy.
const FOUNDER_NOTE: string = ''

const SKELETON_ROWS = 6

export function meta() {
	return pageMeta({
		title: 'Pow Hunter: powder alerts by text for your ski resorts',
		description:
			'Never miss a powder day. Pick your resorts and the snowfall worth chasing, and Pow Hunter texts you when the forecast delivers. Free.',
		path: '/',
	})
}

function BoardSkeleton(): React.ReactElement {
	return (
		<Paper variant="outlined" aria-busy="true" aria-label="Loading snow report">
			<Box sx={{ p: { xs: 2.5, sm: 3 } }}>
				<Skeleton width={120} />
				<Skeleton width={220} height={56} />
			</Box>
			{Array.from({ length: SKELETON_ROWS }, (_, i) => (
				<Box
					key={i}
					sx={{
						display: 'flex',
						gap: 2,
						px: { xs: 2, sm: 3 },
						py: 1.5,
						borderTop: 1,
						borderColor: 'divider',
					}}
				>
					<Skeleton sx={{ flex: 1 }} />
					<Skeleton width={48} />
				</Box>
			))}
		</Paper>
	)
}

function BoardMessage({
	title,
	body,
	action,
}: {
	title: string
	body: string
	action?: React.ReactNode
}): React.ReactElement {
	return (
		<Paper variant="outlined" sx={{ p: { xs: 2.5, sm: 3 } }}>
			<Typography variant="h6" component="h2">
				{title}
			</Typography>
			<Typography color="text.secondary" sx={{ mt: 0.5 }}>
				{body}
			</Typography>
			{action && <Box sx={{ mt: 2 }}>{action}</Box>}
		</Paper>
	)
}

function Board({
	report,
	now,
}: {
	report: SnowReport
	now: Date
}): React.ReactElement {
	if (report.mode === 'live' && report.live?.resorts.length) {
		return (
			<SnowBoard
				resorts={report.live.resorts}
				resortCount={report.live.resort_count}
				days={report.live.days}
				updatedAt={report.updated_at}
				now={now}
			/>
		)
	}

	if (report.mode === 'recap' && report.recap?.resorts.length) {
		return <SeasonRecap recap={report.recap} />
	}

	return (
		<BoardMessage
			title="No snow numbers yet"
			body="The report fills in after the next forecast check, which runs twice a day."
		/>
	)
}

export default function Home({
	now = new Date(),
}: {
	now?: Date
}): React.ReactElement {
	const { data, isPending, isError, refetch } = useSnowReport()
	const offSeason = data?.mode === 'recap'

	return (
		<Container maxWidth="md" sx={{ py: { xs: 5, md: 7 } }}>
			<Typography variant="h3" component="h1">
				Powder alerts by text
			</Typography>
			<Typography color="text.secondary" sx={{ mt: 1.5, maxWidth: 600 }}>
				{offSeason
					? 'No resort has a storm in the forecast yet. Pick your resorts and the snowfall worth chasing, and we text you as soon as the first one shows up.'
					: 'Pick your resorts and the snowfall worth chasing. We check the forecast twice a day and text you when it meets your number, up to 10 days ahead.'}
			</Typography>
			<Button
				component={Link}
				to="/signup"
				variant="contained"
				size="large"
				sx={{ mt: 3 }}
			>
				{offSeason
					? 'Get a text when the first storm of the season shows up'
					: 'Get snow alerts'}
			</Button>

			<Box sx={{ mt: { xs: 4, md: 5 } }}>
				{isPending && <BoardSkeleton />}
				{/* A failed background refetch keeps the last report, so only say it
				failed when there is nothing to show. */}
				{isError && !data && (
					<BoardMessage
						title="The snow report didn't load"
						body="Alerts still go out on schedule. Try again in a minute."
						action={
							<Button variant="outlined" onClick={() => refetch()}>
								Try again
							</Button>
						}
					/>
				)}
				{data && <Board report={data} now={now} />}
			</Box>

			{FOUNDER_NOTE && (
				<Box
					component="section"
					sx={{ mt: 5, pt: 3, borderTop: 1, borderColor: 'divider' }}
				>
					<Typography color="text.secondary" sx={{ maxWidth: 600 }}>
						{FOUNDER_NOTE}
					</Typography>
				</Box>
			)}
		</Container>
	)
}
