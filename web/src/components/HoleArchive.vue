<script setup lang="ts">
import { computed } from 'vue'
import type { Hole } from '../api/types'

const props = defineProps<{ holes: Hole[]; selected: number | null }>()
const emit = defineEmits<{ select: [hole: number] }>()

interface ArchiveRow {
  hole: number
  played: boolean
  topStrokes: number | null
}

// The archive always runs from the latest hole down to 1, so holes with no
// recorded scores still show up as unplayed rather than being skipped.
const rows = computed<ArchiveRow[]>(() => {
  const latest = props.holes[0]?.hole
  if (latest == null) return []
  const byNumber = new Map(props.holes.map((h) => [h.hole, h]))
  const out: ArchiveRow[] = []
  for (let n = latest; n >= 1; n--) {
    const h = byNumber.get(n)
    out.push({ hole: n, played: h != null, topStrokes: h?.top_strokes ?? null })
  }
  return out
})

function select(row: ArchiveRow) {
  if (row.played) emit('select', row.hole)
}
</script>

<template>
  <div class="archive-scroll">
    <table class="data-table archive-table">
      <thead>
        <tr>
          <th>#</th>
          <th>Top</th>
          <th>Par</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="row in rows"
          :key="row.hole"
          :class="[
            'archive-row',
            {
              'archive-row-selected': row.hole === selected,
              'archive-row-disabled': !row.played,
            },
          ]"
          :tabindex="row.played ? 0 : undefined"
          :role="row.played ? 'link' : undefined"
          @click="select(row)"
          @keydown.enter="select(row)"
        >
          <td class="num-cell">{{ row.hole }}</td>
          <td class="num-cell">{{ row.topStrokes ?? '—' }}</td>
          <td class="par-todo">TODO</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.archive-scroll {
  height: 360px;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: 10px;
}

.archive-table {
  font-size: 14px;
}

.archive-table th {
  position: sticky;
  top: 0;
  background: var(--bg);
}

.archive-row {
  cursor: pointer;
}

.archive-row:hover {
  background: var(--row-hover);
}

.archive-row-selected {
  background: var(--accent-soft);
}

.archive-row-selected td:first-child {
  color: var(--accent);
  font-weight: 700;
}

.archive-row-disabled {
  cursor: default;
}

.archive-row-disabled td {
  color: var(--muted);
}

.archive-row-disabled:hover {
  background: transparent;
}

.par-todo {
  font-size: 11px;
  font-style: italic;
  color: var(--muted);
}
</style>
