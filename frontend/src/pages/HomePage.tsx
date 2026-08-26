import { Container, Stack, Typography } from '@mui/material'

export function HomePage() {
  return (
    <Container maxWidth="md">
      <Stack spacing={2} sx={{ py: 8 }}>
        <Typography component="h1" variant="h3">
          Falqon Form Builder
        </Typography>
        <Typography color="text.secondary">
          Está funcionando.
        </Typography>
      </Stack>
    </Container>
  )
}

