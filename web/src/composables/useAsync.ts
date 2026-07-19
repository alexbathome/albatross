import { ref, shallowRef } from 'vue'

/**
 * Wraps an async fetcher with loading/error/data state.
 *
 * Overlapping runs are last-call-wins: a stale response never overwrites the
 * state of a newer call, which matters for type-ahead search.
 */
export function useAsync<Args extends unknown[], T>(fetcher: (...args: Args) => Promise<T>) {
  const data = shallowRef<T | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  let runId = 0

  async function run(...args: Args): Promise<void> {
    const id = ++runId
    loading.value = true
    error.value = null
    try {
      const result = await fetcher(...args)
      if (id === runId) data.value = result
    } catch (e) {
      if (id === runId) {
        error.value = e instanceof Error ? e.message : String(e)
        data.value = null
      }
    } finally {
      if (id === runId) loading.value = false
    }
  }

  /** Clears all state and invalidates any in-flight run. */
  function reset(): void {
    runId++
    data.value = null
    error.value = null
    loading.value = false
  }

  return { data, loading, error, run, reset }
}
