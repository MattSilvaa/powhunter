import React from 'react'
import {
	isRouteErrorResponse,
	Outlet,
	Scripts,
	ScrollRestoration,
} from 'react-router'
import { createTheme, ThemeProvider } from '@mui/material/styles'
import { CssBaseline } from '@mui/material'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

const queryClient = new QueryClient()

const theme = createTheme({
	palette: {
		primary: {
			main: '#1a365d', // Deep blue that matches your current design
		},
		secondary: {
			main: '#4299e1', // Light blue for accents
		},
		background: {
			default: '#f7fafc',
		},
	},
	typography: {
		fontFamily: [
			'-apple-system',
			'BlinkMacSystemFont',
			'"Segoe UI"',
			'Roboto',
			'"Helvetica Neue"',
			'Arial',
			'sans-serif',
		].join(','),
	},
})

export default function Root() {
	return <Outlet />
}

export function Layout({ children }: { children: React.ReactNode }) {
	return (
		<html lang='en' style={{ height: '100%' }}>
			<head>
				<meta charSet='utf-8' />
				<meta name='viewport' content='width=device-width, initial-scale=1' />
				<link rel='stylesheet' href='/src/app.css' />
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
export function ErrorBoundary({ error }: { error: any }) {
	let message = 'Oops!'
	let details = 'An unexpected error occurred.'
	let stack: string | undefined

	if (isRouteErrorResponse(error)) {
		message = error.status === 404 ? '404' : 'Error'
		details = error.status === 404
			? 'The requested page could not be found.'
			: error.statusText || details
	} else if (Deno.env.get('ENV') && error && error instanceof Error) {
		details = error.message
		stack = error.stack
	}

	return (
		<main id='error-page'>
			<h1>{message}</h1>
			<p>{details}</p>
			{stack && (
				<pre>
          <code>{stack}</code>
				</pre>
			)}
		</main>
	)
}
