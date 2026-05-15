<script setup lang="ts">
import { computed } from 'vue'
import { type NovelFullReview } from '@/api/novelWriter'

const props = defineProps<{
  review: NovelFullReview
  loading: boolean
  showReviseAction: boolean
}>()

const emit = defineEmits<{
  revise: []
}>()

const emptyReview = (): NovelFullReview => ({
  total_score: 0,
  coherence_score: 0,
  logic_reasonability_score: 0,
  character_consistency_score: 0,
  trigger_reasonability_score: 0,
  summary: '',
  issues: [],
  revision_advice: '',
  reviewed_at: '',
  applied_at: ''
})

const safeReview = computed(() => props.review || emptyReview())

const hasResult = computed(() => Boolean(
  safeReview.value.reviewed_at
  || safeReview.value.summary?.trim()
  || safeReview.value.issues?.length
  || safeReview.value.revision_advice?.trim()
  || safeReview.value.total_score
))

const scoreCards = computed(() => [
  { label: '剧情连贯性', value: safeReview.value.coherence_score || 0 },
  { label: '逻辑合理性', value: safeReview.value.logic_reasonability_score || 0 },
  { label: '角色一致性', value: safeReview.value.character_consistency_score || 0 },
  { label: '事件触发合理性', value: safeReview.value.trigger_reasonability_score || 0 }
])

const formatSeverity = (severity: string) => {
  if (severity === 'high') return '高风险'
  if (severity === 'medium') return '中风险'
  if (severity === 'low') return '低风险'
  return severity || '未标记'
}
</script>

<template>
  <section class="full-review-panel">
    <div class="full-review-panel__head">
      <div>
        <p class="full-review-panel__kicker">全文核验结果</p>
        <h3 class="full-review-panel__title">全本逻辑与连贯性复检</h3>
      </div>
      <div class="full-review-panel__head-actions">
        <span v-if="safeReview.applied_at" class="full-review-panel__applied-badge">已按核验修订</span>
        <el-button
          v-if="showReviseAction"
          size="small"
          type="success"
          :loading="loading"
          @click="emit('revise')"
        >
          按核验结果修订
        </el-button>
      </div>
    </div>

    <template v-if="hasResult">
      <div class="full-review-panel__meta">
        <span v-if="safeReview.reviewed_at">核验时间：{{ safeReview.reviewed_at }}</span>
        <span v-if="safeReview.applied_at">修订时间：{{ safeReview.applied_at }}</span>
      </div>

      <div class="full-review-score-grid">
        <div class="full-review-score-orb">
          <span>总分</span>
          <strong>{{ safeReview.total_score || 0 }}</strong>
        </div>
        <div class="full-review-score-cards">
          <div v-for="item in scoreCards" :key="item.label" class="full-review-score-card">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
      </div>

      <div v-if="safeReview.summary" class="full-review-summary">
        {{ safeReview.summary }}
      </div>

      <div class="full-review-issue-list">
        <div
          v-for="issue in safeReview.issues"
          :key="`${issue.chapter_id}-${issue.title}-${issue.detail}`"
          class="full-review-issue-card"
        >
          <div class="full-review-issue-card__top">
            <span :class="['full-review-issue-card__severity', `full-review-issue-card__severity--${issue.severity}`]">
              {{ formatSeverity(issue.severity) }}
            </span>
            <span v-if="issue.dimension" class="full-review-issue-card__dimension">{{ issue.dimension }}</span>
            <span v-if="issue.chapter_title" class="full-review-issue-card__chapter">{{ issue.chapter_title }}</span>
          </div>
          <strong>{{ issue.title }}</strong>
          <p>{{ issue.detail }}</p>
          <em>{{ issue.suggestion }}</em>
        </div>
        <el-empty v-if="!safeReview.issues?.length" description="未发现明显的全本结构问题" />
      </div>

      <div v-if="safeReview.revision_advice" class="full-review-advice">
        {{ safeReview.revision_advice }}
      </div>
    </template>

    <el-empty v-else description="完成全文质量核验后，这里会显示全书级问题与修订建议" />
  </section>
</template>

<style scoped>
.full-review-panel {
  display: grid;
  gap: 16px;
  min-width: 0;
  width: 100%;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 24px;
  padding: 22px;
  box-sizing: border-box;
}

.full-review-panel__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.full-review-panel__kicker {
  margin: 0 0 4px;
  color: #0f766e;
  font-size: 12px;
  font-weight: 800;
}

.full-review-panel__title {
  margin: 0;
  color: #0f172a;
  font-size: 20px;
}

.full-review-panel__head-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.full-review-panel__applied-badge {
  display: inline-flex;
  align-items: center;
  height: 30px;
  padding: 0 12px;
  border-radius: 999px;
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
  color: #059669;
  font-size: 12px;
  font-weight: 800;
}

.full-review-panel__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  color: #64748b;
  font-size: 12px;
}

.full-review-score-grid {
  display: grid;
  grid-template-columns: 120px minmax(0, 1fr);
  gap: 14px;
  min-width: 0;
}

.full-review-score-orb {
  display: grid;
  place-items: center;
  gap: 4px;
  min-height: 120px;
  border-radius: 22px;
  background: linear-gradient(135deg, #14b8a6, #0f766e);
  color: #ffffff;
}

.full-review-score-orb span {
  font-size: 12px;
  font-weight: 700;
}

.full-review-score-orb strong {
  font-size: 34px;
  font-weight: 900;
}

.full-review-score-cards {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.full-review-score-card {
  padding: 14px;
  border-radius: 16px;
  background: #f8fafc;
}

.full-review-score-card span,
.full-review-score-card strong {
  display: block;
}

.full-review-score-card span {
  color: #64748b;
  font-size: 12px;
}

.full-review-score-card strong {
  margin-top: 6px;
  color: #0f172a;
  font-size: 22px;
}

.full-review-summary,
.full-review-advice {
  padding: 14px 16px;
  border-radius: 16px;
  background: #f8fafc;
  color: #334155;
  line-height: 1.7;
}

.full-review-advice {
  border: 1px solid #c7f9cc;
  background: #f0fdf4;
  color: #166534;
}

.full-review-issue-list {
  display: grid;
  gap: 10px;
}

.full-review-issue-card {
  padding: 14px;
  border: 1px solid #fee2e2;
  border-radius: 16px;
  background: #fff7ed;
}

.full-review-issue-card__top {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 8px;
}

.full-review-issue-card__severity,
.full-review-issue-card__dimension,
.full-review-issue-card__chapter {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 800;
}

.full-review-issue-card__severity--high {
  background: #fee2e2;
  color: #b91c1c;
}

.full-review-issue-card__severity--medium {
  background: #fef3c7;
  color: #b45309;
}

.full-review-issue-card__severity--low {
  background: #dcfce7;
  color: #15803d;
}

.full-review-issue-card__dimension {
  background: #e0f2fe;
  color: #0369a1;
}

.full-review-issue-card__chapter {
  background: #ecfeff;
  color: #0f766e;
}

.full-review-issue-card strong {
  display: block;
  color: #0f172a;
  margin-bottom: 6px;
}

.full-review-issue-card p,
.full-review-issue-card em {
  display: block;
  margin: 0;
  line-height: 1.65;
}

.full-review-issue-card p {
  color: #475569;
}

.full-review-issue-card em {
  margin-top: 8px;
  color: #0f766e;
  font-style: normal;
}

@media (max-width: 900px) {
  .full-review-score-grid {
    grid-template-columns: 1fr;
  }

  .full-review-score-cards {
    grid-template-columns: 1fr;
  }
}
</style>
