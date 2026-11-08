import React, { useState } from 'react'
import { Autocomplete, Box, TextField } from '@mui/material'
import {
	SEARCH_MIN_LENGTH,
	SnowReportSearch,
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
	const [input, setInput] = useState('')
	const [picked, setPicked] = useState<T | null>(null)
	const search = useSnowReportSearch(input)
	const options = search.data ? select(search.data) : []
	const tooShort = input.trim().length < SEARCH_MIN_LENGTH

	return (
		<Box sx={{ borderTop: 1, borderColor: 'divider' }}>
			<Box sx={{ px: { xs: 2, sm: 3 }, py: 2 }}>
				<Autocomplete<T>
					options={options}
					value={picked}
					onChange={(_, value) => setPicked(value)}
					inputValue={input}
					onInputChange={(_, value) => setInput(value)}
					getOptionLabel={(resort) => resort.name}
					isOptionEqualToValue={(a, b) => a.name === b.name}
					// The server already filtered the options.
					filterOptions={(resorts) => resorts}
					loading={search.isFetching}
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
