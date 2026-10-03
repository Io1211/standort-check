export interface LeadFormValues {
  firstName: string
  lastName: string
  email: string
  phoneCountryCode: string
  phone: string // national number, digits only
  street: string
  houseNumber: string
  postalCode: string
  city: string
  parcelNote: string
  consent: boolean
  website: string // honeypot, stays empty for humans
}

export interface Attribution {
  utmSource: string
  utmMedium: string
  utmCampaign: string
  utmContent: string
  utmTerm: string
  gclid: string
  fbclid: string
  referrer: string
}

export type FieldErrors = Partial<Record<keyof LeadFormValues, string>>
