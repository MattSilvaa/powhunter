import { useQuery } from '@tanstack/react-query'
import { Resort, ResortApiResponse } from './types.ts'
import { API_BASE_URL } from './types'

const fetchResorts = async (): Promise<ResortApiResponse[]> => {
	const response = await fetch(`/api/resorts`, {
		method: 'GET',
		headers: {
			'Content-Type': 'application/json',
		},
		credentials: 'include', // For auth cookies if needed
	})

	if (!response.ok) {
		throw new Error(`Request failed: ${response.status}`)
	}

	return response.json()
}

const transformResortData = (data?: ResortApiResponse[]): Resort[] => {
	return data?.map((resort) => ({
		id: resort.id,
		uuid: resort.uuid,
		name: resort.name,
		urlHost: resort.url_host.Valid ? resort.url_host.String : null,
		urlPathname: resort.url_pathname.Valid ? resort.url_pathname.String : null,
		latitude: resort.latitude.Valid ? resort.latitude.Float64 : null,
		longitude: resort.longitude.Valid ? resort.longitude.Float64 : null,
	})) || [] // Default to empty array if data is undefined
}

export function useResorts() {
	const {
		data,
		isLoading,
		isError,
		error,
		refetch,
	} = useQuery<ResortApiResponse[]>({
		queryKey: ['resorts'],
		queryFn: fetchResorts,
		staleTime: 5 * 60 * 1000,
		retry: 2,
	})

	return {
		resorts: transformResortData(data),
		loading: isLoading,
		error: isError ? error?.message || 'An error occurred' : null,
		refresh: refetch,
	}
}
