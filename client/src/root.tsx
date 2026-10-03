import React from 'react'
import '@fontsource-variable/inter'
import appStylesHref from './app.css?url'
import {
	isRouteErrorResponse,
	Outlet,
	Scripts,
	ScrollRestoration,
} from 'react-router'
import { ThemeProvider } from '@mui/material/styles'
import { Box, Button, Container, CssBaseline, Typography } from '@mui/material'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Footer from './components/footer'
import Header from './components/header'
import theme from './theme'

const queryClient = new QueryClient()

export default function Root() {
	return (
		<Box sx={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
			<Header />
			<Box component="main" sx={{ flex: 1 }}>
				<Outlet />
			</Box>
			<Footer />
		</Box>
	)
}

export function Layout({ children }: { children: React.ReactNode }) {
	return (
		<html lang="en" style={{ height: '100%' }}>
			<head>
				<meta charSet="utf-8" />
				<meta name="viewport" content="width=device-width, initial-scale=1" />
				<meta name="theme-color" content="#ffffff" />
				<link rel="stylesheet" href={appStylesHref} />
				<title>Pow Hunter</title>
			</head>
			<body style={{ height: '100%', margin: 0, padding: 0 }}>
				<QueryClientProvider client={queryClient}>
					<ThemeProvider theme={theme}>
						<CssBaseline />
						<div
							style={{
								height: '100%',
								width: '100%',
								display: 'flex',
								flexDirection: 'column',
							}}
						>
							{children}
						</div>
					</ThemeProvider>
					<ScrollRestoration />
					<Scripts />
				</QueryClientProvider>
			</body>
		</html>
	)
}

// The top most error boundary for the app, rendered when your app throws an error
// For more information, see https://reactrouter.com/start/framework/route-module#errorboundary
export function ErrorBoundary({ error }: { error: unknown }) {
	let message = 'Oops!'
	let details = 'An unexpected error occurred.'
	let stack: string | undefined

	if (isRouteErrorResponse(error)) {
		message = error.status === 404 ? '404' : 'Error'
		details =
			error.status === 404
				? 'The requested page could not be found.'
				: error.statusText || details
	} else if (import.meta.env.DEV && error instanceof Error) {
		details = error.message
		stack = error.stack
	}

	return (
		<Container component="main" maxWidth="sm" sx={{ py: { xs: 8, md: 12 } }}>
			<Typography variant="h3" component="h1" gutterBottom>
				{message}
			</Typography>
			<Typography color="text.secondary" sx={{ mb: 4 }}>
				{details}
			</Typography>
			<Button href="/" variant="contained">
				Back to home
			</Button>
			{stack && (
				<Box
					component="pre"
					sx={{
						mt: 4,
						p: 2,
						overflow: 'auto',
						fontSize: '0.8125rem',
						bgcolor: 'secondary.light',
						borderRadius: 2,
					}}
				>
					<code>{stack}</code>
				</Box>
			)}
		</Container>
	)
}
