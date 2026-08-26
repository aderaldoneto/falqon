import { useQuery } from '@tanstack/react-query'
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Container,
  Divider,
  Paper,
  Stack,
  Typography,
  alpha,
} from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import { listPublishedForms } from '../api/generated'

export function PublicFormsPage() {
  const forms = useQuery({
    queryKey: ['published-forms'],
    queryFn: async () => {
      const response = await listPublishedForms()
      if (response.error) throw new Error('Não foi possível carregar os formulários.')
      return response.data ?? []
    },
    retry: false,
  })

  return (
    <Box
      minHeight="100vh"
      sx={{
        background: `
          radial-gradient(circle at 85% 10%, ${alpha('#b62935', 0.2)}, transparent 32%),
          radial-gradient(circle at 10% 80%, ${alpha('#f4b942', 0.08)}, transparent 30%),
          #090a0d
        `,
      }}
    >
      <Container maxWidth="lg" sx={{ py: { xs: 4, md: 7 } }}>
        <Stack spacing={4}>
          <Stack spacing={2}>
            <Button
              component={RouterLink}
              color="inherit"
              to="/"
              sx={{ alignSelf: 'flex-start', px: 0 }}
            >
              FALQON
            </Button>
            <Divider sx={{ borderColor: alpha('#ffffff', 0.08) }} />
            <Box>
              <Typography
                color="primary.main"
                fontSize={11}
                fontWeight={800}
                letterSpacing="0.18em"
              >
                PARTICIPE DAS REVIEWS
              </Typography>
              <Typography component="h1" variant="h3" fontFamily="Georgia, serif" sx={{ mt: 0.75 }}>
                Formulários disponíveis
              </Typography>
              <Typography color="text.secondary" sx={{ mt: 1 }}>
                Escolha uma pesquisa publicada e compartilhe sua opinião.
              </Typography>
            </Box>
          </Stack>

          {forms.isLoading ? (
            <Stack direction="row" alignItems="center" spacing={2} py={6}>
              <CircularProgress size={24} />
              <Typography color="text.secondary">Carregando formulários...</Typography>
            </Stack>
          ) : forms.isError ? (
            <Alert severity="error" variant="outlined">
              {forms.error.message}
            </Alert>
          ) : forms.data?.length === 0 ? (
            <Paper
              variant="outlined"
              sx={{ py: 8, px: 3, textAlign: 'center', borderColor: alpha('#ffffff', 0.1) }}
            >
              <Typography fontFamily="Georgia, serif" fontSize={28}>
                Nenhum formulário publicado.
              </Typography>
              <Typography color="text.secondary" sx={{ mt: 1 }}>
                Novas pesquisas aparecerão aqui quando estiverem disponíveis.
              </Typography>
            </Paper>
          ) : (
            <Box
              sx={{
                display: 'grid',
                gridTemplateColumns: { xs: '1fr', md: 'repeat(2, 1fr)' },
                gap: 2,
              }}
            >
              {forms.data?.map((form) => (
                <Paper
                  key={form.id}
                  variant="outlined"
                  sx={{
                    p: 3,
                    borderColor: alpha('#ffffff', 0.1),
                    bgcolor: alpha('#ffffff', 0.025),
                  }}
                >
                  <Stack spacing={2.5}>
                    <Box>
                      <Typography variant="h5" fontWeight={750}>
                        {form.title}
                      </Typography>
                      {form.description && (
                        <Typography color="text.secondary" sx={{ mt: 1 }}>
                          {form.description}
                        </Typography>
                      )}
                    </Box>
                    <Button
                      component={RouterLink}
                      to={`/forms/${form.slug}`}
                      variant="contained"
                      size="large"
                    >
                      Responder formulário
                    </Button>
                  </Stack>
                </Paper>
              ))}
            </Box>
          )}
        </Stack>
      </Container>
    </Box>
  )
}
