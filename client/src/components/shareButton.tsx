import React, { useState } from 'react'
import { Button } from '@mui/material'
import { Check, IosShare } from '@mui/icons-material'

type ShareButtonProps = {
	url: string
	text: string
	label?: string
}

// Opens the phone's share sheet where there is one, and copies the link
// everywhere else. A closed share sheet is not an error, so it stays quiet.
export default function ShareButton({
	url,
	text,
	label = 'Share',
}: ShareButtonProps): React.ReactElement {
	const [copied, setCopied] = useState(false)

	const handleClick = async (): Promise<void> => {
		if (typeof navigator.share === 'function') {
			try {
				await navigator.share({ text, url })
			} catch {
				// Cancelled or blocked: nothing to tell the visitor.
			}
			return
		}

		try {
			await navigator.clipboard.writeText(url)
			setCopied(true)
		} catch {
			// Without a clipboard the link is still shown on the page.
		}
	}

	return (
		<Button
			variant="outlined"
			size="large"
			startIcon={copied ? <Check /> : <IosShare />}
			onClick={handleClick}
		>
			{copied ? 'Link copied' : label}
		</Button>
	)
}
