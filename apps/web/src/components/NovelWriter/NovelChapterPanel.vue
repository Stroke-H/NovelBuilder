<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue'
import type { NovelChapter, NovelOutline } from '@/api/novelWriter'

const props = defineProps<{
  outline: NovelOutline
  selectedOutlineId: string
  chapters: NovelChapter[]
  selectedChapterId: string
  running: boolean
  generateLoadingOutlineId: string
  auditLoadingChapterId: string
  bulkGenerateLoading: boolean
  bulkAuditLoading: boolean
  bulkReviseLoading: boolean
  fullReviewLoading: boolean
  bulkGenerateProgressCurrent: number
  bulkGenerateProgressTotal: number
  bulkAuditProgressCurrent: number
  bulkAuditProgressTotal: number
  bulkReviseProgressCurrent: number
  bulkReviseProgressTotal: number
  adoptLoadingVersionId: string
  manualSaveLoading: boolean
  approveLoadingChapterId: string
}>()

const emit = defineEmits<{
  selectOutline: [id: string]
  selectChapter: [id: string]
  generate: [outlineId: string]
  audit: [id: string]
  bulkGenerate: []
  bulkAudit: []
  bulkRevise: []
  fullReview: []
  revise: [id: string]
  adopt: [chapterId: string, versionId: string]
  manualSave: [chapterId: string, content: string, versionId: string]
  manualEditStateChange: [editing: boolean]
  approve: [id: string]
}>()

const previewVersionId = shallowRef('')
const isManualEditing = shallowRef(false)
const manualContent = shallowRef('')

const selectedOutline = computed(() => {
  return props.outline.chapters?.find((chapter) => chapter.id === props.selectedOutlineId) || props.outline.chapters?.[0]
})

const selectedChapter = computed(() => {
  const outlineId = selectedOutline.value?.id || props.selectedOutlineId
  if (outlineId) return props.chapters.find((chapter) => chapter.outline_id === outlineId)
  if (props.selectedChapterId) return props.chapters.find((chapter) => chapter.id === props.selectedChapterId)
  return props.chapters[0]
})

const hasSelectedChapterContent = computed(() => {
  const chapter = selectedChapter.value
  if (!chapter) return false
  if (chapter.content?.trim()) return true
  return Boolean(chapter.versions?.some((item) => item.content?.trim()))
})

const previewContent = computed(() => {
  const chapter = selectedChapter.value
  if (!chapter) return ''
  const version = chapter.versions?.find((item) => item.id === previewVersionId.value)
  return version?.content || chapter.content
})

const activeVersionId = computed(() => {
  const chapter = selectedChapter.value
  if (!chapter) return ''
  if (chapter.active_version_id) return chapter.active_version_id
  const versions = [...(chapter.versions || [])].reverse()
  return versions.find((item) => item.content?.trim() && item.content === chapter.content)?.id || ''
})

const currentVersionType = computed(() => {
  const chapter = selectedChapter.value
  if (!chapter) return ''
  const matched = (chapter.versions || []).find((item) => item.id === activeVersionId.value)
  return matched?.type || (chapter.content?.trim() ? 'draft' : '')
})

const chapterStatusByOutlineId = computed(() => {
  return new Map(props.chapters.map((chapter) => [chapter.outline_id, chapter]))
})

const outlineGenerationText = computed(() => {
  const generated = props.outline.generated_chapters || props.outline.chapters?.length || 0
  const target = props.outline.target_chapters || generated
  const remaining = Math.max(target - generated, 0)

  if (props.outline.generation_status === 'generating' && generated < target) {
    return `已生成前 ${generated} 章完整大纲，剩余 ${remaining} 章正在后台继续加载`
  }
  if (props.outline.generation_status === 'failed') {
    return `后续章节大纲生成失败：${props.outline.generation_error || '请稍后重试'}`
  }
  if (props.outline.generation_status === 'cancelled') {
    return props.outline.generation_error || '大纲后台任务已终止'
  }
  return ''
})

const outlineGenerationClass = computed(() => [
  'outline-progress',
  { 'outline-progress--failed': ['failed', 'cancelled'].includes(props.outline.generation_status) }
])

const compactList = (items?: string[], limit = 3) => {
  return (items || [])
    .map((item) => String(item || '').trim())
    .filter(Boolean)
    .slice(0, limit)
}

const outlineSpecItems = computed(() => {
  const outline = selectedOutline.value
  if (!outline) return []
  const items: { label: string; values: string[] }[] = []
  if (outline.goal) items.push({ label: '目标', values: [outline.goal] })
  if (outline.conflict) items.push({ label: '冲突', values: [outline.conflict] })
  if (compactList(outline.must_happen, 5).length) items.push({ label: '必发', values: compactList(outline.must_happen, 5) })
  if (compactList(outline.key_scenes, 5).length) items.push({ label: '场景', values: compactList(outline.key_scenes, 5) })
  if (compactList(outline.new_hooks, 5).length) items.push({ label: '新钩子', values: compactList(outline.new_hooks, 5) })
  if (outline.hook) items.push({ label: '钩子', values: [outline.hook] })
  return items
})

const selectOutline = (outlineId: string) => {
  previewVersionId.value = ''
  emit('selectOutline', outlineId)
  const chapter = chapterStatusByOutlineId.value.get(outlineId)
  if (chapter) emit('selectChapter', chapter.id)
  else emit('selectChapter', '')
}

const chapterActionLabel = (outlineId: string) => {
  if (props.generateLoadingOutlineId === outlineId) return '生成中...'
  const chapter = chapterStatusByOutlineId.value.get(outlineId)
  return chapter?.content?.trim() ? '重新生成正文' : '生成正文'
}

const triggerChapterGeneration = (outlineId: string) => {
  selectOutline(outlineId)
  emit('generate', outlineId)
}

const isFullReviewRevisionVersion = (version?: { type?: string; reason?: string } | null) => {
  const type = String(version?.type || '').trim()
  const reason = String(version?.reason || '').trim()
  return type === 'full_review_revision' || reason.includes('全文质量核验') || reason.includes('全文核验') || reason.includes('全文修订')
}

const formatVersionType = (type: string) => {
  if (type === 'draft') return '初稿版'
  if (type === 'full_review_revision') return '全文精修版'
  if (type === 'revision') return '修订版'
  return type || '未标记版本'
}

const formatVersionLabel = (version?: { type?: string; reason?: string } | null) => {
  if (isFullReviewRevisionVersion(version)) return '全文精修版'
  return formatVersionType(version?.type || '')
}

const currentVersionLabel = computed(() => {
  const chapter = selectedChapter.value
  const version = chapter?.versions?.find((item) => item.id === activeVersionId.value)
  if (version) return `当前版本（当前使用：${formatVersionLabel(version)}）`
  const type = currentVersionType.value
  return type ? `当前版本（当前使用：${formatVersionType(type)}）` : '当前版本'
})

const displayedVersion = computed(() => {
  const chapter = selectedChapter.value
  if (!chapter) return null
  if (previewVersionId.value) return chapter.versions?.find((item) => item.id === previewVersionId.value) || null
  return chapter.versions?.find((item) => item.id === activeVersionId.value) || null
})

const displayedVersionId = computed(() => displayedVersion.value?.id || '')
const displayedVersionType = computed(() => displayedVersion.value?.type || '')

const displayedVersionAdopted = computed(() => Boolean(displayedVersionId.value) && displayedVersionId.value === activeVersionId.value)

const canAdoptDisplayedVersion = computed(() => {
  return Boolean(displayedVersionId.value) && displayedVersionId.value !== activeVersionId.value
})

const isCurrentVersionView = computed(() => !previewVersionId.value)
const displayedWordCount = computed(() => Array.from(String(previewContent.value || '').replace(/\s+/g, '')).length)

const adoptDisplayedVersion = () => {
  if (!selectedChapter.value || !displayedVersionId.value) return
  emit('adopt', selectedChapter.value.id, displayedVersionId.value)
}

const startManualEdit = () => {
  if (!selectedChapter.value || !isCurrentVersionView.value) return
  manualContent.value = previewContent.value
  isManualEditing.value = true
}

const cancelManualEdit = () => {
  isManualEditing.value = false
  manualContent.value = previewContent.value
}

const saveManualEdit = () => {
  if (!selectedChapter.value || !activeVersionId.value) return
  emit('manualSave', selectedChapter.value.id, manualContent.value, activeVersionId.value)
  isManualEditing.value = false
}

const formatChapterStatus = (status?: string) => {
  if (status === 'approved') return '已确认'
  if (status === 'reviewing') return '待审计'
  if (status === 'revision_needed') return '待修订'
  if (status === 'drafted') return '已生成'
  return status || '待生成'
}

const draftVersion = computed(() => {
  const chapter = selectedChapter.value
  return chapter?.versions?.find((item) => item.type === 'draft') || null
})

const tokenizeDiffText = (text: string) => {
  return String(text || '').match(/[^\n。！？!?；;]+[。！？!?；;]?|\n+/g) || []
}

const buildSequenceDiff = (source: string[], target: string[]) => {
  const rows = source.length
  const cols = target.length
  const table = Array.from({ length: rows + 1 }, () => Array<number>(cols + 1).fill(0))

  for (let row = rows - 1; row >= 0; row -= 1) {
    for (let col = cols - 1; col >= 0; col -= 1) {
      if (source[row] === target[col]) {
        table[row]![col] = (table[row + 1]?.[col + 1] ?? 0) + 1
      } else {
        table[row]![col] = Math.max(table[row + 1]?.[col] ?? 0, table[row]?.[col + 1] ?? 0)
      }
    }
  }

  const operations: { type: 'same' | 'add' | 'remove'; text: string }[] = []
  let row = 0
  let col = 0

  while (row < rows && col < cols) {
    if (source[row] === target[col]) {
      operations.push({ type: 'same', text: target[col] ?? '' })
      row += 1
      col += 1
      continue
    }
    if ((table[row + 1]?.[col] ?? 0) >= (table[row]?.[col + 1] ?? 0)) {
      operations.push({ type: 'remove', text: source[row] ?? '' })
      row += 1
      continue
    }
    operations.push({ type: 'add', text: target[col] ?? '' })
    col += 1
  }

  while (row < rows) {
    operations.push({ type: 'remove', text: source[row] ?? '' })
    row += 1
  }

  while (col < cols) {
    operations.push({ type: 'add', text: target[col] ?? '' })
    col += 1
  }

  return operations
}

const revisionDiff = computed(() => {
  if (
    !previewVersionId.value
    || !['revision', 'full_review_revision'].includes(displayedVersionType.value)
    || !draftVersion.value
    || !displayedVersion.value
  ) {
    return null
  }

  const operations = buildSequenceDiff(
    tokenizeDiffText(draftVersion.value.content),
    tokenizeDiffText(displayedVersion.value.content)
  )

  const segments: Array<
    | { mode: 'same'; text: string }
    | { mode: 'replace'; before: string; after: string }
    | { mode: 'remove'; before: string }
    | { mode: 'add'; after: string }
  > = []
  let pendingBefore: string[] = []
  let pendingAfter: string[] = []

  const flushChange = () => {
    const before = pendingBefore.join('').trim()
    const after = pendingAfter.join('').trim()
    if (!before && !after) {
      pendingBefore = []
      pendingAfter = []
      return
    }
    if (before && after) {
      segments.push({ mode: 'replace', before, after })
    } else if (before) {
      segments.push({ mode: 'remove', before })
    } else {
      segments.push({ mode: 'add', after })
    }
    pendingBefore = []
    pendingAfter = []
  }

  for (const operation of operations) {
    if (operation.type === 'same') {
      flushChange()
      segments.push({ mode: 'same', text: operation.text })
      continue
    }
    if (operation.type === 'remove') {
      pendingBefore.push(operation.text)
      continue
    }
    pendingAfter.push(operation.text)
  }

  flushChange()

  return segments
})

const escapeDiffHtml = (value: string) => {
  return String(value || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

const escapeDiffAttr = (value: string) => {
  return escapeDiffHtml(value).replace(/'/g, '&#39;')
}

const revisionDiffHtml = computed(() => {
  if (!revisionDiff.value) return ''

  return revisionDiff.value.map((segment) => {
    if (segment.mode === 'same') return escapeDiffHtml(segment.text)
    if (segment.mode === 'remove') return ''

    const tooltip = segment.mode === 'replace' ? segment.before : '初稿中无对应内容'
    return `<span class="chapter-diff-highlight" title="${escapeDiffAttr(tooltip)}">${escapeDiffHtml(segment.after)}</span>`
  }).join('')
})

watch(
  [() => selectedChapter.value?.id || '', () => previewVersionId.value],
  ([nextChapterId, nextPreviewVersionId], [previousChapterId, previousPreviewVersionId]) => {
    if (nextChapterId === previousChapterId && nextPreviewVersionId === previousPreviewVersionId) return
    isManualEditing.value = false
    manualContent.value = previewContent.value
  },
  { immediate: true }
)

watch(
  isManualEditing,
  (editing) => {
    emit('manualEditStateChange', editing)
  },
  { immediate: true }
)
</script>

<template>
  <section class="chapter-panel">
    <div class="panel-head">
      <div class="panel-head__title-group">
        <p class="panel-kicker">步骤 3</p>
        <h3 class="panel-title">分章生成 / 审计修订</h3>
      </div>
      <div class="panel-head__actions">
        <el-button size="small" :loading="bulkGenerateLoading" :disabled="running && !bulkGenerateLoading" @click="emit('bulkGenerate')">
          <span>{{ bulkGenerateLoading ? '生成中' : '生成所有章节正文' }}</span>
          <span v-if="bulkGenerateLoading && bulkGenerateProgressTotal" class="panel-head__action-progress">
            当前第 {{ bulkGenerateProgressCurrent }}/{{ bulkGenerateProgressTotal }} 章
          </span>
        </el-button>
        <el-button size="small" :loading="bulkAuditLoading" :disabled="running && !bulkAuditLoading" @click="emit('bulkAudit')">
          <span>{{ bulkAuditLoading ? '审计中' : '审计所有正文' }}</span>
          <span v-if="bulkAuditLoading && bulkAuditProgressTotal" class="panel-head__action-progress">
            当前第 {{ bulkAuditProgressCurrent }}/{{ bulkAuditProgressTotal }} 章
          </span>
        </el-button>
        <el-button size="small" :loading="bulkReviseLoading" :disabled="running && !bulkReviseLoading" @click="emit('bulkRevise')">
          <span>{{ bulkReviseLoading ? '修订中' : '按审计修订所有正文' }}</span>
          <span v-if="bulkReviseLoading && bulkReviseProgressTotal" class="panel-head__action-progress">
            当前第 {{ bulkReviseProgressCurrent }}/{{ bulkReviseProgressTotal }} 章
          </span>
        </el-button>
        <el-button size="small" :loading="fullReviewLoading" :disabled="running && !fullReviewLoading" @click="emit('fullReview')">
          <span>{{ fullReviewLoading ? '核验中' : '全文质量核验' }}</span>
        </el-button>
      </div>
      <div v-if="outlineGenerationText" :class="outlineGenerationClass">
        <span>{{ outlineGenerationText }}</span>
      </div>
    </div>

    <div class="chapter-layout">
      <div class="chapter-tabs">
        <div
          v-for="chapter in outline.chapters || []"
          :key="chapter.id"
          :class="['chapter-tab', { 'chapter-tab--active': chapter.id === selectedOutline?.id }]"
          @click="selectOutline(chapter.id)"
        >
          <span v-if="chapterStatusByOutlineId.get(chapter.id)?.status === 'approved'" class="chapter-tab__confirmed-badge">章节已确认</span>
          <strong>{{ chapter.title }}</strong>
          <span v-if="chapter.summary" class="chapter-tab__summary">{{ chapter.summary }}</span>
          <div class="chapter-tab__footer">
            <small>{{ formatChapterStatus(chapterStatusByOutlineId.get(chapter.id)?.status) }}</small>
            <button
              :class="['chapter-tab__action', { 'chapter-tab__action--loading': generateLoadingOutlineId === chapter.id }]"
              type="button"
              :disabled="running"
              @click.stop="triggerChapterGeneration(chapter.id)"
            >
              <span v-if="generateLoadingOutlineId === chapter.id" class="chapter-tab__action-spinner"></span>
              <span>{{ chapterActionLabel(chapter.id) }}</span>
            </button>
          </div>
        </div>
        <el-empty v-if="!outline.chapters?.length" description="先生成大纲/章节结构" />
      </div>

      <article v-if="selectedChapter && hasSelectedChapterContent" class="chapter-preview">
        <div class="chapter-preview__top">
          <h4>{{ selectedChapter.title }}</h4>
          <div class="chapter-actions">
            <el-button size="small" :loading="auditLoadingChapterId === selectedChapter.id" @click="emit('audit', selectedChapter.id)">审计</el-button>
            <el-button
              v-if="selectedChapter.status !== 'approved'"
              size="small"
              type="success"
              :loading="approveLoadingChapterId === selectedChapter.id"
              @click="emit('approve', selectedChapter.id)"
            >
              确认章节
            </el-button>
          </div>

          <div v-if="outlineSpecItems.length" class="outline-spec-strip">
            <div
              v-for="item in outlineSpecItems"
              :key="item.label"
              class="outline-spec-strip__group"
            >
              <strong>{{ item.label }}</strong>
              <div class="outline-spec-strip__values">
                <span v-for="value in item.values" :key="`${item.label}-${value}`">{{ value }}</span>
              </div>
            </div>
          </div>
        </div>

        <div v-if="selectedChapter.versions?.length" class="version-strip">
          <button
            :class="['version-chip', { 'version-chip--active': !previewVersionId }]"
            @click="previewVersionId = ''"
          >
            {{ currentVersionLabel }}
          </button>
          <button
            v-for="version in selectedChapter.versions"
            :key="version.id"
            :class="['version-chip', { 'version-chip--active': previewVersionId === version.id }]"
            @click="previewVersionId = version.id"
          >
            {{ formatVersionLabel(version) }} · {{ version.created_at }}
          </button>
        </div>

        <div class="chapter-content-shell">
          <div class="chapter-version-floating">
            <span class="chapter-word-count">当前 {{ displayedWordCount }} 字</span>
            <template v-if="isCurrentVersionView">
              <div v-if="isManualEditing" class="chapter-version-edit-actions">
                <button
                  class="chapter-version-adopt-btn"
                  type="button"
                  :disabled="manualSaveLoading"
                  @click="saveManualEdit"
                >
                  {{ manualSaveLoading ? '保存中...' : '保存修改' }}
                </button>
                <button
                  class="chapter-version-secondary-btn"
                  type="button"
                  :disabled="manualSaveLoading"
                  @click="cancelManualEdit"
                >
                  取消
                </button>
              </div>
              <button
                v-else
                class="chapter-version-adopt-btn"
                type="button"
                @click="startManualEdit"
              >
                手动编辑
              </button>
            </template>
            <button
              v-else-if="canAdoptDisplayedVersion"
              class="chapter-version-adopt-btn"
              type="button"
              :disabled="adoptLoadingVersionId === displayedVersionId"
              @click="adoptDisplayedVersion"
            >
              {{ adoptLoadingVersionId === displayedVersionId ? '采用中...' : '选用此版' }}
            </button>
            <span v-else-if="displayedVersionAdopted" class="chapter-version-adopt-badge">已采用</span>
          </div>
          <textarea
            v-if="isManualEditing"
            v-model="manualContent"
            class="chapter-editor"
            spellcheck="false"
          ></textarea>
          <div v-else-if="revisionDiff" class="chapter-content chapter-content--diff" v-html="revisionDiffHtml"></div>
          <pre v-else class="chapter-content">{{ previewContent }}</pre>
        </div>
      </article>

      <article v-else class="chapter-preview chapter-preview--empty chapter-preview--outline-only">
        <h4>{{ selectedOutline?.title || '等待选择章节' }}</h4>
        <p v-if="selectedOutline?.summary" class="chapter-preview__empty-summary">{{ selectedOutline.summary }}</p>
        <div v-if="outlineSpecItems.length" class="outline-spec-strip outline-spec-strip--expanded">
          <div
            v-for="item in outlineSpecItems"
            :key="item.label"
            class="outline-spec-strip__group"
          >
            <strong>{{ item.label }}</strong>
            <div class="outline-spec-strip__values">
              <span v-for="value in item.values" :key="`${item.label}-${value}`">{{ value }}</span>
            </div>
          </div>
        </div>
        <p>当前章节还没有生成正文。确认左侧目标、冲突和钩子后，点击左侧目录中的“生成正文”。</p>
      </article>
    </div>
  </section>
</template>

<style scoped>
.chapter-panel {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  gap: 16px;
  height: 100%;
  box-sizing: border-box;
  min-width: 0;
  overflow: hidden;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 24px;
  padding: 22px;
}

.panel-head {
  display: flex;
  align-items: center;
  gap: 20px;
  min-width: 0;
}

.panel-head__title-group {
  display: flex;
  align-items: center;
  flex: 0 0 auto;
  gap: 12px;
}

.panel-head__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
}

.panel-head__action-progress {
  margin-left: 6px;
  color: #16a34a;
  font-weight: 800;
}

.outline-progress {
  display: flex;
  align-items: center;
  flex: 1 1 auto;
  min-width: 0;
  min-height: 40px;
  padding: 10px 13px;
  border: 1px solid #99f6e4;
  border-radius: 12px;
  background: #f0fdfa;
  color: #0f766e;
  font-size: 13px;
  font-weight: 800;
  line-height: 1.5;
}

.outline-progress span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.outline-progress--failed {
  border-color: #fecaca;
  background: #fef2f2;
  color: #b91c1c;
}

.panel-kicker {
  margin: 0;
  color: #0f766e;
  font-size: 12px;
  font-weight: 800;
}

.panel-title {
  margin: 0;
  color: #0f172a;
  font-size: 20px;
}

.chapter-layout {
  display: grid;
  grid-template-columns: minmax(260px, 330px) minmax(0, 1fr);
  gap: 18px;
  align-items: stretch;
  height: 100%;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
}

.chapter-layout > * {
  min-height: 0;
}

.chapter-tabs {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 10px;
  min-height: 0;
  overflow-y: auto;
  padding-right: 4px;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.chapter-tabs::-webkit-scrollbar {
  display: none;
}

.chapter-tab {
  position: relative;
  display: grid;
  gap: 10px;
  padding: 14px;
  border: 1px solid #e2e8f0;
  border-radius: 15px;
  background: #f8fafc;
  text-align: left;
  cursor: pointer;
}

.chapter-tab--active {
  border-color: #14b8a6;
  background: #ecfeff;
}

.chapter-tab__confirmed-badge {
  position: absolute;
  top: 10px;
  right: 10px;
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
  color: #059669;
  font-size: 11px;
  font-weight: 800;
}

.chapter-tab strong,
.chapter-tab span,
.chapter-tab em,
.chapter-tab small {
  display: block;
}

.chapter-tab strong {
  margin-bottom: 8px;
  color: #0f172a;
}

.chapter-tab__meta,
.chapter-tab__hook,
.chapter-tab__summary,
.chapter-tab__spec {
  color: #64748b;
  font-size: 12px;
  font-style: normal;
  line-height: 1.55;
}

.chapter-tab__summary {
  margin-bottom: 6px;
  color: #475569;
}

.chapter-tab__spec {
  margin-top: 4px;
  color: #0f766e;
}

.chapter-tab small {
  width: fit-content;
  padding: 4px 8px;
  border-radius: 999px;
  background: #ffffff;
  color: #0f766e;
  font-size: 11px;
  font-weight: 800;
}

.chapter-tab__footer {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 10px;
}

.chapter-tab__action {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: 0 0 auto;
  min-height: 30px;
  padding: 0 12px;
  border: 1px solid #cbd5e1;
  border-radius: 10px;
  background: #ffffff;
  color: #0f766e;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: border-color 0.2s ease, background 0.2s ease, color 0.2s ease;
}

.chapter-tab__action:hover:not(:disabled) {
  border-color: #14b8a6;
  background: #f0fdfa;
}

.chapter-tab__action:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.chapter-tab__action--loading {
  border-color: #14b8a6;
  background: #f0fdfa;
}

.chapter-tab__action-spinner {
  width: 12px;
  height: 12px;
  border: 1.5px solid rgba(15, 118, 110, 0.2);
  border-top-color: #0f766e;
  border-radius: 999px;
  animation: chapter-action-spin 0.8s linear infinite;
}

@keyframes chapter-action-spin {
  from {
    transform: rotate(0deg);
  }

  to {
    transform: rotate(360deg);
  }
}

.chapter-preview {
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr);
  gap: 14px;
  height: 100%;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
}

.chapter-preview--empty {
  min-height: 260px;
  padding: 26px;
  border: 1px dashed #cbd5e1;
  border-radius: 18px;
  background: #f8fafc;
}

.chapter-preview--outline-only {
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr) auto;
  gap: 14px;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.chapter-preview--empty h4 {
  margin: 0 0 10px;
  color: #0f172a;
  font-size: 18px;
}

.chapter-preview--empty p {
  margin: 0;
  color: #64748b;
  line-height: 1.7;
}

.chapter-preview__empty-summary {
  overflow: auto;
}

.outline-spec {
  display: grid;
  gap: 12px;
  margin: 0 0 16px;
}

.outline-spec__group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.outline-spec__group strong,
.outline-spec__group span {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 5px 9px;
  border-radius: 999px;
  font-size: 12px;
}

.outline-spec__group strong {
  background: #ecfeff;
  color: #0f766e;
}

.outline-spec__group span {
  background: #f1f5f9;
  color: #475569;
}

.chapter-preview__top {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 14px;
  min-height: 0;
  padding-bottom: 14px;
  border-bottom: 1px solid #e2e8f0;
}

.chapter-preview__top h4 {
  min-width: 0;
  margin: 0 0 6px;
  color: #0f172a;
  font-size: 18px;
}

.outline-spec-strip {
  grid-column: 1 / -1;
  display: grid;
  max-height: 150px;
  gap: 8px;
  overflow: auto;
  padding: 12px;
  border: 1px solid #ccfbf1;
  border-radius: 16px;
  background: #f0fdfa;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.outline-spec-strip::-webkit-scrollbar {
  display: none;
}

.outline-spec-strip--expanded {
  max-height: none;
  height: 100%;
  align-content: start;
}

.outline-spec-strip__group {
  display: grid;
  grid-template-columns: 62px minmax(0, 1fr);
  gap: 8px;
  align-items: start;
}

.outline-spec-strip__group strong {
  color: #0f766e;
  font-size: 12px;
  line-height: 1.6;
}

.outline-spec-strip__values {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.outline-spec-strip__values span {
  color: #334155;
  font-size: 13px;
  line-height: 1.65;
  word-break: break-word;
}

.chapter-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.version-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.chapter-content-shell {
  position: relative;
  display: grid;
  grid-template-rows: minmax(0, 1fr);
  min-height: 0;
  height: 100%;
}

.chapter-version-floating {
  position: absolute;
  top: 12px;
  right: 12px;
  z-index: 2;
  display: flex;
  align-items: center;
  gap: 8px;
  pointer-events: none;
}

.chapter-version-edit-actions {
  display: flex;
  gap: 8px;
  pointer-events: auto;
}

.chapter-version-adopt-btn,
.chapter-version-adopt-badge,
.chapter-version-secondary-btn,
.chapter-word-count {
  display: inline-flex;
  align-items: center;
  min-height: 32px;
  padding: 0 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 800;
}

.chapter-word-count {
  background: rgba(15, 23, 42, 0.72);
  color: #cbd5e1;
  box-shadow: 0 6px 16px rgba(15, 23, 42, 0.16);
}

.chapter-version-adopt-btn {
  pointer-events: auto;
  border: 1px solid #99f6e4;
  background: rgba(240, 253, 250, 0.96);
  color: #0f766e;
  cursor: pointer;
  box-shadow: 0 6px 16px rgba(15, 118, 110, 0.08);
}

.chapter-version-adopt-btn:disabled {
  cursor: not-allowed;
  opacity: 0.7;
}

.chapter-version-secondary-btn {
  pointer-events: auto;
  border: 1px solid rgba(148, 163, 184, 0.5);
  background: rgba(15, 23, 42, 0.72);
  color: #e2e8f0;
  cursor: pointer;
}

.chapter-version-secondary-btn:disabled {
  cursor: not-allowed;
  opacity: 0.7;
}

.chapter-version-adopt-badge {
  background: rgba(20, 184, 166, 0.96);
  color: #ffffff;
  box-shadow: 0 6px 16px rgba(20, 184, 166, 0.2);
}

.version-chip {
  padding: 6px 10px;
  border: 1px solid #cbd5e1;
  border-radius: 999px;
  background: #ffffff;
  color: #475569;
  cursor: pointer;
}

.version-chip--active {
  border-color: #14b8a6;
  background: #ccfbf1;
  color: #0f766e;
}

.chapter-content {
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  min-height: 0;
  margin: 0;
  overflow: auto;
  overscroll-behavior: contain;
  padding: 56px 20px 20px;
  border-radius: 18px;
  background: #0f172a;
  color: #e2e8f0;
  white-space: pre-wrap;
  line-height: 1.9;
  font-family: "Songti SC", "Noto Serif CJK SC", serif;
  font-size: 15px;
}

.chapter-editor {
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  min-height: 0;
  resize: none;
  border: none;
  outline: none;
  padding: 56px 20px 20px;
  border-radius: 18px;
  background: #0f172a;
  color: #e2e8f0;
  white-space: pre-wrap;
  line-height: 1.9;
  font-family: "Songti SC", "Noto Serif CJK SC", serif;
  font-size: 15px;
}

.chapter-content--diff {
  display: block;
  padding-top: 56px;
  overflow-x: hidden;
  white-space: pre-wrap;
}

.chapter-content--diff :deep(.chapter-diff-highlight) {
  display: inline;
  padding: 0 2px;
  border-radius: 4px;
  background: rgba(20, 184, 166, 0.24);
  color: #5eead4;
  cursor: help;
  box-shadow: 0 0 0 1px rgba(45, 212, 191, 0.18);
}

@media (max-width: 1000px) {
  .chapter-layout {
    grid-template-columns: 1fr;
  }

  .panel-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .panel-head__actions {
    flex-wrap: wrap;
  }

  .chapter-preview__top {
    grid-template-columns: 1fr;
  }

  .chapter-actions {
    justify-content: flex-start;
  }

  .chapter-tab__footer {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
