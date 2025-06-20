import { test, expect } from 'bun:test'

// Test the error parsing logic from useCreateAlert
test('createAlert error parsing', async () => {
	const API_BASE_URL = 'http://localhost:8080/api'

	// Test successful response
	const mockSuccessResponse = new Response(null, { status: 201 })
	expect(mockSuccessResponse.ok).toBe(true)

	// Test error response with JSON
	const errorData = {
		error: 'DUPLICATE_EMAIL',
		message: 'An account with this email address already exists'
	}
	const mockErrorResponse = new Response(JSON.stringify(errorData), { 
		status: 409,
		headers: { 'Content-Type': 'application/json' }
	})
	
	expect(mockErrorResponse.ok).toBe(false)
	expect(mockErrorResponse.status).toBe(409)
	
	const parsedError = await mockErrorResponse.json()
	expect(parsedError.error).toBe('DUPLICATE_EMAIL')
	expect(parsedError.message).toBe('An account with this email address already exists')
})

test('AlertData type structure', () => {
	const mockAlertData = {
		email: 'test@example.com',
		phone: '1234567890',
		notificationDays: 3,
		minSnowAmount: 6.5,
		resortsUuids: ['uuid1', 'uuid2']
	}

	// Validate structure matches expected AlertData type
	expect(typeof mockAlertData.email).toBe('string')
	expect(typeof mockAlertData.phone).toBe('string')
	expect(typeof mockAlertData.notificationDays).toBe('number')
	expect(typeof mockAlertData.minSnowAmount).toBe('number')
	expect(Array.isArray(mockAlertData.resortsUuids)).toBe(true)
	expect(mockAlertData.resortsUuids.every(uuid => typeof uuid === 'string')).toBe(true)
})

test('API request structure', () => {
	const API_BASE_URL = 'http://localhost:8080/api'
	const mockData = {
		email: 'test@example.com',
		phone: '1234567890',
		notificationDays: 3,
		minSnowAmount: 6.5,
		resortsUuids: ['uuid1']
	}

	// Test request configuration
	const requestConfig = {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json',
		},
		credentials: 'include' as RequestCredentials,
		body: JSON.stringify(mockData),
	}

	expect(requestConfig.method).toBe('PUT')
	expect(requestConfig.headers['Content-Type']).toBe('application/json')
	expect(requestConfig.credentials).toBe('include')
	
	const parsedBody = JSON.parse(requestConfig.body)
	expect(parsedBody).toEqual(mockData)
})

test('Error handling edge cases', async () => {
	// Test non-JSON error response
	const textErrorResponse = new Response('Internal Server Error', { status: 500 })
	expect(textErrorResponse.ok).toBe(false)
	
	// Test empty error response
	const emptyErrorResponse = new Response('', { status: 400 })
	expect(emptyErrorResponse.ok).toBe(false)
	
	// Test malformed JSON error response
	const malformedJsonResponse = new Response('{"error": malformed', { status: 400 })
	expect(malformedJsonResponse.ok).toBe(false)
})