<script setup lang="ts">
import { computed } from 'vue'
import type { NovelAuditReport } from '@/api/novelWriter'

const props = defineProps<{
  audit: NovelAuditReport
  reviseLoading: boolean
  showReviseAction: boolean
}>()

const emit = defineEmits<{
  revise: []
}>()

const scoreGuide = [
  '统一按 100 分制显示，分数越高越好。',
  '90-100：质量优秀，可直接进入定稿或轻微润色。',
  '75-89：整体良好，有少量可优化问题。',
  '60-74：可读性尚可，但建议按审计意见修订。',
  '0-59：问题较多，建议先修订再继续推进。',
  'AI 味分数越高，表示文字越自然、越不像 AI 拼接生成。',
  '四项维度分别是 AI 味、人物一致性、剧情逻辑、文风贴合。',
  '若历史审计结果来自旧 10 分制，界面会自动换算为 100 分制显示。'
]

const usesLegacyTenScale = computed(() => {
  const values = [
    props.audit?.total_score || 0,
    props.audit?.ai_flavor_score || 0,
    props.audit?.character_score || 0,
    props.audit?.logic_score || 0,
    props.audit?.style_score || 0
  ].filter((value) => value > 0)
  return values.length > 0 && Math.max(...values) <= 10
})

const normalizeScore = (value: number, legacyTenScale: boolean) => {
  const scaled = legacyTenScale ? value * 10 : value
  return Math.max(0, Math.min(100, Math.round(scaled)))
}

const normalizedAudit = computed(() => {
  const legacyTenScale = usesLegacyTenScale.value
  const aiFlavorScore = normalizeScore(props.audit?.ai_flavor_score || 0, legacyTenScale)
  const characterScore = normalizeScore(props.audit?.character_score || 0, legacyTenScale)
  const logicScore = normalizeScore(props.audit?.logic_score || 0, legacyTenScale)
  const styleScore = normalizeScore(props.audit?.style_score || 0, legacyTenScale)
  const averageScore = Math.round((aiFlavorScore + characterScore + logicScore + styleScore) / 4)
  const totalScore = normalizeScore(props.audit?.total_score || 0, legacyTenScale) || averageScore

  return {
    total_score: totalScore,
    ai_flavor_score: aiFlavorScore,
    character_score: characterScore,
    logic_score: logicScore,
    style_score: styleScore,
    issues: props.audit?.issues || [],
    revision_advice: props.audit?.revision_advice || ''
  }
})

const formatSeverity = (severity: string) => {
  if (severity === 'high') return '高风险'
  if (severity === 'medium') return '中风险'
  if (severity === 'low') return '低风险'
  return severity || '未标记'
}
</script>

<template>
  <section class="audit-panel">
    <div class="audit-panel__head">
      <div class="audit-panel__title-row">
        <div>
          <p class="audit-panel__kicker">步骤 4</p>
          <h3 class="audit-panel__title">自动审计/评分</h3>
        </div>
        <el-button
          v-if="showReviseAction"
          size="small"
          class="audit-panel__revise-btn"
          :loading="reviseLoading"
          @click="emit('revise')"
        >
          按审计修订
        </el-button>
      </div>
      <el-tooltip placement="left-start" effect="light" :show-after="120">
        <template #content>
          <div class="score-guide-tooltip">
            <strong>评分说明</strong>
            <p v-for="line in scoreGuide" :key="line">{{ line }}</p>
          </div>
        </template>
        <div class="score-orb">{{ normalizedAudit.total_score }}</div>
      </el-tooltip>
    </div>

    <div class="score-grid">
      <div class="score-card">
        <span>AI 味</span>
        <strong>{{ normalizedAudit.ai_flavor_score }}</strong>
      </div>
      <div class="score-card">
        <span>人物一致性</span>
        <strong>{{ normalizedAudit.character_score }}</strong>
      </div>
      <div class="score-card">
        <span>剧情逻辑</span>
        <strong>{{ normalizedAudit.logic_score }}</strong>
      </div>
      <div class="score-card">
        <span>文风贴合</span>
        <strong>{{ normalizedAudit.style_score }}</strong>
      </div>
    </div>

    <div class="issue-list">
      <div v-for="issue in normalizedAudit.issues" :key="`${issue.title}-${issue.detail}`" class="issue-card">
        <span :class="['issue-card__severity', `issue-card__severity--${issue.severity}`]">{{ formatSeverity(issue.severity) }}</span>
        <strong>{{ issue.title }}</strong>
        <p>{{ issue.detail }}</p>
        <em>{{ issue.suggestion }}</em>
      </div>
      <el-empty v-if="!normalizedAudit.issues.length" description="暂无审计结果" />
    </div>

    <div v-if="normalizedAudit.revision_advice" class="advice-box">
      {{ normalizedAudit.revision_advice }}
    </div>
  </section>
</template>

<style scoped>
.audit-panel {
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr) auto;
  gap: 16px;
  height: 100%;
  box-sizing: border-box;
  min-width: 0;
  width: 100%;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 24px;
  padding: 22px;
}

.audit-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.audit-panel__title-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.audit-panel__kicker {
  margin: 0 0 4px;
  color: #0f766e;
  font-size: 12px;
  font-weight: 800;
}

.audit-panel__title {
  margin: 0;
  color: #0f172a;
  font-size: 20px;
}

.audit-panel__revise-btn {
  flex: 0 0 auto;
}

.score-orb {
  display: grid;
  place-items: center;
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: linear-gradient(135deg, #14b8a6, #0f766e);
  color: #ffffff;
  font-size: 24px;
  font-weight: 900;
  cursor: help;
}

.score-guide-tooltip {
  max-width: 280px;
  display: grid;
  gap: 6px;
  color: #0f172a;
  line-height: 1.55;
}

.score-guide-tooltip strong {
  font-size: 13px;
}

.score-guide-tooltip p {
  margin: 0;
  font-size: 12px;
}

.score-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.score-card {
  padding: 12px;
  border-radius: 16px;
  background: #f8fafc;
}

.score-card span,
.score-card strong {
  display: block;
}

.score-card span {
  color: #64748b;
  font-size: 12px;
  line-height: 1.35;
  word-break: keep-all;
}

.score-card strong {
  margin-top: 6px;
  color: #0f172a;
  font-size: 20px;
}

.issue-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
  overflow-y: auto;
  padding-right: 4px;
}

.issue-card {
  padding: 14px;
  border: 1px solid #fee2e2;
  border-radius: 16px;
  background: #fff7ed;
}

.issue-card__severity {
  display: inline-block;
  margin-bottom: 8px;
  padding: 3px 8px;
  border-radius: 999px;
  background: #fed7aa;
  color: #9a3412;
  font-size: 12px;
}

.issue-card__severity--high {
  background: #fecaca;
  color: #991b1b;
}

.issue-card strong,
.issue-card p,
.issue-card em {
  display: block;
}

.issue-card strong {
  color: #0f172a;
  word-break: break-word;
}

.issue-card p {
  margin: 8px 0;
  color: #475569;
  line-height: 1.65;
  word-break: break-word;
}

.issue-card em {
  color: #0f766e;
  font-style: normal;
  line-height: 1.65;
  word-break: break-word;
}

.advice-box {
  padding: 14px;
  border-radius: 16px;
  background: #ecfeff;
  color: #0f766e;
  line-height: 1.7;
}

@media (max-width: 900px) {
  .audit-panel__head,
  .audit-panel__title-row {
    align-items: flex-start;
    flex-direction: column;
  }

  .score-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
