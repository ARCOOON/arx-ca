import { readonly, ref } from 'vue'
import { fetchHealth } from '@/api/health'
import { fetchSettingsConfig } from '@/api/settings'
import { useAuthStore } from '@/stores/auth'

const LAST_SEEN_VERSION_KEY = 'arx_last_seen_version'

const showChangelog = ref(false)
let driftCheckPromise: Promise<void> | null = null

function canManageSettings(roles: string[]): boolean {
  return roles.includes('SuperAdmin') || roles.includes('CA-Admin')
}

function normalizeVersion(value: string): string {
  return value.trim().replace(/^v/i, '').toLowerCase()
}

/**
 * Detect a running-binary version drift and, when the operator opted into
 * post-update changelogs, surface the release notes modal exactly once.
 */
export function useUpdater() {
  async function checkVersionDrift(): Promise<void> {
    if (typeof window === 'undefined') {
      return
    }

    const authStore = useAuthStore()
    if (!authStore.isAuthenticated || !canManageSettings(authStore.roles)) {
      return
    }

    try {
      const [health, config] = await Promise.all([fetchHealth(), fetchSettingsConfig()])
      const currentVersion = health.api.binary_version || health.api.version
      const previousVersion = localStorage.getItem(LAST_SEEN_VERSION_KEY)

      if (
        config.updater.view_changelog_after_update &&
        previousVersion !== null &&
        normalizeVersion(previousVersion) !== normalizeVersion(currentVersion)
      ) {
        showChangelog.value = true
      }

      localStorage.setItem(LAST_SEEN_VERSION_KEY, currentVersion)
    } catch {
      // Drift detection is best-effort during bootstrap; ignore transient errors.
    }
  }

  function runVersionDriftCheck(): void {
    if (driftCheckPromise) {
      return
    }
    driftCheckPromise = checkVersionDrift().finally(() => {
      driftCheckPromise = null
    })
  }

  function dismissChangelog(): void {
    showChangelog.value = false
  }

  return {
    showChangelog: readonly(showChangelog),
    runVersionDriftCheck,
    dismissChangelog,
  }
}
