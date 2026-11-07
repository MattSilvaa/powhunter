import { Resort } from './types.ts'

// The readable name of a resort in links, such as "mt-bachelor" for
// "Mt. Bachelor", so a shared link says where it leads.
export function resortSlug(name: string): string {
	return name
		.toLowerCase()
		.replace(/&/g, ' and ')
		.replace(/['.]/g, '')
		.replace(/[^a-z0-9]+/g, '-')
		.replace(/^-+|-+$/g, '')
}

// The uuids of the resorts a link names. Names the catalog does not have are
// skipped, so an old or mistyped link still opens the form.
export function resortUuidsFromSlugs(
	slugs: string[],
	resorts: Resort[]
): string[] {
	const wanted = new Set(slugs.map((slug) => slug.trim().toLowerCase()))

	return resorts
		.filter((resort) => wanted.has(resortSlug(resort.name)))
		.map((resort) => resort.uuid)
}

// The signup link that opens with these resorts already picked.
export function signupLink(origin: string, resortNames: string[]): string {
	if (resortNames.length === 0) {
		return `${origin}/signup`
	}

	const slugs = resortNames.map(resortSlug).join(',')

	return `${origin}/signup?resort=${slugs}`
}
