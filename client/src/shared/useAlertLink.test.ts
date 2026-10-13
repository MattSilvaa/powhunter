import { test, expect, describe } from 'bun:test'
import {
	AlertLinkDetails,
	forecastDay,
	formatInches,
	shareMessage,
	signupPath,
} from './useAlertLink.ts'

// Wednesday, Oct 15 2026, late in the evening: a time at which reading the
// date as UTC would land on the wrong day for most of North America.
const now = new Date(2026, 9, 15, 22, 30)

const alta: AlertLinkDetails = {
	resortUuid: '11111111-1111-4111-8111-111111111111',
	resortName: 'Alta',
	snowAmount: 14,
	forecastDate: '2026-10-17',
	reportUrl: 'https://www.alta.com/conditions',
}

describe('forecastDay', () => {
	test('names today and tomorrow', () => {
		expect(forecastDay('2026-10-15', now)).toBe('today')
		expect(forecastDay('2026-10-16', now)).toBe('tomorrow')
	})

	test('spells out later days', () => {
		expect(forecastDay('2026-10-17', now)).toBe('Saturday, Oct 17')
	})
})

test('formatInches drops a trailing .0', () => {
	expect(formatInches(14)).toBe('14″')
	expect(formatInches(8.5)).toBe('8.5″')
	expect(formatInches(6.04)).toBe('6″')
})

test('signupPath carries the resort and the alert code', () => {
	expect(signupPath(alta, 'Ab12Cd34')).toBe(
		'/signup?resort=11111111-1111-4111-8111-111111111111&ref=Ab12Cd34'
	)
})

test('shareMessage reads like a text to friends', () => {
	expect(shareMessage(alta, 'https://powhunter.app/signup', now)).toBe(
		'Alta is expecting 14″ of snow on Saturday, Oct 17. ' +
			'Get free powder alerts: https://powhunter.app/signup'
	)
	expect(
		shareMessage({ ...alta, forecastDate: '2026-10-16' }, 'link', now)
	).toBe('Alta is expecting 14″ of snow tomorrow. Get free powder alerts: link')
})
