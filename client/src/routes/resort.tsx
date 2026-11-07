import {
	Box,
	Button,
	Container,
	Link as MuiLink,
	Paper,
	Typography,
} from '@mui/material'
import { OpenInNew } from '@mui/icons-material'
import { Link, useParams, type MetaArgs } from 'react-router'
import { CATALOG, findResort } from '../shared/catalog.ts'
import { pageMeta } from '../shared/seo.ts'
import NotFound from './notFound.tsx'

export function meta({ params }: MetaArgs) {
	const resort = findResort(params.slug)
	if (!resort) {
		return pageMeta({
			title: 'Page not found | Pow Hunter',
			path: '/resorts',
			noindex: true,
		})
	}

	return pageMeta({
		title: `${resort.name} snow alerts by text | Pow Hunter`,
		description:
			`Get a text when the forecast at ${resort.name}, ${resort.region} ` +
			'calls for fresh snow. Pick how much, from 1″ to 24″, with up to 10 days of notice. Free.',
		path: `/resorts/${resort.slug}`,
	})
}

export default function ResortPage() {
	const resort = findResort(useParams().slug)
	if (!resort) return <NotFound />

	const steps = [
		{
			title: `Follow ${resort.name}`,
			body: 'Add any other mountains you ride too.',
		},
		{
			title: 'Set your number',
			body: 'The minimum snowfall worth chasing, from 1″ to 24″, and up to 10 days of notice.',
		},
		{
			title: 'Get a text',
			body: `We check the ${resort.name} forecast twice a day and text you once when it meets your number.`,
		},
	]
	const nearby = CATALOG.filter(
		(r) => r.region === resort.region && r.slug !== resort.slug
	)

	return (
		<Container maxWidth="sm" sx={{ py: { xs: 5, md: 8 } }}>
			<Typography variant="overline" component="p" color="primary">
				Snow alerts · {resort.region}
			</Typography>
			<Typography variant="h3" component="h1" sx={{ mt: 1, mb: 1.5 }}>
				{resort.name} powder alerts
			</Typography>
			<Typography color="text.secondary" sx={{ mb: 4 }}>
				Get a text when the forecast at {resort.name} calls for fresh snow, so
				you can plan the trip before everyone else does.
			</Typography>

			<Paper variant="outlined">
				{steps.map((step, i) => (
					<Box
						key={step.title}
						sx={{
							display: 'flex',
							gap: 2,
							p: { xs: 2.5, sm: 3 },
							borderTop: i === 0 ? 0 : 1,
							borderColor: 'divider',
						}}
					>
						<Typography
							color="primary"
							sx={{ fontWeight: 600, minWidth: 16 }}
							aria-hidden
						>
							{i + 1}
						</Typography>
						<Box>
							<Typography sx={{ fontWeight: 600 }}>{step.title}</Typography>
							<Typography variant="body2" color="text.secondary">
								{step.body}
							</Typography>
						</Box>
					</Box>
				))}
			</Paper>

			<Box sx={{ display: 'flex', gap: 1.5, flexWrap: 'wrap', mt: 3 }}>
				<Button
					component={Link}
					to={`/signup?resort=${resort.slug}`}
					variant="contained"
					size="large"
				>
					Get {resort.name} alerts
				</Button>
				{resort.reportUrl && (
					<Button
						href={resort.reportUrl}
						target="_blank"
						rel="noopener"
						size="large"
						endIcon={<OpenInNew fontSize="small" />}
					>
						Mountain report
					</Button>
				)}
			</Box>

			<Box sx={{ mt: 6 }}>
				<Typography variant="overline" component="h2" color="text.secondary">
					{nearby.length > 0 ? `More in ${resort.region}` : 'More resorts'}
				</Typography>
				<Box
					component="ul"
					sx={{
						listStyle: 'none',
						p: 0,
						m: 0,
						display: 'flex',
						flexWrap: 'wrap',
						gap: 2,
					}}
				>
					{nearby.map((r) => (
						<li key={r.slug}>
							<MuiLink component={Link} to={`/resorts/${r.slug}`}>
								{r.name}
							</MuiLink>
						</li>
					))}
					<li>
						<MuiLink component={Link} to="/resorts">
							All resorts
						</MuiLink>
					</li>
				</Box>
			</Box>
		</Container>
	)
}
