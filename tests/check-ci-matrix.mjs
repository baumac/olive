// check-ci-matrix fails when the e2e matrix of the CI workflow and the
// projects of playwright.config.ts differ. A project that is not in the
// matrix does not run in CI. See decision 00010.
import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'

const here = new URL('.', import.meta.url)
const list = execFileSync('npx', ['playwright', 'test', '--list', '--reporter=json'], {
  cwd: here,
  encoding: 'utf8',
  maxBuffer: 64 * 1024 * 1024,
})
const config = new Set(JSON.parse(list).config.projects.map((p) => p.name))

const workflow = readFileSync(new URL('../.github/workflows/ci.yml', import.meta.url), 'utf8')
const matrix = new Set([...workflow.matchAll(/^\s*- \{ project: ([\w-]+),/gm)].map((m) => m[1]))

const missing = [...config].filter((p) => !matrix.has(p))
const extra = [...matrix].filter((p) => !config.has(p))
if (missing.length || extra.length) {
  console.error(`The e2e matrix of .github/workflows/ci.yml does not match tests/playwright.config.ts.`)
  if (missing.length) console.error(`Add these projects to the matrix: ${missing.join(', ')}`)
  if (extra.length) console.error(`Remove these projects from the matrix: ${extra.join(', ')}`)
  process.exit(1)
}
console.log(`The e2e matrix has the ${config.size} Playwright projects.`)
