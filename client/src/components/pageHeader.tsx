import React from 'react'
import { Box, Typography } from '@mui/material'

type PageHeaderProps = {
	title: string
	subtitle?: React.ReactNode
	action?: React.ReactNode
}

export default function PageHeader({
	title,
	subtitle,
	action,
}: PageHeaderProps): React.ReactElement {
	return (
		<Box
			sx={{
				display: 'flex',
				justifyContent: 'space-between',
				alignItems: 'flex-start',
				gap: 2,
				mb: { xs: 3, sm: 4 },
			}}
		>
			<Box>
				<Typography variant="h3" component="h1">
					{title}
				</Typography>
				{subtitle && (
					<Typography color="text.secondary" sx={{ mt: 1 }}>
						{subtitle}
					</Typography>
				)}
			</Box>
			{action}
		</Box>
	)
}
