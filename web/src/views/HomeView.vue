<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { listHoles, topScores } from '../api/client'
import { useAsync } from '../composables/useAsync'
import LeaderboardTable from '../components/LeaderboardTable.vue'
import UserSearch from '../components/UserSearch.vue'

const { data: holes, error: holesError, run: loadHoles } = useAsync(listHoles)
const { data: scores, loading: scoresLoading, error: scoresError, run: loadScores } = useAsync(topScores)

const latestHole = computed(() => holes.value?.[0]?.hole ?? null)
const loading = computed(() => holes.value == null || scoresLoading.value)
const error = computed(() => holesError.value ?? scoresError.value)

onMounted(async () => {
  await loadHoles(1)
  if (latestHole.value != null) await loadScores(latestHole.value, 10)
})
</script>

<template>
  <div class="page">
    <div class="home-grid">
      <section class="card panel">
        <h2 class="panel-title">Today's top 10</h2>
        <RouterLink v-if="latestHole != null" class="hole-link" :to="`/holes/${latestHole}`">
          Hole {{ latestHole }} <span aria-hidden="true">→</span>
        </RouterLink>

        <p v-if="error" class="status status-error">{{ error }}</p>
        <p v-else-if="loading" class="status">Loading…</p>
        <p v-else-if="latestHole == null" class="status">No holes recorded yet.</p>
        <p v-else-if="scores == null || scores.length === 0" class="status">
          No scores recorded for this hole yet.
        </p>
        <LeaderboardTable v-else :scores="scores" />
      </section>

      <section class="card panel">
        <h2 class="panel-title">Search by player</h2>
        <UserSearch />
      </section>
    </div>
  </div>
</template>

<style scoped>
.page {
  padding: 8px 24px 48px;

  @media (max-width: 720px) {
    padding: 8px 16px 32px;
  }
}

.home-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
  align-items: start;

  @media (max-width: 860px) {
    grid-template-columns: 1fr;
  }
}

.hole-link {
  display: inline-block;
  margin-bottom: 20px;
  font-size: 40px;
  font-weight: 800;
  letter-spacing: -1px;
  color: var(--accent);
  text-decoration: none;
}

.hole-link:hover {
  text-decoration: underline;
}
</style>
