import catalogJson from '../../../server/internal/seed/resorts.json'
import { resortSlug } from './resortSlug.ts'

// The resort catalog, read from the same file the server seeds its database
// from, so the resort pages can be built ahead of time without the API.
export type CatalogResort = {
	name: string
	slug: string
	region: string
	reportUrl: string | null
}

type CatalogEntry = {
	name: string
	region: string
	url?: { host?: string; pathname?: string }
}

function reportUrl(entry: CatalogEntry): string | null {
	const host = entry.url?.host?.replace(/\/+$/, '')
	if (!host || !/^https?:\/\//.test(host)) return null

	const path = entry.url?.pathname?.replace(/^\/+/, '') ?? ''
	return path ? `${host}/${path}` : host
}

export const CATALOG: CatalogResort[] = (catalogJson as CatalogEntry[])
	.map((entry) => ({
		name: entry.name,
		slug: resortSlug(entry.name),
		region: entry.region,
		reportUrl: reportUrl(entry),
	}))
	.sort((a, b) => a.name.localeCompare(b.name))

export function findResort(slug: string | undefined): CatalogResort | null {
	return CATALOG.find((resort) => resort.slug === slug) ?? null
}

// Resorts grouped by state or province, both in alphabetical order.
export function resortsByRegion(): [string, CatalogResort[]][] {
	const groups = new Map<string, CatalogResort[]>()
	for (const resort of CATALOG) {
		groups.set(resort.region, [...(groups.get(resort.region) ?? []), resort])
	}

	return [...groups.entries()].sort(([a], [b]) => a.localeCompare(b))
}
