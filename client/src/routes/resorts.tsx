import { Box, Container, Paper, Typography } from '@mui/material'
import { ChevronRight } from '@mui/icons-material'
import { Link } from 'react-router'
import PageHeader from '../components/pageHeader.tsx'
import { resortsByRegion } from '../shared/catalog.ts'
import { pageMeta } from '../shared/seo.ts'

export function meta() {
	return pageMeta({
		title: 'Ski resort snow alerts by text | Pow Hunter',
		description:
			'Powder alerts for Alta, Vail, Whistler, Palisades Tahoe and more. Pick a resort and get a text when fresh snow is in the forecast.',
		path: '/resorts',
	})
}

export default function ResortsPage() {
	return (
		<Container maxWidth="sm" sx={{ py: { xs: 5, md: 8 } }}>
			<PageHeader
				title="Resorts"
				subtitle="Pick a mountain to see how its powder alerts work."
			/>

			<Paper variant="outlined">
				{resortsByRegion().map(([region, resorts], i) => (
					<Box
						key={region}
						component="section"
						sx={{ borderTop: i === 0 ? 0 : 1, borderColor: 'divider' }}
					>
						<Typography
							variant="overline"
							component="h2"
							color="text.secondary"
							sx={{ display: 'block', px: { xs: 2.5, sm: 3 }, pt: 2 }}
						>
							{region}
						</Typography>
						<Box component="ul" sx={{ listStyle: 'none', p: 0, m: 0, pb: 1 }}>
							{resorts.map((resort) => (
								<li key={resort.slug}>
									<Box
										component={Link}
										to={`/resorts/${resort.slug}`}
										sx={{
											display: 'flex',
											justifyContent: 'space-between',
											alignItems: 'center',
											px: { xs: 2.5, sm: 3 },
											py: 1.25,
											color: 'text.primary',
											textDecoration: 'none',
											'&:hover': { color: 'primary.main' },
										}}
									>
										{resort.name}
										<ChevronRight
											fontSize="small"
											sx={{ color: 'text.secondary' }}
										/>
									</Box>
								</li>
							))}
						</Box>
					</Box>
				))}
			</Paper>
		</Container>
	)
}
