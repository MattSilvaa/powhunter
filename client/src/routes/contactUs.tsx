import React, { useState } from 'react'
import {
	Alert,
	Box,
	Button,
	CircularProgress,
	Container,
	Paper,
	TextField,
} from '@mui/material'
import { apiRequest } from '../shared/apiClient.ts'
import PageHeader from '../components/pageHeader.tsx'

interface ContactFormData {
	name: string
	email: string
	message: string
}

export default function ContactUs() {
	const [formData, setFormData] = useState<ContactFormData>({
		name: '',
		email: '',
		message: '',
	})
	const [loading, setLoading] = useState(false)
	const [success, setSuccess] = useState(false)
	const [error, setError] = useState<string | null>(null)

	const handleChange = (
		e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
	) => {
		const { name, value } = e.target
		setFormData((prev) => ({
			...prev,
			[name]: value,
		}))
	}

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault()
		setLoading(true)
		setError(null)
		setSuccess(false)

		try {
			await apiRequest('/api/contact', { method: 'POST', body: formData })

			setSuccess(true)
			setFormData({ name: '', email: '', message: '' })
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: 'Failed to send message. Please try again.'
			)
		} finally {
			setLoading(false)
		}
	}

	return (
		<Container maxWidth="sm" sx={{ py: { xs: 5, md: 8 } }}>
			<PageHeader
				title="Contact Us"
				subtitle="Have questions or feedback? We'd love to hear from you!"
			/>

			<Paper variant="outlined" sx={{ p: { xs: 2.5, sm: 4 } }}>
				<Box
					component="form"
					onSubmit={handleSubmit}
					sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}
				>
					{success && (
						<Alert severity="success" onClose={() => setSuccess(false)}>
							Thank you for your message! We'll get back to you soon.
						</Alert>
					)}

					{error && (
						<Alert severity="error" onClose={() => setError(null)}>
							{error}
						</Alert>
					)}

					<TextField
						fullWidth
						label="Name"
						name="name"
						autoComplete="name"
						value={formData.name}
						onChange={handleChange}
						required
						disabled={loading}
					/>

					<TextField
						fullWidth
						label="Email"
						name="email"
						type="email"
						autoComplete="email"
						value={formData.email}
						onChange={handleChange}
						required
						disabled={loading}
					/>

					<TextField
						fullWidth
						label="Message"
						name="message"
						multiline
						rows={6}
						value={formData.message}
						onChange={handleChange}
						required
						disabled={loading}
					/>

					<Button
						type="submit"
						variant="contained"
						size="large"
						disabled={loading}
					>
						{loading ? (
							<>
								<CircularProgress size={18} color="inherit" sx={{ mr: 1 }} />
								Sending...
							</>
						) : (
							'Send Message'
						)}
					</Button>
				</Box>
			</Paper>
		</Container>
	)
}
