<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { Delete, Plus } from '@element-plus/icons-vue'
import type {
  AIProviderGroupConfig,
  NovelWriterSettings
} from '@/api/novelWriter'

const props = defineProps<{
  modelValue: boolean
  settings: NovelWriterSettings
  loading: boolean
  saving: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [value: NovelWriterSettings]
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

watch(
  () => props.settings,
  (value) => Object.assign(draft, cloneSettings(value)),
  { deep: true, immediate: true }
)

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value)
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

const addStyleTemplate = () => {
  draft.style_templates.push({
    id: '',
    name: '',
    description: '',
    content: '',
    updated_at: ''
  })
}

const removeItem = <T>(list: T[], index: number) => {
  list.splice(index, 1)
}

const saveSettings = () => {
  emit('save', cloneSettings(draft))
}
</script>

<template>
  <el-dialog
    v-model="visible"
    title="创作设置"
    width="980px"
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
                <h4>文风模版库</h4>
                <p>这些模版会出现在“编辑文风参考”弹窗的下拉栏里，便于快速带入常用风格。</p>
              </div>
              <el-button type="primary" plain @click="addStyleTemplate">
                <el-icon><Plus /></el-icon>
                新增模版
              </el-button>
            </div>

            <div v-if="draft.style_templates.length" class="settings-stack">
              <div v-for="(template, index) in draft.style_templates" :key="template.id || `template-${index}`" class="settings-card">
                <div class="settings-card__toolbar">
                  <strong>模版 {{ index + 1 }}</strong>
                  <el-button text type="danger" @click="removeItem(draft.style_templates, index)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </div>
                <div class="settings-grid settings-grid--double">
                  <el-input v-model="template.name" placeholder="模版名称" />
                  <el-input v-model="template.description" placeholder="一句话说明（可选）" />
                </div>
                <el-input
                  v-model="template.content"
                  type="textarea"
                  :rows="8"
                  resize="vertical"
                  placeholder="输入这份文风模版的参考内容..."
                />
              </div>
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

@media (max-width: 960px) {
  .settings-grid--triple,
  .settings-grid--double,
  .settings-row,
  .settings-row--provider {
    grid-template-columns: 1fr;
  }
}
</style>
