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

// The uuids of the resorts a link names, each by uuid or by slug. Resorts the
// catalog does not have are skipped, so an old or mistyped link still opens
// the form.
export function resortUuidsFromLink(
	refs: string[],
	resorts: Resort[]
): string[] {
	const wanted = new Set(refs.map((ref) => ref.trim().toLowerCase()))

	return resorts
		.filter(
			(resort) =>
				wanted.has(resort.uuid.toLowerCase()) ||
				wanted.has(resortSlug(resort.name))
		)
		.map((resort) => resort.uuid)
}

// The signup link that opens with these resorts already picked, tagged with
// the source the signup will be counted under, such as "share".
export function signupLink(
	origin: string,
	resortNames: string[],
	source?: string
): string {
	const params = []
	if (resortNames.length > 0) {
		params.push(`resort=${resortNames.map(resortSlug).join(',')}`)
	}
	if (source) {
		params.push(`utm_source=${encodeURIComponent(source)}`)
	}

	return params.length > 0
		? `${origin}/signup?${params.join('&')}`
		: `${origin}/signup`
}
