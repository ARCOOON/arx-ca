import { readonly, ref } from 'vue'
import { fetchSettingsConfig } from '@/api/config'
import { fetchHealth } from '@/api/health'
import { useAuthStore } from '@/store/auth'

const LAST_SEEN_VERSION_KEY = 'arx_last_seen_version'

const triggerChangelogModal = ref(false)
let driftCheckPromise: Promise<void> | null = null

function canManageSettings(roles: string[]): boolean {
  return roles.includes('SuperAdmin') || roles.includes('CA-Admin')
}

function normalizeVersion(value: string): string {
  return value.trim().replace(/^v/i, '').toLowerCase()
}

function versionsDiffer(stored: string, current: string): boolean {
  return normalizeVersion(stored) !== normalizeVersion(current)
}

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
        versionsDiffer(previousVersion, currentVersion)
      ) {
        triggerChangelogModal.value = true
      }

      localStorage.setItem(LAST_SEEN_VERSION_KEY, currentVersion)
    } catch {
      // Drift detection is best-effort during bootstrap.
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

  function dismissChangelogModal(): void {
    triggerChangelogModal.value = false
  }

  return {
    triggerChangelogModal: readonly(triggerChangelogModal),
    runVersionDriftCheck,
    dismissChangelogModal,
  }
}
