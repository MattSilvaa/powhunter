import { describe, expect, test } from 'bun:test'
import { resortSlug, resortUuidsFromSlugs, signupLink } from './resortSlug.ts'
import { Resort } from './types.ts'

const resort = (uuid: string, name: string): Resort => ({
	id: 1,
	uuid,
	name,
	urlHost: null,
	urlPathname: null,
	latitude: null,
	longitude: null,
})

describe('resortSlug', () => {
	test('turns a name into a readable link segment', () => {
		expect(resortSlug('Mt. Bachelor')).toBe('mt-bachelor')
		expect(resortSlug('Summit at Snoqualmie')).toBe('summit-at-snoqualmie')
		expect(resortSlug('Aspen Snowmass')).toBe('aspen-snowmass')
	})
})

describe('resortUuidsFromSlugs', () => {
	const resorts = [resort('a', 'Alta'), resort('b', 'Mt. Bachelor')]

	test('finds the resorts a link names and skips unknown ones', () => {
		expect(resortUuidsFromSlugs(['MT-BACHELOR', 'nowhere'], resorts)).toEqual([
			'b',
		])
	})
})

describe('signupLink', () => {
	test('names the picked resorts', () => {
		expect(signupLink('https://powhunter.app', ['Alta', 'Mt. Bachelor'])).toBe(
			'https://powhunter.app/signup?resort=alta,mt-bachelor'
		)
	})

	test('is the plain signup page without resorts', () => {
		expect(signupLink('https://powhunter.app', [])).toBe(
			'https://powhunter.app/signup'
		)
	})
})
