import { describe, expect, test } from 'bun:test'
import { CATALOG, findResort, resortsByRegion } from './catalog.ts'

describe('CATALOG', () => {
	test('gives every resort a unique link and a region', () => {
		const slugs = CATALOG.map((r) => r.slug)
		expect(new Set(slugs).size).toBe(slugs.length)
		expect(CATALOG.every((r) => r.region)).toBe(true)
	})

	test('builds the mountain report address', () => {
		expect(findResort('alta')?.reportUrl).toStartWith('https://')
		expect(findResort('nowhere')).toBeNull()
	})
})

describe('resortsByRegion', () => {
	test('groups resorts under their state', () => {
		const utah = resortsByRegion().find(([region]) => region === 'Utah')
		expect(utah?.[1].map((r) => r.name)).toContain('Alta')
	})
})
