import { parseDay } from './format'

// The rules match the Public API spec. Checking them in the UI stops most
// errors before a request.

const space = /^[\t\n\v\f\r ]*$/
const tagLetter = /^[\p{Ll}\p{Lm}\p{Lo}\p{Nd}]$/u
const tagMark = /^[\p{Mn}\p{Mc}]$/u
const ignorable = /^\p{Default_Ignorable_Code_Point}$/u
const letter = /^\p{L}$/u
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export const maxTags = 20

function length(v: string): number {
  return Array.from(v).length
}

// hidden has the characters that the API refuses. These are control
// characters, direction marks and overrides, and invisible characters.
const hidden =
  // eslint-disable-next-line no-control-regex -- The API refuses control characters.
  /[\u0000-\u001f\u007f-\u009f\u00ad\u061c\u115f\u1160\u180e\u200b\u200e\u200f\u202a-\u202e\u2060-\u206f\u3164\ufeff\uffa0\ufff9-\ufffb]/
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

// firstReviewDay is the first date that the API allows. See decision 00005.
export const firstReviewDay = '1900-01-01'

// checkReviewDay allows the dates from firstReviewDay to today. A text
// comparison orders dates of the form 2024-11-04.
export function checkReviewDay(v: string, today: string): string {
  if (!parseDay(v)) {
    return 'Enter a date.'
  }
  if (v < firstReviewDay || v > today) {
    return 'Enter a date from 1 Jan 1900 to today.'
  }
  return ''
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

// toTag changes free text to a tag name, such as "wood-fired" for "Wood
// Fired". The API has the same steps. See decision 00011.
export function toTag(text: string): string {
  let out = ''
  let inWord = false
  let gap = false
  for (const char of text.normalize('NFKC')) {
    // The API changes each code point to lowercase. toLowerCase of the whole
    // text changes Σ by its place in the word. It also adds a dot to İ.
    for (const c of char === 'İ' ? 'i' : char.toLowerCase()) {
      if (ignorable.test(c)) {
        continue
      }
      if (tagLetter.test(c)) {
        out += gap && out !== '' ? `-${c}` : c
        inWord = true
        gap = false
      } else if (tagMark.test(c)) {
        out += inWord ? c : ''
      } else {
        inWord = false
        gap = true
      }
    }
  }
  return out.normalize('NFC').split('-').map(finalSigma).join('-')
}

function finalSigma(word: string): string {
  const chars = Array.from(word)
  const n = chars.length
  if (n < 2 || chars[n - 1] !== 'σ' || !letter.test(chars[n - 2]!)) {
    return word
  }
  return chars.slice(0, -1).join('') + 'ς'
}

// isTag is true for a name that the API stores without a change.
export function isTag(v: string): boolean {
  return v !== '' && length(v) <= 63 && toTag(v) === v
}

// checkTag checks a name of toTag before the input adds it.
export function checkTag(tag: string, tags: string[]): string {
  if (tag === '') {
    return 'Use a letter or a digit.'
  }
  if (length(tag) > 63) {
    return 'Use at most 63 characters.'
  }
  if (tags.includes(tag)) {
    return `The tag ${tag} is already added.`
  }
  if (tags.length >= maxTags) {
    return `Use at most ${maxTags} tags.`
  }
  return ''
}

// safeRedirect accepts only a path on this site. Browsers read /\ like //.
export function safeRedirect(value: unknown, fallback = '/'): string {
  return typeof value === 'string' && /^\/(?![/\\])/.test(value) ? value : fallback
}

export const photoTypes = ['image/jpeg', 'image/png', 'image/webp']
export const maxPhotoBytes = 10 * 1024 * 1024
export const maxPhotos = 5

export function checkPhotoFile(file: File): string {
  if (!photoTypes.includes(file.type) || file.size > maxPhotoBytes) {
    return 'Select a JPEG, PNG, or WebP image of at most 10 MB.'
  }
  return ''
}

// checkAlt allows an empty alt text. The API stores it as null.
export function checkAlt(v: string): string {
  return v === '' ? '' : checkName(v, 'Enter a description with at least one character that is not a space.')
}

// eslint-disable-next-line no-control-regex -- A URL has no spaces or control characters.
const websitePattern = /^https?:\/\/[^\u0000- \u007f-\u009f/?#][^\u0000- \u007f-\u009f]*$/
const phonePattern = /^\+?\(?[0-9](?:[ ()./-]*[0-9]){2,19}$/

// checkWebsite checks a website after normalizeWebsite. An empty website is
// allowed.
export function checkWebsite(v: string): string {
  if (v === '') {
    return ''
  }
  return length(v) > 2048 || !websitePattern.test(v) || hidden.test(v)
    ? 'Enter a web address, for example https://cansole.example.'
    : ''
}

export function checkPhone(v: string): string {
  if (v.trim() === '') {
    return ''
  }
  return v.trim().length > 40 || !phonePattern.test(v.trim())
    ? 'Enter a phone number with 3 to 20 digits, for example +34 932 21 50 12.'
    : ''
}

export function checkOpeningHours(v: string): string {
  if (length(v) > 500) {
    return 'Use at most 500 characters.'
  }
  return hidden.test(v.replace(/[\t\n\r]/g, '')) ? hiddenText : ''
}

// The limits of the Ingredients and the Steps schemas of the API.
export const maxIngredients = 100
export const maxSteps = 50

// ingredientLines gives the ingredients of a text with one ingredient on each
// line. Empty lines do not count.
export function ingredientLines(text: string): string[] {
  return text
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
}

export function checkIngredients(lines: string[]): string {
  if (lines.length === 0) {
    return 'Enter at least one ingredient.'
  }
  if (lines.length > maxIngredients) {
    return `Use at most ${maxIngredients} ingredients.`
  }
  if (lines.some((line) => length(line) > 200)) {
    return 'Use at most 200 characters for each ingredient.'
  }
  return lines.some((line) => hidden.test(line)) ? hiddenText : ''
}

export function checkStep(v: string): string {
  if (space.test(v)) {
    return 'Enter the step, or remove it.'
  }
  if (length(v) > 2000) {
    return 'Use at most 2,000 characters for a step.'
  }
  return hidden.test(v.replace(/[\t\n\r]/g, '')) ? hiddenText : ''
}

// checkWholeNumber allows an empty value. The field is then not set.
export function checkWholeNumber(v: string, min: number, max: number): string {
  if (v.trim() === '') {
    return ''
  }
  const n = Number(v)
  return Number.isInteger(n) && n >= min && n <= max ? '' : `Enter a whole number from ${min} to ${max}.`
}
