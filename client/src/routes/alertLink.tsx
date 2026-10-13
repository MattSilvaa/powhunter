import React, { useState } from 'react'
import {
	Alert,
	Box,
	Button,
	Container,
	LinearProgress,
	Link as MuiLink,
	Paper,
	Typography,
} from '@mui/material'
import { IosShare, OpenInNew } from '@mui/icons-material'
import { Link, useParams } from 'react-router'
import { tabularNums } from '../theme.ts'
import {
	AlertLinkDetails,
	forecastDay,
	formatInches,
	shareMessage,
	signupPath,
	useAlertLink,
} from '../shared/useAlertLink.ts'
import { ApiError } from '../shared/apiClient.ts'

type ShareState = 'idle' | 'copied' | 'failed'

// share hands the message to the phone's share sheet, so it can go to any
// group chat. Desktop browsers mostly lack one, so the message is copied
// instead.
const share = async (message: string): Promise<ShareState> => {
	if (typeof navigator.share === 'function') {
		try {
			await navigator.share({ text: message })
			return 'idle'
		} catch (err) {
			// Closing the share sheet is a choice, not a failure.
			if (err instanceof DOMException && err.name === 'AbortError') {
				return 'idle'
			}
		}
	}

	try {
		await navigator.clipboard.writeText(message)
		return 'copied'
	} catch {
		return 'failed'
	}
}

function AlertCard({
	alert,
	code,
}: {
	alert: AlertLinkDetails
	code: string
}): React.ReactElement {
	const [shareState, setShareState] = useState<ShareState>('idle')
	const path = signupPath(alert, code)
	const message = shareMessage(alert, window.location.origin + path)
	const day = forecastDay(alert.forecastDate)

	return (
		<Paper variant="outlined" sx={{ p: { xs: 3, sm: 5 } }}>
			<Typography variant="overline" component="p" color="primary">
				Powder alert
			</Typography>
			<Typography variant="h3" component="h1" sx={{ mt: 0.5 }}>
				{alert.resortName}
			</Typography>

			<Box sx={{ my: { xs: 3, sm: 4 } }}>
				<Typography
					component="p"
					sx={{
						...tabularNums,
						fontSize: { xs: '3.5rem', sm: '4.5rem' },
						fontWeight: 700,
						lineHeight: 1,
						letterSpacing: '-0.03em',
					}}
				>
					{formatInches(alert.snowAmount)}
				</Typography>
				<Typography color="text.secondary" sx={{ mt: 1 }}>
					forecast {day === 'today' || day === 'tomorrow' ? day : `for ${day}`}
				</Typography>
			</Box>

			<Box sx={{ display: 'grid', gap: 1.5 }}>
				<Button
					variant="contained"
					size="large"
					startIcon={<IosShare />}
					onClick={async () => setShareState(await share(message))}
				>
					Text your ski crew
				</Button>
				<Button
					variant="outlined"
					size="large"
					endIcon={<OpenInNew />}
					href={alert.reportUrl}
					target="_blank"
					rel="noopener noreferrer"
				>
					View mountain report
				</Button>
			</Box>

			{shareState === 'copied' && (
				<Alert severity="success" role="status" sx={{ mt: 2 }}>
					Copied. Paste it into your group chat.
				</Alert>
			)}
			{shareState === 'failed' && (
				<Alert severity="info" sx={{ mt: 2 }}>
					Copy this into your group chat: {message}
				</Alert>
			)}

			<Typography color="text.secondary" variant="body2" sx={{ mt: 3 }}>
				Not getting these yet?{' '}
				<MuiLink component={Link} to={path}>
					Get alerts for {alert.resortName}
				</MuiLink>
			</Typography>
		</Paper>
	)
}

export default function AlertLinkPage(): React.ReactElement {
	const { code = '' } = useParams()
	const { alert, loading, error } = useAlertLink(code)

	return (
		<Container maxWidth="sm" sx={{ py: { xs: 5, md: 8 } }}>
			{loading && <LinearProgress />}

			{alert && <AlertCard alert={alert} code={code} />}

			{!loading && !alert && (
				<Paper
					variant="outlined"
					sx={{ p: { xs: 3, sm: 5 }, textAlign: 'center' }}
				>
					<Typography variant="h5" component="h1" gutterBottom>
						We couldn&apos;t find that alert
					</Typography>
					<Typography color="text.secondary" sx={{ mb: 3 }}>
						{error instanceof ApiError && error.status !== 404
							? error.message
							: 'The link may be mistyped or out of date.'}
					</Typography>
					<Button component={Link} to="/signup" variant="contained">
						Create an alert
					</Button>
				</Paper>
			)}
		</Container>
	)
}
