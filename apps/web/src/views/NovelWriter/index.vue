<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, shallowRef, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import NovelProjectList from '@/components/NovelWriter/NovelProjectList.vue'
import NovelMaterialMap from '@/components/NovelWriter/NovelMaterialMap.vue'
import NovelMaterialPanel from '@/components/NovelWriter/NovelMaterialPanel.vue'
import NovelStructurePanel from '@/components/NovelWriter/NovelStructurePanel.vue'
import NovelChapterPanel from '@/components/NovelWriter/NovelChapterPanel.vue'
import NovelAuditPanel from '@/components/NovelWriter/NovelAuditPanel.vue'
import {
  emptyNovelMaterials,
  novelWriterApi,
  type CreateNovelProjectPayload,
  type NovelExtractedInfo,
  type NovelMaterials,
  type NovelStyleProfile,
  type NovelProject
} from '@/api/novelWriter'

const projects = shallowRef<NovelProject[]>([])
const selectedProject = shallowRef<NovelProject | null>(null)
const selectedOutlineId = shallowRef('')
const selectedChapterId = shallowRef('')
const loading = shallowRef(false)
const saving = shallowRef(false)
const running = shallowRef(false)
const createDialogVisible = shallowRef(false)
const activeStep = shallowRef<'materials' | 'generation'>('materials')
const insightsOpen = shallowRef(false)
const materialInsightsImported = shallowRef(false)
const materialInsightsSnapshot = shallowRef<NovelExtractedInfo | null>(null)
const materialPanelRef = shallowRef<InstanceType<typeof NovelMaterialPanel> | null>(null)
const outlinePollingTimer = shallowRef<number | undefined>()
const reviewGridRef = shallowRef<HTMLElement | null>(null)
const reviewGridMinHeight = shallowRef(420)
const runningActionLabel = shallowRef('')

const workspaceSwitchItemClass = (step: 'materials' | 'generation') => [
  'workspace-switch__item',
  { 'workspace-switch__item--active': activeStep.value === step }
]

const createForm = reactive<CreateNovelProjectPayload>({
  title: '',
  genre: '',
  target_words: 80000,
  target_chapters: 30,
  materials: emptyNovelMaterials()
})

const selectedChapter = computed(() => {
  const chapters = selectedProject.value?.chapters || []
  return chapters.find((item) => item.id === selectedChapterId.value)
    || chapters.find((item) => item.outline_id === selectedOutlineId.value)
    || chapters[0]
})

const selectedAudit = computed(() => {
  const chapter = selectedChapter.value
  return chapter?.audit || {
    total_score: 0,
    ai_flavor_score: 0,
    character_score: 0,
    logic_score: 0,
    style_score: 0,
    issues: [],
    revision_advice: ''
  }
})

const currentMaterials = computed({
  get() {
    return selectedProject.value?.materials || emptyNovelMaterials()
  },
  set(value: NovelMaterials) {
    if (!selectedProject.value) return
    selectedProject.value = {
      ...selectedProject.value,
      materials: value
    }
  }
})

const splitMaterialBlocks = (text: string) => {
  return String(text || '')
    .split(/\n\s*\n|\n-/)
    .map((item) => item.replace(/^-/, '').trim())
    .filter(Boolean)
}

const splitMaterialLines = (text: string) => {
  return String(text || '')
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean)
}

const parseMaterialCard = (raw: string, fallbackName: string) => {
  const lines = String(raw || '')
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean)
  const firstLine = lines[0] || ''
  const matched = firstLine.match(/^(?:新人物|人物|角色|世界观|冲突|灵感|事件)\s*[:：]\s*(.+)$/)
  const name = matched?.[1]?.trim() || firstLine.replace(/[:：]$/, '').trim() || fallbackName
  const description = matched ? lines.slice(1).join('；') : lines.slice(1).join('；') || raw
  return {
    name,
    description: description || raw
  }
}

const emptyExtractedInfo = (): NovelExtractedInfo => ({
  characters: [],
  world_rules: [],
  conflicts: [],
  key_events: [],
  open_questions: []
})

const buildMaterialExtracted = (materials: NovelMaterials, includeIdeas = true): NovelExtractedInfo => ({
  characters: splitMaterialBlocks(materials.character_raw).map((description, index) => parseMaterialCard(description, `人物 ${index + 1}`)),
  world_rules: splitMaterialBlocks(materials.world_raw).map((description, index) => parseMaterialCard(description, `世界观 ${index + 1}`)),
  conflicts: splitMaterialLines(materials.conflict_raw).map((description, index) => parseMaterialCard(description, `冲突 ${index + 1}`)),
  key_events: includeIdeas ? splitMaterialLines(materials.raw_text).map((description, index) => parseMaterialCard(description, `灵感 ${index + 1}`)) : [],
  open_questions: []
})

const displayExtracted = computed<NovelExtractedInfo>(() => {
  if (materialInsightsImported.value && materialInsightsSnapshot.value) return materialInsightsSnapshot.value
  return selectedProject.value?.extracted || emptyExtractedInfo()
})

const displayStyleProfile = computed<NovelStyleProfile>(() => {
  if (materialInsightsImported.value) {
    return {
      summary: '',
      narration: '',
      sentence: '',
      dialogue: '',
      rhythm: '',
      do_rules: [],
      avoid_rules: []
    }
  }

  const profile = selectedProject.value?.style_profile
  if (
    profile?.summary
    || profile?.narration
    || profile?.sentence
    || profile?.dialogue
    || profile?.rhythm
    || profile?.do_rules?.length
    || profile?.avoid_rules?.length
  ) {
    return profile
  }

  return {
    summary: '',
    narration: '',
    sentence: '',
    dialogue: '',
    rhythm: '',
    do_rules: [],
    avoid_rules: []
  }
})

const getErrorMessage = (error: unknown) => {
  if (error && typeof error === 'object' && 'customMessage' in error) {
    return String((error as { customMessage?: unknown }).customMessage || error)
  }
  if (error instanceof Error) return error.message
  return String(error)
}

const taskInProgressMessage = (label?: string) => {
  if (label) return `${label}\u4efb\u52a1\u5df2\u7ecf\u5728\u540e\u53f0\u8fdb\u884c\uff0c\u8bf7\u7b49\u5f85\u4efb\u52a1\u7ed3\u675f`
  return '\u540e\u53f0\u4efb\u52a1\u5df2\u7ecf\u5728\u8fdb\u884c\uff0c\u8bf7\u7b49\u5f85\u4efb\u52a1\u7ed3\u675f'
}

const notifyTaskInProgress = (label?: string) => {
  ElMessage.warning(taskInProgressMessage(label))
}

const ensureActionAvailable = (label?: string) => {
  if (!running.value) return true
  notifyTaskInProgress(runningActionLabel.value || label)
  return false
}

const updateReviewGridMinHeight = () => {
  const element = reviewGridRef.value
  if (!element) return
  reviewGridMinHeight.value = Math.max(window.innerHeight - element.getBoundingClientRect().top - 24, 420)
}

const reviewGridStyle = computed(() => ({
  height: `${reviewGridMinHeight.value}px`
}))

const refreshProjects = async () => {
  loading.value = true
  try {
    projects.value = await novelWriterApi.listProjects()
    if (selectedProject.value) {
      const latest = projects.value.find((project) => project.id === selectedProject.value?.id)
      if (latest) selectProject(latest)
    }
  } catch (error) {
    ElMessage.error(`加载小说项目失败：${getErrorMessage(error)}`)
  } finally {
    loading.value = false
  }
}

const selectProject = (project: NovelProject) => {
  selectedProject.value = project
  selectedOutlineId.value = project.outline?.chapters?.[0]?.id || ''
  selectedChapterId.value = project.chapters?.[0]?.id || ''
  activeStep.value = 'materials'
  materialInsightsImported.value = false
  materialInsightsSnapshot.value = null
  if (project.outline?.generation_status === 'generating') {
    startOutlinePolling(project.id)
  } else {
    stopOutlinePolling()
  }
}

const backToProjectList = () => {
  stopOutlinePolling()
  selectedProject.value = null
  selectedOutlineId.value = ''
  selectedChapterId.value = ''
  insightsOpen.value = false
  materialInsightsImported.value = false
  materialInsightsSnapshot.value = null
}

const importMaterialsToInsights = () => {
  materialInsightsSnapshot.value = buildMaterialExtracted(currentMaterials.value, false)
  materialInsightsImported.value = true
  insightsOpen.value = true
  ElMessage.success('素材图谱内容已带入，灵感卡片未带入')
}

const createProject = async () => {
  if (!createForm.title.trim()) {
    ElMessage.warning('请先填写小说名称')
    return
  }
  saving.value = true
  try {
    const project = await novelWriterApi.createProject(createForm)
    projects.value = [project, ...projects.value]
    selectProject(project)
    createDialogVisible.value = false
    Object.assign(createForm, {
      title: '',
      genre: '',
      target_words: 80000,
      target_chapters: 30,
      materials: emptyNovelMaterials()
    })
    ElMessage.success('小说项目已创建')
  } catch (error) {
    ElMessage.error(`创建失败：${getErrorMessage(error)}`)
  } finally {
    saving.value = false
  }
}

const syncSelectedProject = (project: NovelProject) => {
  selectedProject.value = project
  projects.value = projects.value.map((item) => item.id === project.id ? project : item)
  const outlineExists = project.outline?.chapters?.some((chapter) => chapter.id === selectedOutlineId.value)
  if (!selectedOutlineId.value || !outlineExists) selectedOutlineId.value = project.outline?.chapters?.[0]?.id || ''
  const chapterExists = project.chapters?.some((chapter) => chapter.id === selectedChapterId.value)
  if (!selectedChapterId.value || !chapterExists) {
    selectedChapterId.value = project.chapters?.find((chapter) => chapter.outline_id === selectedOutlineId.value)?.id
      || project.chapters?.[0]?.id
      || ''
  }
}

const stopOutlinePolling = () => {
  if (!outlinePollingTimer.value) return
  window.clearInterval(outlinePollingTimer.value)
  outlinePollingTimer.value = undefined
}

const startOutlinePolling = (projectId: string) => {
  stopOutlinePolling()
  outlinePollingTimer.value = window.setInterval(async () => {
    try {
      const project = await novelWriterApi.getProject(projectId)
      syncSelectedProject(project)
      if (project.outline?.generation_status !== 'generating') stopOutlinePolling()
    } catch (error) {
      stopOutlinePolling()
      ElMessage.error(`刷新大纲进度失败：${getErrorMessage(error)}`)
    }
  }, 5000)
}

const saveCurrentProject = async () => {
  if (!selectedProject.value) return
  saving.value = true
  try {
    syncSelectedProject(await novelWriterApi.updateProject(selectedProject.value))
    ElMessage.success('素材已保存')
  } catch (error) {
    ElMessage.error(`保存失败：${getErrorMessage(error)}`)
  } finally {
    saving.value = false
  }
}

const runProjectAction = async (label: string, action: () => Promise<NovelProject>) => {
  if (!selectedProject.value) return
  if (!ensureActionAvailable(label)) return
  running.value = true
  runningActionLabel.value = label
  try {
    const project = await action()
    syncSelectedProject(project)
    ElMessage.success(`${label}完成`)
    return project
  } catch (error) {
    ElMessage.error(`${label}失败：${getErrorMessage(error)}`)
  } finally {
    running.value = false
    runningActionLabel.value = ''
  }
}

const saveBeforeAIAction = async () => {
  if (!selectedProject.value) return
  syncSelectedProject(await novelWriterApi.updateProject(selectedProject.value))
}

const extractInfo = () => runProjectAction('信息提取', async () => {
  await saveBeforeAIAction()
  return novelWriterApi.extractInfo(selectedProject.value!.id)
})

const planOutline = async () => {
  if (!ensureActionAvailable('生成大纲/章节结构')) return
  if (selectedProject.value?.outline?.generation_status === 'generating') {
    notifyTaskInProgress('生成大纲/章节结构')
    return
  }
  const project = await runProjectAction('大纲规划', async () => {
    await saveBeforeAIAction()
    return novelWriterApi.planOutline(selectedProject.value!.id)
  })
  if (project?.outline?.chapters?.length) {
    insightsOpen.value = true
    if (project.outline.generation_status === 'generating') {
      startOutlinePolling(project.id)
      const generated = project.outline.generated_chapters || project.outline.chapters.length
      ElMessage.info(`已生成前 ${generated} 章完整大纲，剩余章节正在后台继续生成`)
    }
  }
}

const analyzeStyle = () => runProjectAction('文风画像', async () => {
  await saveBeforeAIAction()
  return novelWriterApi.analyzeStyle(selectedProject.value!.id)
})

const switchToGeneration = async () => {
  await saveBeforeAIAction()
  activeStep.value = 'generation'
  insightsOpen.value = true
  await nextTick()
}

const handleMaterialMapNext = async () => {
  await switchToGeneration()
  if (currentMaterials.value.reference_raw.trim()) return

  try {
    await ElMessageBox.confirm(
      '未提供文风参考，是否使用模型默认生成？',
      '文风参考确认',
      {
        confirmButtonText: '确认',
        cancelButtonText: '取消',
        type: 'info'
      }
    )
    await analyzeStyle()
  } catch (error) {
    if (error === 'cancel' || error === 'close') {
      await nextTick()
      materialPanelRef.value?.openReferenceDialog()
      return
    }
    ElMessage.error(`文风画像生成失败：${getErrorMessage(error)}`)
  }
}

const ensureOutlineReady = async () => {
  if (!selectedProject.value) throw new Error('请先选择小说项目')
  await saveBeforeAIAction()

  const hasExtractedInfo = Boolean(
    selectedProject.value.extracted?.characters?.length
    || selectedProject.value.extracted?.world_rules?.length
    || selectedProject.value.extracted?.conflicts?.length
    || selectedProject.value.extracted?.key_events?.length
  )
  if (!hasExtractedInfo) {
    ElMessage.info('正在先补充信息提取...')
    syncSelectedProject(await novelWriterApi.extractInfo(selectedProject.value.id))
  }

  if (!selectedProject.value.outline?.chapters?.length) {
    ElMessage.info('正在先生成章节大纲...')
    const project = await novelWriterApi.planOutline(selectedProject.value.id)
    syncSelectedProject(project)
    if (project.outline?.generation_status === 'generating') startOutlinePolling(project.id)
  }

  const outlineId = selectedOutlineId.value || selectedProject.value.outline?.chapters?.[0]?.id || ''
  if (!outlineId) throw new Error('大纲生成失败：没有可用章节')
  selectedOutlineId.value = outlineId
  return outlineId
}

const generateChapter = () => runProjectAction('章节生成', async () => {
  const outlineId = await ensureOutlineReady()
  return novelWriterApi.generateChapter(selectedProject.value!.id, outlineId)
})
const auditChapter = (chapterId: string) => runProjectAction('章节审计', () => novelWriterApi.auditChapter(selectedProject.value!.id, chapterId))
const reviseChapter = (chapterId: string) => runProjectAction('智能修订', () => novelWriterApi.reviseChapter(selectedProject.value!.id, chapterId))
const approveChapter = (chapterId: string) => runProjectAction('章节确认', () => novelWriterApi.approveChapter(selectedProject.value!.id, chapterId))

const exportProjectMarkdown = (project: NovelProject) => {
  const content = [
    `# ${project.title}`,
    '',
    ...(project.chapters || []).map((chapter) => `## ${chapter.title}\n\n${chapter.content}`)
  ].join('\n\n')
  const blob = new Blob([content], { type: 'text/markdown;charset=utf-8' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = `${project.title || 'novel'}.md`
  link.click()
  URL.revokeObjectURL(link.href)
}

const deleteProject = async (project: NovelProject) => {
  try {
    await ElMessageBox.confirm(
      `确认删除《${project.title}》吗？删除后无法在平台内恢复。`,
      '删除小说项目',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )
    try {
      await novelWriterApi.deleteProject(project.id)
    } catch (error: any) {
      if (error?.response?.status !== 404) throw error
      await novelWriterApi.deleteProjectFallback(project.id)
    }
    projects.value = projects.value.filter((item) => item.id !== project.id)
    if (selectedProject.value?.id === project.id) backToProjectList()
    ElMessage.success('小说项目已删除')
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(`删除失败：${getErrorMessage(error)}`)
  }
}

onMounted(refreshProjects)
onMounted(() => {
  window.addEventListener('resize', updateReviewGridMinHeight)
  nextTick(updateReviewGridMinHeight)
})
onBeforeUnmount(stopOutlinePolling)
onBeforeUnmount(() => {
  window.removeEventListener('resize', updateReviewGridMinHeight)
})

watch(
  () => [
    activeStep.value,
    insightsOpen.value,
    selectedProject.value?.id,
    selectedProject.value?.chapters?.length,
    selectedProject.value?.outline?.chapters?.length
  ],
  () => nextTick(updateReviewGridMinHeight)
)
</script>

<template>
  <div class="novel-page">
    <main class="novel-workspace">
      <NovelProjectList
        v-if="!selectedProject"
        :projects="projects"
        selected-id=""
        :loading="loading"
        @select="selectProject"
        @create="createDialogVisible = true"
        @export="exportProjectMarkdown"
        @delete="deleteProject"
      />

      <template v-else>
        <NovelMaterialMap
          v-if="activeStep === 'materials'"
          v-model="currentMaterials"
          :position-storage-key="selectedProject.id"
          :saving="saving"
          @save="saveCurrentProject"
          @extract="extractInfo"
          @next="handleMaterialMapNext"
        >
          <template #workspace-nav>
            <button class="workspace-back" type="button" @click="backToProjectList">
              <el-icon><ArrowLeft /></el-icon>
              <span>返回书籍列表</span>
            </button>
          </template>
          <template #workspace-switch>
            <div class="workspace-switch">
              <button
                :class="workspaceSwitchItemClass('materials')"
                @click="activeStep = 'materials'"
              >
                素材图谱
              </button>
              <button
                :class="workspaceSwitchItemClass('generation')"
                @click="switchToGeneration"
              >
                文风生成
              </button>
            </div>
          </template>
        </NovelMaterialMap>

        <div v-else class="generation-workspace">
          <div class="generation-switch-row">
            <button class="workspace-back" type="button" @click="backToProjectList">
              <el-icon><ArrowLeft /></el-icon>
              <span>返回书籍列表</span>
            </button>
            <div class="workspace-switch">
              <button
                :class="workspaceSwitchItemClass('materials')"
                @click="activeStep = 'materials'"
              >
                素材图谱
              </button>
              <button
                :class="workspaceSwitchItemClass('generation')"
                @click="switchToGeneration"
              >
                文风生成
              </button>
            </div>
          </div>

          <NovelMaterialPanel
            ref="materialPanelRef"
            v-model="currentMaterials"
            :saving="saving"
            @save="saveCurrentProject"
            @outline="planOutline"
            @style="analyzeStyle"
          />

          <NovelStructurePanel
            :extracted="displayExtracted"
            :outline="selectedProject.outline"
            :style-profile="displayStyleProfile"
            :open="insightsOpen"
            :material-imported="materialInsightsImported"
            :position-storage-key="selectedProject.id"
            @update:open="insightsOpen = $event"
            @import-materials="importMaterialsToInsights"
          />

          <div ref="reviewGridRef" class="generation-review-grid" :style="reviewGridStyle">
            <NovelChapterPanel
              :outline="selectedProject.outline"
              :selected-outline-id="selectedOutlineId"
              :chapters="selectedProject.chapters || []"
              :selected-chapter-id="selectedChapterId"
              :running="running"
              @select-outline="selectedOutlineId = $event"
              @select-chapter="selectedChapterId = $event"
              @generate="generateChapter"
              @audit="auditChapter"
              @revise="reviseChapter"
              @approve="approveChapter"
            />

            <aside class="generation-audit-column">
              <NovelAuditPanel :audit="selectedAudit" />
            </aside>
          </div>
        </div>
      </template>
    </main>

    <el-dialog v-model="createDialogVisible" title="新建小说项目" width="560px">
      <el-form label-position="top">
        <el-form-item label="小说名称">
          <el-input v-model="createForm.title" placeholder="例如：风暴之前" />
        </el-form-item>
        <el-form-item label="题材">
          <el-input v-model="createForm.genre" placeholder="都市 / 玄幻 / 科幻 / 短剧改编..." />
        </el-form-item>
        <div class="dialog-grid">
          <el-form-item label="目标字数">
            <el-input-number v-model="createForm.target_words" :min="1000" :step="10000" />
          </el-form-item>
          <el-form-item label="目标章节数">
            <el-input-number v-model="createForm.target_chapters" :min="1" :step="1" />
          </el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="createProject">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.novel-page {
  min-height: 100vh;
  background: #f8fafc;
  background-image: 
    radial-gradient(at 0% 0%, rgba(20, 184, 166, 0.05) 0px, transparent 50%),
    radial-gradient(at 100% 0%, rgba(14, 165, 233, 0.05) 0px, transparent 50%);
}

.novel-workspace {
  display: flex;
  flex-direction: column;
  gap: 24px;
  min-height: 100vh;
  min-width: 0;
}

.generation-workspace {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 24px;
  min-height: 0;
  padding: 24px 32px;
  box-sizing: border-box;
  width: 100%;
}

.generation-review-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(380px, 460px);
  gap: 24px;
  align-items: stretch;
  flex: 1;
  box-sizing: border-box;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
}

.generation-audit-column {
  position: sticky;
  top: 24px;
  height: 100%;
  min-height: 0;
  min-width: 0;
  width: 100%;
  overflow: hidden;
}

.workspace-switch {
  position: relative;
  isolation: isolate;
  display: inline-flex;
  gap: 6px;
  padding: 6px;
  background: rgba(255, 255, 255, 0.6);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(226, 232, 240, 0.8);
  border-radius: 20px;
  box-shadow: 0 10px 30px -10px rgba(15, 23, 42, 0.1);
}

.workspace-switch__item {
  min-width: 110px;
  padding: 10px 24px;
  border: 0;
  border-radius: 14px;
  background: transparent;
  color: #64748b;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.workspace-switch__item:hover {
  color: #0f172a;
  background: rgba(255, 255, 255, 0.5);
}

.workspace-switch__item--active {
  background: #ffffff;
  color: #0d9488;
  box-shadow: 0 4px 12px rgba(13, 148, 136, 0.1);
}

.generation-switch-row {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 64px;
  margin-bottom: 8px;
}

.workspace-back {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  padding: 0 20px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 16px;
  color: #475569;
  font-size: 14px;
  font-weight: 700;
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.03);
  cursor: pointer;
  transition: all 0.2s ease;
}

.workspace-back:hover {
  background: #f8fafc;
  border-color: #0d9488;
  color: #0d9488;
  transform: translateX(-4px);
  box-shadow: 0 6px 15px rgba(13, 148, 136, 0.1);
}

.generation-switch-row .workspace-back {
  position: absolute;
  left: 0;
}

.dialog-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;
}

@media (max-width: 1200px) {
  .generation-review-grid {
    grid-template-columns: 1fr;
  }
  .generation-audit-column {
    position: static;
    height: auto;
    overflow: visible;
  }
  .generation-workspace {
    padding: 16px;
  }
}

@media (max-width: 768px) {
  .generation-switch-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
    height: auto;
  }
  .generation-switch-row .workspace-back {
    position: static;
  }
}
</style>
