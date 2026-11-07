// Where a visitor came from, read from the utm_source and utm_campaign tags on
// the link they first arrived by, such as ?utm_source=reddit&utm_campaign=cosnow.
// It is kept until they sign up, so browsing a few pages first still counts.
export type Attribution = {
	source?: string
	campaign?: string
}

const STORAGE_KEY = 'powhunter:attribution'

// Remembers the tags on the current link. The first tagged link wins: that is
// the marketing that brought the visitor in. Storage can be unavailable, as in
// some private windows, and attribution is never worth an error.
export function captureAttribution(search: string): void {
	const params = new URLSearchParams(search)
	const source = params.get('utm_source')?.trim()
	if (!source) return

	try {
		if (localStorage.getItem(STORAGE_KEY)) return

		const campaign = params.get('utm_campaign')?.trim()
		const attribution: Attribution = campaign
			? { source, campaign }
			: { source }
		localStorage.setItem(STORAGE_KEY, JSON.stringify(attribution))
	} catch {
		// Not recorded; the signup simply counts as direct.
	}
}

export function readAttribution(): Attribution {
	try {
		const stored = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '{}')
		return {
			...(typeof stored.source === 'string' && { source: stored.source }),
			...(typeof stored.campaign === 'string' && { campaign: stored.campaign }),
		}
	} catch {
		return {}
	}
}
