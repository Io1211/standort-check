import { phoneError } from './phone'
import type { FieldErrors, LeadFormValues } from './types'

// Frontend validation is for usability only (instant feedback). The Go backend
// validates again and is the source of truth.

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
// German postal codes: 5 digits, never starting with "00".
const POSTAL_CODE_RE = /^(0[1-9]|[1-9]\d)\d{3}$/

export function validateField(name: keyof LeadFormValues, v: LeadFormValues): string | undefined {
  const value = typeof v[name] === 'string' ? (v[name] as string).trim() : ''
  switch (name) {
    case 'firstName':
      return value ? undefined : 'Bitte Vornamen eingeben'
    case 'lastName':
      return value ? undefined : 'Bitte Nachnamen eingeben'
    case 'email':
      if (!value) return 'Bitte E-Mail-Adresse eingeben'
      return EMAIL_RE.test(value) ? undefined : 'Bitte gültige E-Mail-Adresse eingeben'
    case 'phone':
      return phoneError(v.phoneCountryCode, value)
    case 'street':
      return value ? undefined : 'Bitte Straße eingeben'
    case 'postalCode':
      if (!value) return 'Bitte PLZ eingeben'
      return POSTAL_CODE_RE.test(value) ? undefined : 'Bitte deutsche PLZ eingeben (5 Ziffern)'
    case 'city':
      return value ? undefined : 'Bitte Ort eingeben'
    case 'consent':
      return v.consent ? undefined : 'Bitte stimmen Sie der Datenverarbeitung zu'
    default:
      return undefined
  }
}

export const VALIDATED_FIELDS: (keyof LeadFormValues)[] = [
  'firstName',
  'lastName',
  'email',
  'phone',
  'street',
  'postalCode',
  'city',
  'consent',
]

export function validateAll(v: LeadFormValues): FieldErrors {
  const errors: FieldErrors = {}
  for (const name of VALIDATED_FIELDS) {
    const error = validateField(name, v)
    if (error) errors[name] = error
  }
  return errors
}

// Browser autofill often puts "Hauptstraße 14" into the street field. If the
// house number field is still empty, split it off. The split happens visibly
// in the form, so for rare street names ending in a number ("Straße 101" in
// Berlin) the user can simply move it back.
export function splitHouseNumber(street: string): { street: string; houseNumber: string } | null {
  const match = street.trim().match(/^(.*\D)\s+(\d+\s?[a-zA-Z]?(?:\s?-\s?\d+[a-zA-Z]?)?)$/)
  if (!match) return null
  return { street: match[1].trim(), houseNumber: match[2].trim() }
}
