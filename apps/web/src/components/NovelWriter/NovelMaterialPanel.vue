<template>
  <section class="material-panel">
    <div class="panel-inner">
      <div class="panel-header">
        <div class="header-info">
          <div class="step-badge">Step 2</div>
          <div class="title-row">
            <h3 class="panel-title">文风生成准备</h3>
            <span :class="['reference-status', { 'reference-status--ready': hasReference }]">
              <span class="status-dot"></span>
              {{ hasReference ? '已提供参考' : '无参考' }}
            </span>
          </div>
          <p class="panel-desc">
            文风参考并非必填；如果留空，AI 会根据题材自动匹配最合适的创作风格。
          </p>
        </div>
        <div class="panel-actions">
          <div class="panel-actions__group">
            <el-button class="action-btn" :loading="saving" @click="emit('save')">
              <el-icon><DocumentChecked /></el-icon>
              保存素材
            </el-button>
            <el-button class="action-btn" @click="openReferenceDialog">
              <el-icon><EditPen /></el-icon>
              {{ hasReference ? '编辑文风参考' : '添加文风参考' }}
            </el-button>
          </div>
          <div class="panel-actions__stack">
            <el-button type="primary" class="primary-action-btn" @click="emit('style')">
              <el-icon><MagicStick /></el-icon>
              生成文风画像
            </el-button>
            <el-button type="success" plain class="outline-btn" @click="emit('outline')">
              <el-icon><Memo /></el-icon>
              生成大纲/章节结构
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <el-dialog
      v-model="referenceDialogVisible"
      title="添加文风参考文本"
      width="760px"
      custom-class="premium-dialog"
      append-to-body
    >
      <div class="reference-dialog">
        <div class="dialog-tip">
          <el-icon><InfoFilled /></el-icon>
          <span>上传或粘贴参考片段，AI 仅提取抽象文风规则，不会复制任何具体内容。</span>
        </div>
        <div class="upload-area">
          <label class="file-upload-trigger">
            <el-icon><Upload /></el-icon>
            <span>点击上传 .txt 文件</span>
            <input type="file" accept=".txt,text/plain" @change="handleReferenceFile">
          </label>
        </div>
        <el-input
          v-model="referenceDraft"
          type="textarea"
          :rows="14"
          placeholder="在此粘贴参考小说片段..."
          resize="none"
          class="premium-textarea"
        />
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="referenceDialogVisible = false">取消</el-button>
          <el-button type="primary" class="footer-confirm-btn" @click="applyReferenceText">确认使用</el-button>
        </div>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, shallowRef, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { DocumentChecked, EditPen, MagicStick, Memo, InfoFilled, Upload } from '@element-plus/icons-vue'
import type { NovelMaterials } from '@/api/novelWriter'

const props = defineProps<{
  modelValue: NovelMaterials
  saving: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: NovelMaterials]
  save: []
  outline: []
  style: []
}>()

const form = reactive<NovelMaterials>({ ...props.modelValue })
const referenceDialogVisible = shallowRef(false)
const referenceDraft = shallowRef('')
const hasReference = computed(() => Boolean(form.reference_raw.trim()))

watch(
  () => props.modelValue,
  (value) => Object.assign(form, value),
  { deep: true }
)

watch(
  form,
  () => emit('update:modelValue', { ...form }),
  { deep: true }
)

const openReferenceDialog = () => {
  referenceDraft.value = form.reference_raw || ''
  referenceDialogVisible.value = true
}

const applyReferenceText = () => {
  form.reference_raw = referenceDraft.value
  referenceDialogVisible.value = false
  ElMessage.success(form.reference_raw.trim() ? '文风参考文本已添加' : '已清空文风参考文本')
}

const handleReferenceFile = (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (!file.name.toLowerCase().endsWith('.txt')) {
    ElMessage.warning('目前仅支持上传 .txt 文本文件')
    return
  }

  const reader = new FileReader()
  reader.onload = () => {
    referenceDraft.value = String(reader.result || '')
    form.reference_raw = referenceDraft.value
    ElMessage.success('已读取 .txt 文件内容作为文风参考')
  }
  reader.onerror = () => ElMessage.error('读取文件失败，请重试')
  reader.readAsText(file)
}

defineExpose({
  openReferenceDialog
})
</script>

<style scoped>
.material-panel {
  background: linear-gradient(135deg, #ffffff 0%, #f8fafc 100%);
  border: 1px solid rgba(226, 232, 240, 0.8);
  border-radius: 28px;
  padding: 2px;
  box-shadow: 0 4px 20px -4px rgba(15, 23, 42, 0.05);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.panel-inner {
  background: #ffffff;
  border-radius: 26px;
  padding: 24px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 24px;
}

.step-badge {
  display: inline-flex;
  padding: 4px 12px;
  background: #f0fdfa;
  color: #0d9488;
  font-size: 11px;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  border-radius: 8px;
  margin-bottom: 12px;
}

.title-row {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 8px;
}

.panel-title {
  margin: 0;
  color: #0f172a;
  font-size: 22px;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.reference-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  background: #f1f5f9;
  border: 1px solid #e2e8f0;
  border-radius: 99px;
  font-size: 12px;
  font-weight: 600;
  color: #64748b;
  transition: all 0.3s ease;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #94a3b8;
}

.reference-status--ready {
  background: #ecfdf5;
  border-color: #a7f3d0;
  color: #059669;
}

.reference-status--ready .status-dot {
  background: #10b981;
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.5);
  animation: status-pulse 2s infinite;
}

@keyframes status-pulse {
  0% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(1.2); }
  100% { opacity: 1; transform: scale(1); }
}

.panel-desc {
  margin: 0;
  color: #64748b;
  font-size: 14px;
  line-height: 1.6;
  max-width: 520px;
}

.panel-actions {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.panel-actions__group {
  display: flex;
  gap: 12px;
}

.panel-actions__stack {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.action-btn {
  border-radius: 12px;
  height: 40px;
  font-weight: 600;
  transition: all 0.2s ease;
}

.primary-action-btn {
  border-radius: 12px;
  height: 40px;
  font-weight: 600;
  background: linear-gradient(135deg, #0d9488 0%, #0f766e 100%);
  border: none;
  box-shadow: 0 4px 12px rgba(13, 148, 136, 0.2);
}

.primary-action-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 16px rgba(13, 148, 136, 0.3);
}

.outline-btn {
  border-radius: 12px;
  font-weight: 600;
  height: 40px;
  padding: 0 20px;
}

.dialog-tip {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: #f0f9ff;
  border: 1px solid #bae6fd;
  border-radius: 12px;
  color: #0369a1;
  font-size: 13px;
  margin-bottom: 20px;
}

.upload-area {
  margin-bottom: 16px;
}

.file-upload-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: #ffffff;
  border: 1.5px dashed #cbd5e1;
  border-radius: 10px;
  color: #475569;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.file-upload-trigger:hover {
  border-color: #0d9488;
  color: #0d9488;
  background: #f0fdfa;
}

.file-upload-trigger input {
  display: none;
}

.premium-textarea :deep(.el-textarea__inner) {
  border-radius: 16px;
  padding: 16px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  font-size: 14px;
  line-height: 1.6;
  transition: all 0.2s ease;
}

.premium-textarea :deep(.el-textarea__inner:focus) {
  background: #ffffff;
  box-shadow: 0 0 0 4px rgba(13, 148, 136, 0.08);
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 12px;
}

.footer-confirm-btn {
  border-radius: 10px;
  padding: 0 24px;
  font-weight: 600;
}

@media (max-width: 900px) {
  .panel-header {
    flex-direction: column;
    gap: 20px;
  }

  .panel-actions {
    width: 100%;
    flex-direction: column;
    align-items: stretch;
  }

  .panel-actions__group,
  .panel-actions__stack {
    width: 100%;
  }

  .panel-actions__group {
    flex-wrap: wrap;
  }
}
</style>
