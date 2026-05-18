<script setup lang="ts">
import { computed, reactive, shallowRef, watch } from 'vue'
import { Delete, Plus, Reading } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { AIProviderGroupConfig, GenerateStyleTemplatePayload, NovelWriterGeneralSettings, NovelWriterSettings } from '@/api/novelWriter'

const props = defineProps<{
  modelValue: boolean
  settings: NovelWriterSettings
  loading: boolean
  saving: boolean
  templateGenerationLoading: boolean
  templateGenerationLabel?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [value: NovelWriterSettings]
  generateTemplate: [value: GenerateStyleTemplatePayload]
}>()

type GeneralSettingKey = keyof NovelWriterGeneralSettings

const generalSettingDefaults: NovelWriterGeneralSettings = {
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
}

const generalSettingBounds: Record<GeneralSettingKey, { min: number; max: number; step: number; unit?: string }> = {
  max_chapters: { min: 1, max: 1000, step: 1, unit: '章' },
  max_chapter_words: { min: 1200, max: 200000, step: 1000, unit: '字' },
  db_max_open_conns: { min: 1, max: 100, step: 1, unit: '个' },
  db_max_idle_conns: { min: 1, max: 100, step: 1, unit: '个' },
  db_conn_max_lifetime_minutes: { min: 1, max: 240, step: 1, unit: '分钟' },
  db_timeout_seconds: { min: 1, max: 60, step: 1, unit: '秒' },
  frontend_request_timeout_seconds: { min: 5, max: 120, step: 1, unit: '秒' },
  ai_request_timeout_seconds: { min: 30, max: 600, step: 10, unit: '秒' },
  ai_long_request_timeout_seconds: { min: 60, max: 1800, step: 30, unit: '秒' },
  default_ai_backend_timeout_seconds: { min: 30, max: 600, step: 10, unit: '秒' },
  chapter_ai_backend_timeout_seconds: { min: 60, max: 1800, step: 30, unit: '秒' },
  full_review_ai_backend_timeout_seconds: { min: 60, max: 1800, step: 30, unit: '秒' },
  style_template_chapter_timeout_seconds: { min: 30, max: 900, step: 10, unit: '秒' },
  style_template_summary_timeout_seconds: { min: 30, max: 1200, step: 10, unit: '秒' },
  chapter_max_tokens: { min: 9000, max: 200000, step: 1000, unit: 'tokens' },
  style_reference_sample_runes: { min: 3000, max: 60000, step: 1000, unit: '字' },
  audit_content_max_runes: { min: 3000, max: 80000, step: 1000, unit: '字' },
  revision_content_max_runes: { min: 3000, max: 80000, step: 1000, unit: '字' },
  full_review_payload_max_runes: { min: 20000, max: 500000, step: 10000, unit: '字' },
  style_template_chapter_runes: { min: 2000, max: 30000, step: 1000, unit: '字' },
  style_template_observations_max_runes: { min: 10000, max: 200000, step: 5000, unit: '字' },
  material_raw_max_runes: { min: 1000, max: 20000, step: 1000, unit: '字' },
  material_character_max_runes: { min: 1000, max: 30000, step: 1000, unit: '字' },
  material_world_max_runes: { min: 1000, max: 30000, step: 1000, unit: '字' },
  material_conflict_max_runes: { min: 1000, max: 30000, step: 1000, unit: '字' },
  prompt_card_limit: { min: 5, max: 200, step: 5, unit: '条' },
  prompt_card_name_max_runes: { min: 20, max: 500, step: 10, unit: '字' },
  prompt_card_description_max_runes: { min: 80, max: 3000, step: 50, unit: '字' },
  prompt_question_max_runes: { min: 80, max: 2000, step: 50, unit: '字' },
  outline_initial_batch_size: { min: 1, max: 10, step: 1, unit: '章' },
  outline_batch_size: { min: 1, max: 20, step: 1, unit: '章' },
  outline_small_batch_max_tokens: { min: 3000, max: 60000, step: 1000, unit: 'tokens' },
  outline_medium_batch_max_tokens: { min: 6000, max: 90000, step: 1000, unit: 'tokens' },
  outline_large_batch_max_tokens: { min: 9000, max: 120000, step: 1000, unit: 'tokens' },
  batch_retry_attempts: { min: 1, max: 10, step: 1, unit: '次' },
  outline_wait_timeout_minutes: { min: 1, max: 120, step: 1, unit: '分钟' },
  runtime_polling_interval_ms: { min: 500, max: 10000, step: 100, unit: '毫秒' },
  outline_polling_interval_ms: { min: 1000, max: 30000, step: 500, unit: '毫秒' },
  finished_task_retention_minutes: { min: 1, max: 120, step: 1, unit: '分钟' },
  style_template_retry_attempts: { min: 1, max: 10, step: 1, unit: '次' }
}

const generalSettingSections: Array<{
  title: string
  description: string
  fields: Array<{ key: GeneralSettingKey; label: string; bottleneck: string; safety: string; effect?: string }>
}> = [
  {
    title: '小说生成上限',
    description: '限制新建小说、生成大纲和单章正文目标，避免误填超大数值导致生成链路失控。',
    fields: [
      { key: 'max_chapters', label: '小说章节数量最大值', bottleneck: '章节越多，大纲续生成、批量审计和全文核验耗时越长。', safety: '建议 10-300；安全范围 1-1000。' },
      { key: 'max_chapter_words', label: '单章节字数最大值', bottleneck: '单章越长，模型输出 token、等待时间和失败重试成本越高。', safety: '建议 2000-20000；安全范围 1200-200000。' }
    ]
  },
  {
    title: '数据库连接',
    description: '用于记录数据库连接池的期望上限。数据库真实连接仍以启动配置和环境变量优先，调整后建议重启后端。',
    fields: [
      { key: 'db_max_open_conns', label: '最大打开连接数', bottleneck: '过低会让并发保存排队，过高会压垮 MySQL 连接数。', safety: '建议 5-30；安全范围 1-100。', effect: '需重启后端' },
      { key: 'db_max_idle_conns', label: '最大空闲连接数', bottleneck: '过低会频繁建连，过高会长期占用数据库连接。', safety: '建议不超过最大打开连接数；安全范围 1-100。', effect: '需重启后端' },
      { key: 'db_conn_max_lifetime_minutes', label: '连接最大生命周期', bottleneck: '过短会增加重连，过长可能遇到数据库侧断连。', safety: '建议 10-60 分钟；安全范围 1-240。', effect: '需重启后端' },
      { key: 'db_timeout_seconds', label: '数据库连接超时', bottleneck: '过短容易误判慢查询失败，过长会拖住请求线程。', safety: '建议 3-10 秒；安全范围 1-60。', effect: '需重启后端' }
    ]
  },
  {
    title: '请求与 AI 超时',
    description: '控制前端等待、后端 AI 调用和长任务的超时边界。',
    fields: [
      { key: 'frontend_request_timeout_seconds', label: '普通接口前端超时', bottleneck: '过短会让保存/读取在慢网络下失败，过长会让页面等待过久。', safety: '建议 10-30 秒；安全范围 5-120。' },
      { key: 'ai_request_timeout_seconds', label: '普通 AI 接口前端超时', bottleneck: '过短会中断文风、审计等请求，过长会掩盖卡死。', safety: '建议 90-180 秒；安全范围 30-600。' },
      { key: 'ai_long_request_timeout_seconds', label: '长 AI 接口前端超时', bottleneck: '正文、全文核验和整本提炼会受它限制。', safety: '建议 180-600 秒；安全范围 60-1800。' },
      { key: 'default_ai_backend_timeout_seconds', label: '后端默认 AI 超时', bottleneck: '所有未单独声明的 AI 调用会受它限制。', safety: '建议 90-180 秒；安全范围 30-600。' },
      { key: 'chapter_ai_backend_timeout_seconds', label: '章节生成后端超时', bottleneck: '长章节生成常卡在这里，过大则占用后台任务更久。', safety: '建议 180-600 秒；安全范围 60-1800。' },
      { key: 'full_review_ai_backend_timeout_seconds', label: '全文核验/修订后端超时', bottleneck: '全文 payload 大时需要更长等待。', safety: '建议 240-900 秒；安全范围 60-1800。' },
      { key: 'style_template_chapter_timeout_seconds', label: '文风逐章提炼超时', bottleneck: '过短会跳过更多章节，过长会拖慢整本提炼。', safety: '建议 90-240 秒；安全范围 30-900。' },
      { key: 'style_template_summary_timeout_seconds', label: '文风汇总超时', bottleneck: '逐章观察多时汇总会变慢。', safety: '建议 120-360 秒；安全范围 30-1200。' }
    ]
  },
  {
    title: '上下文与输出上限',
    description: '这些参数直接影响提示词大小、上下文风险和模型输出长度。',
    fields: [
      { key: 'chapter_max_tokens', label: '章节生成最大输出 tokens', bottleneck: '越高越可能变慢或触发提供商限制。', safety: '建议 9000-120000；安全范围 9000-200000。' },
      { key: 'style_reference_sample_runes', label: '文风参考抽样字数', bottleneck: '越高越接近全文风格，但文风画像请求越重。', safety: '建议 8000-20000；安全范围 3000-60000。' },
      { key: 'audit_content_max_runes', label: '审计正文裁剪字数', bottleneck: '越高审计更完整，但更容易超上下文。', safety: '建议 8000-30000；安全范围 3000-80000。' },
      { key: 'revision_content_max_runes', label: '审计修订正文裁剪字数', bottleneck: '越高修订依据更完整，但请求更慢。', safety: '建议 8000-30000；安全范围 3000-80000。' },
      { key: 'full_review_payload_max_runes', label: '全文核验 payload 上限', bottleneck: '越高越能覆盖全文，越容易触发上下文或超时。', safety: '建议 80000-220000；安全范围 20000-500000。' },
      { key: 'style_template_chapter_runes', label: '文风单章提炼字数', bottleneck: '越高单章分析更稳，但整本提炼更慢。', safety: '建议 5000-12000；安全范围 2000-30000。' },
      { key: 'style_template_observations_max_runes', label: '文风观察汇总上限', bottleneck: '超过后会走本地压缩汇总，避免撑爆上下文。', safety: '建议 50000-100000；安全范围 10000-200000。' }
    ]
  },
  {
    title: '素材压缩与卡片数量',
    description: '控制素材图谱写入提示词时的压缩比例。',
    fields: [
      { key: 'material_raw_max_runes', label: '灵感素材裁剪字数', bottleneck: '越高越保留细节，但大纲提示词越长。', safety: '建议 3000-8000；安全范围 1000-20000。' },
      { key: 'material_character_max_runes', label: '人物素材裁剪字数', bottleneck: '人物越多越容易挤压剧情空间。', safety: '建议 4000-10000；安全范围 1000-30000。' },
      { key: 'material_world_max_runes', label: '世界观素材裁剪字数', bottleneck: '设定越多越容易降低生成稳定性。', safety: '建议 3000-8000；安全范围 1000-30000。' },
      { key: 'material_conflict_max_runes', label: '冲突素材裁剪字数', bottleneck: '冲突越多，章节规划越容易发散。', safety: '建议 3000-8000；安全范围 1000-30000。' },
      { key: 'prompt_card_limit', label: '提示词卡片数量上限', bottleneck: '卡片越多，事实库越完整但上下文压力越大。', safety: '建议 20-60；安全范围 5-200。' },
      { key: 'prompt_card_name_max_runes', label: '卡片名称字数上限', bottleneck: '名称过长会浪费上下文。', safety: '建议 60-160；安全范围 20-500。' },
      { key: 'prompt_card_description_max_runes', label: '卡片描述字数上限', bottleneck: '描述越长越准确，但更容易挤占正文输出。', safety: '建议 300-800；安全范围 80-3000。' },
      { key: 'prompt_question_max_runes', label: '开放问题字数上限', bottleneck: '问题越长，模型越容易跑偏。', safety: '建议 120-500；安全范围 80-2000。' }
    ]
  },
  {
    title: '大纲批处理与轮询',
    description: '控制后台生成节奏、失败重试和前端刷新频率。',
    fields: [
      { key: 'outline_initial_batch_size', label: '首批大纲章节数', bottleneck: '越大首屏等待越久，越小越快看到结果。', safety: '建议 1-3；安全范围 1-10。' },
      { key: 'outline_batch_size', label: '后续大纲批量章节数', bottleneck: '越大生成吞吐更高，但失败回滚成本更高。', safety: '建议 3-8；安全范围 1-20。' },
      { key: 'outline_small_batch_max_tokens', label: '单章大纲 tokens', bottleneck: '过低会缺字段，过高会变慢。', safety: '建议 6000-12000；安全范围 3000-60000。' },
      { key: 'outline_medium_batch_max_tokens', label: '中批大纲 tokens', bottleneck: '影响 2-3 章批量规划完整度。', safety: '建议 12000-24000；安全范围 6000-90000。' },
      { key: 'outline_large_batch_max_tokens', label: '大批大纲 tokens', bottleneck: '影响 4 章以上批量规划完整度。', safety: '建议 24000-45000；安全范围 9000-120000。' },
      { key: 'batch_retry_attempts', label: '批量任务重试次数', bottleneck: '越高越稳，但失败时等待更久。', safety: '建议 2-4；安全范围 1-10。' },
      { key: 'outline_wait_timeout_minutes', label: '等待大纲完成超时', bottleneck: 'AI一条龙会受它影响，过短会提前失败。', safety: '建议 10-30 分钟；安全范围 1-120。' },
      { key: 'runtime_polling_interval_ms', label: '后台任务轮询间隔', bottleneck: '越短越实时，但请求更频繁。', safety: '建议 1000-3000 毫秒；安全范围 500-10000。' },
      { key: 'outline_polling_interval_ms', label: '大纲轮询间隔', bottleneck: '越短进度更实时，但会增加后端读压力。', safety: '建议 3000-8000 毫秒；安全范围 1000-30000。' }
    ]
  },
  {
    title: '后台任务与登录模式',
    description: '控制任务面板保留时间，并展示登录模式的安全边界。',
    fields: [
      { key: 'finished_task_retention_minutes', label: '已结束任务保留时间', bottleneck: '越长越方便排查，内存保留越多。', safety: '建议 5-30 分钟；安全范围 1-120。' },
      { key: 'style_template_retry_attempts', label: '文风提炼单章重试次数', bottleneck: '越高越稳，但坏章节会拖慢整本提炼。', safety: '建议 2-4；安全范围 1-10。' }
    ]
  }
]

const normalizeGeneralSettings = (value?: Partial<NovelWriterGeneralSettings> | null): NovelWriterGeneralSettings => {
  const source = value || {}
  const result = { ...generalSettingDefaults }
  for (const key of Object.keys(generalSettingDefaults) as GeneralSettingKey[]) {
    const bound = generalSettingBounds[key]
    const rawValue = Number(source[key]) || generalSettingDefaults[key]
    result[key] = Math.min(Math.max(rawValue, bound.min), bound.max)
  }
  if (result.db_max_idle_conns > result.db_max_open_conns) {
    result.db_max_idle_conns = result.db_max_open_conns
  }
  return result
}

const updateGeneralField = (key: GeneralSettingKey, value?: number) => {
  const bound = generalSettingBounds[key]
  const rawValue = Number(value) || generalSettingDefaults[key]
  draft.general[key] = Math.min(Math.max(rawValue, bound.min), bound.max)
  if (key === 'db_max_open_conns' && draft.general.db_max_idle_conns > draft.general.db_max_open_conns) {
    draft.general.db_max_idle_conns = draft.general.db_max_open_conns
  }
}

const normalizeSettings = (value?: NovelWriterSettings | null): NovelWriterSettings => {
  const source = value || {} as NovelWriterSettings
  const aiConfig = source.ai_config || {} as NovelWriterSettings['ai_config']
  return {
    ai_config: {
      deepseek_api_key: aiConfig.deepseek_api_key || '',
      base_url: aiConfig.base_url || '',
      model: aiConfig.model || '',
      provider_groups: Array.isArray(aiConfig.provider_groups)
        ? aiConfig.provider_groups.map((group) => ({
          name: group?.name || '',
          api_key: group?.api_key || '',
          base_url: group?.base_url || '',
          models: Array.isArray(group?.models)
            ? group.models.map((model) => ({
              name: model?.name || '',
              model: model?.model || '',
              capability: model?.capability || 'chat'
            }))
            : []
        }))
        : [],
      providers: Array.isArray(aiConfig.providers)
        ? aiConfig.providers.map((provider) => ({
          name: provider?.name || '',
          api_key: provider?.api_key || '',
          base_url: provider?.base_url || '',
          model: provider?.model || '',
          capability: provider?.capability || 'chat'
        }))
        : []
    },
    general: normalizeGeneralSettings(source.general),
    style_templates: Array.isArray(source.style_templates)
      ? source.style_templates.map((template) => ({
        id: template?.id || '',
        name: template?.name || '',
        description: template?.description || '',
        content: template?.content || '',
        updated_at: template?.updated_at || ''
      }))
      : []
  }
}

const cloneSettings = (value?: NovelWriterSettings | null): NovelWriterSettings => JSON.parse(JSON.stringify(normalizeSettings(value)))

const draft = reactive<NovelWriterSettings>(cloneSettings(props.settings))
const detailDialogVisible = shallowRef(false)
const detailTemplateIndex = shallowRef(-1)
const generatorDialogVisible = shallowRef(false)
const fileInputRef = shallowRef<HTMLInputElement | null>(null)
const selectedSourceFile = shallowRef<File | null>(null)
const generatorForm = reactive({
  name: '',
  description: ''
})

watch(
  () => props.settings,
  (value) => Object.assign(draft, cloneSettings(value)),
  { deep: true, immediate: true }
)

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value)
})

const currentTemplate = computed(() => {
  if (detailTemplateIndex.value < 0) return null
  return draft.style_templates[detailTemplateIndex.value] || null
})

const addProviderGroup = () => {
  draft.ai_config.provider_groups.push({
    name: '',
    api_key: '',
    base_url: '',
    models: []
  })
}

const addGroupModel = (group: AIProviderGroupConfig) => {
  group.models.push({
    name: '',
    model: '',
    capability: 'chat'
  })
}

const addProvider = () => {
  draft.ai_config.providers.push({
    name: '',
    api_key: '',
    base_url: '',
    model: '',
    capability: 'chat'
  })
}

const createTemplate = () => {
  draft.style_templates.unshift({
    id: '',
    name: '',
    description: '',
    content: '',
    updated_at: ''
  })
  detailTemplateIndex.value = 0
  detailDialogVisible.value = true
}

const openTemplateDetail = (index: number) => {
  detailTemplateIndex.value = index
  detailDialogVisible.value = true
}

const removeItem = <T>(list: T[], index: number) => {
  list.splice(index, 1)
}

const removeTemplate = (index: number) => {
  removeItem(draft.style_templates, index)
  if (detailTemplateIndex.value === index) {
    detailDialogVisible.value = false
    detailTemplateIndex.value = -1
  }
}

const templatePreview = (content: string) => {
  const text = String(content || '').replace(/\s+/g, ' ').trim()
  if (!text) return '尚未填写完整内容'
  return text.length > 80 ? `${text.slice(0, 80)}...` : text
}

const openGeneratorDialog = () => {
  generatorForm.name = ''
  generatorForm.description = ''
  selectedSourceFile.value = null
  if (fileInputRef.value) fileInputRef.value.value = ''
  generatorDialogVisible.value = true
}

const triggerFileSelect = () => {
  fileInputRef.value?.click()
}

const handleFileChange = (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0] || null
  if (!file) {
    selectedSourceFile.value = null
    return
  }

  const isTxtFile = file.type === 'text/plain' || file.name.toLowerCase().endsWith('.txt')
  if (!isTxtFile) {
    selectedSourceFile.value = null
    input.value = ''
    ElMessage.warning('暂时只支持选择 .txt 格式文件')
    return
  }

  selectedSourceFile.value = file
}

const formatFileSize = (size: number) => {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

const startGenerateTemplate = async () => {
  if (!selectedSourceFile.value) {
    ElMessage.warning('请先选择一个 .txt 文件')
    return
  }

  try {
    const sourceText = await selectedSourceFile.value.text()
    if (!sourceText.trim()) {
      ElMessage.warning('当前文件内容为空，请重新选择')
      return
    }
    emit('generateTemplate', {
      name: generatorForm.name.trim(),
      description: generatorForm.description.trim(),
      source_text: sourceText
    })
    generatorDialogVisible.value = false
  } catch (error) {
    console.error('read style template source file failed', error)
    ElMessage.error('读取文件失败，请重新选择后再试')
  }
}

const saveSettings = () => {
  emit('save', cloneSettings(draft))
}
</script>

<template>
  <el-dialog
    v-model="visible"
    title="创作设置"
    width="1080px"
    append-to-body
    class="writer-settings-dialog"
  >
    <div v-loading="loading" class="writer-settings">
      <el-tabs class="writer-settings__tabs">
        <el-tab-pane label="常规设置">
          <div
            v-for="section in generalSettingSections"
            :key="section.title"
            class="settings-block"
          >
            <div class="settings-block__head">
              <div>
                <h4>{{ section.title }}</h4>
                <p>{{ section.description }}</p>
              </div>
            </div>
            <div class="settings-grid settings-grid--advanced">
              <label
                v-for="field in section.fields"
                :key="field.key"
                class="settings-field settings-field--advanced"
              >
                <span class="settings-field__label">
                  {{ field.label }}
                  <em v-if="field.effect">{{ field.effect }}</em>
                </span>
                <el-input-number
                  :model-value="draft.general[field.key]"
                  :min="generalSettingBounds[field.key].min"
                  :max="field.key === 'db_max_idle_conns' ? draft.general.db_max_open_conns : generalSettingBounds[field.key].max"
                  :step="generalSettingBounds[field.key].step"
                  controls-position="right"
                  @update:model-value="(value: number | undefined) => updateGeneralField(field.key, value)"
                />
                <p class="settings-field__meta">
                  <strong>性能瓶颈：</strong>{{ field.bottleneck }}
                </p>
                <p class="settings-field__meta">
                  <strong>安全范围：</strong>{{ field.safety }}
                  <span v-if="generalSettingBounds[field.key].unit">当前单位：{{ generalSettingBounds[field.key].unit }}</span>
                </p>
              </label>
            </div>
            <div v-if="section.title === '后台任务与登录模式'" class="settings-notice">
              登录模块是否禁用仍由 <code>NOVEL_GENERATER_AUTH_DISABLED</code> 控制，不在页面内开放切换。性能瓶颈是本地调试方便性与线上访问安全之间的取舍；安全范围建议线上保持启用登录，本地开发才禁用。
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="API Key 配置">
          <div class="settings-block">
            <div class="settings-block__head">
              <div>
                <h4>兼容旧配置</h4>
                <p>保留现有回退配置，避免本地旧流程失效。</p>
              </div>
            </div>
            <div class="settings-grid settings-grid--triple">
              <el-input v-model="draft.ai_config.deepseek_api_key" type="password" show-password placeholder="默认 API Key" />
              <el-input v-model="draft.ai_config.base_url" placeholder="默认 Base URL" />
              <el-input v-model="draft.ai_config.model" placeholder="默认模型" />
            </div>
          </div>

          <div class="settings-block">
            <div class="settings-block__head">
              <div>
                <h4>提供商分组</h4>
                <p>适合一组模型共用同一个 API Key 与 Base URL。</p>
              </div>
              <el-button type="primary" plain @click="addProviderGroup">
                <el-icon><Plus /></el-icon>
                新增分组
              </el-button>
            </div>

            <div v-if="draft.ai_config.provider_groups.length" class="settings-stack">
              <div v-for="(group, groupIndex) in draft.ai_config.provider_groups" :key="`group-${groupIndex}`" class="settings-card">
                <div class="settings-card__toolbar">
                  <strong>分组 {{ groupIndex + 1 }}</strong>
                  <el-button text type="danger" @click="removeItem(draft.ai_config.provider_groups, groupIndex)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </div>
                <div class="settings-grid settings-grid--triple">
                  <el-input v-model="group.name" placeholder="分组名称" />
                  <el-input v-model="group.base_url" placeholder="Base URL" />
                  <el-input v-model="group.api_key" type="password" show-password placeholder="API Key" />
                </div>

                <div class="settings-subhead">
                  <span>模型列表</span>
                  <el-button size="small" plain @click="addGroupModel(group)">
                    <el-icon><Plus /></el-icon>
                    新增模型
                  </el-button>
                </div>

                <div v-if="group.models.length" class="settings-stack">
                  <div v-for="(model, modelIndex) in group.models" :key="`model-${groupIndex}-${modelIndex}`" class="settings-row">
                    <el-input v-model="model.name" placeholder="显示名称" />
                    <el-input v-model="model.model" placeholder="模型标识" />
                    <el-select v-model="model.capability" placeholder="能力">
                      <el-option label="通用对话" value="chat" />
                      <el-option label="小说写作" value="novel" />
                      <el-option label="日常写作" value="casual_chat_pro" />
                    </el-select>
                    <el-button text type="danger" @click="removeItem(group.models, modelIndex)">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </div>
                </div>
                <el-empty v-else description="暂无模型" :image-size="48" />
              </div>
            </div>
            <el-empty v-else description="暂无提供商分组" :image-size="56" />
          </div>

          <div class="settings-block">
            <div class="settings-block__head">
              <div>
                <h4>独立提供商</h4>
                <p>适合单个模型单独配置。</p>
              </div>
              <el-button type="primary" plain @click="addProvider">
                <el-icon><Plus /></el-icon>
                新增提供商
              </el-button>
            </div>

            <div v-if="draft.ai_config.providers.length" class="settings-stack">
              <div v-for="(provider, providerIndex) in draft.ai_config.providers" :key="`provider-${providerIndex}`" class="settings-row settings-row--provider">
                <el-input v-model="provider.name" placeholder="名称" />
                <el-input v-model="provider.base_url" placeholder="Base URL" />
                <el-input v-model="provider.model" placeholder="模型标识" />
                <el-select v-model="provider.capability" placeholder="能力">
                  <el-option label="通用对话" value="chat" />
                  <el-option label="小说写作" value="novel" />
                  <el-option label="日常写作" value="casual_chat_pro" />
                </el-select>
                <el-input v-model="provider.api_key" type="password" show-password placeholder="API Key" />
                <el-button text type="danger" @click="removeItem(draft.ai_config.providers, providerIndex)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </div>
            </div>
            <el-empty v-else description="暂无独立提供商" :image-size="56" />
          </div>
        </el-tab-pane>

        <el-tab-pane label="文风模版">
          <div class="settings-block">
            <div class="settings-block__head">
              <div>
                <h4>文风模版书架</h4>
                <p>点击卡片查看完整内容；也可以直接用整本参考小说逐章提炼出一份精华版文风模版。</p>
              </div>
              <div class="settings-block__actions">
                <el-button
                  type="warning"
                  :loading="templateGenerationLoading"
                  @click="openGeneratorDialog"
                >
                  <el-icon><Reading /></el-icon>
                  {{ templateGenerationLoading ? (templateGenerationLabel || '整本小说提炼中') : '整本小说提炼' }}
                </el-button>
                <el-button type="primary" plain @click="createTemplate">
                  <el-icon><Plus /></el-icon>
                  新增模版
                </el-button>
              </div>
            </div>

            <div v-if="draft.style_templates.length" class="template-shelf">
              <article
                v-for="(template, index) in draft.style_templates"
                :key="template.id || `template-${index}`"
                class="template-card-shell"
              >
                <button class="template-card" type="button" @click="openTemplateDetail(index)">
                  <div class="template-card__hero">
                    <div class="template-card__cover">
                      <div class="template-card__cover-content">
                        <span>文风模版</span>
                        <strong>{{ template.name || '未命名模版' }}</strong>
                        <small>{{ template.updated_at || '待保存' }}</small>
                      </div>
                    </div>
                    <div class="template-card__body">
                      <strong class="template-card__title">{{ template.name || '未命名模版' }}</strong>
                      <p class="template-card__subtitle">{{ template.description || '点击查看完整文风内容与编辑入口' }}</p>
                      <div class="template-card__stats">
                        <span>{{ template.content ? '已录入内容' : '待补充内容' }}</span>
                        <span>{{ (template.content || '').length }} 字</span>
                      </div>
                      <p class="template-card__summary">{{ templatePreview(template.content) }}</p>
                      <div class="template-card__cta">
                        <span>查看全文</span>
                      </div>
                    </div>
                  </div>
                  <div class="template-card__footer">
                    <span class="template-card__stage">
                      <i class="template-card__stage-dot" />
                      {{ template.updated_at || '待保存' }}
                    </span>
                    <button class="template-card__action" type="button" @click.stop="removeTemplate(index)">删除</button>
                  </div>
                </button>
              </article>
            </div>
            <el-empty v-else description="暂无文风模版" :image-size="64" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <template #footer>
      <div class="writer-settings__footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveSettings">保存设置</el-button>
      </div>
    </template>
  </el-dialog>

  <el-dialog
    v-model="detailDialogVisible"
    title="文风模版详情"
    width="760px"
    append-to-body
  >
    <template v-if="currentTemplate">
      <div class="template-detail">
        <div class="settings-grid settings-grid--double">
          <el-input v-model="currentTemplate.name" placeholder="模版名称" />
          <el-input v-model="currentTemplate.description" placeholder="一句话说明（可选）" />
        </div>
        <el-input
          v-model="currentTemplate.content"
          type="textarea"
          :rows="16"
          resize="vertical"
          placeholder="输入完整文风模版内容..."
        />
      </div>
    </template>
  </el-dialog>

  <el-dialog
    v-model="generatorDialogVisible"
    title="整本参考小说提炼文风模版"
    width="820px"
    append-to-body
  >
    <div class="template-detail">
      <div class="dialog-tip dialog-tip--accent">
        <el-icon><Reading /></el-icon>
        <span>系统会按章节逐章提炼文风特征，再汇总成一份精华版文风模版。处理期间可在右下角后台任务球里查看进度。</span>
      </div>
      <div class="settings-grid settings-grid--double">
        <el-input v-model="generatorForm.name" placeholder="模版名称（可选）" />
        <el-input v-model="generatorForm.description" placeholder="模版说明（可选）" />
      </div>
      <input
        ref="fileInputRef"
        type="file"
        accept=".txt,text/plain"
        class="file-input-hidden"
        @change="handleFileChange"
      >
      <div class="file-picker-card">
        <div class="file-picker-card__info">
          <strong>{{ selectedSourceFile ? selectedSourceFile.name : '尚未选择参考小说文件' }}</strong>
          <p>
            {{ selectedSourceFile
              ? `已选择 TXT 文件 · ${formatFileSize(selectedSourceFile.size)}`
              : '点击右侧按钮选择整本参考小说的 .txt 文件，确认后会自动读取内容并开始提炼。' }}
          </p>
        </div>
        <el-button plain @click="triggerFileSelect">
          选择 TXT 文件
        </el-button>
      </div>
    </div>

    <template #footer>
      <div class="writer-settings__footer">
        <el-button @click="generatorDialogVisible = false">取消</el-button>
        <el-button type="warning" :loading="templateGenerationLoading" @click="startGenerateTemplate">
          {{ templateGenerationLoading ? (templateGenerationLabel || '提炼中') : '开始提炼' }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.writer-settings {
  min-height: 420px;
}

.writer-settings__tabs {
  min-width: 0;
}

.settings-block {
  display: grid;
  gap: 16px;
}

.settings-block + .settings-block {
  margin-top: 20px;
}

.settings-block__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.settings-block__actions {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.settings-block__head h4 {
  margin: 0;
  color: #0f172a;
  font-size: 16px;
}

.settings-block__head p {
  margin: 6px 0 0;
  color: #64748b;
  font-size: 13px;
  line-height: 1.6;
}

.settings-grid {
  display: grid;
  gap: 12px;
}

.settings-grid--triple {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.settings-grid--double {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.settings-grid--advanced {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: stretch;
}

.settings-field {
  display: grid;
  gap: 8px;
  color: #334155;
  font-size: 13px;
  font-weight: 700;
}

.settings-field--advanced {
  align-content: start;
  padding: 14px;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  background: #f8fafc;
}

.settings-field__label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.settings-field__label em {
  flex: 0 0 auto;
  padding: 2px 7px;
  border-radius: 999px;
  background: #fff7ed;
  color: #c2410c;
  font-size: 11px;
  font-style: normal;
  font-weight: 800;
}

.settings-field__meta {
  margin: 0;
  color: #64748b;
  font-size: 12px;
  font-weight: 500;
  line-height: 1.5;
}

.settings-field__meta strong {
  color: #0f766e;
}

.settings-field__meta span {
  display: block;
  margin-top: 2px;
  color: #94a3b8;
}

.settings-field :deep(.el-input-number) {
  width: 100%;
}

.settings-notice {
  padding: 12px 14px;
  border: 1px solid #bae6fd;
  border-radius: 10px;
  background: #f0f9ff;
  color: #0369a1;
  font-size: 13px;
  line-height: 1.6;
}

.settings-notice code {
  color: #075985;
  font-weight: 800;
}

.settings-stack {
  display: grid;
  gap: 14px;
}

.settings-card {
  display: grid;
  gap: 14px;
  padding: 16px;
  border: 1px solid #e2e8f0;
  border-radius: 18px;
  background: #f8fafc;
}

.settings-card__toolbar,
.settings-subhead {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.settings-card__toolbar strong,
.settings-subhead span {
  color: #0f172a;
  font-size: 14px;
  font-weight: 700;
}

.settings-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) 180px auto;
  gap: 12px;
  align-items: center;
}

.settings-row--provider {
  grid-template-columns: 150px minmax(0, 1.2fr) minmax(0, 1fr) 140px minmax(0, 1fr) auto;
}

.writer-settings__footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.template-shelf {
  display: grid;
  grid-template-columns: repeat(3, minmax(260px, 1fr));
  gap: 28px 20px;
}

.template-card-shell {
  height: 291px;
  min-width: 0;
}

.template-card {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  padding: 0;
  border: 0;
  border-radius: 10px;
  background: #eef2f6;
  text-align: left;
  cursor: pointer;
  overflow: hidden;
  color: #64748b;
  transition: 0.2s ease;
}

.template-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 18px 40px rgba(15, 23, 42, 0.12);
}

.template-card__hero {
  position: relative;
  display: flex;
  height: 220px;
  min-height: 0;
  padding: 28px 28px 0 28px;
  background: #f6a3ab;
  border-radius: 10px 10px 0 0;
  overflow: visible;
}

.template-card-shell:nth-child(5n + 2) .template-card__hero {
  background: #9edce8;
}

.template-card-shell:nth-child(5n + 3) .template-card__hero {
  background: #e7b7d4;
}

.template-card-shell:nth-child(5n + 4) .template-card__hero {
  background: #f7ca96;
}

.template-card-shell:nth-child(5n) .template-card__hero {
  background: #c8b3e3;
}

.template-card__cover {
  position: relative;
  z-index: 1;
  width: 165px;
  height: 248px;
  flex-shrink: 0;
  overflow: hidden;
  margin: 0 0 -50px;
  border-radius: 3px;
  background:
    radial-gradient(circle at 18% 22%, rgba(86, 204, 242, 0.95) 0 18%, transparent 32%),
    radial-gradient(circle at 78% 45%, rgba(255, 214, 80, 0.92) 0 16%, transparent 31%),
    radial-gradient(circle at 42% 62%, rgba(243, 84, 127, 0.9) 0 22%, transparent 38%),
    linear-gradient(160deg, #dcecff 0%, #f7bdd0 38%, #ff7f5f 64%, #692348 100%);
  box-shadow: -2px 6px 19px 0 #7f818e;
}

.template-card__cover-content {
  position: relative;
  z-index: 1;
  display: flex;
  min-width: 0;
  height: 100%;
  flex-direction: column;
  justify-content: center;
  padding: 18px 14px;
  color: #ffffff;
  text-align: center;
}

.template-card__cover-content span,
.template-card__cover-content small {
  position: absolute;
  left: 14px;
  right: 14px;
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.template-card__cover-content span {
  top: 18px;
}

.template-card__cover-content small {
  bottom: 16px;
}

.template-card__cover-content strong {
  display: block;
  font-size: 24px;
  line-height: 1.25;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  word-break: break-word;
}

.template-card__body {
  display: grid;
  grid-template-rows: auto auto auto 1fr auto;
  align-content: start;
  min-width: 0;
  flex: 1;
  padding-left: 22px;
  max-height: 180px;
  overflow: hidden;
}

.template-card__title {
  display: block;
  margin-top: 6px;
  color: #0f172a;
  font-size: 22px;
  line-height: 1.2;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  word-break: break-word;
}

.template-card__subtitle,
.template-card__summary {
  margin: 10px 0 0;
  color: #475569;
  font-size: 13px;
  line-height: 1.6;
}

.template-card__summary {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
}

.template-card__subtitle {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
}

.template-card__stats {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 12px;
  max-height: 26px;
  overflow: hidden;
}

.template-card__stats span {
  display: inline-flex;
  align-items: center;
  min-height: 26px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.78);
  color: #0f766e;
  font-size: 11px;
  font-weight: 800;
}

.template-card__cta {
  margin-top: 10px;
  color: #0f766e;
  font-size: 13px;
  font-weight: 800;
}

.template-card__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 24px 18px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-top: none;
  border-radius: 0 0 10px 10px;
}

.template-card__stage {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: #475569;
  font-size: 12px;
  font-weight: 700;
}

.template-card__stage-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #14b8a6;
}

.template-card__action {
  border: 0;
  background: transparent;
  color: #dc2626;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
}

.template-detail {
  display: grid;
  gap: 14px;
}

.dialog-tip {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  color: #475569;
  font-size: 13px;
  line-height: 1.6;
}

.dialog-tip--accent {
  background: #fff7ed;
  border-color: #fed7aa;
  color: #9a3412;
}

.file-input-hidden {
  display: none;
}

.file-picker-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
  border: 1px solid #dbe4ee;
  border-radius: 14px;
  background: #f8fafc;
}

.file-picker-card__info {
  min-width: 0;
}

.file-picker-card__info strong {
  display: block;
  color: #0f172a;
  font-size: 14px;
  line-height: 1.5;
  word-break: break-all;
}

.file-picker-card__info p {
  margin: 6px 0 0;
  color: #64748b;
  font-size: 13px;
  line-height: 1.6;
}

@media (max-width: 1200px) {
  .template-shelf {
    grid-template-columns: repeat(2, minmax(260px, 1fr));
  }
}

@media (max-width: 960px) {
  .settings-grid--triple,
  .settings-grid--double,
  .settings-row,
  .settings-row--provider,
  .template-shelf {
    grid-template-columns: 1fr;
  }

  .template-card__hero {
    flex-direction: column;
    gap: 18px;
    min-height: 0;
  }

  .template-card__cover {
    margin-bottom: 0;
  }

  .template-card__body {
    padding-left: 0;
  }
}
</style>
