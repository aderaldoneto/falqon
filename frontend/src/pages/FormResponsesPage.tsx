import { useQuery } from '@tanstack/react-query'
import { Alert, Box, Button, CircularProgress, Container, Divider, Paper, Stack, Typography, alpha } from '@mui/material'
import { Link as RouterLink, Navigate, useParams } from 'react-router-dom'
import { getAuthSession, listFormSubmissions } from '../api/generated'
import { displayValue } from './pageHelpers'

export function FormResponsesPage() {
  const { formId = '' } = useParams()
  const id = Number(formId)
  const session = useQuery({
    queryKey: ['auth', 'session'],
    queryFn: async () => (await getAuthSession()).data ?? null,
    retry: false,
  })
  const responses = useQuery({
    queryKey: ['form-submissions', id],
    queryFn: async () => {
      const response = await listFormSubmissions({ path: { formId: id } })
      if (response.error) throw new Error('Não foi possível carregar as respostas.')
      return response.data
    },
    enabled: Boolean(session.data && Number.isInteger(id) && id > 0),
    retry: false,
  })

  if (session.isLoading) return <Box minHeight="100vh" display="grid" sx={{ placeItems: 'center' }}><CircularProgress /></Box>
  if (!session.data) return <Navigate replace to="/" />

  return (
    <Box minHeight="100vh" bgcolor="background.default">
      <Container component="main" maxWidth="md" sx={{ py: { xs: 4, md: 7 } }}>
        <Stack spacing={4}>
          <Button component={RouterLink} to="/admin/forms" color="inherit" sx={{ alignSelf: 'flex-start' }}>← Voltar aos formulários</Button>
          {responses.isLoading ? <CircularProgress /> : responses.isError || !responses.data ? (
            <Alert severity="error">{responses.error?.message ?? 'Formulário não encontrado.'}</Alert>
          ) : (
            <>
              <Box>
                <Typography color="primary.main" fontSize={11} fontWeight={800} letterSpacing="0.18em">RESPOSTAS</Typography>
                <Typography component="h1" variant="h3" fontFamily="Georgia, serif" sx={{ mt: 1 }}>{responses.data.title}</Typography>
                <Typography color="text.secondary" sx={{ mt: 1 }}>{responses.data.submissions.length} resposta(s) recebida(s)</Typography>
              </Box>
              <Divider sx={{ borderColor: alpha('#ffffff', 0.08) }} />
              {responses.data.submissions.length === 0 ? (
                <Paper variant="outlined" sx={{ p: 6, textAlign: 'center', borderColor: alpha('#ffffff', 0.1) }}>
                  <Typography color="text.secondary">Este formulário ainda não recebeu respostas.</Typography>
                </Paper>
              ) : responses.data.submissions.map((submission, index) => (
                <Paper key={submission.id} variant="outlined" sx={{ p: { xs: 3, sm: 4 }, borderColor: alpha('#ffffff', 0.1) }}>
                  <Stack spacing={3}>
                    <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between">
                      <Typography fontWeight={800}>Resposta #{responses.data!.submissions.length - index}</Typography>
                      <Typography color="text.secondary" fontSize={13}>{new Intl.DateTimeFormat('pt-BR', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(submission.created_at))}</Typography>
                    </Stack>
                    <Divider />
                    {submission.answers.map((answer) => (
                      <Box key={answer.field_id}>
                        <Typography color="text.secondary" fontSize={12} fontWeight={700}>{answer.label}</Typography>
                        <Typography sx={{ mt: 0.5, whiteSpace: 'pre-wrap' }}>{displayValue(answer.value)}</Typography>
                      </Box>
                    ))}
                  </Stack>
                </Paper>
              ))}
            </>
          )}
        </Stack>
      </Container>
    </Box>
  )
}
