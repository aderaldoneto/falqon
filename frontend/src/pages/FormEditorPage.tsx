import { type FormEvent, useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Container,
  FormControlLabel,
  MenuItem,
  Paper,
  Stack,
  Switch,
  TextField,
  Typography,
  alpha,
} from '@mui/material'
import { Link as RouterLink, Navigate, useNavigate, useParams } from 'react-router-dom'
import {
  createForm,
  getForm,
  getAuthSession,
  updateForm,
  type CreateFormField,
  type FormFieldType,
} from '../api/generated'

type EditorField = CreateFormField & { key: number }

const fieldTypes: Array<{ type: FormFieldType; label: string }> = [
  { type: 'SHORT_TEXT', label: 'Texto curto' },
  { type: 'LONG_TEXT', label: 'Texto longo' },
  { type: 'NUMBER', label: 'Número' },
  { type: 'SINGLE_CHOICE', label: 'Escolha única' },
  { type: 'MULTIPLE_CHOICE', label: 'Múltipla escolha' },
  { type: 'RATING', label: 'Avaliação' },
]

const defaultConfiguration = (type: FormFieldType): Record<string, unknown> => {
  if (type === 'SHORT_TEXT') return { max_length: 255 }
  if (type === 'LONG_TEXT') return { max_length: 2000 }
  if (type === 'NUMBER') return { step: 1 }
  if (type === 'RATING') return { min: 1, max: 5 }
  if (type === 'SINGLE_CHOICE' || type === 'MULTIPLE_CHOICE') {
    return {
      choices: [
        { value: 'option-1', label: 'Opção 1' },
        { value: 'option-2', label: 'Opção 2' },
      ],
    }
  }
  return {}
}

const newField = (key: number, type: FormFieldType = 'SHORT_TEXT'): EditorField => ({
  key,
  type,
  label: '',
  description: '',
  required: false,
  configuration: defaultConfiguration(type),
})

const slugify = (value: string) =>
  value
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')

export function FormEditorPage() {
  const navigate = useNavigate()
  const { formId: formIdParam } = useParams()
  const formId = formIdParam ? Number(formIdParam) : undefined
  const isEditing = Number.isInteger(formId)
  const queryClient = useQueryClient()
  const [title, setTitle] = useState('')
  const [slug, setSlug] = useState('')
  const [slugEdited, setSlugEdited] = useState(false)
  const [description, setDescription] = useState('')
  const [fields, setFields] = useState<EditorField[]>([newField(1)])
  const [nextKey, setNextKey] = useState(2)
  const [validationError, setValidationError] = useState<string | null>(null)

  const session = useQuery({
    queryKey: ['auth', 'session'],
    queryFn: async () => (await getAuthSession()).data ?? null,
    retry: false,
  })
  const existingForm = useQuery({
    queryKey: ['form', formId],
    queryFn: async () => {
      const response = await getForm({ path: { formId: formId! } })
      if (response.error) throw new Error('Não foi possível carregar o formulário.')
      return response.data
    },
    enabled: Boolean(session.data) && isEditing,
    retry: false,
  })
  useEffect(() => {
    if (!existingForm.data) return
    setTitle(existingForm.data.title)
    setSlug(existingForm.data.slug)
    setSlugEdited(true)
    setDescription(existingForm.data.description ?? '')
    setFields(existingForm.data.fields.map((field) => ({
      key: field.id,
      type: field.type,
      label: field.label,
      description: field.description ?? '',
      required: field.required,
      configuration: field.configuration,
    })))
    setNextKey(existingForm.data.fields.length + 1)
  }, [existingForm.data])
  const save = useMutation({
    mutationFn: async () => {
      const body = {
        title: title.trim(),
        slug,
        description: description.trim() || undefined,
        fields: fields.map((field) => ({
          type: field.type,
          label: field.label,
          description: field.description,
          required: field.required,
          configuration: field.configuration,
        })),
      }
      const response = isEditing
        ? await updateForm({ path: { formId: formId! }, body })
        : await createForm({ body })
      if (response.error) {
        const error = response.error as { code?: string; message?: string }
        if (error.code === 'slug_already_exists') throw new Error('Este slug já está em uso.')
        if (error.code === 'invalid_form_state') throw new Error('Apenas rascunhos podem ser editados.')
        throw new Error(error.message ?? 'Não foi possível salvar o formulário.')
      }
      return response.data
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['forms'] })
      navigate('/admin/forms')
    },
  })

  const updateField = (key: number, changes: Partial<EditorField>) => {
    setFields((current) => current.map((field) => (field.key === key ? { ...field, ...changes } : field)))
  }

  const changeFieldType = (field: EditorField, type: FormFieldType) => {
    updateField(field.key, { type, configuration: defaultConfiguration(type) })
  }

  const moveField = (index: number, direction: -1 | 1) => {
    const target = index + direction
    if (target < 0 || target >= fields.length) return
    const reordered = [...fields]
    ;[reordered[index], reordered[target]] = [reordered[target], reordered[index]]
    setFields(reordered)
  }

  const addField = (type: FormFieldType) => {
    setFields((current) => [...current, newField(nextKey, type)])
    setNextKey((current) => current + 1)
  }

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setValidationError(null)
    if (title.trim().length < 2 || slug.length < 2) {
      setValidationError('Informe o título e um slug válido.')
      return
    }
    if (fields.some((field) => !field.label.trim())) {
      setValidationError('Todas as perguntas precisam de um título.')
      return
    }
    save.mutate()
  }

  if (session.isLoading) {
    return <Box minHeight="100vh" display="grid" sx={{ placeItems: 'center' }}><CircularProgress /></Box>
  }
  if (!session.data) return <Navigate replace to="/" />
  if (isEditing && existingForm.isLoading) {
    return <Box minHeight="100vh" display="grid" sx={{ placeItems: 'center' }}><CircularProgress aria-label="Carregando formulário" /></Box>
  }
  if (isEditing && (existingForm.isError || !existingForm.data)) {
    return <Alert severity="error">{existingForm.error?.message ?? 'Formulário não encontrado.'}</Alert>
  }

  return (
    <Box minHeight="100vh" bgcolor="background.default">
      <Box component="header" sx={{ borderBottom: '1px solid', borderColor: alpha('#fff', 0.08) }}>
        <Container maxWidth="lg">
          <Stack direction="row" justifyContent="space-between" alignItems="center" py={2}>
            <Button component={RouterLink} color="inherit" to="/admin/forms">← Meus formulários</Button>
            <Typography color="text.secondary" fontSize={13}>{isEditing ? 'Editando rascunho' : 'Rascunho não salvo'}</Typography>
          </Stack>
        </Container>
      </Box>

      <Container component="main" maxWidth="lg" sx={{ py: { xs: 4, md: 6 } }}>
        <Box component="form" onSubmit={submit}>
          <Stack spacing={4}>
            <Box>
              <Typography color="primary.main" fontSize={11} fontWeight={800} letterSpacing="0.18em">
                EDITOR DINÂMICO
              </Typography>
              <Typography component="h1" variant="h3" fontFamily="Georgia, serif" sx={{ mt: 0.75 }}>
                {isEditing ? 'Editar formulário' : 'Novo formulário'}
              </Typography>
            </Box>

            {(validationError || save.error) && (
              <Alert severity="error" variant="outlined">{validationError ?? save.error?.message}</Alert>
            )}

            <Paper variant="outlined" sx={{ p: { xs: 2.5, md: 4 }, borderColor: alpha('#fff', 0.1) }}>
              <Stack spacing={2.5}>
                <Typography variant="h6" fontWeight={750}>Sobre o filme</Typography>
                <TextField
                  label="Título do formulário"
                  onChange={(event) => {
                    setTitle(event.target.value)
                    if (!slugEdited) setSlug(slugify(event.target.value))
                  }}
                  required
                  value={title}
                />
                <TextField
                  helperText={`O formulário será acessado em /forms/${slug || 'slug-do-filme'}`}
                  label="Slug"
                  onChange={(event) => {
                    setSlugEdited(true)
                    setSlug(slugify(event.target.value))
                  }}
                  required
                  value={slug}
                />
                <TextField
                  label="Descrição"
                  multiline
                  onChange={(event) => setDescription(event.target.value)}
                  rows={3}
                  value={description}
                />
              </Stack>
            </Paper>

            <Stack spacing={2}>
              <Typography variant="h5" fontWeight={750}>Perguntas</Typography>
              {fields.map((field, index) => (
                <Paper key={field.key} variant="outlined" sx={{ p: { xs: 2.5, md: 3 }, borderColor: alpha('#fff', 0.1) }}>
                  <Stack spacing={2.25}>
                    <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} alignItems={{ sm: 'center' }}>
                      <Typography color="primary.main" fontFamily="monospace">{String(index + 1).padStart(2, '0')}</Typography>
                      <TextField
                        fullWidth
                        label="Pergunta"
                        onChange={(event) => updateField(field.key, { label: event.target.value })}
                        required
                        size="small"
                        value={field.label}
                      />
                      <TextField
                        label="Tipo"
                        onChange={(event) => changeFieldType(field, event.target.value as FormFieldType)}
                        select
                        size="small"
                        sx={{ minWidth: 190 }}
                        value={field.type}
                      >
                        {fieldTypes.map((option) => <MenuItem key={option.type} value={option.type}>{option.label}</MenuItem>)}
                      </TextField>
                    </Stack>
                    <TextField
                      label="Descrição de apoio (opcional)"
                      onChange={(event) => updateField(field.key, { description: event.target.value })}
                      size="small"
                      value={field.description ?? ''}
                    />
                    <FieldConfiguration field={field} update={(configuration) => updateField(field.key, { configuration })} />
                    <Stack direction="row" justifyContent="space-between" alignItems="center">
                      <FormControlLabel
                        control={<Switch checked={field.required} onChange={(event) => updateField(field.key, { required: event.target.checked })} />}
                        label="Obrigatória"
                      />
                      <Stack direction="row" spacing={0.5}>
                        <Button disabled={index === 0} onClick={() => moveField(index, -1)} size="small">↑</Button>
                        <Button disabled={index === fields.length - 1} onClick={() => moveField(index, 1)} size="small">↓</Button>
                        <Button color="error" disabled={fields.length === 1} onClick={() => setFields(fields.filter((item) => item.key !== field.key))} size="small">Remover</Button>
                      </Stack>
                    </Stack>
                  </Stack>
                </Paper>
              ))}
            </Stack>

            <Paper variant="outlined" sx={{ p: 2.5, borderStyle: 'dashed', borderColor: alpha('#f4b942', 0.35) }}>
              <Typography fontWeight={700} mb={1.5}>Adicionar pergunta</Typography>
              <Stack direction="row" flexWrap="wrap" gap={1}>
                {fieldTypes.map((option) => (
                  <Button key={option.type} onClick={() => addField(option.type)} size="small" variant="outlined">+ {option.label}</Button>
                ))}
              </Stack>
            </Paper>

            <Stack direction="row" justifyContent="flex-end" spacing={1.5}>
              <Button component={RouterLink} color="inherit" to="/admin/forms">Cancelar</Button>
              <Button disabled={save.isPending} size="large" type="submit" variant="contained">
                {save.isPending ? 'Salvando...' : 'Salvar rascunho'}
              </Button>
            </Stack>
          </Stack>
        </Box>
      </Container>
    </Box>
  )
}

function FieldConfiguration({ field, update }: { field: EditorField; update: (value: Record<string, unknown>) => void }) {
  const configuration = field.configuration
  if (field.type === 'SHORT_TEXT' || field.type === 'LONG_TEXT') {
    return <TextField label="Máximo de caracteres" type="number" size="small" value={String(configuration.max_length ?? '')} onChange={(event) => update({ max_length: Number(event.target.value) })} />
  }
  if (field.type === 'NUMBER') {
    return (
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5}>
        {['min', 'max', 'step'].map((name) => <TextField key={name} fullWidth label={{ min: 'Mínimo', max: 'Máximo', step: 'Intervalo' }[name]} type="number" size="small" value={String(configuration[name] ?? '')} onChange={(event) => update({ ...configuration, [name]: event.target.value === '' ? undefined : Number(event.target.value) })} />)}
      </Stack>
    )
  }
  if (field.type === 'RATING') {
    return <TextField label="Nota máxima" type="number" inputProps={{ min: 2, max: 10 }} size="small" value={String(configuration.max ?? 5)} onChange={(event) => update({ min: 1, max: Number(event.target.value) })} />
  }
  const choices = (configuration.choices as Array<{ value: string; label: string }>) ?? []
  return (
    <Stack spacing={1}>
      <Typography color="text.secondary" fontSize={12}>Opções de resposta</Typography>
      {choices.map((choice, index) => (
        <Stack key={choice.value} direction="row" spacing={1}>
          <TextField fullWidth size="small" value={choice.label} onChange={(event) => update({ ...configuration, choices: choices.map((item, itemIndex) => itemIndex === index ? { ...item, label: event.target.value } : item) })} />
          <Button disabled={choices.length <= 2} onClick={() => update({ ...configuration, choices: choices.filter((_, itemIndex) => itemIndex !== index) })}>Remover</Button>
        </Stack>
      ))}
      <Button onClick={() => update({ ...configuration, choices: [...choices, { value: `option-${Date.now()}`, label: `Opção ${choices.length + 1}` }] })} sx={{ alignSelf: 'flex-start' }}>+ Adicionar opção</Button>
    </Stack>
  )
}
