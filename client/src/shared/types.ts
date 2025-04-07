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
	noaa_station: NullableString
}

export type Resort = {
	id: number
	uuid: string
	name: string
	urlHost: string | null
	urlPathname: string | null
	latitude: number | null
	longitude: number | null
	noaaStation: string | null
}
