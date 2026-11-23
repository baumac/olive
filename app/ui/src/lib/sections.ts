// The sections of the navigation. See decision 00009.
export const sections = [
  { path: '/', label: 'Table' },
  { path: '/restaurants', label: 'Restaurants' },
  { path: '/recipes', label: 'Recipes' },
] as const

// A section is current on each page of its section, such as
// /recipes/{id}/reviews/new. The table is current only on its own page.
export function isCurrentSection(sectionPath: string, routePath: string): boolean {
  if (sectionPath === '/') {
    return routePath === '/'
  }
  return routePath === sectionPath || routePath.startsWith(`${sectionPath}/`)
}
