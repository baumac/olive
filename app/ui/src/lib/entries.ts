import { parseDay, parseTimestamp } from './format'
import type { AnyReview } from './reviewKinds'

// An entry is a review of any kind. The table and the journal merge the
// restaurant reviews and the recipe reviews. See decision 00009.

// created sorts by the time a review was written. date sorts by the day of
// the visit or the cook, then by created.
export type EntryOrder = 'created' | 'date'

export function compareEntries(order: EntryOrder) {
  return (a: AnyReview, b: AnyReview): number => {
    if (order === 'date' && a.date !== b.date) {
      return a.date < b.date ? 1 : -1
    }
    const ta = parseTimestamp(a.created).getTime()
    const tb = parseTimestamp(b.created).getTime()
    if (ta !== tb) {
      return tb - ta
    }
    return a.id < b.id ? 1 : a.id > b.id ? -1 : 0
  }
}

export interface EntrySource {
  // buffer has the loaded entries that the list does not show yet.
  buffer: AnyReview[]
  cursor: string | null
}

// takeMerged moves the entries of the buffers into one ordered list. It
// stops when a buffer is empty and its source has more pages. The next
// page can have a newer entry.
export function takeMerged(sources: EntrySource[], order: EntryOrder): AnyReview[] {
  const compare = compareEntries(order)
  const out: AnyReview[] = []
  for (;;) {
    if (sources.some((s) => s.buffer.length === 0 && s.cursor !== null)) {
      return out
    }
    const live = sources.filter((s) => s.buffer.length > 0)
    if (live.length === 0) {
      return out
    }
    let best = live[0]!
    for (const s of live) {
      if (compare(s.buffer[0]!, best.buffer[0]!) < 0) {
        best = s
      }
    }
    out.push(best.buffer.shift()!)
  }
}

export interface EntryGroup {
  key: string
  label: string
  entries: AnyReview[]
}

const weekMs = 7 * 24 * 60 * 60 * 1000
const monthYear = new Intl.DateTimeFormat('en-GB', { month: 'long', year: 'numeric' })
const monthName = new Intl.DateTimeFormat('en-GB', { month: 'long' })

function addTo(groups: EntryGroup[], key: string, label: string, entry: AnyReview) {
  const last = groups[groups.length - 1]
  if (last?.key === key) {
    last.entries.push(entry)
  } else {
    groups.push({ key, label, entries: [entry] })
  }
}

// groupByWritten groups the entries of the table: "Past week", then
// "Earlier in November" for the current month, then "October 2026".
export function groupByWritten(entries: AnyReview[], now: Date = new Date()): EntryGroup[] {
  const groups: EntryGroup[] = []
  for (const e of entries) {
    const written = parseTimestamp(e.created)
    if (now.getTime() - written.getTime() < weekMs) {
      addTo(groups, 'week', 'Past week', e)
      continue
    }
    const key = `${written.getFullYear()}-${written.getMonth()}`
    const thisMonth = written.getFullYear() === now.getFullYear() && written.getMonth() === now.getMonth()
    addTo(groups, key, thisMonth ? `Earlier in ${monthName.format(written)}` : monthYear.format(written), e)
  }
  return groups
}

// groupByMonth groups the entries of a journal by the month of their date.
export function groupByMonth(entries: AnyReview[]): EntryGroup[] {
  const groups: EntryGroup[] = []
  for (const e of entries) {
    const day = parseDay(e.date)
    const key = e.date.slice(0, 7)
    addTo(groups, key, day ? monthYear.format(day) : key, e)
  }
  return groups
}
