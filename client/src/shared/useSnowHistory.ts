import { useQuery } from '@tanstack/react-query'
import { apiRequest, retryOnlyTransport } from './apiClient.ts'
import { SnowHistory } from './types.ts'

const HISTORY_RETRIES = 1

// Past seasons don't change, so there is no reason to refetch within a visit.
const HISTORY_STALE_MS = 60 * 60 * 1000

const fetchSnowHistory = (
	resortUuid: string,
	threshold: number | null
): Promise<SnowHistory> => {
	const query = threshold === null ? '' : `?threshold=${threshold}`
	return apiRequest<SnowHistory>(
		`/api/resorts/${encodeURIComponent(resortUuid)}/snow-history${query}`
	)
}

export function useSnowHistory(
	resortUuid: string,
	threshold: number | null,
	enabled: boolean
) {
	return useQuery<SnowHistory>({
		queryKey: ['snowHistory', resortUuid, threshold],
		queryFn: () => fetchSnowHistory(resortUuid, threshold),
		enabled,
		staleTime: HISTORY_STALE_MS,
		retry: retryOnlyTransport(HISTORY_RETRIES),
	})
}
