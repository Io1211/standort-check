// Country codes offered in the dropdown. Germany first (default), then the
// neighbouring countries most likely to appear. Text labels instead of flag
// emojis because Windows does not render flag emojis.
export const COUNTRY_CODES = [
  { code: '+49', label: 'DE +49' },
  { code: '+43', label: 'AT +43' },
  { code: '+41', label: 'CH +41' },
  { code: '+31', label: 'NL +31' },
  { code: '+32', label: 'BE +32' },
  { code: '+352', label: 'LU +352' },
  { code: '+33', label: 'FR +33' },
  { code: '+45', label: 'DK +45' },
  { code: '+48', label: 'PL +48' },
  { code: '+420', label: 'CZ +420' },
  { code: '+39', label: 'IT +39' },
] as const

export const DEFAULT_COUNTRY_CODE = '+49'
export const MIN_PHONE_DIGITS = 7
const MAX_PHONE_DIGITS = 15

// Turns whatever the user typed or pasted into { countryCode, digits }.
// A full international number ("+43 699 …" or "0043 699 …") switches the
// dropdown to the matching country instead of being glued behind "+49".
export function parsePhoneInput(input: string, currentCode: string): { countryCode: string; digits: string } {
  const trimmed = input.trim()
  const international = trimmed.startsWith('+') ? trimmed.slice(1) : trimmed.startsWith('00') ? trimmed.slice(2) : null

  if (international !== null) {
    const digits = international.replace(/\D/g, '')
    // Longest code first, so "+352" wins over a hypothetical "+35".
    const match = [...COUNTRY_CODES]
      .sort((a, b) => b.code.length - a.code.length)
      .find((c) => digits.startsWith(c.code.slice(1)))
    if (match) return { countryCode: match.code, digits: digits.slice(match.code.length - 1) }
  }
  return { countryCode: currentCode, digits: trimmed.replace(/\D/g, '') }
}

export function phoneError(countryCode: string, digits: string): string | undefined {
  if (!digits) return 'Bitte Telefonnummer eingeben'
  const nsn = digits.replace(/^0+/, '')
  if (nsn.length < MIN_PHONE_DIGITS || countryCode.length - 1 + nsn.length > MAX_PHONE_DIGITS) {
    return `Bitte gültige Telefonnummer eingeben (mind. ${MIN_PHONE_DIGITS} Ziffern)`
  }
  return undefined
}
