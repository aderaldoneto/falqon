import { type FormEvent, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Alert,
  Box,
  Button,
  Container,
  Divider,
  Link,
  Paper,
  Stack,
  TextField,
  Typography,
  alpha,
} from '@mui/material'
import { Link as RouterLink, useNavigate } from 'react-router-dom'
import { registerUser } from '../api/generated'

type RegistrationForm = {
  name: string
  email: string
  password: string
  passwordConfirmation: string
}

const initialForm: RegistrationForm = {
  name: '',
  email: '',
  password: '',
  passwordConfirmation: '',
}

export function RegisterPage() {
  const apiURL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [form, setForm] = useState(initialForm)
  const [validationError, setValidationError] = useState<string | null>(null)

  const registration = useMutation({
    mutationFn: async () => {
      const response = await registerUser({
        body: {
          name: form.name.trim(),
          email: form.email.trim(),
          password: form.password,
        },
      })
      if (response.error) {
        const error = response.error as { code?: string }
        if (response.response?.status === 409 || error.code === 'email_already_registered') {
          throw new Error('Este e-mail já está cadastrado.')
        }
        throw new Error('Não foi possível concluir o cadastro.')
      }
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['auth', 'session'] })
      navigate('/')
    },
  })

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setValidationError(null)

    if (form.password.length < 8) {
      setValidationError('A senha deve ter pelo menos 8 caracteres.')
      return
    }
    if (form.password !== form.passwordConfirmation) {
      setValidationError('As senhas informadas não são iguais.')
      return
    }
    registration.mutate()
  }

  const beginGoogleRegistration = () => {
    window.location.assign(`${apiURL}/auth/google`)
  }

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'grid',
        placeItems: 'center',
        py: 5,
        background: `
          radial-gradient(circle at 15% 15%, ${alpha('#f4b942', 0.1)}, transparent 28%),
          radial-gradient(circle at 85% 85%, ${alpha('#b62935', 0.18)}, transparent 32%),
          #090a0d
        `,
      }}
    >
      <Container maxWidth="sm">
        <Stack spacing={3}>
          <Link
            component={RouterLink}
            color="inherit"
            to="/"
            underline="none"
            sx={{ alignSelf: 'flex-start' }}
          >
            <Stack direction="row" alignItems="center" spacing={1.5}>
              <Box
                sx={{
                  display: 'grid',
                  placeItems: 'center',
                  width: 34,
                  height: 34,
                  border: '1px solid',
                  borderColor: 'primary.main',
                  borderRadius: '50%',
                  color: 'primary.main',
                  fontWeight: 900,
                }}
              >
                F
              </Box>
              <Typography letterSpacing="0.18em" fontSize={14} fontWeight={800}>
                FALQON
              </Typography>
            </Stack>
          </Link>

          <Paper
            component="main"
            elevation={0}
            sx={{
              p: { xs: 3, sm: 5 },
              border: '1px solid',
              borderColor: alpha('#ffffff', 0.1),
              bgcolor: alpha('#15171c', 0.94),
              boxShadow: `0 30px 80px ${alpha('#000000', 0.4)}`,
            }}
          >
            <Stack spacing={3}>
              <Box>
                <Typography color="primary.main" fontSize={11} fontWeight={800} letterSpacing="0.18em">
                  CRIE SUA CONTA
                </Typography>
                <Typography component="h1" variant="h4" fontWeight={750} sx={{ mt: 1 }}>
                  Comece sua curadoria.
                </Typography>
                <Typography color="text.secondary" sx={{ mt: 1 }}>
                  Cadastre-se para criar e publicar formulários sobre seus filmes favoritos.
                </Typography>
              </Box>

              <Button fullWidth onClick={beginGoogleRegistration} size="large" variant="outlined">
                Cadastrar com Google
              </Button>

              <Divider>ou use seu e-mail</Divider>

              <Stack component="form" onSubmit={handleSubmit} spacing={2.25}>
                {(validationError || registration.error) && (
                  <Alert severity="error" variant="outlined">
                    {validationError ?? registration.error?.message}
                  </Alert>
                )}
                <TextField
                  autoComplete="name"
                  label="Nome"
                  onChange={(event) => setForm({ ...form, name: event.target.value })}
                  required
                  value={form.name}
                />
                <TextField
                  autoComplete="email"
                  label="E-mail"
                  onChange={(event) => setForm({ ...form, email: event.target.value })}
                  required
                  type="email"
                  value={form.email}
                />
                <TextField
                  autoComplete="new-password"
                  helperText="Use pelo menos 8 caracteres."
                  label="Senha"
                  onChange={(event) => setForm({ ...form, password: event.target.value })}
                  required
                  type="password"
                  value={form.password}
                />
                <TextField
                  autoComplete="new-password"
                  label="Confirme a senha"
                  onChange={(event) =>
                    setForm({ ...form, passwordConfirmation: event.target.value })
                  }
                  required
                  type="password"
                  value={form.passwordConfirmation}
                />
                <Button
                  disabled={registration.isPending}
                  fullWidth
                  size="large"
                  type="submit"
                  variant="contained"
                  sx={{ py: 1.35 }}
                >
                  {registration.isPending ? 'Criando conta...' : 'Criar minha conta'}
                </Button>
              </Stack>

              <Typography color="text.secondary" fontSize={13} textAlign="center">
                Já tem uma conta?{' '}
                <Link component={RouterLink} to="/" underline="hover">
                  Voltar para o início
                </Link>
              </Typography>
            </Stack>
          </Paper>
        </Stack>
      </Container>
    </Box>
  )
}
