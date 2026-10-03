import { useState, type ChangeEvent, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router'
import { ApiError, apiFetch } from '../api'
import { getAttribution } from '../attribution'
import type { FieldErrors, LeadFormValues } from '../types'
import { COUNTRY_CODES, DEFAULT_COUNTRY_CODE, parsePhoneInput } from '../phone'
import { splitHouseNumber, validateAll, validateField } from '../validation'
import { Field, FieldError } from './Field'

const initialValues: LeadFormValues = {
  firstName: '',
  lastName: '',
  email: '',
  phoneCountryCode: DEFAULT_COUNTRY_CODE,
  phone: '',
  street: '',
  houseNumber: '',
  postalCode: '',
  city: '',
  parcelNote: '',
  consent: false,
  website: '',
}

export function LeadForm() {
  const navigate = useNavigate()
  const [values, setValues] = useState(initialValues)
  const [errors, setErrors] = useState<FieldErrors>({})
  const [formError, setFormError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  function update(name: keyof LeadFormValues, value: string | boolean) {
    setAndRevalidate({ ...values, [name]: value }, name)
  }

  // Once a field shows an error, re-check it while the user types.
  function setAndRevalidate(next: LeadFormValues, name: keyof LeadFormValues) {
    setValues(next)
    if (errors[name]) setErrors({ ...errors, [name]: validateField(name, next) })
  }

  // Digits only; a pasted "+43 699 …" switches the country code dropdown.
  function onPhoneChange(e: ChangeEvent<HTMLInputElement>) {
    const { countryCode, digits } = parsePhoneInput(e.target.value, values.phoneCountryCode)
    setAndRevalidate({ ...values, phoneCountryCode: countryCode, phone: digits }, 'phone')
  }

  function onCountryCodeChange(e: ChangeEvent<HTMLSelectElement>) {
    setAndRevalidate({ ...values, phoneCountryCode: e.target.value }, 'phone')
  }

  const onChange = (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    update(e.target.name as keyof LeadFormValues, e.target.value)

  const onBlur = (e: ChangeEvent<HTMLInputElement>) => {
    const name = e.target.name as keyof LeadFormValues
    let next = values

    if (name === 'street' && !values.houseNumber) {
      const split = splitHouseNumber(values.street)
      if (split) {
        next = { ...values, ...split }
        setValues(next)
      }
    }
    // Only show "required" errors after the user actually typed something,
    // not when tabbing through empty fields.
    if (String(next[name]).trim() !== '') {
      setErrors({ ...errors, [name]: validateField(name, next) })
    }
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (submitting) return

    const found = validateAll(values)
    setErrors(found)
    setFormError('')
    if (Object.keys(found).length > 0) {
      focusFirstError(found)
      return
    }

    setSubmitting(true)
    try {
      await apiFetch('/leads', {
        method: 'POST',
        body: JSON.stringify({ ...values, ...getAttribution() }),
      })
      navigate('/standort-check/danke', { replace: true })
    } catch (err) {
      if (err instanceof ApiError && err.fields) {
        setErrors(err.fields as FieldErrors)
        focusFirstError(err.fields as FieldErrors)
      } else if (err instanceof ApiError) {
        setFormError(err.message)
      } else {
        setFormError('Keine Verbindung. Bitte prüfen Sie Ihre Internetverbindung und versuchen Sie es erneut.')
      }
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={onSubmit} noValidate className="space-y-4">
      <div className="grid grid-cols-1 items-start gap-4 sm:grid-cols-2">
        <Field id="firstName" label="Vorname" placeholder="Vorname" autoComplete="given-name"
          value={values.firstName} onChange={onChange} onBlur={onBlur} error={errors.firstName} />
        <Field id="lastName" label="Nachname" placeholder="Nachname" autoComplete="family-name"
          value={values.lastName} onChange={onChange} onBlur={onBlur} error={errors.lastName} />
      </div>

      <Field id="email" label="E-Mail-Adresse" placeholder="E-Mail-Adresse" type="email" autoComplete="email"
        inputMode="email" value={values.email} onChange={onChange} onBlur={onBlur} error={errors.email} />

      <div className="overflow-hidden rounded-md bg-white focus-within:ring-2 focus-within:ring-accent/50">
        <div className="flex items-stretch">
          <label htmlFor="phoneCountryCode" className="sr-only">Ländervorwahl</label>
          <select id="phoneCountryCode" name="phoneCountryCode" value={values.phoneCountryCode}
            onChange={onCountryCodeChange} autoComplete="tel-country-code"
            className="shrink-0 cursor-pointer border-r border-neutral-200 bg-transparent pr-2 pl-4 text-base text-ink outline-none">
            {COUNTRY_CODES.map((c) => (
              <option key={c.code} value={c.code}>{c.label}</option>
            ))}
          </select>
          <label htmlFor="phone" className="sr-only">Telefonnummer</label>
          <input id="phone" name="phone" type="tel" inputMode="numeric" autoComplete="tel-national"
            placeholder="Telefonnummer, z. B. 170 1234567" value={values.phone}
            onChange={onPhoneChange} onBlur={onBlur}
            aria-invalid={errors.phone ? true : undefined}
            aria-describedby={errors.phone ? 'phone-error' : undefined}
            className="w-full min-w-0 bg-transparent px-4 py-4 text-base text-ink outline-none placeholder:text-neutral-400" />
        </div>
        <FieldError id="phone-error" message={errors.phone} />
      </div>

      <fieldset className="space-y-4 pt-4">
        <legend className="text-lg font-semibold">Adresse des Grundstücks</legend>
        <p className="-mt-2 text-sm text-ink/70">Der Standort-Check ist derzeit nur für Grundstücke in Deutschland möglich.</p>

        <div className="grid grid-cols-[1fr_7.5rem] items-start gap-4">
          <Field id="street" label="Straße" placeholder="Straße" autoComplete="address-line1"
            value={values.street} onChange={onChange} onBlur={onBlur} error={errors.street} />
          <Field id="houseNumber" label="Hausnummer (falls vorhanden)" placeholder="Nr." autoComplete="off"
            value={values.houseNumber} onChange={onChange} onBlur={onBlur} error={errors.houseNumber} />
        </div>

        <div className="grid grid-cols-[7.5rem_1fr] items-start gap-4">
          <Field id="postalCode" label="Postleitzahl" placeholder="PLZ" autoComplete="postal-code"
            inputMode="numeric" maxLength={5} value={values.postalCode} onChange={onChange} onBlur={onBlur}
            error={errors.postalCode} />
          <Field id="city" label="Ort" placeholder="Ort" autoComplete="address-level2"
            value={values.city} onChange={onChange} onBlur={onBlur} error={errors.city} />
        </div>

        <div className="overflow-hidden rounded-md bg-white focus-within:ring-2 focus-within:ring-accent/50">
          <label htmlFor="parcelNote" className="sr-only">Flurstück oder Hinweise (optional)</label>
          <textarea id="parcelNote" name="parcelNote" rows={2} value={values.parcelNote} onChange={onChange}
            placeholder="Noch keine Hausnummer? Flurstück oder Hinweise (optional)"
            className="block w-full resize-y bg-transparent px-5 py-4 text-base text-ink outline-none placeholder:text-neutral-400" />
          <FieldError id="parcelNote-error" message={errors.parcelNote} />
        </div>
      </fieldset>

      {/* Honeypot: hidden from humans and screen readers, bots fill it in. */}
      <div aria-hidden="true" className="absolute -left-[9999px] h-0 w-0 overflow-hidden">
        <label htmlFor="website">Website</label>
        <input id="website" name="website" tabIndex={-1} autoComplete="off"
          value={values.website} onChange={onChange} />
      </div>

      <div className="overflow-hidden rounded-md">
        <label className="flex cursor-pointer items-start gap-3 py-2 text-sm leading-relaxed">
          <input type="checkbox" name="consent" checked={values.consent}
            onChange={(e) => update('consent', e.target.checked)}
            aria-invalid={errors.consent ? true : undefined}
            aria-describedby={errors.consent ? 'consent-error' : undefined}
            className="mt-1 size-5 shrink-0 accent-ink" />
          <span>
            Ich bin einverstanden, dass meine Angaben zur Bearbeitung meiner Anfrage gespeichert werden und mich
            das Team per Telefon oder E-Mail kontaktiert. Details in der{' '}
            <Link to="/datenschutz" target="_blank" className="underline underline-offset-2">Datenschutzerklärung</Link>.
          </span>
        </label>
        <FieldError id="consent-error" message={errors.consent} />
      </div>

      {formError && (
        <p role="alert" className="rounded-md bg-error-bg px-5 py-4 text-sm text-error">{formError}</p>
      )}

      <button type="submit" disabled={submitting}
        className="mt-2 flex w-full flex-col items-center rounded-full bg-accent px-6 py-4 text-white transition hover:bg-accent-dark disabled:cursor-wait disabled:opacity-70">
        <span className="text-xl font-bold">{submitting ? 'Wird gesendet …' : 'Standort-Check anfragen!'}</span>
        <span className="text-sm opacity-90">Kostenlos und unverbindlich</span>
      </button>
    </form>
  )
}

function focusFirstError(errors: FieldErrors) {
  const first = Object.keys(errors)[0]
  if (first) document.getElementsByName(first)[0]?.focus()
}
