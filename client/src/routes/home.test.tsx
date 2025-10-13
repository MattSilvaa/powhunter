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
		</BrowserRouter>,
	)
}

describe('Home Component', () => {
	afterEach(() => {
		cleanup()
	})
	test('renders main heading', () => {
		renderWithProviders(<Home />)
		expect(
			screen.getByRole('heading', { level: 1, name: /pow hunter/i }),
		).toBeTruthy()
	})

	test('renders tagline', () => {
		renderWithProviders(<Home />)
		const taglines = screen.getAllByRole('heading', {
			level: 5,
			name: /never miss a powder day/i,
		})
		expect(taglines[0]).toBeTruthy()
	})

	test('renders Sign Up for Alerts button with link', () => {
		renderWithProviders(<Home />)
		const signupButton = screen.getByRole('link', { name: /sign up for alerts/i })
		expect(signupButton.getAttribute('href')).toBe('/signup')
	})

	test('renders Manage Subscriptions button with link', () => {
		renderWithProviders(<Home />)
		const manageButton = screen.getByRole('link', {
			name: /manage subscriptions/i,
		})
		expect(manageButton.getAttribute('href')).toBe('/manage')
	})

	test('renders Why Choose Pow Hunter section', () => {
		renderWithProviders(<Home />)
		const sections = screen.getAllByRole('heading', { name: /why choose pow hunter/i })
		expect(sections[0]).toBeTruthy()
	})

	test('renders Smart SMS Alerts feature card', () => {
		renderWithProviders(<Home />)
		const alerts = screen.getAllByRole('heading', { name: /smart sms alerts/i })
		const descriptions = screen.getAllByText(
			/set your minimum snow amount and get sms alerts/i,
		)
		expect(alerts[0]).toBeTruthy()
		expect(descriptions[0]).toBeTruthy()
	})

	test('renders Multiple Resort Tracking feature card', () => {
		renderWithProviders(<Home />)
		const tracking = screen.getAllByRole('heading', {
			name: /multiple resort tracking/i,
		})
		const descriptions = screen.getAllByText(
			/subscribe to alerts for multiple resorts/i,
		)
		expect(tracking[0]).toBeTruthy()
		expect(descriptions[0]).toBeTruthy()
	})

	test('renders More Features Coming card', () => {
		renderWithProviders(<Home />)
		const features = screen.getAllByRole('heading', {
			name: /more features coming/i,
		})
		const descriptions = screen.getAllByText(
			/account management, email alerts, detailed weather data/i,
		)
		expect(features[0]).toBeTruthy()
		expect(descriptions[0]).toBeTruthy()
	})

	test('renders all three feature cards', () => {
		renderWithProviders(<Home />)

		const alerts = screen.getAllByRole('heading', { name: /smart sms alerts/i })
		const tracking = screen.getAllByRole('heading', {
			name: /multiple resort tracking/i,
		})
		const features = screen.getAllByRole('heading', {
			name: /more features coming/i,
		})

		expect(alerts[0]).toBeTruthy()
		expect(tracking[0]).toBeTruthy()
		expect(features[0]).toBeTruthy()
	})

	test('has proper semantic structure', () => {
		const { container } = renderWithProviders(<Home />)

		const h1Elements = container.querySelectorAll('h1')
		const h2Elements = container.querySelectorAll('h2')
		const h3Elements = container.querySelectorAll('h3')

		expect(h1Elements.length).toBeGreaterThan(0)
		expect(h2Elements.length).toBeGreaterThan(0)
		expect(h3Elements.length).toBe(3) // Three feature card headings
	})
})
