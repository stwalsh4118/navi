import { mkdir, readFile, rename, writeFile } from "node:fs/promises"
import { randomUUID } from "node:crypto"
import { homedir } from "node:os"
import path from "node:path"

const STATUS_DIR = path.join(homedir(), ".claude-sessions")
const OPENCODE_AGENT_KEY = "opencode"

function nowUnixSeconds() {
  return Math.floor(Date.now() / 1000)
}

async function resolveSessionName($) {
  try {
    const paneId = process.env.TMUX_PANE
    const result = paneId
      ? await $`tmux display-message -p -t ${paneId} '#{session_name}'`
      : await $`tmux display-message -p '#{session_name}'`
    const output =
      typeof result === "string"
        ? result
        : typeof result?.stdout === "string"
          ? result.stdout
          : typeof result?.text === "function"
            ? await result.text()
            : ""
    const sessionName = output.trim()
    return sessionName.length > 0 ? sessionName : null
  } catch {
    return null
  }
}

async function readStatusFile(filePath) {
  try {
    const contents = await readFile(filePath, "utf8")
    if (!contents.trim()) {
      return {}
    }
    const parsed = JSON.parse(contents)
    return parsed && typeof parsed === "object" ? parsed : {}
  } catch {
    return {}
  }
}

async function updateStatus(sessionName, status) {
  if (!sessionName) {
    return
  }

  await mkdir(STATUS_DIR, { recursive: true })

  const statusPath = path.join(STATUS_DIR, `${sessionName}.json`)
  const existing = await readStatusFile(statusPath)
  const existingAgents =
    existing.agents && typeof existing.agents === "object" && !Array.isArray(existing.agents)
      ? existing.agents
      : {}

  const updated = {
    ...existing,
    tmux_session: existing.tmux_session || sessionName,
    agents: {
      ...existingAgents,
      [OPENCODE_AGENT_KEY]: {
        status,
        timestamp: nowUnixSeconds(),
      },
    },
  }

  const tempPath = `${statusPath}.tmp-${process.pid}-${Date.now()}-${randomUUID()}`
  await writeFile(tempPath, `${JSON.stringify(updated, null, 2)}\n`, "utf8")
  try {
    await rename(tempPath, statusPath)
  } catch {
    await writeFile(statusPath, `${JSON.stringify(updated, null, 2)}\n`, "utf8")
  }
}

export const NaviPlugin = async ({ $ }) => {
  let cachedSessionName = await resolveSessionName($)
  let writeQueue = Promise.resolve()

  const writeStatus = async (status) => {
    writeQueue = writeQueue
      .catch(() => undefined)
      .then(async () => {
        const runtimeSessionName = await resolveSessionName($)
        if (runtimeSessionName) {
          cachedSessionName = runtimeSessionName
        }
        await updateStatus(cachedSessionName, status)
      })
    await writeQueue
  }

  return {
    "tool.execute.after": async () => writeStatus("working"),
    "chat.message": async () => writeStatus("working"),
    event: async ({ event }) => {
      switch (event.type) {
        case "session.created":
          await writeStatus("working")
          break
        case "session.idle":
          await writeStatus("idle")
          break
        case "session.error":
          await writeStatus("error")
          break
        case "permission.updated":
          await writeStatus("permission")
          break
        case "permission.replied":
          await writeStatus("working")
          break
        case "session.status":
          if (event.properties?.status?.type === "busy") {
            await writeStatus("working")
          }
          if (event.properties?.status?.type === "idle") {
            await writeStatus("idle")
          }
          break
        default:
          break
      }
    },
  }
}
