import request from '@/api/request'

const novelWriterClientRuntimeSettings = {
  frontend_request_timeout_seconds: 15,
  ai_request_timeout_seconds: 120,
  ai_long_request_timeout_seconds: 300
}

export function setNovelWriterClientRuntimeSettings(settings?: Partial<NovelWriterGeneralSettings>) {
  novelWriterClientRuntimeSettings.frontend_request_timeout_seconds = Number(settings?.frontend_request_timeout_seconds) || 15
  novelWriterClientRuntimeSettings.ai_request_timeout_seconds = Number(settings?.ai_request_timeout_seconds) || 120
  novelWriterClientRuntimeSettings.ai_long_request_timeout_seconds = Number(settings?.ai_long_request_timeout_seconds) || 300
}

const novelRequestConfig = () => ({
  timeout: novelWriterClientRuntimeSettings.frontend_request_timeout_seconds * 1000
})

const novelAIRequestConfig = () => ({
  timeout: novelWriterClientRuntimeSettings.ai_request_timeout_seconds * 1000
})

const novelLongAIRequestConfig = () => ({
  timeout: novelWriterClientRuntimeSettings.ai_long_request_timeout_seconds * 1000
})

export interface NovelInfoCard {
  name: string
  description: string
}

export interface NovelCharacterState {
  name: string
  state: string
  location: string
}

export interface NovelChapterState {
  characters: NovelCharacterState[]
  plot_hooks: string[]
  plot_advances: string[]
}

export interface NovelTensionPoint {
  position: number
  value: number
  note: string
}

export interface NovelMaterials {
  raw_text: string
  character_raw: string
  world_raw: string
  conflict_raw: string
  reference_raw: string
}

export interface NovelExtractedInfo {
  characters: NovelInfoCard[]
  world_rules: NovelInfoCard[]
  conflicts: NovelInfoCard[]
  key_events: NovelInfoCard[]
  open_questions: string[]
}

export interface NovelChapterOutline {
  id: string
  title: string
  goal: string
  conflict: string
  hook: string
  summary: string
  before_state: NovelChapterState
  after_state: NovelChapterState
  must_happen: string[]
  tension_curve: NovelTensionPoint[]
  key_scenes: string[]
  new_hooks: string[]
}

export interface NovelOutline {
  logline: string
  acts: NovelInfoCard[]
  chapters: NovelChapterOutline[]
  generation_status: string
  target_chapters: number
  generated_chapters: number
  batch_size: number
  generation_error: string
}

export interface NovelStyleProfile {
  summary: string
  narration: string
  sentence: string
  dialogue: string
  rhythm: string
  do_rules: string[]
  avoid_rules: string[]
}

export interface NovelAuditItem {
  severity: string
  title: string
  detail: string
  suggestion: string
}

export interface NovelFullReviewIssue {
  severity: string
  dimension: string
  chapter_id: string
  chapter_title: string
  title: string
  detail: string
  suggestion: string
}

export interface NovelAuditReport {
  total_score: number
  ai_flavor_score: number
  character_score: number
  logic_score: number
  style_score: number
  issues: NovelAuditItem[]
  revision_advice: string
}

export interface NovelFullReview {
  total_score: number
  coherence_score: number
  logic_reasonability_score: number
  character_consistency_score: number
  trigger_reasonability_score: number
  summary: string
  issues: NovelFullReviewIssue[]
  revision_advice: string
  reviewed_at: string
  applied_at: string
}

export interface NovelChapterVersion {
  id: string
  type: string
  content: string
  reason: string
  created_at: string
}

export interface NovelChapter {
  id: string
  outline_id: string
  title: string
  status: string
  content: string
  summary: string
  audit: NovelAuditReport
  versions: NovelChapterVersion[]
  active_version_id: string
  created_at: string
  updated_at: string
}

export interface NovelMemory {
  chapter_summaries: NovelInfoCard[]
  character_states: NovelInfoCard[]
  open_hooks: NovelInfoCard[]
  timeline: NovelInfoCard[]
}

export interface NovelRuntimeTask {
  id: string
  project_id: string
  project_title: string
  kind: string
  title: string
  status: string
  error?: string
  started_at: string
  updated_at: string
  finished_at?: string
}

export interface AIProviderModelConfig {
  name: string
  model: string
  capability: string
}

export interface AIProviderGroupConfig {
  name: string
  api_key: string
  base_url: string
  models: AIProviderModelConfig[]
}

export interface AIProviderConfig {
  name: string
  api_key: string
  base_url: string
  model: string
  capability: string
}

export interface NovelWriterAISettings {
  deepseek_api_key: string
  base_url: string
  model: string
  provider_groups: AIProviderGroupConfig[]
  providers: AIProviderConfig[]
}

export interface NovelWriterGeneralSettings {
  max_chapters: number
  max_chapter_words: number
  db_max_open_conns: number
  db_max_idle_conns: number
  db_conn_max_lifetime_minutes: number
  db_timeout_seconds: number
  frontend_request_timeout_seconds: number
  ai_request_timeout_seconds: number
  ai_long_request_timeout_seconds: number
  default_ai_backend_timeout_seconds: number
  chapter_ai_backend_timeout_seconds: number
  full_review_ai_backend_timeout_seconds: number
  style_template_chapter_timeout_seconds: number
  style_template_summary_timeout_seconds: number
  chapter_max_tokens: number
  style_reference_sample_runes: number
  audit_content_max_runes: number
  revision_content_max_runes: number
  full_review_payload_max_runes: number
  style_template_chapter_runes: number
  style_template_observations_max_runes: number
  material_raw_max_runes: number
  material_character_max_runes: number
  material_world_max_runes: number
  material_conflict_max_runes: number
  prompt_card_limit: number
  prompt_card_name_max_runes: number
  prompt_card_description_max_runes: number
  prompt_question_max_runes: number
  outline_initial_batch_size: number
  outline_batch_size: number
  outline_small_batch_max_tokens: number
  outline_medium_batch_max_tokens: number
  outline_large_batch_max_tokens: number
  batch_retry_attempts: number
  outline_wait_timeout_minutes: number
  runtime_polling_interval_ms: number
  outline_polling_interval_ms: number
  finished_task_retention_minutes: number
  style_template_retry_attempts: number
}

export interface NovelStyleTemplate {
  id: string
  name: string
  description: string
  content: string
  updated_at: string
}

export interface NovelWriterSettings {
  ai_config: NovelWriterAISettings
  general: NovelWriterGeneralSettings
  style_templates: NovelStyleTemplate[]
}

export interface GenerateStyleTemplatePayload {
  name: string
  description: string
  source_text: string
}

export interface NovelProject {
  id: string
  title: string
  genre: string
  target_words: number
  target_chapters: number
  status: string
  current_stage: string
  created_by: string
  created_at: string
  updated_at: string
  materials: NovelMaterials
  extracted: NovelExtractedInfo
  outline: NovelOutline
  style_profile: NovelStyleProfile
  chapters: NovelChapter[]
  memory: NovelMemory
  full_review: NovelFullReview
}

export interface CreateNovelProjectPayload {
  title: string
  genre: string
  target_words: number
  target_chapters: number
  materials: NovelMaterials
}

export function emptyNovelMaterials(): NovelMaterials {
  return {
    raw_text: '',
    character_raw: '',
    world_raw: '',
    conflict_raw: '',
    reference_raw: ''
  }
}

export function emptyNovelWriterSettings(): NovelWriterSettings {
  return {
    ai_config: {
      deepseek_api_key: '',
      base_url: '',
      model: '',
      provider_groups: [],
      providers: []
    },
    general: {
      max_chapters: 200,
      max_chapter_words: 80000,
      db_max_open_conns: 10,
      db_max_idle_conns: 5,
      db_conn_max_lifetime_minutes: 30,
      db_timeout_seconds: 5,
      frontend_request_timeout_seconds: 15,
      ai_request_timeout_seconds: 120,
      ai_long_request_timeout_seconds: 300,
      default_ai_backend_timeout_seconds: 110,
      chapter_ai_backend_timeout_seconds: 240,
      full_review_ai_backend_timeout_seconds: 240,
      style_template_chapter_timeout_seconds: 120,
      style_template_summary_timeout_seconds: 180,
      chapter_max_tokens: 120000,
      style_reference_sample_runes: 12000,
      audit_content_max_runes: 16000,
      revision_content_max_runes: 16000,
      full_review_payload_max_runes: 220000,
      style_template_chapter_runes: 7000,
      style_template_observations_max_runes: 70000,
      material_raw_max_runes: 4000,
      material_character_max_runes: 6000,
      material_world_max_runes: 5000,
      material_conflict_max_runes: 5000,
      prompt_card_limit: 40,
      prompt_card_name_max_runes: 120,
      prompt_card_description_max_runes: 500,
      prompt_question_max_runes: 300,
      outline_initial_batch_size: 1,
      outline_batch_size: 5,
      outline_small_batch_max_tokens: 9000,
      outline_medium_batch_max_tokens: 18000,
      outline_large_batch_max_tokens: 30000,
      batch_retry_attempts: 3,
      outline_wait_timeout_minutes: 15,
      runtime_polling_interval_ms: 1500,
      outline_polling_interval_ms: 5000,
      finished_task_retention_minutes: 10,
      style_template_retry_attempts: 3
    },
    style_templates: []
  }
}

export const novelWriterApi = {
  listProjects() {
    return request.get('/novel-writer/projects', novelRequestConfig()) as Promise<NovelProject[]>
  },
  createProject(payload: CreateNovelProjectPayload) {
    return request.post('/novel-writer/projects', payload, novelRequestConfig()) as Promise<NovelProject>
  },
  getProject(projectId: string) {
    return request.get(`/novel-writer/projects/${projectId}`, novelRequestConfig()) as Promise<NovelProject>
  },
  updateProject(project: NovelProject) {
    return request.put(`/novel-writer/projects/${project.id}`, project, novelRequestConfig()) as Promise<NovelProject>
  },
  deleteProject(projectId: string) {
    return request.delete(`/novel-writer/projects/${projectId}`) as Promise<void>
  },
  deleteProjectFallback(projectId: string) {
    return request.post(`/novel-writer/projects/${projectId}/delete`) as Promise<void>
  },
  extractInfo(projectId: string) {
    return request.post(`/novel-writer/projects/${projectId}/extract`, undefined, novelAIRequestConfig()) as Promise<NovelProject>
  },
  planOutline(projectId: string) {
    return request.post(`/novel-writer/projects/${projectId}/outline`, undefined, novelAIRequestConfig()) as Promise<NovelProject>
  },
  analyzeStyle(projectId: string) {
    return request.post(`/novel-writer/projects/${projectId}/style`, undefined, novelAIRequestConfig()) as Promise<NovelProject>
  },
  generateChapter(projectId: string, outlineId?: string) {
    return request.post(`/novel-writer/projects/${projectId}/chapters/generate`, {
      outline_id: outlineId || ''
    }, novelLongAIRequestConfig()) as Promise<NovelProject>
  },
  auditChapter(projectId: string, chapterId: string) {
    return request.post(`/novel-writer/projects/${projectId}/chapters/${chapterId}/audit`, undefined, novelAIRequestConfig()) as Promise<NovelProject>
  },
  reviseChapter(projectId: string, chapterId: string) {
    return request.post(`/novel-writer/projects/${projectId}/chapters/${chapterId}/revise`, undefined, novelAIRequestConfig()) as Promise<NovelProject>
  },
  fullReviewProject(projectId: string) {
    return request.post(`/novel-writer/projects/${projectId}/full-review`, undefined, novelLongAIRequestConfig()) as Promise<NovelProject>
  },
  reviseProjectByFullReview(projectId: string) {
    return request.post(`/novel-writer/projects/${projectId}/full-review/revise`, undefined, novelLongAIRequestConfig()) as Promise<NovelProject>
  },
  adoptChapterVersion(projectId: string, chapterId: string, versionId: string) {
    return request.post(`/novel-writer/projects/${projectId}/chapters/${chapterId}/versions/${versionId}/adopt`, undefined, novelRequestConfig()) as Promise<NovelProject>
  },
  approveChapter(projectId: string, chapterId: string) {
    return request.post(`/novel-writer/projects/${projectId}/chapters/${chapterId}/approve`, undefined, novelRequestConfig()) as Promise<NovelProject>
  },
  getSettings() {
    return request.get('/novel-writer/settings', novelRequestConfig()) as Promise<NovelWriterSettings>
  },
  updateSettings(payload: NovelWriterSettings) {
    return request.put('/novel-writer/settings', payload, novelRequestConfig()) as Promise<NovelWriterSettings>
  },
  generateStyleTemplate(payload: GenerateStyleTemplatePayload) {
    return request.post('/novel-writer/settings/style-templates/generate', payload, novelLongAIRequestConfig()) as Promise<{ ok: boolean; task_id: string }>
  },
  listRuntimeTasks() {
    return request.get('/novel-writer/tasks', novelRequestConfig()) as Promise<NovelRuntimeTask[]>
  },
  cancelRuntimeTask(taskId: string) {
    return request.post(`/novel-writer/tasks/${taskId}/cancel`, undefined, novelRequestConfig()) as Promise<{ ok: boolean }>
  }
}
