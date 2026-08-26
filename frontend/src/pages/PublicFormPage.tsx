import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Alert,
  Box,
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
import { getPublicForm, type PublicFormField } from '../api/generated'

type Choice = { value: string; label: string }

function DynamicField({ field }: { field: PublicFormField }) {
  const [rating, setRating] = useState<number | null>(null)
  const configuration = field.configuration as Record<string, unknown>
  const choices = (configuration.choices ?? []) as Choice[]
  const label = `${field.label}${field.required ? ' *' : ''}`

  if (field.type === 'LONG_TEXT') {
    return <TextField fullWidth helperText={field.description} label={label} multiline minRows={4} required={field.required} />
  }
  if (field.type === 'NUMBER') {
    return (
      <TextField
        fullWidth
        helperText={field.description}
        inputProps={{ min: configuration.min, max: configuration.max, step: configuration.step }}
        label={label}
        required={field.required}
        type="number"
      />
    )
  }
  if (field.type === 'SINGLE_CHOICE') {
    return (
      <FormControl required={field.required}>
        <FormLabel>{field.label}</FormLabel>
        <RadioGroup>
          {choices.map((choice) => <FormControlLabel key={choice.value} control={<Radio />} label={choice.label} value={choice.value} />)}
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
          {choices.map((choice) => <FormControlLabel key={choice.value} control={<Checkbox />} label={choice.label} />)}
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
          onChange={(_, value) => setRating(value)}
          size="large"
          value={rating}
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
      required={field.required}
    />
  )
}

export function PublicFormPage() {
  const { slug = '' } = useParams()
  const form = useQuery({
    queryKey: ['public-form', slug],
    queryFn: async () => {
      const response = await getPublicForm({ path: { slug } })
      if (response.error) throw new Error('Este formulário não existe ou ainda não foi publicado.')
      return response.data
    },
    retry: false,
  })

  if (form.isLoading) {
    return <Box minHeight="100vh" display="grid" sx={{ placeItems: 'center' }}><CircularProgress /></Box>
  }

  return (
    <Box minHeight="100vh" sx={{ background: 'radial-gradient(circle at 50% 0%, #282113 0%, #090a0d 42%)' }}>
      <Container maxWidth="sm" sx={{ py: { xs: 4, md: 8 } }}>
        <Typography component={RouterLink} to="/" color="primary.main" fontWeight={900} letterSpacing="0.18em" sx={{ textDecoration: 'none' }}>
          FALQON
        </Typography>
        {form.isError || !form.data ? (
          <Alert severity="error" sx={{ mt: 5 }}>{form.error?.message ?? 'Formulário não encontrado.'}</Alert>
        ) : (
          <Paper variant="outlined" sx={{ mt: 4, p: { xs: 3, sm: 5 }, borderColor: alpha('#ffffff', 0.1), bgcolor: alpha('#121419', 0.94) }}>
            <Stack spacing={4}>
              <Box>
                <Typography color="primary.main" fontSize={11} fontWeight={800} letterSpacing="0.16em">REVIEW DE FILME</Typography>
                <Typography component="h1" fontFamily="Georgia, serif" variant="h3" sx={{ mt: 1 }}>{form.data.title}</Typography>
                {form.data.description && <Typography color="text.secondary" lineHeight={1.7} sx={{ mt: 2 }}>{form.data.description}</Typography>}
              </Box>
              {form.data.fields.map((field) => <DynamicField field={field} key={field.id} />)}
            </Stack>
          </Paper>
        )}
      </Container>
    </Box>
  )
}
