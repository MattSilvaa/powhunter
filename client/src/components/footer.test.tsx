import { test, expect, describe } from 'bun:test'
import { render, screen } from '@testing-library/react'
import { BrowserRouter } from 'react-router'
import Footer from './Footer'

describe('Footer Component', () => {
	test('renders copyright with current year', () => {
		render(
			<BrowserRouter>
				<Footer />
			</BrowserRouter>,
		)

		const currentYear = new Date().getFullYear()
		expect(screen.getByText(`© ${currentYear} Pow Hunter`)).toBeTruthy()
	})

	test('renders Home link', () => {
		render(
			<BrowserRouter>
				<Footer />
			</BrowserRouter>,
		)

		const homeLink = screen.getByText('Home')
		expect(homeLink).toBeTruthy()
		expect(homeLink.closest('a')?.getAttribute('href')).toBe('/')
	})

	test('renders Contact link', () => {
		render(
			<BrowserRouter>
				<Footer />
			</BrowserRouter>,
		)

		const contactLink = screen.getByText('Contact')
		expect(contactLink).toBeTruthy()
		expect(contactLink.closest('a')?.getAttribute('href')).toBe('/contact')
	})

	test('has footer semantic element', () => {
		const { container } = render(
			<BrowserRouter>
				<Footer />
			</BrowserRouter>,
		)

		const footer = container.querySelector('footer')
		expect(footer).toBeTruthy()
	})
})
