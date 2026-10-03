export interface Campaign {
  id: string
  name: string
  utmSource: string
  utmMedium: string
  utmCampaign: string
  utmContent: string
  notes: string
  archived: boolean
  createdAt: string
  leads: number
  unique: number
  qualified: number
}

export type CampaignInput = Pick<Campaign, 'name' | 'utmSource' | 'utmMedium' | 'utmCampaign' | 'utmContent' | 'notes' | 'archived'>

export const PRESETS = [
  { label: 'Google Ads', utmSource: 'google', utmMedium: 'cpc' },
  { label: 'Meta Ads', utmSource: 'meta', utmMedium: 'paid_social' },
] as const

// "Dresden Herbst-Aktion 2026" → "dresden-herbst-aktion-2026"
// While typing (keepTrailing), a trailing "-" or "_" must survive, otherwise
// "paid_social" could never be typed.
export function slugify(s: string, { keepTrailing = false } = {}): string {
  const slug = s
    .toLowerCase()
    .replace(/ä/g, 'ae')
    .replace(/ö/g, 'oe')
    .replace(/ü/g, 'ue')
    .replace(/ß/g, 'ss')
    .normalize('NFKD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/[^a-z0-9._-]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^[-._]+/, '')
    .slice(0, 100)
  return keepTrailing ? slug : slug.replace(/[-._]+$/, '')
}

// The tracking link points at this deployment's form, so it is always
// correct for the environment the admin is using (local, preview, production).
export function trackingLink(c: Pick<Campaign, 'utmSource' | 'utmMedium' | 'utmCampaign' | 'utmContent'>): string {
  const params = new URLSearchParams({
    utm_source: c.utmSource,
    utm_medium: c.utmMedium,
    utm_campaign: c.utmCampaign,
  })
  if (c.utmContent) params.set('utm_content', c.utmContent)
  // Keep platform placeholders like {{ad.name}} readable: the ad platforms
  // replace them before the click and expect them unencoded.
  const query = params.toString().replace(/%7B/gi, '{').replace(/%7D/gi, '}')
  return `${window.location.origin}/standort-check?${query}`
}
