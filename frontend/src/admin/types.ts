export type LeadStatus = 'new' | 'contacted' | 'qualified' | 'not_qualified'

export interface AdminLead {
  id: string
  firstName: string
  lastName: string
  email: string
  phone: string
  street: string
  houseNumber: string
  postalCode: string
  city: string
  parcelNote: string
  utmSource: string
  utmMedium: string
  utmCampaign: string
  utmContent: string
  utmTerm: string
  gclid: string
  fbclid: string
  referrer: string
  status: LeadStatus
  duplicateOf: string | null
  duplicateCount: number
  geo: GeoInfo
  createdAt: string
}

export interface LeadSummary {
  id: string
  firstName: string
  lastName: string
  email: string
  status: LeadStatus
  createdAt: string
}

export interface LeadDetail {
  lead: AdminLead
  original: LeadSummary | null
  duplicates: LeadSummary[]
}

export interface LeadListResponse {
  leads: AdminLead[]
  total: number
  page: number
  pageSize: number
}

export interface CampaignStats {
  source: string
  campaign: string
  total: number
  unique: number
  new: number
  contacted: number
  qualified: number
  notQualified: number
}

export type GeoStatus = '' | 'ok' | 'not_found' | 'error'

export interface GeoInfo {
  status: GeoStatus
  lat: number | null
  lon: number | null
  municipality: string
  county: string
  state: string
  postcode: string
  formatted: string
  resultType: string
  confidence: number | null
  checkedAt: string | null
  inServiceArea: boolean | null
  postcodeMismatch: boolean
  imprecise: boolean
}

export interface GeoStats {
  inArea: number
  outOfArea: number
  noGeo: number
  notFound: number
  states: string[]
  configured: boolean
}
