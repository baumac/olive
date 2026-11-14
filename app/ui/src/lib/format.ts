const locale = 'en'
// en-GB puts the day before the month, for example "4 Nov 2024". The page
// language stays en.
const dateLocale = 'en-GB'
const day = 24 * 60 * 60 * 1000

// parseTimestamp reads the RFC 3339 format of the API, such as
// "2024-11-04T01:26:55.002Z". A session that the browser saved before
// decision 00023 has a space in the place of the T.
export function parseTimestamp(value: string): Date {
  return new Date(value.replace(' ', 'T'))
}

const relative = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })
const short = new Intl.DateTimeFormat(dateLocale, { day: 'numeric', month: 'short', year: 'numeric' })
const fullDay = new Intl.DateTimeFormat(dateLocale, { day: 'numeric', month: 'long', year: 'numeric' })
const fullTime = new Intl.DateTimeFormat(dateLocale, { hour: '2-digit', minute: '2-digit' })

// maxSkew is how far the browser clock can be behind the server. A time
// that is this close in the future shows as "just now".
const maxSkew = 5 * 60 * 1000

export function formatDate(value: string, now: Date = new Date()): string {
  const date = parseTimestamp(value)
  let ms = now.getTime() - date.getTime()
  if (ms < 0 && ms > -maxSkew) {
    ms = 0
  }
  if (ms < 0 || ms >= 7 * day) {
    return short.format(date)
  }
  const minutes = Math.floor(ms / 60000)
  if (minutes < 1) {
    return 'just now'
  }
  if (minutes < 60) {
    return relative.format(-minutes, 'minute')
  }
  const hours = Math.floor(minutes / 60)
  if (hours < 24) {
    return relative.format(-hours, 'hour')
  }
  return relative.format(-Math.floor(hours / 24), 'day')
}

// formatFullDate gives "4 November 2024, 02:26" in the time zone of the
// browser.
export function formatFullDate(value: string): string {
  const date = parseTimestamp(value)
  return `${fullDay.format(date)}, ${fullTime.format(date)}`
}

export function isoDate(value: string): string {
  return parseTimestamp(value).toISOString()
}

const plural = new Intl.PluralRules(locale)
const number = new Intl.NumberFormat(locale)
const score = new Intl.NumberFormat(locale, { maximumFractionDigits: 1 })

// formatScore keeps the one decimal of an average from the API.
export function formatScore(n: number): string {
  return score.format(n)
}

// formatCount gives "1 review" or "12 reviews".
export function formatCount(n: number, one: string, other = `${one}s`): string {
  return `${number.format(n)} ${plural.select(n) === 'one' ? one : other}`
}

export function initials(name: string | null | undefined): string {
  if (!name) {
    return '?'
  }
  return Array.from(name.trim()).slice(0, 2).join('').toUpperCase() || '?'
}
