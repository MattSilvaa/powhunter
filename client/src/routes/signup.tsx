import React, { useState } from "react";
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Container,
  FormControl,
  FormControlLabel,
  Grid,
  InputLabel,
  LinearProgress,
  MenuItem,
  Paper,
  Select,
  SelectChangeEvent,
  Slider,
  Switch,
  TextField,
  Typography,
} from "@mui/material";
import { useResorts } from "../shared/useResorts.ts";
import { Resort } from "../shared/types.ts";
import { useCreateAlert } from "../shared/useCreateAlert.ts";

export default function SignUpPage() {
  const [formData, setFormData] = useState({
    email: "",
    phone: "",
    notificationDays: 3,
    minSnowAmount: 6,
    resorts: [] as string[],
  });
  const {
    createAlert,
    loading: isCreateAlertLoading,
    error: createAlertError,
  } = useCreateAlert();

  const { resorts, loading, error } = useResorts();

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleSelectChange = (e: SelectChangeEvent<string[]>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createAlert({
      email: formData.email,
      phone: formData.phone,
      minSnowAmount: formData.minSnowAmount,
      notificationDays: formData.notificationDays,
      resorts: formData.resorts,
    });
  };

  return (
    <Container maxWidth="md" sx={{ py: 4 }}>
      <Paper elevation={3} sx={{ p: 4 }}>
        {loading ? (
          <LinearProgress />
        ) : (
          <>
            <Typography variant="h2" component="h1" gutterBottom align="center">
              Create Your Account
            </Typography>
            <Typography
              variant="body1"
              color="text.secondary"
              align="center"
              sx={{ mb: 4 }}
            >
              Sign up to start receiving powder alerts for your favorite resorts
            </Typography>

            <Grid container spacing={3}>
              <Grid size={4}>
                <TextField
                  required
                  fullWidth
                  label="Email"
                  name="email"
                  type="email"
                  value={formData.email}
                  onChange={handleChange}
                />
              </Grid>

              <Grid size={4}>
                <TextField
                  required
                  fullWidth
                  label="Phone Number"
                  name="phone"
                  type="tel"
                  value={formData.phone}
                  onChange={handleChange}
                  helperText="We'll send SMS alerts to this number"
                />
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
                    }))
                  }
                  min={1}
                  max={5}
                  marks
                  valueLabelDisplay="auto"
                />
                <Typography
                  variant="body2"
                  color="text.secondary"
                  align="center"
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
                    }))
                  }
                  min={1}
                  max={24}
                  marks
                  valueLabelDisplay="auto"
                />
                <Typography
                  variant="body2"
                  color="text.secondary"
                  align="center"
                >
                  {formData.minSnowAmount} inches
                </Typography>
              </Grid>

              <Grid size={{ xs: 12 }}>
                <FormControl fullWidth>
                  <InputLabel>Select Resorts</InputLabel>
                  {loading && (
                    <Box display="flex" justifyContent="center" p={2}>
                      <CircularProgress />
                    </Box>
                  )}
                  {error && <Alert severity="error">{error}</Alert>}

                  <Select
                    required
                    multiple
                    name="resorts"
                    value={formData.resorts}
                    onChange={handleSelectChange}
                    label="Select Resorts"
                  >
                    {resorts.map((resort: Resort) => (
                      <MenuItem key={resort.name} value={resort.name}>
                        {resort.name}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Grid>

              <Grid size={{ xs: 12 }}>
                <Button
                  type="submit"
                  variant="contained"
                  size="large"
                  fullWidth
                  sx={{ mt: 2 }}
                  disabled={
                    !formData.email || !formData.phone || !formData.resorts
                  }
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
  );
}
