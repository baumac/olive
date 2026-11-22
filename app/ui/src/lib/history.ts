import type { Creator, RecipeVersion, RestaurantVersion } from '@reviews/sdk'
import { formatMinutes } from './format'
import { priceLevels } from './restaurantDetails'
import { type ReviewKind, kinds } from './reviewKinds'
import { tagList } from './reviews'

// The history of restaurants and recipes. See decision 00004.
export type AnyVersion = RestaurantVersion | RecipeVersion

// The properties have the camel case names of the SDK.
const labels: Record<ReviewKind, Record<string, string>> = {
  restaurant: {
    name: 'Name',
    description: 'Description',
    tags: 'Tags',
    location: 'Location',
    website: 'Website',
    phone: 'Phone',
    openingHours: 'Opening hours',
    priceLevel: 'Price level',
  },
  recipe: {
    name: 'Name',
    description: 'Description',
    tags: 'Tags',
    ingredients: 'Ingredients',
    steps: 'Steps',
    source: 'Source',
    totalTime: 'Total time',
    servings: 'Servings',
  },
}

// required has the properties that a patch cannot clear.
const required: Record<ReviewKind, string[]> = {
  restaurant: ['name', 'location'],
  recipe: ['name', 'ingredients', 'steps'],
}

export function propertyLabel(kind: ReviewKind, property: string): string {
  return labels[kind][property] ?? property
}

function valuesOf(values: object): Record<string, unknown> {
  return values as Record<string, unknown>
}

// changedProperties keeps the order of the form. The SDK sets each property
// of the values object. A property that did not change is undefined.
export function changedProperties(kind: ReviewKind, version: AnyVersion): string[] {
  const after = valuesOf(version.after)
  return Object.keys(labels[kind]).filter((p) => after[p] !== undefined)
}

// valueText is null for a property without a value. A list has one line for
// each item.
export function valueText(property: string, value: unknown): string | string[] | null {
  if (value === null || value === undefined) {
    return null
  }
  switch (property) {
    case 'tags':
      return tagList(value as Set<string>).join(', ') || null
    case 'location':
      return (value as { address: string }).address
    case 'priceLevel':
      return priceLevels.find((l) => l.value === value)?.label ?? String(value)
    case 'totalTime':
      return formatMinutes(value as number)
    case 'ingredients':
    case 'steps':
      return value as string[]
    default:
      return String(value)
  }
}

export interface KeptChange {
  property: string
  number: number
  editor: Creator | null
}

export interface Undo {
  number: number
  // values has the values before the version, in the names of the SDK.
  values: Record<string, unknown>
  // kept has the properties that a newer version changed again.
  kept: KeptChange[]
}

// undoChange gets the values before one version of versions, the newest
// first. A property that a newer version changed keeps its current value. It
// returns null for an unknown version and for a create.
export function undoChange(kind: ReviewKind, versions: AnyVersion[], number: number): Undo | null {
  const i = versions.findIndex((v) => v.number === number)
  const version = versions[i]
  if (!version || version.action !== 'update') {
    return null
  }
  const newer = versions.slice(0, i)
  const before = valuesOf(version.before)
  const undo: Undo = { number, values: {}, kept: [] }
  for (const property of changedProperties(kind, version)) {
    const later = newer.find((v) => changedProperties(kind, v).includes(property))
    if (later) {
      undo.kept.push({ property, number: later.number, editor: later.editor })
    } else if (before[property] !== null || !required[kind].includes(property)) {
      undo.values[property] = before[property]
    }
  }
  return undo
}

// findUndo loads the versions from the newest down to number.
export async function findUndo(kind: ReviewKind, parentId: string, number: number): Promise<Undo | null> {
  const versions: AnyVersion[] = []
  let cursor: string | undefined
  do {
    const page = await kinds[kind].listVersions(parentId, { limit: 100, cursor })
    versions.push(...page.items)
    if (versions.some((v) => v.number <= number)) {
      break
    }
    cursor = page.nextCursor ?? undefined
  } while (cursor)
  return undoChange(kind, versions, number)
}

// undoNumber reads the ?undo query of the edit form.
export function undoNumber(query: unknown): number | null {
  const n = Number(query)
  return Number.isInteger(n) && n > 0 ? n : null
}
