import React, {useState} from 'react'
import {
    Alert,
    Box,
    Button,
    CircularProgress,
    Container,
    FormControl,
    Grid,
    InputLabel,
    LinearProgress,
    MenuItem,
    Paper,
    Select,
    SelectChangeEvent,
    Slider,
    TextField,
    Typography,
} from '@mui/material'
import {useNavigate} from 'react-router'
import {useResorts} from '../shared/useResorts'
import {Resort} from '../shared/types'
import {useCreateAlert} from '../shared/useCreateAlert'

export default function SignUpPage() {
    const navigate = useNavigate()
    const [formData, setFormData] = useState({
        email: '',
        phone: '',
        notificationDays: 3,
        minSnowAmount: 6,
        resorts: [] as string[],
    })
    const [fieldErrors, setFieldErrors] = useState({
        email: '',
        phone: '',
        resorts: '',
    })
    const {
        createAlert,
        loading: isCreateAlertLoading,
        error: createAlertError,
    } = useCreateAlert()

    const {resorts = [], loading, error} = useResorts()

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const {name, value} = e.target
        setFormData((prev) => ({
            ...prev,
            [name]: value,
        }))
        
        // Clear field-specific error when user starts typing
        if (fieldErrors[name as keyof typeof fieldErrors]) {
            setFieldErrors(prev => ({
                ...prev,
                [name]: '',
            }))
        }
    }

    const handleSelectChange = (e: SelectChangeEvent<string[]>) => {
        const {name, value} = e.target
        setFormData((prev) => ({
            ...prev,
            [name]: value,
        }))
        
        // Clear resorts error when user selects resorts
        if (name === 'resorts' && fieldErrors.resorts) {
            setFieldErrors(prev => ({
                ...prev,
                resorts: '',
            }))
        }
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()

        // Clear previous field errors
        setFieldErrors({
            email: '',
            phone: '',
            resorts: '',
        })

        // Validate form fields
        const errors = {
            email: '',
            phone: '',
            resorts: '',
        }

        if (!formData.email.trim()) {
            errors.email = 'Email is required'
        } else if (!/\S+@\S+\.\S+/.test(formData.email)) {
            errors.email = 'Please enter a valid email address'
        }

        if (!formData.phone.trim()) {
            errors.phone = 'Phone number is required'
        }

        if (formData.resorts.length === 0) {
            errors.resorts = 'Please select at least one resort'
        }

        // If there are validation errors, show them and don't submit
        if (errors.email || errors.phone || errors.resorts) {
            setFieldErrors(errors)
            return
        }

        const resortsUuids = formData.resorts
            .map((resortName) => {
                const resort = resorts.find((r) => r.name === resortName)
                if (!resort?.uuid) {
                    console.error(`Resort UUID not found for: ${resortName}`)
                }
                return resort?.uuid
            })
            .filter((uuid) => !!uuid) as string[]

        if (resortsUuids.length !== formData.resorts.length) {
            console.error(
                "Some selected resorts couldn't be properly mapped to UUIDs",
            )
            return
        }

        createAlert({
            email: formData.email.trim(),
            phone: formData.phone.trim(),
            minSnowAmount: formData.minSnowAmount,
            notificationDays: formData.notificationDays,
            resortsUuids,
        }, {
            onSuccess: () => {
                navigate('/success')
            },
            onError: (error) => {
                console.error('Failed to create alert:', error)
            }
        })
    }

    return (
        <Container maxWidth='md' sx={{py: 4}}>
            <Paper elevation={3} sx={{p: 4}}>
                {loading ? <LinearProgress/> : (
                    <>
                        <Typography variant='h2' component='h1' gutterBottom align='center'>
                            Start Hunting Powder
                        </Typography>
                        <Typography
                            variant='body1'
                            color='text.secondary'
                            align='center'
                            sx={{mb: 4}}
                        >
                            Sign up to start receiving powder alerts for your favorite resorts
                        </Typography>

                        {createAlertError && (
                            <Alert 
                                severity='error' 
                                sx={{mb: 3}}
                                onClose={() => {
                                    // Reset error when user dismisses
                                }}
                            >
                                <strong>Oops!</strong> {createAlertError}
                            </Alert>
                        )}

                        <Grid container spacing={3}>
                            <Grid container spacing={3} size={12}>
                                <Grid size={6}>
                                    <TextField
                                        required
                                        fullWidth
                                        label='Email'
                                        name='email'
                                        type='email'
                                        value={formData.email}
                                        onChange={handleChange}
                                        error={!!fieldErrors.email}
                                        helperText={fieldErrors.email || 'We\'ll use this to send you powder alerts'}
                                    />
                                </Grid>
                                <Grid size={6}>
                                    <TextField
                                        required
                                        fullWidth
                                        label='Phone Number'
                                        name='phone'
                                        type='tel'
                                        value={formData.phone}
                                        onChange={handleChange}
                                        error={!!fieldErrors.phone}
                                        helperText={fieldErrors.phone || "We'll send SMS alerts to this number"}
                                    />
                                </Grid>
                            </Grid>

                            <Grid size={12}>
                                <Typography gutterBottom>
                                    How many days in advance would you like to receive alerts?
                                </Typography>
                                <Slider
                                    value={formData.notificationDays}
                                    onChange={(_, value) =>
                                        setFormData((prev) => ({
                                            ...prev,
                                            notificationDays: value as number,
                                        }))}
                                    min={1}
                                    max={10}
                                    marks
                                    valueLabelDisplay='auto'
                                />
                                <Typography
                                    variant='body2'
                                    color='text.secondary'
                                    align='center'
                                >
                                    {formData.notificationDays} days
                                </Typography>
                            </Grid>

                            <Grid size={12}>
                                <Typography gutterBottom>
                                    Minimum snow amount for alerts (inches)?
                                </Typography>
                                <Slider
                                    value={formData.minSnowAmount}
                                    onChange={(_, value) =>
                                        setFormData((prev) => ({
                                            ...prev,
                                            minSnowAmount: value as number,
                                        }))}
                                    min={0}
                                    max={24}
                                    marks
                                    valueLabelDisplay='auto'
                                />
                                <Typography
                                    variant='body2'
                                    color='text.secondary'
                                    align='center'
                                >
                                    {formData.minSnowAmount} inches
                                </Typography>
                            </Grid>

                            <Grid size={{xs: 12}}>
                                <FormControl fullWidth error={!!fieldErrors.resorts}>
                                    <InputLabel>Select Resorts</InputLabel>
                                    {loading && (
                                        <Box display='flex' justifyContent='center' p={2}>
                                            <CircularProgress/>
                                        </Box>
                                    )}
                                    {error && <Alert severity='error'>{error}</Alert>}

                                    <Select
                                        required
                                        multiple
                                        name='resorts'
                                        value={formData.resorts}
                                        onChange={handleSelectChange}
                                        label='Select Resorts'
                                    >
                                        {resorts.map((resort: Resort) => (
                                            <MenuItem key={resort.uuid} value={resort.name}>
                                                {resort.name}
                                            </MenuItem>
                                        ))}
                                    </Select>
                                    {fieldErrors.resorts && (
                                        <Typography variant='caption' color='error' sx={{mt: 0.5, ml: 1.5}}>
                                            {fieldErrors.resorts}
                                        </Typography>
                                    )}
                                </FormControl>
                            </Grid>

                            <Grid size={{xs: 12}}>
                                <Button
                                    type='submit'
                                    variant='contained'
                                    size='large'
                                    fullWidth
                                    sx={{mt: 2}}
                                    disabled={isCreateAlertLoading}
                                    onClick={handleSubmit}
                                >
                                    {isCreateAlertLoading ? 'Creating Alert...' : 'Create Alert'}
                                </Button>
                            </Grid>
                        </Grid>
                    </>
                )}
            </Paper>
        </Container>
    )
}
