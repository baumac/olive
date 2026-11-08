// The rules match the Public API spec. Checking them in the UI stops most
// errors before a request.

const space = /^[\t\n\v\f\r ]*$/
const tagPattern = /^[a-z0-9]+(-[a-z0-9]+)*$/
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export const maxTags = 20

function length(v: string): number {
  return Array.from(v).length
}

// hidden matches the characters that the API refuses: control characters,
// direction marks and overrides, and zero-width spaces.
const hidden = /[\u0000-\u001f\u007f-\u009f\u200b\u200e\u200f\u202a-\u202e\u2060\u2066-\u2069\ufeff]/
const hiddenText = 'Remove the hidden characters, such as direction marks or zero-width spaces.'

export function checkName(v: string, empty: string): string {
  if (space.test(v)) {
    return empty
  }
  if (length(v) > 255) {
    return 'Use at most 255 characters.'
  }
  return hidden.test(v) ? hiddenText : ''
}

export function checkLongText(v: string): string {
  if (length(v) > 10000) {
    return 'Use at most 10,000 characters.'
  }
  return hidden.test(v.replace(/[\t\n\r]/g, '')) ? hiddenText : ''
}

export function checkEmail(v: string): string {
  if (v.trim() === '') {
    return 'Enter your email address.'
  }
  if (length(v) > 255 || !emailPattern.test(v)) {
    return 'Enter an email address, for example ana@example.com.'
  }
  return ''
}

export function checkPassword(v: string): string {
  if (v === '') {
    return 'Enter a password.'
  }
  const n = length(v)
  if (n < 8 || n > 71 || new TextEncoder().encode(v).length > 72) {
    return 'Use 8 to 71 characters.'
  }
  return ''
}

// suggestTag gives the valid form of a tag, such as "wood-fired" for
// "Wood Fired".
export function suggestTag(v: string): string {
  return v
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 63)
}

export function isTag(v: string): boolean {
  return v.length <= 63 && tagPattern.test(v)
}

export function checkTag(tag: string, tags: string[]): string {
  if (tags.includes(tag)) {
    return `The tag ${tag} is already added.`
  }
  if (tags.length >= maxTags) {
    return `Use at most ${maxTags} tags.`
  }
  if (tag.length > 63 || !tagPattern.test(tag)) {
    const suggestion = suggestTag(tag)
    return suggestion
      ? `Use lowercase letters, digits and single hyphens, for example ${suggestion}.`
      : 'Use lowercase letters, digits and single hyphens.'
  }
  return ''
}

// safeRedirect accepts only a path on this site. Browsers read /\ like //.
export function safeRedirect(value: unknown, fallback = '/restaurants'): string {
  return typeof value === 'string' && /^\/(?![/\\])/.test(value) ? value : fallback
}
