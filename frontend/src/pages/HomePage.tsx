import { useMemo } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
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
import { getAuthSession, logout } from '../api/generated'
import { Link as RouterLink } from 'react-router-dom'

const sessionQueryKey = ['auth', 'session'] as const

const reviewHighlights = [
  { title: 'The Godfather', detail: 'Drama · 1972', score: '9.2' },
  { title: 'Past Lives', detail: 'Romance · 2023', score: '8.4' },
  { title: 'Parasite', detail: 'Thriller · 2019', score: '8.8' },
]

const benefits = [
  {
    number: '01',
    title: 'Crie sua curadoria',
    description: 'Monte perguntas que combinam com cada filme.',
  },
  {
    number: '02',
    title: 'Compartilhe a sessão',
    description: 'Publique um link simples para sua audiência.',
  },
  {
    number: '03',
    title: 'Leia novas perspectivas',
    description: 'Acompanhe todas as reviews em um só lugar.',
  },
]

export function HomePage() {
  const queryClient = useQueryClient()
  const apiURL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'
  const authError = useMemo(
    () => new URLSearchParams(window.location.search).get('auth_error'),
    [],
  )

  const session = useQuery({
    queryKey: sessionQueryKey,
    queryFn: async () => {
      const response = await getAuthSession()
      return response.data ?? null
    },
    retry: false,
  })

  const logoutMutation = useMutation({
    mutationFn: async () => {
      await logout()
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: sessionQueryKey })
    },
  })

  const beginGoogleLogin = () => {
    window.location.assign(`${apiURL}/auth/google`)
  }

  return (
    <Box
      sx={{
        minHeight: '100vh',
        overflow: 'hidden',
        background: `
          radial-gradient(circle at 80% 8%, ${alpha('#b62935', 0.22)}, transparent 32%),
          radial-gradient(circle at 8% 70%, ${alpha('#f4b942', 0.08)}, transparent 28%),
          #090a0d
        `,
      }}
    >
      <Container maxWidth="lg">
        <Stack
          component="header"
          direction="row"
          alignItems="center"
          justifyContent="space-between"
          sx={{ py: 3 }}
        >
          <Stack direction="row" alignItems="center" spacing={1.5}>
            <Box
              aria-hidden="true"
              sx={{
                display: 'grid',
                placeItems: 'center',
                width: 34,
                height: 34,
                border: '1px solid',
                borderColor: 'primary.main',
                borderRadius: '50%',
                color: 'primary.main',
                fontSize: 15,
                fontWeight: 900,
              }}
            >
              F
            </Box>
            <Typography letterSpacing="0.18em" fontSize={14} fontWeight={800}>
              FALQON
            </Typography>
          </Stack>
          <Stack direction="row" alignItems="center" spacing={{ xs: 1, sm: 2 }}>
            <Button component={RouterLink} size="small" to="/forms" variant="outlined">
              Ver formulários para responder
            </Button>
          </Stack>
        </Stack>

        <Box
          component="main"
          sx={{
            display: 'grid',
            gridTemplateColumns: { xs: '1fr', md: 'minmax(0, 1.35fr) minmax(330px, 0.65fr)' },
            alignItems: 'center',
            gap: { xs: 6, md: 8 },
            minHeight: { md: 'calc(100vh - 210px)' },
            py: { xs: 6, md: 8 },
          }}
        >
          <Stack spacing={4}>
            <Stack spacing={2.5}>
              <Typography
                color="primary.main"
                fontSize={12}
                fontWeight={800}
                letterSpacing="0.2em"
              >
                FORMULÁRIOS PARA CINÉFILOS
              </Typography>
              <Typography
                component="h1"
                sx={{
                  maxWidth: 720,
                  fontFamily: 'Georgia, "Times New Roman", serif',
                  fontSize: { xs: '3rem', sm: '4.5rem', md: '5.2rem' },
                  fontWeight: 500,
                  letterSpacing: '-0.055em',
                  lineHeight: 0.98,
                }}
              >
                Todo filme merece uma boa conversa.
              </Typography>
              <Typography
                color="text.secondary"
                sx={{ maxWidth: 590, fontSize: { xs: 17, sm: 19 }, lineHeight: 1.65 }}
              >
                Crie questionários únicos, compartilhe com quem assistiu e
                descubra novos olhares sobre as histórias que ficaram com você.
              </Typography>
            </Stack>

            <Stack spacing={1.25} sx={{ maxWidth: 590 }}>
              {reviewHighlights.map((review, index) => (
                <Paper
                  key={review.title}
                  variant="outlined"
                  sx={{
                    p: 1.7,
                    pl: 2,
                    borderColor: alpha('#ffffff', 0.09),
                    bgcolor: alpha('#ffffff', index === 0 ? 0.055 : 0.025),
                    transform: `translateX(${index * 12}px)`,
                  }}
                >
                  <Stack direction="row" alignItems="center" justifyContent="space-between">
                    <Box>
                      <Typography fontWeight={700}>{review.title}</Typography>
                      <Typography color="text.secondary" fontSize={12}>
                        {review.detail}
                      </Typography>
                    </Box>
                    <Stack direction="row" alignItems="baseline" spacing={0.6}>
                      <Typography color="primary.main" fontSize={13}>
                        ★
                      </Typography>
                      <Typography fontFamily="Georgia, serif" fontSize={22}>
                        {review.score}
                      </Typography>
                    </Stack>
                  </Stack>
                </Paper>
              ))}
            </Stack>
          </Stack>

          <Paper
            elevation={0}
            sx={{
              position: 'relative',
              overflow: 'hidden',
              p: { xs: 3, sm: 4 },
              border: '1px solid',
              borderColor: alpha('#ffffff', 0.1),
              bgcolor: alpha('#15171c', 0.9),
              boxShadow: `0 30px 80px ${alpha('#000000', 0.38)}`,
              '&::before': {
                content: '""',
                position: 'absolute',
                inset: '0 0 auto',
                height: 3,
                background: 'linear-gradient(90deg, #f4b942, #b62935)',
              },
            }}
          >
            <Stack spacing={3}>
              <Box>
                <Typography
                  color="primary.main"
                  fontSize={11}
                  fontWeight={800}
                  letterSpacing="0.18em"
                >
                  ÁREA DO CURADOR
                </Typography>
                <Typography component="h2" variant="h5" fontWeight={750} sx={{ mt: 1 }}>
                  Sua próxima sessão começa aqui.
                </Typography>
              </Box>

              {authError && (
                <Alert severity="error" variant="outlined">
                  Não foi possível concluir a autenticação com o Google.
                </Alert>
              )}

              <Divider sx={{ borderColor: alpha('#ffffff', 0.08) }} />

              {session.isLoading ? (
                <Stack direction="row" alignItems="center" spacing={2}>
                  <CircularProgress aria-label="Carregando sessão" size={24} />
                  <Typography color="text.secondary">Carregando sua sessão...</Typography>
                </Stack>
              ) : session.data ? (
                <Stack spacing={2.5}>
                  <Box>
                    <Typography fontSize={13} color="text.secondary">
                      Bem-vindo de volta
                    </Typography>
                    <Typography variant="h6" fontWeight={750}>
                      {session.data.name}
                    </Typography>
                    <Typography color="text.secondary" fontSize={14}>
                      {session.data.email}
                    </Typography>
                  </Box>
                  <Button
                    component={RouterLink}
                    fullWidth
                    size="large"
                    to="/admin/forms"
                    variant="contained"
                  >
                    Acessar meus formulários
                  </Button>
                  <Button
                    color="inherit"
                    disabled={logoutMutation.isPending}
                    onClick={() => logoutMutation.mutate()}
                    size="small"
                  >
                    Sair da conta
                  </Button>
                </Stack>
              ) : (
                <Stack spacing={2.5}>
                  <Typography color="text.secondary" lineHeight={1.65}>
                    Entre para criar seus formulários de review e acompanhar o
                    que sua audiência tem a dizer.
                  </Typography>
                  <Button
                    fullWidth
                    onClick={beginGoogleLogin}
                    size="large"
                    variant="contained"
                    sx={{ py: 1.35 }}
                  >
                    Continuar com Google
                  </Button>
                  <Button
                    component={RouterLink}
                    fullWidth
                    size="large"
                    to="/register"
                    variant="outlined"
                  >
                    Cadastrar
                  </Button>
                  <Typography color="text.disabled" fontSize={11} textAlign="center">
                    A página pública dos formulários não exige login.
                  </Typography>
                </Stack>
              )}
            </Stack>
          </Paper>
        </Box>

        <Box
          component="section"
          aria-label="Como funciona"
          sx={{
            display: 'grid',
            gridTemplateColumns: { xs: '1fr', sm: 'repeat(3, 1fr)' },
            borderTop: '1px solid',
            borderColor: alpha('#ffffff', 0.08),
          }}
        >
          {benefits.map((benefit) => (
            <Stack
              key={benefit.number}
              spacing={1}
              sx={{
                py: 3.5,
                pr: 3,
                borderRight: { sm: '1px solid' },
                borderColor: { sm: alpha('#ffffff', 0.08) },
                '&:not(:first-of-type)': { pl: { sm: 3 } },
                '&:last-of-type': { borderRight: 0 },
              }}
            >
              <Typography color="primary.main" fontFamily="monospace" fontSize={11}>
                {benefit.number}
              </Typography>
              <Typography fontWeight={700}>{benefit.title}</Typography>
              <Typography color="text.secondary" fontSize={13} lineHeight={1.55}>
                {benefit.description}
              </Typography>
            </Stack>
          ))}
        </Box>
      </Container>
    </Box>
  )
}
