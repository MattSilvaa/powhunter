import React, { useState } from 'react'
import {
	Box,
	Button,
	Container,
	TextField,
	Typography,
	Alert,
	CircularProgress,
} from '@mui/material'
import { Email as EmailIcon } from '@mui/icons-material'
import {BASE_SERVER_URL} from "../shared/types";

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
			const response = await fetch(`${BASE_SERVER_URL}/api/contact`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify(formData),
			})

			if (!response.ok) {
				const errorData = await response.json()
				throw new Error(errorData.message || 'Failed to send message')
			}

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
		<Box
			sx={{
				minHeight: '100vh',
				display: 'flex',
				flexDirection: 'column',
				justifyContent: 'center',
				bgcolor: 'background.default',
				py: { xs: 4, sm: 6, md: 8 },
			}}
		>
			<Container maxWidth="sm">
				<Box sx={{ textAlign: 'center', mb: { xs: 4, sm: 5 } }}>
					<EmailIcon
						sx={{
							fontSize: { xs: 40, sm: 48 },
							color: 'primary.main',
							mb: 2,
						}}
					/>
					<Typography
						variant="h2"
						component="h1"
						gutterBottom
						sx={{ fontSize: { xs: '2rem', sm: '2.5rem' } }}
					>
						Contact Us
					</Typography>
					<Typography variant="body1" color="text.secondary">
						Have questions or feedback? We'd love to hear from you!
					</Typography>
				</Box>

				<Box
					component="form"
					onSubmit={handleSubmit}
					sx={{
						display: 'flex',
						flexDirection: 'column',
						gap: { xs: 2.5, sm: 3 },
					}}
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
						sx={{
							py: 1.5,
							position: 'relative',
						}}
					>
						{loading ? (
							<>
								<CircularProgress size={24} sx={{ mr: 1 }} />
								Sending...
							</>
						) : (
							'Send Message'
						)}
					</Button>
				</Box>
			</Container>
		</Box>
	)
}
