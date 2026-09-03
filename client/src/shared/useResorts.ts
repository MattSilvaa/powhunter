import { useQuery } from '@tanstack/react-query'
import { apiRequest, retryOnlyTransport } from './apiClient.ts'
import { Resort, ResortApiResponse } from './types.ts'

const RESORT_RETRIES = 2
const RESORT_STALE_TIME_MS = 5 * 60 * 1000

const fetchResorts = (): Promise<ResortApiResponse[]> =>
	apiRequest<ResortApiResponse[]>('/api/resorts')

// Nullable columns arrive from Go as {Valid, ...} wrappers. Guarding each one
// matters because this runs during render, where React Query cannot catch a
// throw, so a single incomplete row would blank the signup page.
const transformResortData = (data?: ResortApiResponse[]): Resort[] =>
	(data ?? []).map((resort) => ({
		id: resort.id,
		uuid: resort.uuid,
		name: resort.name,
		urlHost: resort.url_host?.Valid ? resort.url_host.String : null,
		urlPathname: resort.url_pathname?.Valid
			? resort.url_pathname.String
			: null,
		latitude: resort.latitude?.Valid ? resort.latitude.Float64 : null,
		longitude: resort.longitude?.Valid ? resort.longitude.Float64 : null,
	}))

export function useResorts() {
	const { data, isLoading, isError, error, refetch } = useQuery<
		ResortApiResponse[]
	>({
		queryKey: ['resorts'],
		queryFn: fetchResorts,
		staleTime: RESORT_STALE_TIME_MS,
		retry: retryOnlyTransport(RESORT_RETRIES),
	})

	return {
		resorts: transformResortData(data),
		loading: isLoading,
		error: isError ? error?.message || 'An error occurred' : null,
		refresh: refetch,
	}
}
