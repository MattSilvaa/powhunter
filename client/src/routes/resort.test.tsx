import { describe, expect, test } from 'bun:test'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import ResortPage from './resort'

const renderAt = (path: string) =>
	render(
		<MemoryRouter initialEntries={[path]}>
			<Routes>
				<Route path="/resorts/:slug" element={<ResortPage />} />
			</Routes>
		</MemoryRouter>
	)

describe('ResortPage', () => {
	test('links to signup with the resort picked', () => {
		renderAt('/resorts/alta')
		expect(
			screen.getByRole('heading', { level: 1, name: 'Alta powder alerts' })
		).toBeTruthy()
		expect(
			screen.getByRole('link', { name: 'Get Alta alerts' }).getAttribute('href')
		).toBe('/signup?resort=alta')
		expect(screen.getByRole('link', { name: 'Snowbird' })).toBeTruthy()
	})

	test('shows not found for an unknown resort', () => {
		renderAt('/resorts/nowhere')
		expect(screen.getByRole('heading', { name: 'Page not found' })).toBeTruthy()
	})
})
