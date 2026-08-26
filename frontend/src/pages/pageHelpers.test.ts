import { describe, expect, it } from 'vitest'
import { displayValue, slugify } from './pageHelpers'

describe('slugify', () => {
  it('normaliza acentos, espaços e pontuação', () => {
    expect(slugify('  O Poderoso Chefão!  ')).toBe('o-poderoso-chefao')
  })

  it('remove separadores das extremidades', () => {
    expect(slugify('---Movie---')).toBe('movie')
  })
})

describe('displayValue', () => {
  it('formata respostas de múltipla escolha', () => {
    expect(displayValue(['Drama', 'Crime'])).toBe('Drama, Crime')
  })

  it('identifica respostas vazias', () => {
    expect(displayValue(null)).toBe('Sem resposta')
    expect(displayValue('')).toBe('Sem resposta')
  })

  it('converte valores escalares', () => {
    expect(displayValue(5)).toBe('5')
  })
})
