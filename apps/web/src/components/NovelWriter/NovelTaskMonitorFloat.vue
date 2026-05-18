<script setup lang="ts">
import { computed, shallowRef } from 'vue'
import { CloseBold, Loading, Operation } from '@element-plus/icons-vue'
import type { NovelRuntimeTask } from '@/api/novelWriter'

const props = defineProps<{
  tasks: NovelRuntimeTask[]
  loading: boolean
  cancelLoadingTaskId: string
}>()

const emit = defineEmits<{
  refresh: []
  cancel: [taskId: string]
}>()

const open = shallowRef(false)

const isTaskActive = (task: NovelRuntimeTask) => task.status === 'running' || task.status === 'cancelling'
const runningCount = computed(() => props.tasks.filter(isTaskActive).length)

const panelTitle = computed(() => {
  if (runningCount.value === 0) return '后台任务'
  return `后台任务（${runningCount.value}）`
})

const statusText = (status: string) => {
  if (status === 'cancelling') return '终止中'
  if (status === 'completed') return '已完成'
  if (status === 'failed') return '失败'
  if (status === 'cancelled') return '已终止'
  return '执行中'
}

const toggleOpen = () => {
  open.value = !open.value
  if (open.value) emit('refresh')
}
</script>

<template>
  <teleport to="body">
    <div class="task-float">
      <transition name="task-panel-fade">
        <section v-if="open" class="task-panel">
          <header class="task-panel__header">
            <div>
              <p class="task-panel__kicker">任务监控</p>
              <h3>{{ panelTitle }}</h3>
            </div>
            <el-button text class="task-panel__refresh" @click="emit('refresh')">
              刷新
            </el-button>
          </header>

          <div v-if="tasks.length" class="task-list">
            <article
              v-for="task in tasks"
              :key="task.id"
              :class="['task-item', `task-item--${task.status}`]"
            >
              <div class="task-item__meta">
                <span class="task-item__status">{{ statusText(task.status) }}</span>
                <span class="task-item__time">{{ task.finished_at || task.started_at }}</span>
              </div>
              <strong class="task-item__title">{{ task.title }}</strong>
              <p v-if="task.project_title" class="task-item__project">{{ task.project_title }}</p>
              <p v-if="task.error" class="task-item__error">{{ task.error }}</p>
              <div class="task-item__actions">
                <el-button
                  size="small"
                  type="danger"
                  plain
                  :loading="cancelLoadingTaskId === task.id"
                  :disabled="!isTaskActive(task) || task.status === 'cancelling'"
                  @click="emit('cancel', task.id)"
                >
                  {{ isTaskActive(task) ? (task.status === 'cancelling' ? '终止中' : '结束进程') : '已结束' }}
                </el-button>
              </div>
            </article>
          </div>

          <el-empty v-else description="当前没有后台任务" />
        </section>
      </transition>

      <button
        type="button"
        :class="['task-fab', { 'task-fab--active': open }]"
        @click="toggleOpen"
      >
        <el-icon v-if="loading"><Loading /></el-icon>
        <el-icon v-else-if="open"><CloseBold /></el-icon>
        <el-icon v-else><Operation /></el-icon>
        <span class="task-fab__count">{{ runningCount }}</span>
      </button>
    </div>
  </teleport>
</template>

<style scoped>
.task-float {
  position: fixed;
  right: 20px;
  bottom: 20px;
  z-index: 80;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 10px;
  font-size: 14px;
  line-height: 1.4;
}

.task-fab {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  border: 1px solid rgba(255, 255, 255, 0.72);
  border-radius: 999px;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.92), rgba(236, 254, 255, 0.74)),
    rgba(255, 255, 255, 0.76);
  color: #0f766e;
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.92),
    0 18px 44px rgba(15, 23, 42, 0.16);
  backdrop-filter: blur(18px) saturate(150%);
  -webkit-backdrop-filter: blur(18px) saturate(150%);
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.task-fab:hover {
  transform: translateY(-1px) scale(1.02);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.92),
    0 22px 52px rgba(15, 23, 42, 0.2);
}

.task-fab--active {
  color: #0f172a;
}

.task-fab :deep(.el-icon) {
  font-size: 20px;
}

.task-fab__count {
  position: absolute;
  top: -4px;
  right: -2px;
  min-width: 22px;
  height: 22px;
  padding: 0 6px;
  border-radius: 999px;
  background: #14b8a6;
  color: #ffffff;
  font-size: 12px;
  font-weight: 800;
  line-height: 22px;
  text-align: center;
  box-shadow: 0 8px 18px rgba(20, 184, 166, 0.35);
}

.task-panel {
  width: min(332px, calc(100vw - 28px));
  max-height: min(66vh, 520px);
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.74);
  border-radius: 22px;
  background:
    linear-gradient(145deg, rgba(255, 255, 255, 0.94), rgba(248, 250, 252, 0.88)),
    rgba(255, 255, 255, 0.84);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.92),
    0 24px 60px rgba(15, 23, 42, 0.18);
  backdrop-filter: blur(20px) saturate(150%);
  -webkit-backdrop-filter: blur(20px) saturate(150%);
}

.task-panel__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 16px 12px;
  border-bottom: 1px solid rgba(226, 232, 240, 0.84);
}

.task-panel__kicker {
  margin: 0 0 4px;
  color: #0f766e;
  font-size: 12px;
  font-weight: 800;
}

.task-panel__header h3 {
  margin: 0;
  color: #0f172a;
  font-size: 18px;
}

.task-panel__refresh {
  color: #0f766e;
}

.task-list {
  display: grid;
  gap: 12px;
  max-height: calc(min(70vh, 560px) - 78px);
  overflow-y: auto;
  padding: 14px 16px 16px;
}

.task-item {
  display: grid;
  gap: 8px;
  padding: 14px;
  border: 1px solid #dbeafe;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.8);
}

.task-item__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: #64748b;
  font-size: 12px;
}

.task-item__status {
  color: #0f766e;
  font-weight: 700;
}

.task-item--completed .task-item__status {
  color: #16a34a;
}

.task-item--failed .task-item__status {
  color: #dc2626;
}

.task-item--cancelled .task-item__status {
  color: #64748b;
}

.task-item__title {
  color: #0f172a;
  font-size: 15px;
}

.task-item__project {
  margin: 0;
  color: #475569;
  font-size: 13px;
}

.task-item__error {
  margin: 0;
  padding: 8px 10px;
  border-radius: 10px;
  background: #fef2f2;
  color: #b91c1c;
  font-size: 12px;
  line-height: 1.5;
  word-break: break-word;
}

.task-item__actions {
  display: flex;
  justify-content: flex-end;
}

.task-panel-fade-enter-active,
.task-panel-fade-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.task-panel-fade-enter-from,
.task-panel-fade-leave-to {
  opacity: 0;
  transform: translateY(10px) scale(0.98);
}
</style>
