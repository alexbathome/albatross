<script setup lang="ts">
import type { ScoreRecord } from '../api/types'

defineProps<{ scores: ScoreRecord[] }>()

function formatRecordedAt(recordedAt: string): string {
  return new Date(recordedAt).toLocaleString()
}
</script>

<template>
  <div class="table-scroll">
    <table class="data-table">
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
        <tr v-for="(s, i) in scores" :key="s.share_link" :class="{ 'row-top': i === 0 }">
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
  </div>
</template>
