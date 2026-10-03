import { test, expect, describe, afterEach } from 'bun:test'
import { render, screen, cleanup } from '@testing-library/react'
import { BrowserRouter } from 'react-router'
import { ThemeProvider, createTheme } from '@mui/material/styles'
import { type ReactElement } from 'react'
import Home from './home'

const theme = createTheme()

const renderWithProviders = (component: ReactElement) => {
	return render(
		<BrowserRouter>
			<ThemeProvider theme={theme}>{component}</ThemeProvider>
		</BrowserRouter>
	)
}

describe('Home Component', () => {
	afterEach(() => {
		cleanup()
	})
	test('renders the headline', () => {
		renderWithProviders(<Home />)
		expect(
			screen.getByRole('heading', {
				level: 1,
				name: /never miss a powder day/i,
			})
		).toBeTruthy()
	})

	test('renders Sign up for alerts button with link', () => {
		renderWithProviders(<Home />)
		const signupButton = screen.getByRole('link', {
			name: /sign up for alerts/i,
		})
		expect(signupButton.getAttribute('href')).toBe('/signup')
	})

	test('renders Manage subscriptions button with link', () => {
		renderWithProviders(<Home />)
		const manageButton = screen.getByRole('link', {
			name: /manage subscriptions/i,
		})
		expect(manageButton.getAttribute('href')).toBe('/manage')
	})

	test('labels the forecast card as an example', () => {
		renderWithProviders(<Home />)
		expect(screen.getByText('Example')).toBeTruthy()
	})

	test('renders the three how it works steps', () => {
		renderWithProviders(<Home />)
		expect(
			screen.getByRole('heading', { level: 2, name: /how it works/i })
		).toBeTruthy()

		const steps = screen.getAllByRole('heading', { level: 3 })
		expect(steps.map((h) => h.textContent)).toEqual([
			'Pick your resorts',
			'Set your number',
			'Get a text',
		])
	})
})
