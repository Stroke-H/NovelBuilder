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
import NovelFullReviewPanel from '@/components/NovelWriter/NovelFullReviewPanel.vue'
import NovelWriterSettingsDialog from '@/components/NovelWriter/NovelWriterSettingsDialog.vue'
import NovelTaskMonitorFloat from '@/components/NovelWriter/NovelTaskMonitorFloat.vue'
import {
  emptyNovelMaterials,
  emptyNovelWriterSettings,
  type GenerateStyleTemplatePayload,
  novelWriterApi,
  type CreateNovelProjectPayload,
  type NovelExtractedInfo,
  type NovelFullReview,
  type NovelMaterials,
  type NovelOutline,
  type NovelRuntimeTask,
  type NovelStyleProfile,
  type NovelWriterSettings,
  type NovelProject
} from '@/api/novelWriter'

const projects = shallowRef<NovelProject[]>([])
const selectedProject = shallowRef<NovelProject | null>(null)
const selectedOutlineId = shallowRef('')
const selectedChapterId = shallowRef('')
const loading = shallowRef(false)
const saving = shallowRef(false)
const settingsLoading = shallowRef(false)
const settingsSaving = shallowRef(false)
const running = shallowRef(false)
type WriterActionKey = '' | 'extract' | 'outline' | 'style' | 'generate' | 'audit' | 'revise' | 'adopt' | 'manualEdit' | 'approve' | 'bulkGenerate' | 'bulkAudit' | 'bulkRevise' | 'fullReview' | 'fullReviewRevise'
const createDialogVisible = shallowRef(false)
const settingsDialogVisible = shallowRef(false)
const activeStep = shallowRef<'materials' | 'generation'>('materials')
const insightsOpen = shallowRef(false)
const generationVisited = shallowRef(false)
const materialInsightsImported = shallowRef(false)
const styleProfileSuppressedByMaterialImport = shallowRef(false)
const materialInsightsSnapshot = shallowRef<NovelExtractedInfo | null>(null)
const materialPanelRef = shallowRef<InstanceType<typeof NovelMaterialPanel> | null>(null)
const outlinePollingTimer = shallowRef<number | undefined>()
const runningActionLabel = shallowRef('')
const activeAction = shallowRef<WriterActionKey>('')
const activeActionTargetId = shallowRef('')
const chapterPanelEditing = shallowRef(false)
const materialMapEditing = shallowRef(false)
const pendingPolledProject = shallowRef<NovelProject | null>(null)
const runtimeTasks = shallowRef<NovelRuntimeTask[]>([])
const runtimeTasksLoading = shallowRef(false)
const runtimeTaskCancelId = shallowRef('')
const runtimeTaskTimer = shallowRef<number | undefined>()
const runtimeTasksHydrated = shallowRef(false)
const styleTemplateTaskWasActive = shallowRef(false)
const styleTemplateTaskLastStatus = shallowRef('')
const bulkGenerateProgress = reactive({ current: 0, total: 0 })
const bulkAuditProgress = reactive({ current: 0, total: 0 })
const bulkReviseProgress = reactive({ current: 0, total: 0 })
const pipelineLoading = shallowRef(false)
const pipelineLabel = shallowRef('AI一条龙')
const writerSettings = shallowRef<NovelWriterSettings>(emptyNovelWriterSettings())

const normalizeWriterGeneralSettings = (settings?: NovelWriterSettings['general']) => ({
  max_chapters: Math.min(Math.max(Number(settings?.max_chapters) || 200, 1), 1000),
  max_chapter_words: Math.min(Math.max(Number(settings?.max_chapter_words) || 80000, 1200), 200000)
})

const writerGeneralSettings = computed(() => normalizeWriterGeneralSettings(writerSettings.value.general))

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

const createTargetWordsMax = computed(() => (
  Math.max(1000, createForm.target_chapters * writerGeneralSettings.value.max_chapter_words)
))

const clampCreateFormTargets = () => {
  const general = writerGeneralSettings.value
  if (createForm.target_chapters > general.max_chapters) {
    createForm.target_chapters = general.max_chapters
  }
  if (createForm.target_chapters < 1) {
    createForm.target_chapters = 1
  }
  if (createForm.target_words > createTargetWordsMax.value) {
    createForm.target_words = createTargetWordsMax.value
  }
  if (createForm.target_words < 1000) {
    createForm.target_words = 1000
  }
}

const selectedChapter = computed(() => {
  const chapters = selectedProject.value?.chapters || []
  const selectedById = chapters.find((item) => item.id === selectedChapterId.value)
  if (selectedById) return selectedById
  if (selectedOutlineId.value) {
    return chapters.find((item) => item.outline_id === selectedOutlineId.value)
  }
  return chapters[0]
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

const emptyFullReview = (): NovelFullReview => ({
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

const selectedFullReview = computed(() => selectedProject.value?.full_review || emptyFullReview())

const hasLocalEditing = computed(() => chapterPanelEditing.value || materialMapEditing.value)

const hasAuditResult = computed(() => {
  const audit = selectedAudit.value
  return Boolean(
    audit.issues?.length
    || audit.revision_advice?.trim()
    || audit.total_score
    || audit.ai_flavor_score
    || audit.character_score
    || audit.logic_score
    || audit.style_score
  )
})

const fullReviewNeedsRevision = (review?: NovelFullReview | null) => Boolean(
  review?.issues?.length
  || review?.revision_advice?.trim()
)

const chapterHasContent = (chapter?: NovelProject['chapters'][number] | null) => {
  if (!chapter) return false
  if (chapter.content?.trim()) return true
  return Boolean(chapter.versions?.some((item) => item.content?.trim()))
}

const chapterHasAuditResult = (chapter?: NovelProject['chapters'][number] | null) => {
  if (!chapter) return false
  const audit = chapter.audit
  if (!audit) return false
  return Boolean(
    audit.issues?.length
    || audit.revision_advice?.trim()
    || audit.total_score
    || audit.ai_flavor_score
    || audit.character_score
    || audit.logic_score
    || audit.style_score
  )
}

const chapterHasRevisionVersion = (chapter?: NovelProject['chapters'][number] | null) => {
  if (!chapter) return false
  return Boolean(chapter.versions?.some((version) => version.type === 'revision' && version.content?.trim()))
}

const hasExtractedInfoData = (project: NovelProject) => Boolean(
  project.extracted?.characters?.length
  || project.extracted?.world_rules?.length
  || project.extracted?.conflicts?.length
  || project.extracted?.key_events?.length
)

const hasStyleProfileData = (project: NovelProject) => Boolean(
  project.style_profile?.summary
  || project.style_profile?.narration
  || project.style_profile?.sentence
  || project.style_profile?.dialogue
  || project.style_profile?.rhythm
  || project.style_profile?.do_rules?.length
  || project.style_profile?.avoid_rules?.length
)

const latestRevisionVersionId = (chapter?: NovelProject['chapters'][number] | null) => {
  if (!chapter) return ''
  const revisions = (chapter.versions || []).filter((version) => version.type === 'revision' && version.content?.trim())
  return revisions[revisions.length - 1]?.id || ''
}

const retryAsync = async <T>(action: () => Promise<T>, maxAttempts = 3) => {
  let lastError: unknown
  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    try {
      return await action()
    } catch (error) {
      lastError = error
      if (attempt >= maxAttempts) break
    }
  }
  throw lastError
}

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
  if (styleProfileSuppressedByMaterialImport.value) {
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

const styleTemplates = computed(() => writerSettings.value.style_templates || [])
const isRuntimeTaskActive = (task: NovelRuntimeTask) => task.status === 'running' || task.status === 'cancelling'
const styleTemplateLatestTask = computed(() => runtimeTasks.value.find((task) => task.kind === 'style_template_generate') || null)
const styleTemplateGenerationTask = computed(() => runtimeTasks.value.find((task) => task.kind === 'style_template_generate' && isRuntimeTaskActive(task)) || null)
const styleTemplateGenerationLoading = computed(() => Boolean(styleTemplateGenerationTask.value))
const styleTemplateGenerationLabel = computed(() => styleTemplateGenerationTask.value?.title || '整本小说提炼中')

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

const isActionLoading = (action: WriterActionKey, targetId = '') => {
  if (!running.value || activeAction.value !== action) return false
  return !targetId || activeActionTargetId.value === targetId
}

const outlineProgress = (outline?: NovelOutline | null) => {
  const chapters = outline?.chapters?.length || 0
  const generated = outline?.generated_chapters || chapters
  const target = outline?.target_chapters || 0
  return {
    generated,
    target,
    complete: target > 0 && generated >= target
  }
}

const hasActiveRuntimeTask = (projectId: string, kind: string) => {
  return runtimeTasks.value.some((task) => task.project_id === projectId && task.kind === kind && isRuntimeTaskActive(task))
}

const normalizeStaleOutlineProject = (project: NovelProject) => {
  if (!runtimeTasksHydrated.value) return project
  if (project.outline?.generation_status !== 'generating') return project
  if (hasActiveRuntimeTask(project.id, 'outline')) return project
  return {
    ...project,
    outline: {
      ...project.outline,
      generation_status: 'cancelled',
      generation_error: project.outline.generation_error || '服务已重启，原大纲后台任务已中断'
    }
  }
}

const stopRuntimeTaskPolling = () => {
  if (!runtimeTaskTimer.value) return
  window.clearInterval(runtimeTaskTimer.value)
  runtimeTaskTimer.value = undefined
}

const fetchRuntimeTasks = async (silent = false) => {
  if (!silent) runtimeTasksLoading.value = true
  try {
    runtimeTasks.value = await novelWriterApi.listRuntimeTasks()
    runtimeTasksHydrated.value = true
    if (selectedProject.value) {
      const normalizedProject = normalizeStaleOutlineProject(selectedProject.value)
      if (normalizedProject !== selectedProject.value) {
        syncSelectedProject(normalizedProject)
        stopOutlinePolling()
      }
    }
    projects.value = projects.value.map((project) => normalizeStaleOutlineProject(project))
  } catch (error) {
    if (!silent) ElMessage.error(`加载后台任务失败：${getErrorMessage(error)}`)
  } finally {
    if (!silent) runtimeTasksLoading.value = false
  }
}

const loadWriterSettings = async (silent = false) => {
  if (!silent) settingsLoading.value = true
  try {
    writerSettings.value = await novelWriterApi.getSettings()
  } catch (error) {
    if (!silent) ElMessage.error(`加载创作设置失败：${getErrorMessage(error)}`)
  } finally {
    if (!silent) settingsLoading.value = false
  }
}

const saveWriterSettings = async (payload: NovelWriterSettings) => {
  settingsSaving.value = true
  try {
    writerSettings.value = await novelWriterApi.updateSettings(payload)
    settingsDialogVisible.value = false
    ElMessage.success('创作设置已保存')
  } catch (error) {
    ElMessage.error(`保存创作设置失败：${getErrorMessage(error)}`)
  } finally {
    settingsSaving.value = false
  }
}

const startStyleTemplateGeneration = async (payload: GenerateStyleTemplatePayload) => {
  try {
    await novelWriterApi.generateStyleTemplate(payload)
    await fetchRuntimeTasks(true)
    ElMessage.success('整本参考小说提炼任务已进入后台')
  } catch (error) {
    ElMessage.error(`启动文风模版提炼失败：${getErrorMessage(error)}`)
  }
}

const openWriterSettingsDialog = async () => {
  settingsDialogVisible.value = true
  await loadWriterSettings()
}

const startRuntimeTaskPolling = () => {
  stopRuntimeTaskPolling()
  fetchRuntimeTasks(true)
  runtimeTaskTimer.value = window.setInterval(() => {
    fetchRuntimeTasks(true)
  }, 1500)
}

const cancelRuntimeTask = async (taskId: string) => {
  runtimeTaskCancelId.value = taskId
  try {
    await novelWriterApi.cancelRuntimeTask(taskId)
    await fetchRuntimeTasks(true)
    ElMessage.success('后台任务终止指令已发送')
  } catch (error) {
    ElMessage.error(`终止后台任务失败：${getErrorMessage(error)}`)
  } finally {
    runtimeTaskCancelId.value = ''
  }
}

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
  const normalizedProject = normalizeStaleOutlineProject(project)
  chapterPanelEditing.value = false
  materialMapEditing.value = false
  pendingPolledProject.value = null
  selectedProject.value = normalizedProject
  selectedOutlineId.value = normalizedProject.outline?.chapters?.[0]?.id || ''
  selectedChapterId.value = normalizedProject.chapters?.find((chapter) => chapter.outline_id === selectedOutlineId.value)?.id || ''
  activeStep.value = 'materials'
  generationVisited.value = false
  materialInsightsImported.value = false
  styleProfileSuppressedByMaterialImport.value = false
  materialInsightsSnapshot.value = null
  if (normalizedProject.outline?.generation_status === 'generating' && hasActiveRuntimeTask(normalizedProject.id, 'outline')) {
    startOutlinePolling(normalizedProject.id)
  } else {
    stopOutlinePolling()
  }
}

const backToProjectList = () => {
  stopOutlinePolling()
  chapterPanelEditing.value = false
  materialMapEditing.value = false
  pendingPolledProject.value = null
  selectedProject.value = null
  selectedOutlineId.value = ''
  selectedChapterId.value = ''
  insightsOpen.value = false
  generationVisited.value = false
  materialInsightsImported.value = false
  styleProfileSuppressedByMaterialImport.value = false
  materialInsightsSnapshot.value = null
}

const importMaterialsToInsights = () => {
  materialInsightsSnapshot.value = buildMaterialExtracted(currentMaterials.value, false)
  materialInsightsImported.value = true
  styleProfileSuppressedByMaterialImport.value = true
  insightsOpen.value = true
  ElMessage.success('素材图谱内容已带入，灵感卡片未带入')
}

const createProject = async () => {
  if (!createForm.title.trim()) {
    ElMessage.warning('请先填写小说名称')
    return
  }
  clampCreateFormTargets()
  saving.value = true
  try {
    const project = await novelWriterApi.createProject({
      ...createForm,
      materials: { ...createForm.materials }
    })
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
    clampCreateFormTargets()
    ElMessage.success('小说项目已创建')
  } catch (error) {
    ElMessage.error(`创建失败：${getErrorMessage(error)}`)
  } finally {
    saving.value = false
  }
}

const syncSelectedProject = (project: NovelProject) => {
  const normalizedProject = normalizeStaleOutlineProject(project)
  if (pendingPolledProject.value?.id === project.id) {
    pendingPolledProject.value = null
  }
  selectedProject.value = normalizedProject
  projects.value = projects.value.map((item) => item.id === normalizedProject.id ? normalizedProject : item)
  const outlineExists = normalizedProject.outline?.chapters?.some((chapter) => chapter.id === selectedOutlineId.value)
  if (!selectedOutlineId.value || !outlineExists) selectedOutlineId.value = normalizedProject.outline?.chapters?.[0]?.id || ''
  const chapterExists = normalizedProject.chapters?.some((chapter) => chapter.id === selectedChapterId.value)
  if (!selectedChapterId.value || !chapterExists) {
    selectedChapterId.value = selectedOutlineId.value
      ? normalizedProject.chapters?.find((chapter) => chapter.outline_id === selectedOutlineId.value)?.id || ''
      : normalizedProject.chapters?.[0]?.id || ''
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
      if (!hasActiveRuntimeTask(projectId, 'outline')) {
        stopOutlinePolling()
        syncSelectedProject(normalizeStaleOutlineProject(project))
        return
      }
      const progress = outlineProgress(project.outline)
      if (hasLocalEditing.value) pendingPolledProject.value = project
      else syncSelectedProject(project)
      if (project.outline?.generation_status !== 'generating' || progress.complete) stopOutlinePolling()
    } catch (error) {
      stopOutlinePolling()
      ElMessage.error(`刷新大纲进度失败：${getErrorMessage(error)}`)
    }
  }, 5000)
}

watch(
  hasLocalEditing,
  (editing) => {
    if (editing || !pendingPolledProject.value) return
    const project = pendingPolledProject.value
    pendingPolledProject.value = null
    syncSelectedProject(project)
  }
)

watch(
  styleTemplateGenerationLoading,
  async (active) => {
    if (active) {
      styleTemplateTaskWasActive.value = true
      styleTemplateTaskLastStatus.value = styleTemplateGenerationTask.value?.status || ''
      return
    }
    if (!styleTemplateTaskWasActive.value) return
    styleTemplateTaskWasActive.value = false
    await loadWriterSettings(true)
    const latestTask = styleTemplateLatestTask.value
    const latestStatus = latestTask?.status || styleTemplateTaskLastStatus.value
    if (latestStatus === 'failed') {
      ElMessage.error(`文风模版提炼失败：${latestTask?.error || '请展开后台任务查看原因'}`)
    } else if (latestStatus === 'cancelled' || latestStatus === 'cancelling') {
      ElMessage.info('文风模版提炼任务已终止')
    } else {
      ElMessage.success('文风模版提炼已完成，书架已刷新')
    }
    styleTemplateTaskLastStatus.value = ''
  }
)

watch(
  () => styleTemplateGenerationTask.value?.status || '',
  (status) => {
    if (status) styleTemplateTaskLastStatus.value = status
  }
)

watch(
  [writerGeneralSettings, () => createForm.target_chapters],
  clampCreateFormTargets,
  { immediate: true }
)

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

const runProjectAction = async (
  actionKey: WriterActionKey,
  label: string,
  action: () => Promise<NovelProject>,
  targetId = ''
) => {
  if (!selectedProject.value) return
  if (!ensureActionAvailable(label)) return
  running.value = true
  runningActionLabel.value = label
  activeAction.value = actionKey
  activeActionTargetId.value = targetId
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
    activeAction.value = ''
    activeActionTargetId.value = ''
  }
}

const saveBeforeAIAction = async () => {
  if (!selectedProject.value) return
  syncSelectedProject(await novelWriterApi.updateProject(selectedProject.value))
}

const extractInfo = () => runProjectAction('extract', '信息提取', async () => {
  await saveBeforeAIAction()
  return novelWriterApi.extractInfo(selectedProject.value!.id)
})

const planOutline = async () => {
  if (!ensureActionAvailable('生成大纲/章节结构')) return
  if (
    selectedProject.value?.outline?.generation_status === 'generating'
    && selectedProject.value
    && hasActiveRuntimeTask(selectedProject.value.id, 'outline')
  ) {
    notifyTaskInProgress('生成大纲/章节结构')
    return
  }
  const project = await runProjectAction('outline', '大纲规划', async () => {
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

const analyzeStyle = async () => {
  const project = await runProjectAction('style', '文风画像', async () => {
    await saveBeforeAIAction()
    return novelWriterApi.analyzeStyle(selectedProject.value!.id)
  })
  if (project) {
    styleProfileSuppressedByMaterialImport.value = false
    insightsOpen.value = true
  }
  return project
}

const wait = (ms: number) => new Promise((resolve) => window.setTimeout(resolve, ms))

const waitForOutlineReady = async (projectId: string, timeoutMs = 15 * 60 * 1000) => {
  const startedAt = Date.now()
  while (Date.now() - startedAt < timeoutMs) {
    const project = normalizeStaleOutlineProject(await novelWriterApi.getProject(projectId))
    syncSelectedProject(project)
    const progress = outlineProgress(project.outline)
    if (project.outline?.generation_status === 'failed') {
      throw new Error(project.outline.generation_error || '大纲生成失败')
    }
    if (project.outline?.generation_status === 'cancelled') {
      throw new Error(project.outline.generation_error || '大纲生成已终止')
    }
    if (project.outline?.generation_status !== 'generating' || progress.complete) {
      return project
    }
    await fetchRuntimeTasks(true)
    await wait(2000)
  }
  throw new Error('等待大纲生成完成超时，请稍后重试')
}

const switchToGeneration = async () => {
  await saveBeforeAIAction()
  activeStep.value = 'generation'
  if (!generationVisited.value) {
    insightsOpen.value = false
    generationVisited.value = true
  }
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

const ensureOutlineReady = async (preferredOutlineId = '') => {
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

  const preferredExists = selectedProject.value.outline?.chapters?.some((chapter) => chapter.id === preferredOutlineId)
  const outlineId = (preferredExists ? preferredOutlineId : '')
    || selectedOutlineId.value
    || selectedProject.value.outline?.chapters?.[0]?.id
    || ''
  if (!outlineId) throw new Error('大纲生成失败：没有可用章节')
  selectedOutlineId.value = outlineId
  return outlineId
}

const generateChapter = (outlineId = '') => runProjectAction('generate', '章节生成', async () => {
  const resolvedOutlineId = await ensureOutlineReady(outlineId)
  return novelWriterApi.generateChapter(selectedProject.value!.id, resolvedOutlineId)
}, outlineId || selectedOutlineId.value)
const auditChapter = (chapterId: string) => runProjectAction('audit', '章节审计', () => novelWriterApi.auditChapter(selectedProject.value!.id, chapterId), chapterId)
const reviseChapter = (chapterId: string) => runProjectAction('revise', '智能修订', () => novelWriterApi.reviseChapter(selectedProject.value!.id, chapterId), chapterId)
const adoptChapterVersion = (chapterId: string, versionId: string) => runProjectAction('adopt', '版本选用', async () => {
  try {
    return await novelWriterApi.adoptChapterVersion(selectedProject.value!.id, chapterId, versionId)
  } catch (error: any) {
    if (error?.response?.status !== 404 || !selectedProject.value) throw error

    const nextProject: NovelProject = {
      ...selectedProject.value,
      chapters: (selectedProject.value.chapters || []).map((chapter) => {
        if (chapter.id !== chapterId) return chapter
        const targetVersion = chapter.versions?.find((version) => version.id === versionId)
        if (!targetVersion) return chapter
        return {
          ...chapter,
          content: targetVersion.content,
          active_version_id: versionId
        }
      })
    }

    return novelWriterApi.updateProject(nextProject)
  }
}, versionId)
const saveManualChapterEdit = async (chapterId: string, content: string, versionId: string) => {
  if (!selectedProject.value) return

  const previousProject = selectedProject.value
  const editedProject: NovelProject = {
    ...selectedProject.value,
    chapters: (selectedProject.value.chapters || []).map((chapter) => {
      if (chapter.id !== chapterId) return chapter
      return {
        ...chapter,
        content,
        versions: (chapter.versions || []).map((version) => version.id === versionId ? { ...version, content } : version)
      }
    })
  }

  syncSelectedProject(editedProject)
  const project = await runProjectAction('manualEdit', '正文编辑保存', () => novelWriterApi.updateProject(editedProject), chapterId)
  if (!project) syncSelectedProject(previousProject)
}
const approveChapter = (chapterId: string) => runProjectAction('approve', '章节确认', () => novelWriterApi.approveChapter(selectedProject.value!.id, chapterId), chapterId)
const reviewFullNovelQuality = async (managedByWorkflow = false) => {
  if (!selectedProject.value) return
  if (!managedByWorkflow && !ensureActionAvailable('全文质量核验')) return

  if (!managedByWorkflow) {
    return runProjectAction('fullReview', '全文质量核验', () => novelWriterApi.fullReviewProject(selectedProject.value!.id))
  }

  activeAction.value = 'fullReview'
  activeActionTargetId.value = selectedProject.value.id
  try {
    const project = await novelWriterApi.fullReviewProject(selectedProject.value.id)
    syncSelectedProject(project)
    return project
  } finally {
    activeAction.value = ''
    activeActionTargetId.value = ''
  }
}

const reviseNovelByFullReview = async (managedByWorkflow = false) => {
  if (!selectedProject.value) return
  if (!managedByWorkflow && !ensureActionAvailable('按核验结果修订')) return

  if (!managedByWorkflow) {
    return runProjectAction('fullReviewRevise', '按核验结果修订', () => novelWriterApi.reviseProjectByFullReview(selectedProject.value!.id))
  }

  activeAction.value = 'fullReviewRevise'
  activeActionTargetId.value = selectedProject.value.id
  try {
    const project = await novelWriterApi.reviseProjectByFullReview(selectedProject.value.id)
    syncSelectedProject(project)
    return project
  } finally {
    activeAction.value = ''
    activeActionTargetId.value = ''
  }
}

const generateAllChapters = async (managedByWorkflow = false) => {
  if (!selectedProject.value) return
  if (!managedByWorkflow && !ensureActionAvailable('生成所有章节正文')) return

  await saveBeforeAIAction()
  const latestProject = selectedProject.value
  const outlineChapters = latestProject.outline?.chapters || []
  if (!outlineChapters.length) {
    if (!managedByWorkflow) ElMessage.warning('请先生成大纲/章节结构')
    return { successCount: 0, failedCount: 0, skipped: 0 }
  }

  const pendingOutlineIds = outlineChapters
    .filter((outlineChapter) => {
      const chapter = latestProject.chapters?.find((item) => item.outline_id === outlineChapter.id)
      return !chapterHasContent(chapter) && !chapterHasAuditResult(chapter)
    })
    .map((outlineChapter) => outlineChapter.id)

  if (!pendingOutlineIds.length) {
    if (!managedByWorkflow) ElMessage.info('没有需要生成正文的章节')
    return { successCount: 0, failedCount: 0, skipped: outlineChapters.length }
  }

  if (!managedByWorkflow) running.value = true
  runningActionLabel.value = managedByWorkflow ? 'AI一条龙 · 生成正文' : '批量生成正文'
  activeAction.value = 'bulkGenerate'
  activeActionTargetId.value = ''
  bulkGenerateProgress.current = 0
  bulkGenerateProgress.total = pendingOutlineIds.length

  let successCount = 0
  const failedOutlineIds: string[] = []
  try {
    for (const [index, outlineId] of pendingOutlineIds.entries()) {
      bulkGenerateProgress.current = index + 1
      activeActionTargetId.value = outlineId
      try {
        const project = await retryAsync(
          () => novelWriterApi.generateChapter(selectedProject.value!.id, outlineId),
          3
        )
        syncSelectedProject(project)
        successCount += 1
      } catch (error) {
        failedOutlineIds.push(outlineId)
        console.error(`generate chapter failed for outline ${outlineId}`, error)
      }
    }
    if (!failedOutlineIds.length) {
      if (!managedByWorkflow) ElMessage.success(`已生成 ${successCount} 章正文`)
    } else if (successCount > 0) {
      if (!managedByWorkflow) ElMessage.warning(`已生成 ${successCount} 章正文，另有 ${failedOutlineIds.length} 章生成失败`)
    } else {
      if (!managedByWorkflow) ElMessage.error('批量生成正文失败：所有待生成章节都未成功')
    }
    return { successCount, failedCount: failedOutlineIds.length, skipped: outlineChapters.length - pendingOutlineIds.length }
  } finally {
    if (!managedByWorkflow) running.value = false
    if (!managedByWorkflow) runningActionLabel.value = ''
    activeAction.value = ''
    activeActionTargetId.value = ''
    bulkGenerateProgress.current = 0
    bulkGenerateProgress.total = 0
  }
}

const auditAllChapters = async (managedByWorkflow = false) => {
  if (!selectedProject.value) return
  if (!managedByWorkflow && !ensureActionAvailable('审计所有正文')) return

  const outlineOrder = new Map(
    (selectedProject.value.outline?.chapters || []).map((chapter, index) => [chapter.id, index])
  )
  const allChapters = selectedProject.value.chapters || []
  const pendingChapterIds = [...(selectedProject.value.chapters || [])]
    .sort((left, right) => {
      const leftOrder = outlineOrder.get(left.outline_id) ?? Number.MAX_SAFE_INTEGER
      const rightOrder = outlineOrder.get(right.outline_id) ?? Number.MAX_SAFE_INTEGER
      return leftOrder - rightOrder
    })
    .filter((chapter) => chapterHasContent(chapter) && !chapterHasAuditResult(chapter))
    .map((chapter) => chapter.id)

  if (!pendingChapterIds.length) {
    if (!managedByWorkflow) ElMessage.info('没有需要审计的正文')
    return { successCount: 0, failedCount: 0, skipped: allChapters.length }
  }

  if (!managedByWorkflow) running.value = true
  runningActionLabel.value = managedByWorkflow ? 'AI一条龙 · 审计正文' : '批量审计正文'
  activeAction.value = 'bulkAudit'
  activeActionTargetId.value = ''
  bulkAuditProgress.current = 0
  bulkAuditProgress.total = pendingChapterIds.length

  let successCount = 0
  const failedChapterIds: string[] = []
  try {
    for (const [index, chapterId] of pendingChapterIds.entries()) {
      bulkAuditProgress.current = index + 1
      activeActionTargetId.value = chapterId
      try {
        const project = await retryAsync(
          () => novelWriterApi.auditChapter(selectedProject.value!.id, chapterId),
          3
        )
        syncSelectedProject(project)
        successCount += 1
      } catch (error) {
        failedChapterIds.push(chapterId)
        console.error(`audit chapter failed for chapter ${chapterId}`, error)
      }
    }
    if (!failedChapterIds.length) {
      if (!managedByWorkflow) ElMessage.success(`已审计 ${successCount} 章正文`)
    } else if (successCount > 0) {
      if (!managedByWorkflow) ElMessage.warning(`已审计 ${successCount} 章正文，另有 ${failedChapterIds.length} 章审计失败`)
    } else {
      if (!managedByWorkflow) ElMessage.error('批量审计正文失败：所有待审计章节都未成功')
    }
    return { successCount, failedCount: failedChapterIds.length, skipped: allChapters.length - pendingChapterIds.length }
  } finally {
    if (!managedByWorkflow) running.value = false
    if (!managedByWorkflow) runningActionLabel.value = ''
    activeAction.value = ''
    activeActionTargetId.value = ''
    bulkAuditProgress.current = 0
    bulkAuditProgress.total = 0
  }
}

const reviseAllChapters = async (managedByWorkflow = false) => {
  if (!selectedProject.value) return
  if (!managedByWorkflow && !ensureActionAvailable('按审计修订所有正文')) return

  const outlineOrder = new Map(
    (selectedProject.value.outline?.chapters || []).map((chapter, index) => [chapter.id, index])
  )
  const allChapters = selectedProject.value.chapters || []
  const pendingChapterIds = [...(selectedProject.value.chapters || [])]
    .sort((left, right) => {
      const leftOrder = outlineOrder.get(left.outline_id) ?? Number.MAX_SAFE_INTEGER
      const rightOrder = outlineOrder.get(right.outline_id) ?? Number.MAX_SAFE_INTEGER
      return leftOrder - rightOrder
    })
    .filter((chapter) => chapterHasContent(chapter) && chapterHasAuditResult(chapter) && !chapterHasRevisionVersion(chapter))
    .map((chapter) => chapter.id)

  if (!pendingChapterIds.length) {
    if (!managedByWorkflow) ElMessage.info('没有需要按审计修订的正文')
    return { successCount: 0, failedCount: 0, skipped: allChapters.length }
  }

  if (!managedByWorkflow) running.value = true
  runningActionLabel.value = managedByWorkflow ? 'AI一条龙 · 修订正文' : '批量审计修订正文'
  activeAction.value = 'bulkRevise'
  activeActionTargetId.value = ''
  bulkReviseProgress.current = 0
  bulkReviseProgress.total = pendingChapterIds.length

  let successCount = 0
  const failedChapterIds: string[] = []
  try {
    for (const [index, chapterId] of pendingChapterIds.entries()) {
      bulkReviseProgress.current = index + 1
      activeActionTargetId.value = chapterId
      try {
        const project = await retryAsync(
          () => novelWriterApi.reviseChapter(selectedProject.value!.id, chapterId),
          3
        )
        syncSelectedProject(project)
        successCount += 1
      } catch (error) {
        failedChapterIds.push(chapterId)
        console.error(`revise chapter failed for chapter ${chapterId}`, error)
      }
    }
    if (!failedChapterIds.length) {
      if (!managedByWorkflow) ElMessage.success(`已修订 ${successCount} 章正文`)
    } else if (successCount > 0) {
      if (!managedByWorkflow) ElMessage.warning(`已修订 ${successCount} 章正文，另有 ${failedChapterIds.length} 章修订失败`)
    } else {
      if (!managedByWorkflow) ElMessage.error('批量审计修订失败：所有待修订章节都未成功')
    }
    return { successCount, failedCount: failedChapterIds.length, skipped: allChapters.length - pendingChapterIds.length }
  } finally {
    if (!managedByWorkflow) running.value = false
    if (!managedByWorkflow) runningActionLabel.value = ''
    activeAction.value = ''
    activeActionTargetId.value = ''
    bulkReviseProgress.current = 0
    bulkReviseProgress.total = 0
  }
}

const adoptAllRevisionVersions = async () => {
  if (!selectedProject.value) return { adoptedCount: 0, failedCount: 0 }
  const outlineOrder = new Map(
    (selectedProject.value.outline?.chapters || []).map((chapter, index) => [chapter.id, index])
  )
  const targetChapters = [...(selectedProject.value.chapters || [])]
    .sort((left, right) => {
      const leftOrder = outlineOrder.get(left.outline_id) ?? Number.MAX_SAFE_INTEGER
      const rightOrder = outlineOrder.get(right.outline_id) ?? Number.MAX_SAFE_INTEGER
      return leftOrder - rightOrder
    })
    .filter((chapter) => {
      const revisionVersionId = latestRevisionVersionId(chapter)
      return Boolean(revisionVersionId) && chapter.active_version_id !== revisionVersionId
    })

  let adoptedCount = 0
  let failedCount = 0
  for (const chapter of targetChapters) {
    const revisionVersionId = latestRevisionVersionId(chapter)
    if (!revisionVersionId) continue
    try {
      const project = await retryAsync(
        () => novelWriterApi.adoptChapterVersion(selectedProject.value!.id, chapter.id, revisionVersionId),
        3
      )
      syncSelectedProject(project)
      adoptedCount += 1
    } catch (error) {
      failedCount += 1
      console.error(`adopt revision failed for chapter ${chapter.id}`, error)
    }
  }
  return { adoptedCount, failedCount }
}

const approveAllChapters = async () => {
  if (!selectedProject.value) return { approvedCount: 0, failedCount: 0 }
  const outlineOrder = new Map(
    (selectedProject.value.outline?.chapters || []).map((chapter, index) => [chapter.id, index])
  )
  const targetChapterIds = [...(selectedProject.value.chapters || [])]
    .sort((left, right) => {
      const leftOrder = outlineOrder.get(left.outline_id) ?? Number.MAX_SAFE_INTEGER
      const rightOrder = outlineOrder.get(right.outline_id) ?? Number.MAX_SAFE_INTEGER
      return leftOrder - rightOrder
    })
    .filter((chapter) => chapterHasContent(chapter) && chapter.status !== 'approved')
    .map((chapter) => chapter.id)

  let approvedCount = 0
  let failedCount = 0
  for (const chapterId of targetChapterIds) {
    try {
      const project = await retryAsync(
        () => novelWriterApi.approveChapter(selectedProject.value!.id, chapterId),
        3
      )
      syncSelectedProject(project)
      approvedCount += 1
    } catch (error) {
      failedCount += 1
      console.error(`approve chapter failed for chapter ${chapterId}`, error)
    }
  }
  return { approvedCount, failedCount }
}

const runAiPipeline = async () => {
  if (!selectedProject.value) return
  if (!ensureActionAvailable('AI一条龙')) return

  running.value = true
  pipelineLoading.value = true
  pipelineLabel.value = 'AI一条龙 · 准备中'
  runningActionLabel.value = 'AI一条龙'
  activeAction.value = ''
  activeActionTargetId.value = ''

  try {
    await saveBeforeAIAction()
    await fetchRuntimeTasks(true)

    if (!selectedProject.value) throw new Error('请先选择小说项目')

    if (!hasExtractedInfoData(selectedProject.value)) {
      pipelineLabel.value = 'AI一条龙 · 信息提取'
      activeAction.value = 'extract'
      syncSelectedProject(await novelWriterApi.extractInfo(selectedProject.value.id))
    }

    if (!hasStyleProfileData(selectedProject.value)) {
      pipelineLabel.value = 'AI一条龙 · 文风画像'
      activeAction.value = 'style'
      syncSelectedProject(await novelWriterApi.analyzeStyle(selectedProject.value.id))
    }

    pipelineLabel.value = 'AI一条龙 · 生成大纲'
    activeAction.value = 'outline'
    const outlinedProject = await novelWriterApi.planOutline(selectedProject.value.id)
    syncSelectedProject(outlinedProject)
    if (outlinedProject.outline?.generation_status === 'generating') {
      startOutlinePolling(outlinedProject.id)
      pipelineLabel.value = 'AI一条龙 · 等待大纲完成'
      await waitForOutlineReady(outlinedProject.id)
    }

    pipelineLabel.value = 'AI一条龙 · 生成正文'
    await generateAllChapters(true)

    pipelineLabel.value = 'AI一条龙 · 审计正文'
    await auditAllChapters(true)

    pipelineLabel.value = 'AI一条龙 · 修订正文'
    await reviseAllChapters(true)

    pipelineLabel.value = 'AI一条龙 · 采用修订版'
    const adoptSummary = await adoptAllRevisionVersions()
    if (adoptSummary.failedCount > 0) {
      ElMessage.warning(`修订版采用完成 ${adoptSummary.adoptedCount} 章，另有 ${adoptSummary.failedCount} 章采用失败`)
    }

    pipelineLabel.value = 'AI一条龙 · 全文核验'
    const reviewedProject = await reviewFullNovelQuality(true)

    if (fullReviewNeedsRevision(reviewedProject?.full_review)) {
      pipelineLabel.value = 'AI一条龙 · 全文修订'
      await reviseNovelByFullReview(true)
      pipelineLabel.value = 'AI一条龙 · 复审精修章节'
      await auditAllChapters(true)
    }

    pipelineLabel.value = 'AI一条龙 · 确认章节'
    const approveSummary = await approveAllChapters()
    if (approveSummary.failedCount > 0) {
      ElMessage.warning(`章节确认完成 ${approveSummary.approvedCount} 章，另有 ${approveSummary.failedCount} 章确认失败`)
    }

    pipelineLabel.value = 'AI一条龙 · 已完成'
    ElMessage.success('AI一条龙工作流已执行完成')
  } catch (error) {
    ElMessage.error(`AI一条龙执行失败：${getErrorMessage(error)}`)
  } finally {
    running.value = false
    runningActionLabel.value = ''
    activeAction.value = ''
    activeActionTargetId.value = ''
    pipelineLoading.value = false
    pipelineLabel.value = 'AI一条龙'
  }
}

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
onMounted(() => loadWriterSettings(true))
onMounted(startRuntimeTaskPolling)
onBeforeUnmount(stopOutlinePolling)
onBeforeUnmount(stopRuntimeTaskPolling)
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
        @settings="openWriterSettingsDialog"
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
          @editing-state-change="materialMapEditing = $event"
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
          <div class="generation-sticky-shell">
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
          </div>

          <NovelMaterialPanel
            ref="materialPanelRef"
            v-model="currentMaterials"
            :saving="saving"
            :style-loading="isActionLoading('style')"
            :outline-loading="isActionLoading('outline')"
            :pipeline-loading="pipelineLoading"
            :pipeline-label="pipelineLabel"
            :style-templates="styleTemplates"
            :style-profile="selectedProject.style_profile"
            @save="saveCurrentProject"
            @outline="planOutline"
            @style="analyzeStyle"
            @pipeline="runAiPipeline"
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

          <div class="generation-review-grid">
            <NovelChapterPanel
              :outline="selectedProject.outline"
              :selected-outline-id="selectedOutlineId"
              :chapters="selectedProject.chapters || []"
              :selected-chapter-id="selectedChapterId"
              :running="running"
              :generate-loading-outline-id="(isActionLoading('generate') || isActionLoading('bulkGenerate')) ? activeActionTargetId : ''"
              :audit-loading-chapter-id="(isActionLoading('audit') || isActionLoading('bulkAudit')) ? activeActionTargetId : ''"
              :bulk-generate-loading="isActionLoading('bulkGenerate')"
              :bulk-audit-loading="isActionLoading('bulkAudit')"
              :bulk-revise-loading="isActionLoading('bulkRevise')"
              :full-review-loading="isActionLoading('fullReview')"
              :bulk-generate-progress-current="bulkGenerateProgress.current"
              :bulk-generate-progress-total="bulkGenerateProgress.total"
              :bulk-audit-progress-current="bulkAuditProgress.current"
              :bulk-audit-progress-total="bulkAuditProgress.total"
              :bulk-revise-progress-current="bulkReviseProgress.current"
              :bulk-revise-progress-total="bulkReviseProgress.total"
              :adopt-loading-version-id="isActionLoading('adopt') ? activeActionTargetId : ''"
              :manual-save-loading="isActionLoading('manualEdit', selectedChapter?.id || '')"
              :approve-loading-chapter-id="isActionLoading('approve') ? activeActionTargetId : ''"
              @select-outline="selectedOutlineId = $event"
              @select-chapter="selectedChapterId = $event"
              @generate="generateChapter"
              @audit="auditChapter"
              @bulk-generate="generateAllChapters"
              @bulk-audit="auditAllChapters"
              @bulk-revise="reviseAllChapters"
              @full-review="reviewFullNovelQuality"
              @revise="reviseChapter"
              @adopt="adoptChapterVersion"
              @manual-save="saveManualChapterEdit"
              @manual-edit-state-change="chapterPanelEditing = $event"
              @approve="approveChapter"
            />

            <aside class="generation-audit-column">
              <NovelAuditPanel
                :audit="selectedAudit"
                :revise-loading="isActionLoading('revise', selectedChapter?.id || '') || isActionLoading('bulkRevise', selectedChapter?.id || '')"
                :show-revise-action="hasAuditResult"
                @revise="selectedChapter?.id && reviseChapter(selectedChapter.id)"
              />
            </aside>
          </div>

          <NovelFullReviewPanel
            class="generation-full-review"
            :review="selectedFullReview"
            :loading="isActionLoading('fullReviewRevise')"
            :show-revise-action="fullReviewNeedsRevision(selectedFullReview)"
            @revise="reviseNovelByFullReview"
          />
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
            <el-input-number v-model="createForm.target_words" :min="1000" :max="createTargetWordsMax" :step="10000" />
          </el-form-item>
          <el-form-item label="目标章节数">
            <el-input-number v-model="createForm.target_chapters" :min="1" :max="writerGeneralSettings.max_chapters" :step="1" />
          </el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="createProject">创建</el-button>
      </template>
    </el-dialog>

    <NovelWriterSettingsDialog
      v-model="settingsDialogVisible"
      :settings="writerSettings"
      :loading="settingsLoading"
      :saving="settingsSaving"
      :template-generation-loading="styleTemplateGenerationLoading"
      :template-generation-label="styleTemplateGenerationLabel"
      @save="saveWriterSettings"
      @generate-template="startStyleTemplateGeneration"
    />

    <NovelTaskMonitorFloat
      :tasks="runtimeTasks"
      :loading="runtimeTasksLoading"
      :cancel-loading-task-id="runtimeTaskCancelId"
      @refresh="fetchRuntimeTasks()"
      @cancel="cancelRuntimeTask"
    />
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
  flex-direction: column;
  gap: 24px;
  min-height: 100vh;
  padding: 24px 32px;
  box-sizing: border-box;
  width: 100%;
}

.generation-sticky-shell {
  position: sticky;
  top: 0;
  z-index: 30;
  display: block;
  padding-top: 12px;
  background:
    linear-gradient(180deg, rgba(248, 250, 252, 0.98) 0%, rgba(248, 250, 252, 0.95) 78%, rgba(248, 250, 252, 0) 100%);
  backdrop-filter: blur(10px);
}

.generation-review-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(380px, 460px);
  gap: 24px;
  align-items: stretch;
  box-sizing: border-box;
  height: calc(100vh - 32px);
  min-height: 720px;
  min-width: 0;
  overflow: hidden;
}

.generation-audit-column {
  height: 100%;
  min-height: 0;
  align-self: stretch;
  min-width: 0;
  width: 100%;
  overflow: hidden;
}

.generation-full-review {
  width: 100%;
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
    height: auto;
    min-height: 0;
  }
  .generation-audit-column {
    position: static;
    height: auto;
    overflow: visible;
  }
  .generation-workspace {
    padding: 16px;
  }
  .generation-sticky-shell {
    top: 0;
    padding-top: 0;
    background: transparent;
    backdrop-filter: none;
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
