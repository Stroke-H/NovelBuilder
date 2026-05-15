<script setup lang="ts">
import { computed, reactive, shallowRef, watch } from 'vue'
import { Delete, Plus, Reading } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { AIProviderGroupConfig, GenerateStyleTemplatePayload, NovelWriterSettings } from '@/api/novelWriter'

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
const generatorForm = reactive<GenerateStyleTemplatePayload>({
  name: '',
  description: '',
  source_text: ''
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
  generatorForm.source_text = ''
  generatorDialogVisible.value = true
}

const startGenerateTemplate = () => {
  if (!generatorForm.source_text.trim()) {
    ElMessage.warning('请先粘贴整本参考小说内容')
    return
  }
  emit('generateTemplate', {
    name: generatorForm.name.trim(),
    description: generatorForm.description.trim(),
    source_text: generatorForm.source_text
  })
  generatorDialogVisible.value = false
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
      <el-input
        v-model="generatorForm.source_text"
        type="textarea"
        :rows="20"
        resize="vertical"
        placeholder="在这里粘贴整本参考小说内容..."
      />
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
  min-height: 291px;
}

.template-card {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
  min-height: 291px;
  padding: 0;
  border: 0;
  border-radius: 10px;
  background: #eef2f6;
  text-align: left;
  cursor: pointer;
  overflow: visible;
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
  min-height: 220px;
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
  word-break: break-word;
}

.template-card__body {
  min-width: 0;
  flex: 1;
  padding-left: 22px;
}

.template-card__title {
  display: block;
  margin-top: 6px;
  color: #0f172a;
  font-size: 22px;
  line-height: 1.2;
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
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.template-card__stats {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 12px;
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
  margin-top: 14px;
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
