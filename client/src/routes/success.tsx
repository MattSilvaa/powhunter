import { Box, Button, Container, Paper, Typography } from '@mui/material'
import { Check } from '@mui/icons-material'
import { Link, useLocation } from 'react-router'
import ShareButton from '../components/shareButton.tsx'
import { signupLink } from '../shared/resortSlug.ts'
import { pageMeta } from '../shared/seo.ts'

type SuccessState = { resortNames?: string[] } | null

export function meta() {
	return pageMeta({
		title: 'Check your email | Pow Hunter',
		path: '/success',
		noindex: true,
	})
}

export default function SuccessPage() {
	const state = useLocation().state as SuccessState
	const resortNames = state?.resortNames ?? []
	const shareUrl = signupLink(window.location.origin, resortNames, 'share')
	const shareText =
		resortNames.length === 1
			? `Get a text when ${resortNames[0]} gets fresh snow:`
			: 'Get a text when our mountains get fresh snow:'

	return (
		<Container maxWidth="sm" sx={{ py: { xs: 6, md: 10 } }}>
			<Paper
				variant="outlined"
				sx={{ p: { xs: 3, sm: 5 }, textAlign: 'center' }}
			>
				<Box
					sx={{
						display: 'grid',
						placeItems: 'center',
						width: 48,
						height: 48,
						mx: 'auto',
						mb: 3,
						borderRadius: '50%',
						bgcolor: 'primary.light',
						color: 'primary.main',
					}}
				>
					<Check />
				</Box>

				<Typography variant="h3" component="h1" gutterBottom>
					Check your email
				</Typography>
				<Typography
					color="text.secondary"
					sx={{ mb: 4, maxWidth: 420, mx: 'auto' }}
				>
					{'We sent you a link to confirm your email. Your powder alerts ' +
						'start as soon as you click it.'}
				</Typography>

				<Box
					sx={{
						display: 'flex',
						gap: 1.5,
						justifyContent: 'center',
						flexWrap: 'wrap',
					}}
				>
					<Button component={Link} to="/manage" variant="outlined" size="large">
						Manage Subscriptions
					</Button>
					<Button component={Link} to="/" variant="contained" size="large">
						Back to Home
					</Button>
				</Box>

				<Box sx={{ mt: 4, pt: 4, borderTop: 1, borderColor: 'divider' }}>
					<Typography variant="overline" color="text.secondary">
						Ski with friends
					</Typography>
					<Typography
						color="text.secondary"
						sx={{ mb: 2, maxWidth: 420, mx: 'auto' }}
					>
						{resortNames.length > 0
							? 'Send your crew a link with your resorts already picked.'
							: 'Send your crew a link so they hear about powder days too.'}
					</Typography>
					<ShareButton
						url={shareUrl}
						text={shareText}
						label="Share with your ski crew"
					/>
				</Box>
			</Paper>
		</Container>
	)
}
