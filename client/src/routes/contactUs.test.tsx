import { test, expect, describe, beforeEach, afterEach } from 'bun:test'
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react'
import ContactUs from './contactUs'

const originalFetch = global.fetch

describe('ContactUs Component', () => {
	beforeEach(() => {
		global.fetch = originalFetch
	})

	afterEach(() => {
		cleanup()
	})

	test('renders contact form with all fields', () => {
		render(<ContactUs />)

		expect(screen.getByRole('textbox', { name: /name/i })).toBeTruthy()
		expect(screen.getByRole('textbox', { name: /email/i })).toBeTruthy()
		expect(screen.getByRole('textbox', { name: /message/i })).toBeTruthy()
		expect(screen.getByRole('button', { name: /send message/i })).toBeTruthy()
	})

	test('renders heading and description', () => {
		render(<ContactUs />)

		expect(screen.getByRole('heading', { name: /contact us/i })).toBeTruthy()
		expect(
			screen.getByText("Have questions or feedback? We'd love to hear from you!"),
		).toBeTruthy()
	})

	test('updates form fields on user input', () => {
		render(<ContactUs />)

		const nameInput = screen.getByRole('textbox', { name: /name/i }) as HTMLInputElement
		const emailInput = screen.getByRole('textbox', { name: /email/i }) as HTMLInputElement
		const messageInput = screen.getByRole('textbox', { name: /message/i }) as HTMLTextAreaElement

		fireEvent.change(nameInput, { target: { value: 'John Doe' } })
		fireEvent.change(emailInput, { target: { value: 'john@example.com' } })
		fireEvent.change(messageInput, { target: { value: 'Test message' } })

		expect(nameInput.value).toBe('John Doe')
		expect(emailInput.value).toBe('john@example.com')
		expect(messageInput.value).toBe('Test message')
	})

	test('shows loading state when submitting', async () => {
		global.fetch = () =>
			new Promise((resolve) =>
				setTimeout(
					() =>
						resolve({
							ok: true,
							json: async () => ({}),
						} as Response),
					100,
				),
			)

		render(<ContactUs />)

		fireEvent.change(screen.getByRole('textbox', { name: /name/i }), {
			target: { value: 'John Doe' },
		})
		fireEvent.change(screen.getByRole('textbox', { name: /email/i }), {
			target: { value: 'john@example.com' },
		})
		fireEvent.change(screen.getByRole('textbox', { name: /message/i }), {
			target: { value: 'Test message' },
		})

		fireEvent.click(screen.getByRole('button', { name: /send message/i }))

		await waitFor(() => {
			expect(screen.getByRole('button', { name: /sending/i })).toBeTruthy()
		})
	})

	test('shows success message on successful submission', async () => {
		global.fetch = () =>
			Promise.resolve({
				ok: true,
				json: async () => ({}),
			} as Response)

		render(<ContactUs />)

		fireEvent.change(screen.getByRole('textbox', { name: /name/i }), {
			target: { value: 'John Doe' },
		})
		fireEvent.change(screen.getByRole('textbox', { name: /email/i }), {
			target: { value: 'john@example.com' },
		})
		fireEvent.change(screen.getByRole('textbox', { name: /message/i }), {
			target: { value: 'Test message' },
		})

		fireEvent.click(screen.getByRole('button', { name: /send message/i }))

		await waitFor(() => {
			expect(
				screen.getByText("Thank you for your message! We'll get back to you soon."),
			).toBeTruthy()
		})
	})

	test('shows error message on failed submission', async () => {
		global.fetch = () =>
			Promise.resolve({
				ok: false,
				json: async () => ({ message: 'Server error' }),
			} as Response)

		render(<ContactUs />)

		fireEvent.change(screen.getByRole('textbox', { name: /name/i }), {
			target: { value: 'John Doe' },
		})
		fireEvent.change(screen.getByRole('textbox', { name: /email/i }), {
			target: { value: 'john@example.com' },
		})
		fireEvent.change(screen.getByRole('textbox', { name: /message/i }), {
			target: { value: 'Test message' },
		})

		fireEvent.click(screen.getByRole('button', { name: /send message/i }))

		await waitFor(() => {
			expect(screen.getByText('Server error')).toBeTruthy()
		})
	})

	test('clears form after successful submission', async () => {
		global.fetch = () =>
			Promise.resolve({
				ok: true,
				json: async () => ({}),
			} as Response)

		render(<ContactUs />)

		const nameInput = screen.getByRole('textbox', {
			name: /name/i,
		}) as HTMLInputElement
		const emailInput = screen.getByRole('textbox', {
			name: /email/i,
		}) as HTMLInputElement
		const messageInput = screen.getByRole('textbox', {
			name: /message/i,
		}) as HTMLTextAreaElement

		fireEvent.change(nameInput, { target: { value: 'John Doe' } })
		fireEvent.change(emailInput, { target: { value: 'john@example.com' } })
		fireEvent.change(messageInput, { target: { value: 'Test message' } })

		fireEvent.click(screen.getByRole('button', { name: /send message/i }))

		await waitFor(() => {
			expect(nameInput.value).toBe('')
			expect(emailInput.value).toBe('')
			expect(messageInput.value).toBe('')
		})
	})

	test('disables form fields during submission', async () => {
		global.fetch = () =>
			new Promise((resolve) =>
				setTimeout(
					() =>
						resolve({
							ok: true,
							json: async () => ({}),
						} as Response),
					100,
				),
			)

		render(<ContactUs />)

		const nameInput = screen.getByRole('textbox', {
			name: /name/i,
		}) as HTMLInputElement
		const emailInput = screen.getByRole('textbox', {
			name: /email/i,
		}) as HTMLInputElement
		const messageInput = screen.getByRole('textbox', {
			name: /message/i,
		}) as HTMLTextAreaElement

		fireEvent.change(nameInput, { target: { value: 'John Doe' } })
		fireEvent.change(emailInput, { target: { value: 'john@example.com' } })
		fireEvent.change(messageInput, { target: { value: 'Test message' } })

		fireEvent.click(screen.getByRole('button', { name: /send message/i }))

		await waitFor(() => {
			expect(nameInput.disabled).toBe(true)
			expect(emailInput.disabled).toBe(true)
			expect(messageInput.disabled).toBe(true)
		})
	})
})
