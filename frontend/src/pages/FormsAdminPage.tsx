import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Container,
  Divider,
  Paper,
  Stack,
  Typography,
  alpha,
} from '@mui/material'
import { Link as RouterLink, Navigate, useNavigate } from 'react-router-dom'
import { getAuthSession, listForms, logout, publishForm, type FormState } from '../api/generated'

const statePresentation: Record<FormState, { label: string; color: string }> = {
  DRAFT: { label: 'Rascunho', color: '#f4b942' },
  PUBLISHED: { label: 'Publicado', color: '#63c48d' },
  CANCELED: { label: 'Cancelado', color: '#a3a7b0' },
}

export function FormsAdminPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const session = useQuery({
    queryKey: ['auth', 'session'],
    queryFn: async () => (await getAuthSession()).data ?? null,
    retry: false,
  })
  const forms = useQuery({
    queryKey: ['forms'],
    queryFn: async () => {
      const response = await listForms()
      if (response.error) throw new Error('Não foi possível carregar os formulários.')
      return response.data ?? []
    },
    enabled: Boolean(session.data),
    retry: false,
  })
  const logoutMutation = useMutation({
    mutationFn: async () => logout(),
    onSuccess: async () => {
      queryClient.clear()
      navigate('/')
    },
  })
  const publishMutation = useMutation({
    mutationFn: async (formId: number) => {
      const response = await publishForm({ path: { formId } })
      if (response.error) throw new Error('Não foi possível publicar o formulário.')
      return response.data
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['forms'] })
    },
  })

  if (session.isLoading) {
    return (
      <Box minHeight="100vh" display="grid" sx={{ placeItems: 'center' }}>
        <CircularProgress aria-label="Verificando sessão" />
      </Box>
    )
  }
  if (!session.data) return <Navigate replace to="/" />

  return (
    <Box minHeight="100vh" bgcolor="background.default">
      <Box
        component="header"
        sx={{
          borderBottom: '1px solid',
          borderColor: alpha('#ffffff', 0.08),
          bgcolor: alpha('#121419', 0.92),
        }}
      >
        <Container maxWidth="lg">
          <Stack direction="row" alignItems="center" justifyContent="space-between" py={2.25}>
            <Button component={RouterLink} color="inherit" to="/" sx={{ p: 0 }}>
              <Stack direction="row" alignItems="center" spacing={1.25}>
                <Box
                  sx={{
                    display: 'grid',
                    placeItems: 'center',
                    width: 32,
                    height: 32,
                    border: '1px solid',
                    borderColor: 'primary.main',
                    borderRadius: '50%',
                    color: 'primary.main',
                    fontWeight: 900,
                  }}
                >
                  F
                </Box>
                <Typography letterSpacing="0.16em" fontSize={13} fontWeight={800}>
                  FALQON
                </Typography>
              </Stack>
            </Button>
            <Stack direction="row" alignItems="center" spacing={2}>
              <Box textAlign="right" sx={{ display: { xs: 'none', sm: 'block' } }}>
                <Typography fontSize={13} fontWeight={700}>{session.data.name}</Typography>
                <Typography color="text.secondary" fontSize={11}>{session.data.email}</Typography>
              </Box>
              <Button
                color="inherit"
                disabled={logoutMutation.isPending}
                onClick={() => logoutMutation.mutate()}
                size="small"
              >
                Sair
              </Button>
            </Stack>
          </Stack>
        </Container>
      </Box>

      <Container component="main" maxWidth="lg" sx={{ py: { xs: 5, md: 7 } }}>
        <Stack spacing={4}>
          <Stack
            direction={{ xs: 'column', sm: 'row' }}
            alignItems={{ sm: 'flex-end' }}
            justifyContent="space-between"
            spacing={2}
          >
            <Box>
              <Typography color="primary.main" fontSize={11} fontWeight={800} letterSpacing="0.18em">
                ÁREA ADMINISTRATIVA
              </Typography>
              <Typography component="h1" variant="h3" fontFamily="Georgia, serif" sx={{ mt: 0.75 }}>
                Meus formulários
              </Typography>
              <Typography color="text.secondary" sx={{ mt: 1 }}>
                Gerencie suas pesquisas e acompanhe o estado de cada publicação.
              </Typography>
            </Box>
            <Button component={RouterLink} size="large" to="/admin/forms/new" variant="contained">
              Novo formulário
            </Button>
          </Stack>

          <Divider sx={{ borderColor: alpha('#ffffff', 0.08) }} />

          {forms.isLoading ? (
            <Stack direction="row" alignItems="center" spacing={2} py={6}>
              <CircularProgress size={24} />
              <Typography color="text.secondary">Carregando formulários...</Typography>
            </Stack>
          ) : forms.isError ? (
            <Alert severity="error" variant="outlined">{forms.error.message}</Alert>
          ) : forms.data?.length === 0 ? (
            <Paper
              variant="outlined"
              sx={{
                py: 8,
                px: 3,
                textAlign: 'center',
                borderColor: alpha('#ffffff', 0.1),
                bgcolor: alpha('#ffffff', 0.02),
              }}
            >
              <Typography fontFamily="Georgia, serif" fontSize={28}>Sua tela ainda está vazia.</Typography>
              <Typography color="text.secondary" sx={{ mt: 1, maxWidth: 500, mx: 'auto' }}>
                Crie seu primeiro formulário e comece uma conversa sobre cinema.
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
              {forms.data?.map((form) => {
                const state = statePresentation[form.state]
                return (
                  <Paper
                    key={form.id}
                    variant="outlined"
                    sx={{ p: 3, borderColor: alpha('#ffffff', 0.09), bgcolor: alpha('#ffffff', 0.025) }}
                  >
                    <Stack spacing={2.5}>
                      <Stack direction="row" justifyContent="space-between" spacing={2}>
                        <Box>
                          <Typography variant="h6" fontWeight={750}>{form.title}</Typography>
                          <Typography color="text.secondary" fontFamily="monospace" fontSize={12}>
                            /forms/{form.slug}
                          </Typography>
                        </Box>
                        <Chip
                          label={state.label}
                          size="small"
                          sx={{ color: state.color, borderColor: alpha(state.color, 0.45) }}
                          variant="outlined"
                        />
                      </Stack>
                      {form.description && (
                        <Typography color="text.secondary" fontSize={14} lineHeight={1.6}>
                          {form.description}
                        </Typography>
                      )}
                      <Typography color="text.disabled" fontSize={11}>
                        Atualizado em {new Intl.DateTimeFormat('pt-BR', { dateStyle: 'medium' }).format(new Date(form.updated_at))}
                      </Typography>
                      {form.state === 'DRAFT' && (
                        <Button
                          disabled={publishMutation.isPending}
                          onClick={() => publishMutation.mutate(form.id)}
                          variant="contained"
                        >
                          {publishMutation.isPending ? 'Publicando...' : 'Publicar formulário'}
                        </Button>
                      )}
                      {form.state === 'PUBLISHED' && (
                        <Button component={RouterLink} to={`/forms/${form.slug}`} variant="outlined">
                          Abrir formulário
                        </Button>
                      )}
                    </Stack>
                  </Paper>
                )
              })}
            </Box>
          )}
        </Stack>
      </Container>
    </Box>
  )
}
