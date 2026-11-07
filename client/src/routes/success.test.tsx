import { describe, expect, test } from 'bun:test'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import SuccessPage from './success'

describe('SuccessPage', () => {
	test('offers to share the resorts that were just picked', () => {
		render(
			<MemoryRouter
				initialEntries={[
					{ pathname: '/success', state: { resortNames: ['Alta'] } },
				]}
			>
				<SuccessPage />
			</MemoryRouter>
		)

		expect(
			screen.getByRole('button', { name: /share with your ski crew/i })
		).toBeTruthy()
		expect(screen.getByText(/your resorts already picked/i)).toBeTruthy()
	})

	// Alerts only start once the emailed link is clicked, so the page must not
	// claim they are already live.
	test('asks the subscriber to confirm their email', () => {
		render(
			<MemoryRouter initialEntries={['/success']}>
				<SuccessPage />
			</MemoryRouter>
		)

		expect(
			screen.getByRole('heading', { level: 1, name: /check your email/i })
		).toBeTruthy()
	})
})
