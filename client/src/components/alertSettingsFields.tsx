import React from 'react'
import { Box, Slider } from '@mui/material'
import SliderField from './sliderField.tsx'
import { AlertSettings } from '../shared/types.ts'

type AlertSettingsFieldsProps = {
	value: AlertSettings
	onChange: (value: AlertSettings) => void
}

// The two thresholds every alert carries. Signup and the manage page share
// them so the ranges stay in step with what the server accepts.
export default function AlertSettingsFields({
	value,
	onChange,
}: AlertSettingsFieldsProps): React.ReactElement {
	return (
		<Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
			<SliderField
				label="Minimum snowfall"
				hint="Only alert me when at least this much is forecast"
				value={`${value.minSnowAmount}″`}
			>
				<Slider
					aria-label="Minimum snow amount in inches"
					value={value.minSnowAmount}
					onChange={(_, amount) =>
						onChange({ ...value, minSnowAmount: amount as number })
					}
					min={1}
					max={24}
					valueLabelDisplay="off"
				/>
			</SliderField>

			<SliderField
				label="Advance notice"
				hint="How far ahead of the snow you want to hear about it"
				value={`${value.notificationDays} ${value.notificationDays === 1 ? 'day' : 'days'}`}
			>
				<Slider
					aria-label="Days of advance notice"
					value={value.notificationDays}
					onChange={(_, days) =>
						onChange({ ...value, notificationDays: days as number })
					}
					min={1}
					max={10}
					valueLabelDisplay="off"
				/>
			</SliderField>
		</Box>
	)
}
