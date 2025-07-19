import React from 'react'
import { Box, Button, Container, Paper, Typography } from '@mui/material'
import { CheckCircle } from '@mui/icons-material'
import { Link } from 'react-router'

export default function SuccessPage() {
  return (
    <Container maxWidth='md' sx={{ py: 4 }}>
      <Paper elevation={3} sx={{ p: 4, textAlign: 'center' }}>
        <Box sx={{ mb: 3 }}>
          <CheckCircle
            sx={{
              fontSize: 80,
              color: 'success.main',
              mb: 2,
            }}
          />
        </Box>

        <Typography variant='h3' component='h1' gutterBottom>
          {"You're All Set!"}
        </Typography>
        <Typography
          variant='body1'
          color='text.secondary'
          sx={{ mb: 4, maxWidth: 600, mx: 'auto', lineHeight: 1.6 }}
        >
          {"Your powder alert has been created successfully. We'll notify you when " +
            'fresh snow is forecasted at your selected resorts. Get ready to hunt ' +
            'some powder!'}
        </Typography>

        <Box sx={{ display: 'flex', gap: 2, justifyContent: 'center' }}>
          <Button
            component={Link}
            to='/manage'
            variant='outlined'
            size='large'
          >
            Manage Subscriptions
          </Button>
          <Button
            component={Link}
            to='/'
            variant='contained'
            size='large'
          >
            Back to Home
          </Button>
        </Box>
      </Paper>
    </Container>
  )
}
