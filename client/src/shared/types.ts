export const BASE_SERVER_URL = Bun.env.BASE_SERVER_URL || 'http://localhost:8080'

export type NullableString = {
	String: string
	Valid: boolean
}

export type NullableFloat = {
	Float64: number
	Valid: boolean
}

export type ResortApiResponse = {
	id: number
	uuid: string
	name: string
	url_host: NullableString
	url_pathname: NullableString
	latitude: NullableFloat
	longitude: NullableFloat
}

export type Resort = {
	id: number
	uuid: string
	name: string
	urlHost: string | null
	urlPathname: string | null
	latitude: number | null
	longitude: number | null
}
