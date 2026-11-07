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
})
