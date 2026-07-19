<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { searchScores } from '../api/client'
import { useAsync } from '../composables/useAsync'

const minQueryLength = 2
const debounceMs = 300

const query = ref('')
const { data: results, loading, error, run, reset } = useAsync(searchScores)

let timer: ReturnType<typeof setTimeout> | undefined
watch(query, (q) => {
  clearTimeout(timer)
  const trimmed = q.trim()
  if (trimmed.length < minQueryLength) {
    reset()
    return
  }
  timer = setTimeout(() => run(trimmed), debounceMs)
})
onBeforeUnmount(() => clearTimeout(timer))

function formatRecordedAt(recordedAt: string): string {
  return new Date(recordedAt).toLocaleDateString()
}
</script>

<template>
  <div class="user-search">
    <input
      v-model="query"
      class="search-input"
      type="search"
      placeholder="Search by username…"
      aria-label="Search by username"
    />

    <p v-if="loading" class="status">Searching…</p>
    <p v-else-if="error" class="status status-error">{{ error }}</p>
    <template v-else-if="results != null">
      <p v-if="results.length === 0" class="status">No scores found for “{{ query.trim() }}”.</p>
      <div v-else class="table-scroll">
        <table class="data-table">
          <thead>
            <tr>
              <th>Hole</th>
              <th>Strokes</th>
              <th>Player</th>
              <th>Recorded</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="s in results" :key="s.share_link">
              <td class="num-cell">
                <RouterLink class="cell-link" :to="`/holes/${s.hole}`">{{ s.hole }}</RouterLink>
              </td>
              <td class="num-cell">{{ s.strokes }}</td>
              <td>{{ s.username || s.user_id }}</td>
              <td class="muted-cell">{{ formatRecordedAt(s.recorded_at) }}</td>
              <td class="replay-cell">
                <a class="replay-link" :href="s.share_link" target="_blank" rel="noopener">View replay</a>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
    <p v-else class="status">Find a player's best score on every hole they've played.</p>
  </div>
</template>

<style scoped>
.user-search {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.search-input {
  width: 100%;
  padding: 12px 16px;
  font: inherit;
  color: var(--text-h);
  background: var(--page);
  border: 1px solid var(--border);
  border-radius: 10px;
  outline: none;
}

.search-input::placeholder {
  color: var(--muted);
}

.search-input:focus {
  border-color: var(--accent);
}
</style>
