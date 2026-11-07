import { describe, expect, test } from 'bun:test'
import { resortSlug, resortUuidsFromLink, signupLink } from './resortSlug.ts'
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

describe('resortUuidsFromLink', () => {
	const resorts = [resort('a', 'Alta'), resort('b', 'Mt. Bachelor')]

	test('finds resorts by name or uuid and skips unknown ones', () => {
		expect(resortUuidsFromLink(['MT-BACHELOR', 'nowhere'], resorts)).toEqual([
			'b',
		])
		expect(resortUuidsFromLink(['a'], resorts)).toEqual(['a'])
	})
})

describe('signupLink', () => {
	test('names the picked resorts', () => {
		expect(signupLink('https://powhunter.app', ['Alta', 'Mt. Bachelor'])).toBe(
			'https://powhunter.app/signup?resort=alta,mt-bachelor'
		)
		expect(signupLink('https://powhunter.app', ['Alta'], 'share')).toBe(
			'https://powhunter.app/signup?resort=alta&utm_source=share'
		)
	})

	test('is the plain signup page without resorts', () => {
		expect(signupLink('https://powhunter.app', [])).toBe(
			'https://powhunter.app/signup'
		)
	})
})
