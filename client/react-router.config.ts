import { type Config } from '@react-router/dev/config'
import { writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import catalog from '../server/internal/seed/resorts.json' with { type: 'json' }
import { resortSlug } from './src/shared/resortSlug.ts'

const SITE_URL = (process.env.VITE_SITE_URL || 'https://powhunter.app').replace(
	/\/+$/,
	''
)

// Pages search engines should find. Each is rendered to HTML at build time,
// so crawlers and link previews see real content instead of an empty shell.
// Everything else stays a client-rendered single-page app.
const PUBLIC_PAGES = [
	'/',
	'/signup',
	'/contact',
	'/resorts',
	...catalog.map((resort) => `/resorts/${resortSlug(resort.name)}`),
]

const sitemap = (): string =>
	'<?xml version="1.0" encoding="UTF-8"?>\n' +
	'<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n' +
	PUBLIC_PAGES.map(
		(path) => `  <url><loc>${SITE_URL}${path}</loc></url>\n`
	).join('') +
	'</urlset>\n'

// Shared alert links (/alert/...) stay crawlable: some link-preview bots
// obey robots.txt, and a blocked link shares without a preview card.
const robots = (): string =>
	'User-agent: *\n' +
	'Disallow: /manage\n' +
	'Disallow: /login\n' +
	`\nSitemap: ${SITE_URL}/sitemap.xml\n`

export default {
	ssr: false,
	appDirectory: './src',
	prerender: PUBLIC_PAGES,
	async buildEnd({ viteConfig }) {
		const out = join(viteConfig.build.outDir)
		await writeFile(join(out, 'sitemap.xml'), sitemap())
		await writeFile(join(out, 'robots.txt'), robots())
	},
} satisfies Config
