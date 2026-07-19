<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { listHoles, topScores } from '../api/client'
import type { Hole, ScoreRecord } from '../api/types'

// Large enough to cover the full archive of daily holes in one request —
// see maxHolesLimit in internal/api/holes.go.
const holesFetchLimit = 1000

const props = defineProps<{ hole?: number }>()
const router = useRouter()

const holes = ref<Hole[]>([])
const holesLoading = ref(true)
const holesError = ref<string | null>(null)

const scores = ref<ScoreRecord[]>([])
const scoresLoading = ref(true)
const scoresError = ref<string | null>(null)

onMounted(async () => {
  try {
    holes.value = await listHoles(holesFetchLimit)
  } catch (e) {
    holesError.value = e instanceof Error ? e.message : String(e)
  } finally {
    holesLoading.value = false
  }
})

const latestHole = computed(() => holes.value[0]?.hole ?? null)
const selectedHole = computed(() => props.hole ?? latestHole.value)

interface HoleRow {
  hole: number
  played: boolean
  topStrokes: number | null
}

// The archive always runs from the latest hole down to 1, so holes with no
// recorded scores still show up as unplayed rather than being skipped.
const holeRows = computed<HoleRow[]>(() => {
  if (latestHole.value == null) return []
  const byNumber = new Map(holes.value.map((h) => [h.hole, h]))
  const rows: HoleRow[] = []
  for (let n = latestHole.value; n >= 1; n--) {
    const h = byNumber.get(n)
    rows.push({ hole: n, played: h != null, topStrokes: h?.top_strokes ?? null })
  }
  return rows
})

async function loadScores(hole: number) {
  scoresLoading.value = true
  scoresError.value = null
  try {
    scores.value = await topScores(hole, 25)
  } catch (e) {
    scoresError.value = e instanceof Error ? e.message : String(e)
  } finally {
    scoresLoading.value = false
  }
}

watch(
  selectedHole,
  (hole) => {
    if (hole != null) loadScores(hole)
  },
  { immediate: true },
)

function selectHole(row: HoleRow) {
  if (!row.played) return
  router.push(`/holes/${row.hole}`)
}

function goToToday() {
  router.push('/')
}

function formatRecordedAt(recordedAt: string): string {
  return new Date(recordedAt).toLocaleString()
}
</script>

<template>
  <div class="page">
    <div class="dashboard">
      <div class="hero-col">
        <div class="hero">
          <p class="hero-label">{{ selectedHole === latestHole ? "Today's hole" : 'Viewing hole' }}</p>
          <p class="hero-number">{{ selectedHole ?? '—' }}</p>
          <button
            type="button"
            class="today-button"
            :class="{ 'today-button-hidden': latestHole == null || selectedHole === latestHole }"
            :tabindex="latestHole == null || selectedHole === latestHole ? -1 : 0"
            @click="goToToday"
          >
            Go to today
          </button>
        </div>
        <section v-if="holesLoading || holesError || holeRows.length > 0" class="panel">
          <h2 class="panel-title">Archive</h2>

          <p v-if="holesLoading" class="status">Loading holes…</p>
          <p v-else-if="holesError" class="status status-error">{{ holesError }}</p>
          <div v-else class="scroll-root">
            <table class="holes-table">
              <thead>
                <tr>
                  <th>#</th>
                  <th>Top</th>
                  <th>Par</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="row in holeRows"
                  :key="row.hole"
                  :class="[
                    'holes-row',
                    {
                      'holes-row-selected': row.hole === selectedHole,
                      'holes-row-disabled': !row.played,
                    },
                  ]"
                  :tabindex="row.played ? 0 : undefined"
                  :role="row.played ? 'link' : undefined"
                  @click="selectHole(row)"
                  @keydown.enter="selectHole(row)"
                >
                  <td class="num-cell">{{ row.hole }}</td>
                  <td class="num-cell">{{ row.topStrokes ?? '—' }}</td>
                  <td class="par-todo">TODO</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>
        

      <div class="content-col">
        <section class="panel">
          <h2 class="panel-title">Top 25</h2>

          <p v-if="scoresLoading" class="status">Loading…</p>
          <p v-else-if="scoresError" class="status status-error">{{ scoresError }}</p>
          <p v-else-if="scores.length === 0" class="status">No scores recorded for this hole yet.</p>

          <table v-else class="scores-table">
            <thead>
              <tr>
                <th>#</th>
                <th>Strokes</th>
                <th>Player</th>
                <th>Recorded</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(s, i) in scores" :key="s.share_link" :class="{ 'scores-row-top': i === 0 }">
                <td class="num-cell">{{ i + 1 }}</td>
                <td class="num-cell strokes-cell">{{ s.strokes }}</td>
                <td>{{ s.username || s.user_id }}</td>
                <td class="muted-cell">{{ formatRecordedAt(s.recorded_at) }}</td>
                <td class="replay-cell">
                  <a class="replay-link" :href="s.share_link" target="_blank" rel="noopener">View replay</a>
                </td>
              </tr>
            </tbody>
          </table>
        </section>

        
      </div>
    </div>
  </div>
</template>

<style scoped>
.page {
  padding: 8px 24px 48px;
}

.dashboard {
  display: flex;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: var(--bg);
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

.hero-col .panel {
  border-top: 1px solid var(--border);
}

.hero {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 16px;
  padding: 32px;
  width: 100%;
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

.content-col {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.panel {
  padding: 28px 32px;
}

.panel + .panel {
  border-top: 1px solid var(--border);
}

.panel-title {
  margin: 0 0 16px;
  font-size: 15px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--muted);
}

.scores-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 15px;
}

.scores-table th,
.scores-table td {
  text-align: left;
  padding: 10px 8px;
  border-bottom: 1px solid var(--border);
}

.scores-table th {
  color: var(--muted);
  font-weight: 600;
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.scores-table td {
  color: var(--text-h);
}

.muted-cell {
  color: var(--muted);
}

.scores-row-top .strokes-cell {
  color: var(--accent);
  font-weight: 800;
}

.replay-cell {
  text-align: right;
}

.replay-link {
  display: inline-block;
  padding: 5px 14px;
  font-size: 13px;
  font-weight: 700;
  color: var(--accent-ink);
  background: var(--accent-soft);
  border-radius: 999px;
  text-decoration: none;
  white-space: nowrap;
}

.replay-link:hover {
  background: var(--accent);
}

.scroll-root {
  height: 360px;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: 10px;
}

.holes-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

.holes-table th {
  position: sticky;
  top: 0;
  text-align: left;
  padding: 8px 12px;
  font-weight: 600;
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--muted);
  background: var(--bg);
  border-bottom: 1px solid var(--border);
}

.holes-table td {
  padding: 8px 12px;
  border-bottom: 1px solid var(--border);
  color: var(--text-h);
}

.holes-row {
  cursor: pointer;
}

.holes-row:hover {
  background: var(--row-hover);
}

.holes-row-selected {
  background: var(--accent-soft);
}

.holes-row-selected td:first-child {
  color: var(--accent);
  font-weight: 700;
}

.holes-row-disabled {
  cursor: default;
  color: var(--muted);
}

.holes-row-disabled td {
  color: var(--muted);
}

.holes-row-disabled:hover {
  background: transparent;
}

.par-todo {
  font-size: 11px;
  font-style: italic;
  color: var(--muted);
}

.num-cell {
  font-variant-numeric: tabular-nums;
}
</style>
