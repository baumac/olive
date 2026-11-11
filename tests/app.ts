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
  stop: () => Promise<void>
}

// startApp starts the built app with a new data folder. stop deletes only
// this folder. appURL sets the Application URL of the PocketBase settings.
export async function startApp(port: number, env: Record<string, string> = {}, appURL?: string): Promise<RunningApp> {
  if (!existsSync(binary)) {
    throw new Error(`${binary} does not exist. Run make build-app.`)
  }
  const dir = mkdtempSync(join(tmpdir(), 'reviews-e2e-'))
  let superuser: Superuser | undefined
  try {
    superuser = appURL ? makeSuperuser(dir) : undefined
  } catch (err) {
    rmSync(dir, { recursive: true, force: true })
    throw err
  }
  const baseURL = `http://127.0.0.1:${port}`
  const child: ChildProcess = spawn(binary, ['serve', '--dir', dir, '--http', `127.0.0.1:${port}`], {
    env: { ...process.env, ...env },
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
      if (appURL && superuser) {
        try {
          await setAppURL(baseURL, superuser, appURL)
        } catch (err) {
          await stop()
          throw err
        }
      }
      return { baseURL, stop }
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

async function setAppURL(baseURL: string, superuser: Superuser, appURL: string) {
  const auth = await fetch(`${baseURL}/api/collections/_superusers/auth-with-password`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(superuser),
  })
  if (!auth.ok) {
    throw new Error(`The superuser sign in failed with ${auth.status}`)
  }
  const { token } = (await auth.json()) as { token: string }
  const settings = await fetch(`${baseURL}/api/settings`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json', Authorization: token },
    body: JSON.stringify({ meta: { appURL } }),
  })
  if (!settings.ok) {
    throw new Error(`The Application URL was not set: ${settings.status} ${await settings.text()}`)
  }
}
