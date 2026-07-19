<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { listHoles, topScores } from '../api/client'
import { useAsync } from '../composables/useAsync'
import HoleArchive from '../components/HoleArchive.vue'
import LeaderboardTable from '../components/LeaderboardTable.vue'

const props = defineProps<{ hole: number }>()
const router = useRouter()

const valid = computed(() => Number.isInteger(props.hole) && props.hole >= 1)

const { data: holes, loading: holesLoading, error: holesError, run: loadHoles } = useAsync(listHoles)
const { data: scores, loading: scoresLoading, error: scoresError, run: loadScores } = useAsync(topScores)

const latestHole = computed(() => holes.value?.[0]?.hole ?? null)
const onToday = computed(() => latestHole.value != null && props.hole === latestHole.value)

onMounted(() => loadHoles())
watch(
  () => props.hole,
  (hole) => {
    if (valid.value) loadScores(hole, 25)
  },
  { immediate: true },
)

function goToToday() {
  if (latestHole.value != null) router.push(`/holes/${latestHole.value}`)
}

function selectHole(hole: number) {
  router.push(`/holes/${hole}`)
}
</script>

<template>
  <div class="page">
    <p v-if="!valid" class="status status-error">Unknown hole.</p>
    <div v-else class="dashboard card">
      <div class="hero-col">
        <div class="hero">
          <p class="hero-label">{{ onToday ? "Today's hole" : 'Viewing hole' }}</p>
          <p class="hero-number">{{ hole }}</p>
          <button
            type="button"
            class="today-button"
            :class="{ 'today-button-hidden': latestHole == null || onToday }"
            :tabindex="latestHole == null || onToday ? -1 : 0"
            @click="goToToday"
          >
            Go to today
          </button>
        </div>

        <section class="panel archive-panel">
          <h2 class="panel-title">Archive</h2>
          <p v-if="holesLoading" class="status">Loading holes…</p>
          <p v-else-if="holesError" class="status status-error">{{ holesError }}</p>
          <HoleArchive v-else :holes="holes ?? []" :selected="hole" @select="selectHole" />
        </section>
      </div>

      <div class="content-col">
        <section class="panel">
          <h2 class="panel-title">Top 25</h2>
          <p v-if="scoresLoading" class="status">Loading…</p>
          <p v-else-if="scoresError" class="status status-error">{{ scoresError }}</p>
          <p v-else-if="scores == null || scores.length === 0" class="status">
            No scores recorded for this hole yet.
          </p>
          <LeaderboardTable v-else :scores="scores" />
        </section>
      </div>
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

.dashboard {
  display: flex;
  overflow: hidden;

  @media (max-width: 720px) {
    flex-direction: column;
  }
}

.hero-col {
  flex: 0 0 320px;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--border);

  @media (max-width: 720px) {
    flex: 1 1 auto;
    border-right: none;
    border-bottom: 1px solid var(--border);
  }
}

.hero {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 16px;
  padding: 32px;
}

.hero-label {
  margin: 0;
  font-size: 13px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--muted);
}

.hero-number {
  margin: 0;
  font-size: 124px;
  line-height: 1;
  font-weight: 800;
  letter-spacing: -4px;
  color: var(--accent);

  @media (max-width: 720px) {
    font-size: 88px;
    letter-spacing: -3px;
  }
}

.today-button {
  padding: 10px 20px;
  font: inherit;
  font-size: 14px;
  font-weight: 700;
  color: var(--accent-ink);
  background: var(--accent);
  border: none;
  border-radius: 999px;
  cursor: pointer;
  transition: transform 0.1s ease;
}

.today-button-hidden {
  visibility: hidden;
  pointer-events: none;
}

.today-button:hover {
  transform: translateY(-1px);
}

.archive-panel {
  border-top: 1px solid var(--border);
}

.content-col {
  flex: 1 1 auto;
  min-width: 0;
}
</style>
