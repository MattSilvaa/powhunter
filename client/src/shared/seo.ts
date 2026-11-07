import { type MetaDescriptor } from 'react-router'

// The public address of the site, used for canonical links and link previews.
// Previews need absolute URLs, so this cannot be worked out from the page.
export const SITE_URL = (
	import.meta.env.VITE_SITE_URL || 'https://powhunter.app'
).replace(/\/+$/, '')

export const SITE_NAME = 'Pow Hunter'

const DEFAULT_DESCRIPTION =
	'Pick your resorts and the snowfall worth chasing. Pow Hunter texts you when the forecast delivers.'

type PageMetaOptions = {
	title: string
	description?: string
	path: string
	// Pages that only make sense to a signed-in visitor stay out of search.
	noindex?: boolean
}

// The title, description, canonical link and link-preview tags for a page.
export function pageMeta({
	title,
	description = DEFAULT_DESCRIPTION,
	path,
	noindex = false,
}: PageMetaOptions): MetaDescriptor[] {
	const url = `${SITE_URL}${path}`
	const image = `${SITE_URL}/og.png`

	return [
		{ title },
		{ name: 'description', content: description },
		{ tagName: 'link', rel: 'canonical', href: url },
		{ property: 'og:site_name', content: SITE_NAME },
		{ property: 'og:type', content: 'website' },
		{ property: 'og:title', content: title },
		{ property: 'og:description', content: description },
		{ property: 'og:url', content: url },
		{ property: 'og:image', content: image },
		{ property: 'og:image:width', content: '1200' },
		{ property: 'og:image:height', content: '630' },
		{ name: 'twitter:card', content: 'summary_large_image' },
		...(noindex ? [{ name: 'robots', content: 'noindex' }] : []),
	]
}
