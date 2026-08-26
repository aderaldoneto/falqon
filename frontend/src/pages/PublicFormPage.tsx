import { type FormEvent, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import {
  Alert,
  Box,
  Button,
  Checkbox,
  CircularProgress,
  Container,
  FormControl,
  FormControlLabel,
  FormGroup,
  FormHelperText,
  FormLabel,
  Paper,
  Radio,
  RadioGroup,
  Rating,
  Stack,
  TextField,
  Typography,
  alpha,
} from '@mui/material'
import { Link as RouterLink, useParams } from 'react-router-dom'
import { createSubmission, getPublicForm, type PublicFormField } from '../api/generated'

type Choice = { value: string; label: string }

function DynamicField({
  field,
  value,
  onChange,
}: {
  field: PublicFormField
  value: unknown
  onChange: (value: unknown) => void
}) {
  const configuration = field.configuration as Record<string, unknown>
  const choices = (configuration.choices ?? []) as Choice[]
  const label = `${field.label}${field.required ? ' *' : ''}`

  if (field.type === 'LONG_TEXT') {
    return (
      <TextField
        fullWidth
        helperText={field.description}
        label={label}
        multiline
        minRows={4}
        onChange={(event) => onChange(event.target.value)}
        required={field.required}
        value={value ?? ''}
      />
    )
  }
  if (field.type === 'NUMBER') {
    return (
      <TextField
        fullWidth
        helperText={field.description}
        inputProps={{ min: configuration.min, max: configuration.max, step: configuration.step }}
        label={label}
        onChange={(event) =>
          onChange(event.target.value === '' ? undefined : Number(event.target.value))
        }
        required={field.required}
        type="number"
        value={value ?? ''}
      />
    )
  }
  if (field.type === 'SINGLE_CHOICE') {
    return (
      <FormControl required={field.required}>
        <FormLabel>{field.label}</FormLabel>
        <RadioGroup onChange={(event) => onChange(event.target.value)} value={value ?? ''}>
          {choices.map((choice) => (
            <FormControlLabel
              key={choice.value}
              control={<Radio />}
              label={choice.label}
              value={choice.value}
            />
          ))}
        </RadioGroup>
        {field.description && <FormHelperText>{field.description}</FormHelperText>}
      </FormControl>
    )
  }
  if (field.type === 'MULTIPLE_CHOICE') {
    return (
      <FormControl required={field.required}>
        <FormLabel>{field.label}</FormLabel>
        <FormGroup>
          {choices.map((choice) => {
            const selected = Array.isArray(value) ? (value as string[]) : []
            return (
              <FormControlLabel
                key={choice.value}
                control={
                  <Checkbox
                    checked={selected.includes(choice.value)}
                    onChange={(_, checked) =>
                      onChange(
                        checked
                          ? [...selected, choice.value]
                          : selected.filter((item) => item !== choice.value),
                      )
                    }
                  />
                }
                label={choice.label}
              />
            )
          })}
        </FormGroup>
        {field.description && <FormHelperText>{field.description}</FormHelperText>}
      </FormControl>
    )
  }
  if (field.type === 'RATING') {
    return (
      <FormControl required={field.required}>
        <FormLabel>{field.label}</FormLabel>
        <Rating
          max={Number(configuration.max ?? 5)}
          onChange={(_, rating) => onChange(rating ?? undefined)}
          size="large"
          value={typeof value === 'number' ? value : null}
        />
        {field.description && <FormHelperText>{field.description}</FormHelperText>}
      </FormControl>
    )
  }
  return (
    <TextField
      fullWidth
      helperText={field.description}
      inputProps={{ maxLength: configuration.max_length }}
      label={label}
      onChange={(event) => onChange(event.target.value)}
      required={field.required}
      value={value ?? ''}
    />
  )
}

export function PublicFormPage() {
  const { slug = '' } = useParams()
  const [answers, setAnswers] = useState<Record<number, unknown>>({})
  const [submitted, setSubmitted] = useState(false)
  const form = useQuery({
    queryKey: ['public-form', slug],
    queryFn: async () => {
      const response = await getPublicForm({ path: { slug } })
      if (response.error) throw new Error('Este formulário não existe ou ainda não foi publicado.')
      return response.data
    },
    retry: false,
  })
  const submission = useMutation({
    mutationFn: async () => {
      const response = await createSubmission({
        path: { slug },
        body: {
          answers: Object.entries(answers)
            .filter(
              ([, value]) =>
                value !== undefined && value !== '' && (!Array.isArray(value) || value.length > 0),
            )
            .map(([fieldId, value]) => ({ field_id: Number(fieldId), value })),
        },
      })
      if (response.error) throw new Error('Revise os campos obrigatórios e os valores informados.')
      return response.data
    },
    onSuccess: () => setSubmitted(true),
  })

  const submit = (event: FormEvent) => {
    event.preventDefault()
    submission.mutate()
  }

  if (form.isLoading) {
    return (
      <Box minHeight="100vh" display="grid" sx={{ placeItems: 'center' }}>
        <CircularProgress />
      </Box>
    )
  }

  return (
    <Box
      minHeight="100vh"
      sx={{ background: 'radial-gradient(circle at 50% 0%, #282113 0%, #090a0d 42%)' }}
    >
      <Container maxWidth="sm" sx={{ py: { xs: 4, md: 8 } }}>
        <Typography
          component={RouterLink}
          to="/"
          color="primary.main"
          fontWeight={900}
          letterSpacing="0.18em"
          sx={{ textDecoration: 'none' }}
        >
          FALQON
        </Typography>
        {form.isError || !form.data ? (
          <Alert severity="error" sx={{ mt: 5 }}>
            {form.error?.message ?? 'Formulário não encontrado.'}
          </Alert>
        ) : (
          <Paper
            component="form"
            onSubmit={submit}
            variant="outlined"
            sx={{
              mt: 4,
              p: { xs: 3, sm: 5 },
              borderColor: alpha('#ffffff', 0.1),
              bgcolor: alpha('#121419', 0.94),
            }}
          >
            <Stack spacing={4}>
              <Box>
                <Typography
                  color="primary.main"
                  fontSize={11}
                  fontWeight={800}
                  letterSpacing="0.16em"
                >
                  REVIEW DE FILME
                </Typography>
                <Typography component="h1" fontFamily="Georgia, serif" variant="h3" sx={{ mt: 1 }}>
                  {form.data.title}
                </Typography>
                {form.data.description && (
                  <Typography color="text.secondary" lineHeight={1.7} sx={{ mt: 2 }}>
                    {form.data.description}
                  </Typography>
                )}
              </Box>
              {submitted ? (
                <Alert severity="success">Obrigado! Sua resposta foi registrada.</Alert>
              ) : (
                <>
                  {form.data.fields.map((field) => (
                    <DynamicField
                      field={field}
                      key={field.id}
                      value={answers[field.id]}
                      onChange={(value) =>
                        setAnswers((current) => ({ ...current, [field.id]: value }))
                      }
                    />
                  ))}
                  {submission.isError && <Alert severity="error">{submission.error.message}</Alert>}
                  <Button
                    disabled={submission.isPending}
                    size="large"
                    type="submit"
                    variant="contained"
                  >
                    {submission.isPending ? 'Enviando...' : 'Enviar respostas'}
                  </Button>
                </>
              )}
            </Stack>
          </Paper>
        )}
      </Container>
    </Box>
  )
}
