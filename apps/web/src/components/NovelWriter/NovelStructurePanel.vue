<template>
  <aside
    :class="['insight-container', { 'insight-container--open': open, 'insight-container--dragging': dragState.active }]"
    :style="containerStyle"
  >
    <div
      class="premium-toggle"
      role="button"
      tabindex="0"
      @pointerdown="handleDragStart"
      @click="handleToggleClick"
      @keydown.enter.prevent="toggleOpen"
      @keydown.space.prevent="toggleOpen"
    >
      <div class="toggle-content">
        <el-icon class="toggle-icon"><Opportunity /></el-icon>
        <span class="toggle-text">创作灵感 / 文风画像</span>
      </div>
      <div class="toggle-status">
        <button class="material-import-btn" type="button" @click.stop="emit('importMaterials')">
          {{ materialImported ? '素材更新' : '素材带入' }}
        </button>
        <span class="status-label">{{ open ? '收起' : '展开' }}</span>
        <el-icon :class="['arrow-icon', { 'arrow-icon--rotated': open }]"><ArrowRight /></el-icon>
      </div>
    </div>

    <transition name="panel-slide">
      <div v-if="open" class="premium-panel">
        <div class="panel-scroll-content">
          <section class="premium-section">
            <div class="section-header">
              <div class="section-title-box">
                <span class="section-tag">事实库</span>
                <h3 class="section-title">事实卡片</h3>
              </div>
              <div class="section-count" v-if="factCards.length">{{ factCards.length }}</div>
            </div>
            <div class="fact-list">
              <div v-for="(item, index) in factCards" :key="`${item.type}-${item.name}-${index}`" class="premium-fact-card">
                <div class="fact-type-tag" :class="`type-${item.type}`">{{ item.type }}</div>
                <div class="fact-body">
                  <strong class="fact-name">{{ item.name }}</strong>
                  <p class="fact-desc">{{ item.description }}</p>
                </div>
              </div>
              <div v-if="factCards.length === 0" class="premium-empty-state">
                <el-icon><Compass /></el-icon>
                <p>暂无事实卡片</p>
                <span>建议先在素材图谱中执行信息提取</span>
              </div>
            </div>
          </section>

          <div class="section-divider"></div>

          <section class="premium-section">
            <div class="section-header">
              <div class="section-title-box">
                <span class="section-tag">风格规则</span>
                <h3 class="section-title">文风画像</h3>
              </div>
            </div>
            <div class="style-profile-box">
              <div class="style-summary-card">
                <el-icon class="quote-icon"><ChatLineSquare /></el-icon>
                <p class="style-summary-text">
                  {{ styleProfile.summary || '暂无文风画像。AI 将根据题材为您匹配最合适的文字质感。' }}
                </p>
              </div>
              <div class="rules-container">
                <p class="rules-label">写作准则</p>
                <div class="rule-tags">
                  <div v-for="rule in styleProfile.do_rules || []" :key="rule" class="premium-rule-tag">
                    <span class="rule-dot"></span>
                    {{ rule }}
                  </div>
                  <div v-if="!styleProfile.do_rules?.length" class="premium-rule-tag default">
                    <span class="rule-dot"></span>
                    默认原创文风
                  </div>
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>
    </transition>
  </aside>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, watch } from 'vue'
import { Opportunity, ArrowRight, Compass, ChatLineSquare } from '@element-plus/icons-vue'
import type { NovelExtractedInfo, NovelOutline, NovelStyleProfile } from '@/api/novelWriter'

const props = defineProps<{
  extracted: NovelExtractedInfo
  outline: NovelOutline
  styleProfile: NovelStyleProfile
  open: boolean
  materialImported: boolean
  positionStorageKey: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  importMaterials: []
}>()

interface PanelPosition {
  x: number
  y: number
}

const defaultPosition: PanelPosition = {
  x: 32,
  y: 82
}

const panelPosition = reactive<PanelPosition>({ ...defaultPosition })
const dragState = reactive({
  active: false,
  moved: false,
  pointerId: -1,
  startX: 0,
  startY: 0,
  originX: 0,
  originY: 0
})

const positionStorageNamespace = 'novel-generater:insight-panel-position'

const positionStorageKey = computed(() => {
  const projectKey = String(props.positionStorageKey || '').trim()
  return projectKey ? `${positionStorageNamespace}:${projectKey}` : `${positionStorageNamespace}:draft`
})

const containerStyle = computed(() => ({
  left: `${panelPosition.x}px`,
  top: `${panelPosition.y}px`
}))

const factCards = computed(() => [
  ...(props.extracted.characters || []).map((item) => ({ ...item, type: '人物' })),
  ...(props.extracted.conflicts || []).map((item) => ({ ...item, type: '冲突' })),
  ...(props.extracted.world_rules || []).map((item) => ({ ...item, type: '世界观' })),
  ...(props.extracted.key_events || []).map((item) => ({ ...item, type: '灵感' }))
])

const toggleOpen = () => emit('update:open', !props.open)

const clamp = (value: number, min: number, max: number) => Math.min(Math.max(value, min), max)

const loadPanelPosition = () => {
  if (typeof window === 'undefined') return
  try {
    const raw = window.localStorage.getItem(positionStorageKey.value)
    if (!raw) {
      panelPosition.x = defaultPosition.x
      panelPosition.y = defaultPosition.y
      return
    }
    const parsed = JSON.parse(raw) as Partial<PanelPosition>
    if (!Number.isFinite(parsed.x) || !Number.isFinite(parsed.y)) return
    panelPosition.x = clamp(parsed.x!, 12, Math.max(window.innerWidth - 390, 12))
    panelPosition.y = clamp(parsed.y!, 12, Math.max(window.innerHeight - 72, 12))
  } catch (error) {
    console.warn('Failed to load insight panel position', error)
  }
}

const persistPanelPosition = () => {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(positionStorageKey.value, JSON.stringify(panelPosition))
  } catch (error) {
    console.warn('Failed to persist insight panel position', error)
  }
}

const handleDragMove = (event: PointerEvent) => {
  if (!dragState.active || dragState.pointerId !== event.pointerId) return
  event.preventDefault()
  const deltaX = event.clientX - dragState.startX
  const deltaY = event.clientY - dragState.startY
  if (Math.abs(deltaX) + Math.abs(deltaY) > 4) dragState.moved = true
  panelPosition.x = clamp(dragState.originX + deltaX, 12, Math.max(window.innerWidth - 390, 12))
  panelPosition.y = clamp(dragState.originY + deltaY, 12, Math.max(window.innerHeight - 72, 12))
}

const handleDragEnd = (event: PointerEvent) => {
  if (!dragState.active || dragState.pointerId !== event.pointerId) return
  dragState.active = false
  dragState.pointerId = -1
  const target = event.currentTarget as HTMLElement
  target?.removeEventListener('pointermove', handleDragMove)
  target?.releasePointerCapture(event.pointerId)
  persistPanelPosition()
}

const handleDragStart = (event: PointerEvent) => {
  if (event.button !== 0) return
  if ((event.target as HTMLElement).closest('.material-import-btn')) return
  dragState.active = true
  dragState.moved = false
  dragState.pointerId = event.pointerId
  dragState.startX = event.clientX
  dragState.startY = event.clientY
  dragState.originX = panelPosition.x
  dragState.originY = panelPosition.y
  ;(event.currentTarget as HTMLElement)?.setPointerCapture(event.pointerId)
  ;(event.currentTarget as HTMLElement)?.addEventListener('pointermove', handleDragMove)
  ;(event.currentTarget as HTMLElement)?.addEventListener('pointerup', handleDragEnd, { once: true })
  ;(event.currentTarget as HTMLElement)?.addEventListener('pointercancel', handleDragEnd, { once: true })
}

const handleToggleClick = () => {
  if (dragState.moved) {
    dragState.moved = false
    return
  }
  toggleOpen()
}

watch(positionStorageKey, loadPanelPosition)
onMounted(loadPanelPosition)
</script>

<style scoped>
.insight-container {
  position: fixed;
  z-index: 100;
  width: min(390px, calc(100vw - 48px));
  transition: width 0.4s cubic-bezier(0.16, 1, 0.3, 1);
}

.insight-container--open {
  width: min(440px, calc(100vw - 48px));
}

.premium-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  width: 100%;
  box-sizing: border-box;
  padding: 10px 18px;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(13, 148, 136, 0.2);
  border-radius: 100px;
  box-shadow: 
    0 4px 6px -1px rgba(0, 0, 0, 0.05),
    0 10px 25px -5px rgba(13, 148, 136, 0.1);
  cursor: grab;
  touch-action: none;
  transition: all 0.3s ease;
}

.insight-container--dragging .premium-toggle {
  cursor: grabbing;
}

.premium-toggle:hover {
  transform: translateY(-2px);
  background: rgba(255, 255, 255, 0.95);
  box-shadow: 
    0 10px 15px -3px rgba(0, 0, 0, 0.1),
    0 20px 40px -10px rgba(13, 148, 136, 0.15);
}

.toggle-content {
  display: flex;
  align-items: center;
  flex: 1 1 auto;
  gap: 10px;
  min-width: 0;
  color: #0f766e;
}

.toggle-icon {
  font-size: 18px;
}

.toggle-text {
  overflow: hidden;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.02em;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.toggle-status {
  display: flex;
  align-items: center;
  flex: 0 0 auto;
  gap: 6px;
  color: #64748b;
  white-space: nowrap;
}

.material-import-btn {
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
  height: 26px;
  padding: 0 10px;
  border: 1px solid rgba(13, 148, 136, 0.2);
  border-radius: 999px;
  background: rgba(240, 253, 250, 0.78);
  color: #0f766e;
  font-size: 12px;
  font-weight: 800;
  cursor: pointer;
  transition: border-color 0.18s ease, background 0.18s ease, transform 0.18s ease;
}

.material-import-btn:hover {
  border-color: rgba(13, 148, 136, 0.42);
  background: #ccfbf1;
  transform: translateY(-1px);
}

.status-label {
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.arrow-icon {
  font-size: 14px;
  transition: transform 0.3s ease;
}

.arrow-icon--rotated {
  transform: rotate(90deg);
}

.premium-panel {
  margin-top: 12px;
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(24px);
  border: 1px solid rgba(226, 232, 240, 0.6);
  border-radius: 24px;
  box-shadow: 0 25px 60px -12px rgba(15, 23, 42, 0.18);
  overflow: hidden;
  max-height: calc(100vh - 200px);
  display: flex;
  flex-direction: column;
}

.panel-scroll-content {
  padding: 20px;
  overflow-y: auto;
  scrollbar-width: thin;
}

.premium-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
}

.section-tag {
  display: block;
  font-size: 10px;
  font-weight: 900;
  color: #0d9488;
  letter-spacing: 0.1em;
  margin-bottom: 4px;
}

.section-title {
  margin: 0;
  font-size: 18px;
  font-weight: 800;
  color: #0f172a;
}

.section-count {
  background: #f1f5f9;
  color: #475569;
  font-size: 11px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 6px;
}

.fact-list {
  display: grid;
  gap: 10px;
}

.premium-fact-card {
  padding: 14px;
  background: #ffffff;
  border: 1px solid #f1f5f9;
  border-radius: 16px;
  transition: all 0.2s ease;
}

.premium-fact-card:hover {
  border-color: #0d948840;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.03);
  transform: translateX(4px);
}

.fact-type-tag {
  display: inline-block;
  font-size: 10px;
  font-weight: 800;
  padding: 2px 8px;
  border-radius: 6px;
  margin-bottom: 8px;
}

.type-人物 { background: #eff6ff; color: #2563eb; }
.type-冲突 { background: #fef2f2; color: #dc2626; }
.type-世界观 { background: #f0fdf4; color: #16a34a; }
.type-灵感 { background: #fffbeb; color: #d97706; }

.fact-name {
  display: block;
  font-size: 14px;
  font-weight: 700;
  color: #0f172a;
  margin-bottom: 4px;
}

.fact-desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: #64748b;
}

.section-divider {
  height: 1px;
  background: linear-gradient(to right, transparent, #e2e8f0, transparent);
  margin: 20px 0;
}

.style-profile-box {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.style-summary-card {
  position: relative;
  padding: 16px;
  background: linear-gradient(135deg, #0d94880a 0%, #0d948805 100%);
  border: 1px dashed #0d948840;
  border-radius: 16px;
}

.quote-icon {
  position: absolute;
  top: 10px;
  right: 10px;
  font-size: 20px;
  color: #0d948820;
}

.style-summary-text {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: #334155;
  font-style: italic;
}

.rules-label {
  font-size: 12px;
  font-weight: 700;
  color: #64748b;
  margin-bottom: 8px;
}

.rule-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.premium-rule-tag {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 600;
  color: #475569;
}

.rule-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #0d9488;
}

.premium-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 40px 0;
  color: #94a3b8;
}

.premium-empty-state .el-icon {
  font-size: 32px;
  margin-bottom: 12px;
  opacity: 0.5;
}

.premium-empty-state p {
  margin: 0;
  font-size: 14px;
  font-weight: 700;
}

.premium-empty-state span {
  font-size: 12px;
}

/* Animations */
.panel-slide-enter-active,
.panel-slide-leave-active {
  transition: all 0.4s cubic-bezier(0.16, 1, 0.3, 1);
}

.panel-slide-enter-from,
.panel-slide-leave-to {
  opacity: 0;
  transform: translateY(20px) scale(0.95);
}

@media (max-width: 900px) {
  .insight-container {
    width: min(380px, calc(100vw - 32px));
  }
}
</style>
