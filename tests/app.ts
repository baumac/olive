import type { ChildProcess } from 'node:child_process'
import { spawn, spawnSync } from 'node:child_process'
import { randomUUID } from 'node:crypto'
import { existsSync, mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'

// The path does not depend on the folder that runs Playwright.
const binary = fileURLToPath(new URL('../bin/reviews-app', import.meta.url))

export interface RunningApp {
  baseURL: string
  // superuserToken approves the users of the tests. See decision 00003.
  superuserToken: string
  stop: () => Promise<void>
}

// startApp starts the built app with a new data folder. stop deletes only
// this folder. appURL sets the Application URL of the PocketBase settings.
export async function startApp(port: number, env: Record<string, string> = {}, appURL?: string): Promise<RunningApp> {
  if (!existsSync(binary)) {
    throw new Error(`${binary} does not exist. Run make build-app.`)
  }
  const dir = mkdtempSync(join(tmpdir(), 'reviews-e2e-'))
  let superuser: Superuser
  try {
    superuser = makeSuperuser(dir)
  } catch (err) {
    rmSync(dir, { recursive: true, force: true })
    throw err
  }
  const baseURL = `http://127.0.0.1:${port}`
  const child: ChildProcess = spawn(binary, ['serve', '--dir', dir, '--http', `127.0.0.1:${port}`], {
    // Only an app with the fake server of photon.ts has the place search and
    // the maps.
    env: { ...process.env, REVIEWS_PLACES_URL: 'off', REVIEWS_MAP_TILE_URL: 'off', ...env },
    stdio: ['ignore', 'ignore', 'pipe'],
  })
  let stderr = ''
  child.stderr?.on('data', (chunk) => (stderr += chunk))

  async function stop() {
    if (child.exitCode === null) {
      const exited = new Promise((resolve) => child.once('exit', resolve))
      child.kill()
      await exited
    }
    rmSync(dir, { recursive: true, force: true })
  }

  for (let i = 0; i < 100; i++) {
    if (child.exitCode !== null) {
      await stop()
      throw new Error(`The app stopped with code ${child.exitCode}: ${stderr}`)
    }
    if (await answers(baseURL)) {
      try {
        const superuserToken = await superuserSignIn(baseURL, superuser)
        if (appURL) {
          await setAppURL(baseURL, superuserToken, appURL)
        }
        return { baseURL, superuserToken, stop }
      } catch (err) {
        await stop()
        throw err
      }
    }
    await new Promise((resolve) => setTimeout(resolve, 100))
  }
  await stop()
  throw new Error(`The app did not answer on ${baseURL}: ${stderr}`)
}

async function answers(baseURL: string): Promise<boolean> {
  try {
    return (await fetch(`${baseURL}/api/health`)).ok
  } catch {
    return false
  }
}

interface Superuser {
  identity: string
  password: string
}

// makeSuperuser runs before serve. The CLI and serve then do not write the
// database at the same time.
function makeSuperuser(dir: string): Superuser {
  const superuser = { identity: 'e2e-superuser@example.com', password: randomUUID() }
  const made = spawnSync(binary, ['superuser', 'upsert', superuser.identity, superuser.password, '--dir', dir], { encoding: 'utf8' })
  if (made.status !== 0) {
    throw new Error(`The superuser was not made: ${made.stderr}`)
  }
  return superuser
}

async function superuserSignIn(baseURL: string, superuser: Superuser): Promise<string> {
  const auth = await fetch(`${baseURL}/api/collections/_superusers/auth-with-password`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(superuser),
  })
  if (!auth.ok) {
    throw new Error(`The superuser sign in failed with ${auth.status}`)
  }
  return ((await auth.json()) as { token: string }).token
}

async function setAppURL(baseURL: string, token: string, appURL: string) {
  const settings = await fetch(`${baseURL}/api/settings`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json', Authorization: token },
    body: JSON.stringify({ meta: { appURL } }),
  })
  if (!settings.ok) {
    throw new Error(`The Application URL was not set: ${settings.status} ${await settings.text()}`)
  }
}
