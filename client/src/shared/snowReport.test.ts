import { describe, expect, test } from 'bun:test'
import { act, renderHook } from '@testing-library/react'
import {
	byDay,
	dateRange,
	formatInches,
	resortRef,
	seasonLabel,
	updatedAgo,
	useDebouncedValue,
} from './snowReport.ts'

// A Thursday afternoon, UTC.
const now = new Date('2026-01-15T15:30:00Z')

describe('snow report formatting', () => {
	test('formats inches like a readout', () => {
		expect(formatInches(18.4)).toBe('18″')
		expect(formatInches(3)).toBe('3″')
		expect(formatInches(0.46)).toBe('0.5″')
		expect(formatInches(0)).toBe('0″')
	})

	test('names the day a storm runs through', () => {
		expect(byDay('2026-01-15', now)).toBe('today')
		expect(byDay('2026-01-16', now)).toBe('tomorrow')
		expect(byDay('2026-01-18', now)).toBe('Sunday')
		expect(byDay('2026-01-23', now)).toBe('Jan 23')
	})

	test('writes storm date ranges compactly', () => {
		expect(dateRange('2026-01-03', '2026-01-05')).toBe('Jan 3–5')
		expect(dateRange('2025-12-30', '2026-01-01')).toBe('Dec 30–Jan 1')
		expect(dateRange('2026-02-07', '2026-02-07')).toBe('Feb 7')
	})

	test('labels seasons and freshness', () => {
		expect(seasonLabel('2025-26')).toBe('2025–26')
		expect(updatedAgo('2026-01-15T15:00:00Z', now)).toBe('Updated just now')
		expect(updatedAgo('2026-01-15T12:00:00Z', now)).toBe('Updated 4h ago')
		expect(updatedAgo('2026-01-12T12:00:00Z', now)).toBe('Updated 3 days ago')
	})

	test('finds slugs and regions in the catalog', () => {
		expect(resortRef('Palisades Tahoe')).toEqual({
			slug: 'palisades-tahoe',
			region: 'California',
		})
		expect(resortRef('Nowhere Bowl')).toEqual({
			slug: 'nowhere-bowl',
			region: '',
		})
	})
})

// Search waits for typing to pause, so one word is one request, not one per
// keystroke.
test('debounces a changing value', async () => {
	const { result, rerender } = renderHook(
		({ value }: { value: string }) => useDebouncedValue(value, 20),
		{ initialProps: { value: 'a' } }
	)

	rerender({ value: 'al' })
	rerender({ value: 'alt' })
	expect(result.current).toBe('a')

	await act(() => new Promise((resolve) => setTimeout(resolve, 40)))
	expect(result.current).toBe('alt')
})
