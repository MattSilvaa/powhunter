import React from 'react'
import {
	Box,
	Checkbox,
	Chip,
	FormControl,
	FormHelperText,
	InputLabel,
	ListItemText,
	MenuItem,
	Select,
} from '@mui/material'
import { Resort } from '../shared/types.ts'

type ResortSelectProps = {
	id: string
	resorts: Resort[]
	value: string[]
	onChange: (value: string[]) => void
	error?: string
	helperText?: string
}

// A checkbox list of resorts that shows the picks as chips. Values are uuids so
// resorts sharing a name cannot collide, but the field still reads as names.
export default function ResortSelect({
	id,
	resorts,
	value,
	onChange,
	error,
	helperText = 'Pick as many as you like',
}: ResortSelectProps): React.ReactElement {
	const labelId = `${id}-label`

	return (
		<FormControl fullWidth error={!!error}>
			<InputLabel id={labelId}>Resorts</InputLabel>
			<Select
				required
				multiple
				labelId={labelId}
				name="resorts"
				value={value}
				onChange={(e) => {
					const next = e.target.value
					onChange(typeof next === 'string' ? next.split(',') : next)
				}}
				label="Resorts"
				MenuProps={{
					slotProps: { paper: { sx: { maxHeight: 320 } } },
				}}
				renderValue={(selected) => (
					<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
						{selected.map((uuid) => (
							<Chip
								key={uuid}
								size="small"
								label={resorts.find((r) => r.uuid === uuid)?.name ?? uuid}
								sx={{
									bgcolor: 'primary.light',
									color: 'primary.dark',
								}}
							/>
						))}
					</Box>
				)}
			>
				{resorts.map((resort) => (
					<MenuItem key={resort.uuid} value={resort.uuid} dense>
						<Checkbox
							size="small"
							checked={value.includes(resort.uuid)}
							sx={{ py: 0.5, pl: 0 }}
						/>
						<ListItemText primary={resort.name} />
					</MenuItem>
				))}
			</Select>
			<FormHelperText>{error || helperText}</FormHelperText>
		</FormControl>
	)
}
