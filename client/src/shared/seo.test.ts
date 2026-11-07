import { describe, expect, test } from 'bun:test'
import { pageMeta } from './seo.ts'

describe('pageMeta', () => {
	test('points previews and the canonical link at the page', () => {
		const tags = pageMeta({ title: 'Alta', path: '/resorts/alta' })
		expect(tags).toContainEqual({ title: 'Alta' })
		expect(tags).toContainEqual({
			property: 'og:url',
			content: 'https://powhunter.app/resorts/alta',
		})
		expect(tags).not.toContainEqual({ name: 'robots', content: 'noindex' })
	})

	test('keeps private pages out of search', () => {
		expect(
			pageMeta({ title: 'Manage', path: '/manage', noindex: true })
		).toContainEqual({ name: 'robots', content: 'noindex' })
	})
})
