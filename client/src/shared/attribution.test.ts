import { beforeEach, describe, expect, test } from 'bun:test'
import { captureAttribution, readAttribution } from './attribution.ts'

describe('attribution', () => {
	beforeEach(() => localStorage.clear())

	test('keeps the first tagged link the visitor arrived by', () => {
		captureAttribution('?resort=alta&utm_source=reddit&utm_campaign=cosnow')
		captureAttribution('?utm_source=tiktok')
		expect(readAttribution()).toEqual({ source: 'reddit', campaign: 'cosnow' })
	})

	test('records nothing for an untagged link', () => {
		captureAttribution('?resort=alta')
		expect(readAttribution()).toEqual({})
	})
})
