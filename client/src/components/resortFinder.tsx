import React, { useState } from 'react'
import { Autocomplete, Box, TextField } from '@mui/material'
import {
	SEARCH_DEBOUNCE_MS,
	SEARCH_MAX_LENGTH,
	SEARCH_MIN_LENGTH,
	searchLength,
	SnowReportSearch,
	useDebouncedValue,
	useSnowReportSearch,
} from '../shared/snowReport.ts'

type ResortFinderProps<T extends { name: string }> = {
	// Picks this board's list (live or recap) out of a search response.
	select: (result: SnowReportSearch) => T[]
	// Renders a picked resort as a row of the board.
	renderRow: (resort: T) => React.ReactNode
}

// "Find your resort", for everything outside the board's top rows. The server
// does the searching, so this stays one small request however many resorts
// there are; the picked resort shows as a normal board row underneath.
export default function ResortFinder<T extends { name: string }>({
	select,
	renderRow,
}: ResortFinderProps<T>): React.ReactElement {
	// input is what the box shows; term is what was typed. They differ after
	// a pick, when the box shows the resort's name, which is not a search.
	const [input, setInput] = useState('')
	const [term, setTerm] = useState('')
	const [picked, setPicked] = useState<T | null>(null)

	const typed = term.trim()
	const debounced = useDebouncedValue(typed, SEARCH_DEBOUNCE_MS)
	const search = useSnowReportSearch(debounced)
	const tooShort = searchLength(typed) < SEARCH_MIN_LENGTH
	// Until the debounce catches up, the last results are for another term,
	// so show "Searching…" rather than them or "No resort matches".
	const pending = typed !== debounced
	const results = !pending && search.data ? select(search.data) : []
	// Keep the picked resort among the options so the box can show it.
	const options =
		picked && !results.some((resort) => resort.name === picked.name)
			? [picked, ...results]
			: results

	return (
		<Box sx={{ borderTop: 1, borderColor: 'divider' }}>
			<Box sx={{ px: { xs: 2, sm: 3 }, py: 2 }}>
				<Autocomplete<T>
					options={options}
					value={picked}
					onChange={(_, value) => setPicked(value)}
					inputValue={input}
					onInputChange={(_, value, reason) => {
						setInput(value)
						if (reason === 'input' || reason === 'clear') setTerm(value)
					}}
					getOptionLabel={(resort) => resort.name}
					isOptionEqualToValue={(a, b) => a.name === b.name}
					// The server already filtered the options.
					filterOptions={(resorts) => resorts}
					loading={!tooShort && (pending || search.isFetching)}
					loadingText="Searching…"
					noOptionsText={
						tooShort
							? `Type ${SEARCH_MIN_LENGTH} or more letters`
							: search.isError
								? "Search isn't working right now"
								: 'No resort matches'
					}
					size="small"
					renderInput={(params) => (
						<TextField
							{...params}
							label="Find your resort"
							placeholder="Resort name"
							inputProps={{
								...params.inputProps,
								maxLength: SEARCH_MAX_LENGTH,
							}}
						/>
					)}
				/>
			</Box>
			<Box
				component="ul"
				aria-live="polite"
				sx={{ listStyle: 'none', m: 0, p: 0 }}
			>
				{picked && renderRow(picked)}
			</Box>
		</Box>
	)
}
