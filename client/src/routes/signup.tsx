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
import {useResorts} from '../shared/useResorts.ts'
import {Resort} from '../shared/types.ts'
import {useCreateAlert} from '../shared/useCreateAlert.ts'

export default function SignUpPage() {
    const [formData, setFormData] = useState({
        email: '',
        phone: '',
        notificationDays: 3,
        minSnowAmount: 6,
        resorts: [] as string[],
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
    }

    const handleSelectChange = (e: SelectChangeEvent<string[]>) => {
        const {name, value} = e.target
        setFormData((prev) => ({
            ...prev,
            [name]: value,
        }))
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()

        if (!formData.email.trim()) {
            return
        }

        if (!formData.phone.trim()) {
            return
        }

        if (formData.resorts.length === 0) {
            return
        }

        // Map resort names to UUIDs more safely
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

        try {
            await createAlert({
                email: formData.email.trim(),
                phone: formData.phone.trim(),
                minSnowAmount: formData.minSnowAmount,
                notificationDays: formData.notificationDays,
                resortsUuids,
            })
            // TODO: Create a success page
            // TODO: Add check to remove duplicate
            setFormData({
                email: '',
                phone: '',
                notificationDays: 3,
                minSnowAmount: 6,
                resorts: [],
            })
        } catch (error) {
            console.error('Failed to create alert:', error)
        }
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
                                        helperText="We'll send SMS alerts to this number"
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
                                    max={5}
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
                                    min={1}
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
                                <FormControl fullWidth>
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
                                </FormControl>
                            </Grid>

                            <Grid size={{xs: 12}}>
                                <Button
                                    type='submit'
                                    variant='contained'
                                    size='large'
                                    fullWidth
                                    sx={{mt: 2}}
                                    disabled={!formData.email || !formData.phone ||
                                        !formData.resorts}
                                    onClick={handleSubmit}
                                >
                                    Create Alert
                                </Button>
                            </Grid>
                        </Grid>
                    </>
                )}
            </Paper>
        </Container>
    )
}
