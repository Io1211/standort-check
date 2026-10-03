import type { Attribution } from './types'

// Campaign attribution is read from the URL on the first page view and kept in
// sessionStorage, so it survives a reload or a detour to the privacy page.
// A new visit with campaign parameters overwrites it (last click wins).

const STORAGE_KEY = 'standort-check:attribution'

const PARAMS: Record<string, keyof Attribution> = {
  utm_source: 'utmSource',
  utm_medium: 'utmMedium',
  utm_campaign: 'utmCampaign',
  utm_content: 'utmContent',
  utm_term: 'utmTerm',
  gclid: 'gclid',
  fbclid: 'fbclid',
}

const empty: Attribution = {
  utmSource: '',
  utmMedium: '',
  utmCampaign: '',
  utmContent: '',
  utmTerm: '',
  gclid: '',
  fbclid: '',
  referrer: '',
}

export function captureAttribution(): void {
  const params = new URLSearchParams(window.location.search)
  const fromUrl: Partial<Attribution> = {}
  for (const [param, key] of Object.entries(PARAMS)) {
    const value = params.get(param)
    if (value) fromUrl[key] = value
  }

  const stored = load()
  const hasCampaign = Object.keys(fromUrl).length > 0

  // Only an external referrer is interesting (not our own pages).
  const referrer = isExternal(document.referrer) ? document.referrer : ''

  if (hasCampaign) {
    save({ ...empty, ...fromUrl, referrer })
  } else if (!stored) {
    save({ ...empty, referrer })
  }
}

export function getAttribution(): Attribution {
  return load() ?? empty
}

function isExternal(url: string): boolean {
  if (!url) return false
  try {
    return new URL(url).origin !== window.location.origin
  } catch {
    return false
  }
}

// Storage may be unavailable (private mode, blocked cookies). Attribution is
// nice to have and must never break the form.
function load(): Attribution | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    return raw ? { ...empty, ...JSON.parse(raw) } : null
  } catch {
    return null
  }
}

function save(a: Attribution): void {
  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(a))
  } catch {
    // ignore
  }
}
