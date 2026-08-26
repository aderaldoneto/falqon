export const slugify = (value: string) =>
  value
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')

export const displayValue = (value: unknown) => {
  if (Array.isArray(value)) return value.join(', ')
  if (value === null || value === undefined || value === '') return 'Sem resposta'
  return String(value)
}
