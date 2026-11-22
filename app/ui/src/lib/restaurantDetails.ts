// priceLevels has the names of the price levels of the API. The names have
// no currency. See decision 00004.
export const priceLevels = [
  { value: 1, label: 'Inexpensive' },
  { value: 2, label: 'Moderate' },
  { value: 3, label: 'Expensive' },
  { value: 4, label: 'Very expensive' },
] as const

export function priceLevelLabel(level: number | undefined | null): string {
  return priceLevels.find((l) => l.value === level)?.label ?? ''
}

// normalizeWebsite adds https:// to a value without a scheme, such as
// cansole.example. A colon before a digit starts a port, not a scheme.
// checkWebsite refuses other schemes.
export function normalizeWebsite(v: string): string {
  const text = v.trim()
  if (text === '') {
    return ''
  }
  const scheme = /^([a-z][a-z0-9+.-]*):(?!\d)/i.exec(text)
  if (!scheme) {
    return `https://${text}`
  }
  return text.slice(0, scheme[1]!.length).toLowerCase() + text.slice(scheme[1]!.length)
}

// websiteHost is the short text of a website link.
export function websiteHost(website: string): string {
  try {
    return new URL(website).hostname.replace(/^www\./, '')
  } catch {
    return website
  }
}

export function telLink(phone: string): string {
  return `tel:${phone.replace(/[^+0-9]/g, '')}`
}
