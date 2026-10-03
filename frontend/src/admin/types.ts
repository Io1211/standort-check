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
