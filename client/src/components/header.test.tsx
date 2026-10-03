import { test, expect, describe, afterEach } from 'bun:test'
import { render, screen, cleanup } from '@testing-library/react'
import { BrowserRouter } from 'react-router'
import Header from './header'

describe('Header Component', () => {
	afterEach(() => {
		cleanup()
	})

	test('links the wordmark home and exposes the main actions', () => {
		render(
			<BrowserRouter>
				<Header />
			</BrowserRouter>
		)

		const linkTo = (name: RegExp) =>
			screen.getByRole('link', { name }).getAttribute('href')

		expect(linkTo(/pow hunter home/i)).toBe('/')
		expect(linkTo(/create alert/i)).toBe('/signup')
		expect(linkTo(/manage alerts/i)).toBe('/manage')
	})
})
