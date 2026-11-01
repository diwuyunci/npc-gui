import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { backend } from '@/lib/backend'
import type { OperationResult, Profile, ProfileSaveResult, ProfilesSaveResult } from '@/lib/types'

export const useProfilesStore = defineStore('profiles', () => {
  const items = ref<Profile[]>([])
  const selectedId = ref('')
  const search = ref('')

  const filtered = computed(() => {
    const q = search.value.trim().toLowerCase()
    if (!q) return items.value
    return items.value.filter(
      (p) =>
        p.name.toLowerCase().includes(q) ||
        p.server.toLowerCase().includes(q) ||
        (p.remark || '').toLowerCase().includes(q),
    )
  })

  const selected = computed(() => items.value.find((p) => p.id === selectedId.value) ?? null)

  function setAll(next: Profile[]): void {
    items.value = next
    if (!next.some((p) => p.id === selectedId.value)) {
      selectedId.value = next[0]?.id ?? ''
    }
  }

  function upsert(profile: Profile): void {
    const idx = items.value.findIndex((p) => p.id === profile.id)
    if (idx >= 0) items.value = items.value.map((p) => (p.id === profile.id ? profile : p))
    else items.value = [...items.value, profile]
  }

  function remove(id: string): void {
    items.value = items.value.filter((p) => p.id !== id)
    if (selectedId.value === id) selectedId.value = items.value[0]?.id ?? ''
  }

  function select(id: string): void {
    selectedId.value = id
  }

  async function save(profile: Profile): Promise<ProfileSaveResult> {
    const result = await backend.saveProfile(profile)
    if (result.result.ok && result.profile?.id) {
      upsert(result.profile)
      selectedId.value = result.profile.id
    }
    return result
  }

  async function deleteProfile(id: string): Promise<OperationResult> {
    const result = await backend.deleteProfile(id)
    if (result.ok) remove(id)
    return result
  }

  async function duplicate(id: string): Promise<ProfileSaveResult> {
    const result = await backend.duplicateProfile(id)
    if (result.result.ok && result.profile?.id) upsert(result.profile)
    return result
  }

  async function commitImport(profiles: Profile[]): Promise<ProfilesSaveResult> {
    const result = await backend.commitImport(profiles)
    if (result.result.ok) {
      items.value = [...items.value, ...result.profiles]
      selectedId.value = result.profiles[0]?.id ?? selectedId.value
    }
    return result
  }

  async function importFromFile(): Promise<ProfilesSaveResult> {
    const result = await backend.importProfiles()
    if (result.result.ok) items.value = [...items.value, ...result.profiles]
    return result
  }

  return {
    items,
    selectedId,
    search,
    filtered,
    selected,
    setAll,
    upsert,
    remove,
    select,
    save,
    deleteProfile,
    duplicate,
    commitImport,
    importFromFile,
  }
})
