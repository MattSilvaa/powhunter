import React from 'react'
import { Box, Typography } from '@mui/material'
import { tabularNums } from '../theme.ts'

type SliderFieldProps = {
	label: string
	hint: string
	value: string
	children: React.ReactNode
}

// Shows a slider's current value beside its label, like a forecast readout,
// instead of hiding it in a tooltip.
export default function SliderField({
	label,
	hint,
	value,
	children,
}: SliderFieldProps): React.ReactElement {
	return (
		<Box>
			<Box
				sx={{
					display: 'flex',
					justifyContent: 'space-between',
					alignItems: 'flex-start',
					gap: 2,
				}}
			>
				<Box>
					<Typography sx={{ fontWeight: 500 }}>{label}</Typography>
					<Typography variant="body2" color="text.secondary">
						{hint}
					</Typography>
				</Box>
				<Typography
					aria-hidden
					sx={{
						...tabularNums,
						fontSize: '1.5rem',
						fontWeight: 600,
						letterSpacing: '-0.02em',
						whiteSpace: 'nowrap',
					}}
				>
					{value}
				</Typography>
			</Box>
			<Box sx={{ px: 1, mt: 1 }}>{children}</Box>
		</Box>
	)
}
