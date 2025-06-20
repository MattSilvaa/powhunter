export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api'

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
