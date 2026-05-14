<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, shallowRef, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection, DocumentChecked, EditPen, MagicStick, Memo, Minus, Plus, UserFilled } from '@element-plus/icons-vue'
import type { NovelMaterials } from '@/api/novelWriter'

const props = defineProps<{
  modelValue: NovelMaterials
  positionStorageKey: string
  saving: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: NovelMaterials]
  editingStateChange: [editing: boolean]
  save: []
  extract: []
  next: []
}>()

interface CanvasNode {
  id: string
  type: 'world' | 'character' | 'conflict' | 'idea'
  title: string
  value: string
  x: number
  y: number
  width: number
  height: number
}

interface CanvasEdge {
  id: string
  from: CanvasNode
  to: CanvasNode
  dashed?: boolean
}

interface CharacterRoleOption {
  label: string
  value: string
  template: string
}

interface CharacterFormState {
  role: string
  name: string
  personality: string
  identity: string
  desire: string
  weakness: string
}

interface CanvasViewport {
  x: number
  y: number
  zoom: number
}

const form = reactive<NovelMaterials>({ ...props.modelValue })
const canvasRef = shallowRef<HTMLElement | null>(null)
const zoomLevel = shallowRef(0.92)
const translateX = shallowRef(80)
const translateY = shallowRef(42)
const isPanning = shallowRef(false)
const startPos = reactive({ x: 0, y: 0, tx: 0, ty: 0 })
const draggingIdea = shallowRef('')
const draggingIdeaIndex = shallowRef(-1)
const dropActive = shallowRef(false)
const activeDropCharacterId = shallowRef('')
const conflictDialogVisible = shallowRef(false)
const selectedConflictCharacterIds = shallowRef<string[]>([])
const nodePositionOverrides = reactive<Record<string, { x: number; y: number }>>({})
const characterEditingMap = reactive<Record<string, boolean>>({})
const materialEditingMap = reactive<Record<string, boolean>>({})
const touchPoints = reactive<Record<number, { x: number; y: number }>>({})
const touchGesture = reactive({
  active: false,
  distance: 0,
  centerX: 0,
  centerY: 0,
  tx: 0,
  ty: 0
})
const nodeDragState = reactive({
  active: false,
  nodeKey: '',
  pointerId: -1,
  startX: 0,
  startY: 0,
  originX: 0,
  originY: 0
})

let panFrame = 0
let viewportPersistFrame = 0
let pendingPan: PointerEvent | null = null
const defaultCharacterTemplate = '姓名：\n性格：\n身份：\n欲望：\n弱点：'
const positionStorageNamespace = 'novel-generater:material-map-positions'
const viewportStorageNamespace = 'novel-generater:material-map-viewport'

const characterRoleOptions: CharacterRoleOption[] = [
  { label: '男主角', value: '男主角', template: defaultCharacterTemplate },
  { label: '女主角', value: '女主角', template: defaultCharacterTemplate },
  { label: '主要男配角', value: '主要男配角', template: defaultCharacterTemplate },
  { label: '主要女配角', value: '主要女配角', template: defaultCharacterTemplate },
  { label: '次要男配角', value: '次要男配角', template: defaultCharacterTemplate },
  { label: '次要女配角', value: '次要女配角', template: defaultCharacterTemplate },
  { label: '重要NPC', value: '重要NPC', template: defaultCharacterTemplate },
  { label: '普通NPC', value: '普通NPC', template: defaultCharacterTemplate }
]

const splitBlocks = (text: string) => {
  return String(text || '')
    .split(/\n\s*\n|\n-/)
    .map((item) => item.replace(/^-/, '').trim())
    .filter(Boolean)
}

const splitLines = (text: string) => {
  return String(text || '')
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean)
}

const appendWithBlankLine = (current: string, next: string) => {
  const trimmed = String(current || '').trim()
  return trimmed ? `${trimmed}\n\n${next}` : next
}

const appendLine = (current: string, next: string) => {
  const trimmed = String(current || '').trim()
  return trimmed ? `${trimmed}\n${next}` : next
}

const readLabeledLine = (text: string, label: string) => {
  const matched = String(text || '').match(new RegExp(`^${label}[:：][ \\t]*(.*)$`, 'm'))
  return matched?.[1]?.trim() || ''
}

const readCharacterName = (value: string) => readLabeledLine(value, '姓名') || readLabeledLine(value, '名称')

const getCharacterRole = (value: string) => {
  const role = readLabeledLine(value, '角色类型')
  return role || '人物'
}

const getCharacterName = (value: string, fallback: string) => {
  const name = readCharacterName(value)
  if (name) return name
  const firstLine = String(value || '').split('\n').map((item) => item.trim()).find(Boolean) || ''
  if (/^(?:角色类型|姓名|名称|性格|身份|欲望|弱点)[:：]/.test(firstLine)) return fallback
  return firstLine.replace(/^(?:新人物|人物|角色|男主角|女主角|主要男配角|主要女配角|次要男配角|次要女配角|重要NPC|普通NPC)[ \t]*[:：]?/, '').trim() || fallback
}

const getCharacterNodeLabel = (value: string, fallback: string) => {
  const role = getCharacterRole(value)
  const name = getCharacterName(value, fallback)
  return role === '人物' ? name : `${role}：${name}`
}

const normalizeName = (value: string) => String(value || '').replace(/[，,；;。.\s:：]/g, '').toLowerCase()

const replaceOrPrependLabeledLine = (text: string, label: string, nextValue: string) => {
  const normalized = String(text || '')
  const line = `${label}：${nextValue}`
  const pattern = new RegExp(`^${label}[:：].*$`, 'm')
  if (pattern.test(normalized)) return normalized.replace(pattern, line)
  return normalized.trim() ? `${line}\n${normalized}` : line
}

const buildCharacterValue = (state: CharacterFormState) => {
  return [
    `角色类型：${state.role || '男主角'}`,
    `姓名：${state.name}`,
    `性格：${state.personality}`,
    `身份：${state.identity}`,
    `欲望：${state.desire}`,
    `弱点：${state.weakness}`
  ].join('\n')
}

const parseCharacterForm = (value: string): CharacterFormState => ({
  role: getCharacterRole(value),
  name: readCharacterName(value),
  personality: readLabeledLine(value, '性格'),
  identity: readLabeledLine(value, '身份'),
  desire: readLabeledLine(value, '欲望'),
  weakness: readLabeledLine(value, '弱点')
})

const hashString = (value: string) => {
  let hash = 0
  for (let index = 0; index < value.length; index += 1) {
    hash = ((hash << 5) - hash) + value.charCodeAt(index)
    hash |= 0
  }
  return Math.abs(hash).toString(36)
}

const clamp = (value: number, min: number, max: number) => Math.min(Math.max(value, min), max)

const nodePositionKey = (node: CanvasNode) => `${node.type}:${node.id}`

const positionStorageKey = computed(() => {
  const projectKey = String(props.positionStorageKey || '').trim()
  if (projectKey) return `${positionStorageNamespace}:${projectKey}`
  return `${positionStorageNamespace}:draft:${hashString(JSON.stringify(props.modelValue))}`
})

const viewportStorageKey = computed(() => {
  const projectKey = String(props.positionStorageKey || '').trim()
  if (projectKey) return `${viewportStorageNamespace}:${projectKey}`
  return `${viewportStorageNamespace}:draft:${hashString(JSON.stringify(props.modelValue))}`
})

const getNodePosition = (node: CanvasNode) => nodePositionOverrides[nodePositionKey(node)] || { x: node.x, y: node.y }

const loadNodePositions = () => {
  if (typeof window === 'undefined') return
  Object.keys(nodePositionOverrides).forEach((key) => delete nodePositionOverrides[key])
  try {
    const raw = window.localStorage.getItem(positionStorageKey.value)
    if (!raw) return
    const parsed = JSON.parse(raw) as Record<string, { x: number; y: number }>
    Object.entries(parsed).forEach(([key, value]) => {
      if (!Number.isFinite(value?.x) || !Number.isFinite(value?.y)) return
      nodePositionOverrides[key] = { x: value.x, y: value.y }
    })
  } catch (error) {
    console.warn('Failed to load material map positions', error)
  }
}

const persistNodePositions = () => {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(positionStorageKey.value, JSON.stringify(nodePositionOverrides))
  } catch (error) {
    console.warn('Failed to persist material map positions', error)
  }
}

const loadViewport = () => {
  if (typeof window === 'undefined') return
  try {
    const raw = window.localStorage.getItem(viewportStorageKey.value)
    if (!raw) return
    const parsed = JSON.parse(raw) as Partial<CanvasViewport>
    if (!Number.isFinite(parsed.x) || !Number.isFinite(parsed.y) || !Number.isFinite(parsed.zoom)) return
    translateX.value = parsed.x!
    translateY.value = parsed.y!
    zoomLevel.value = clamp(parsed.zoom!, 0.45, 1.8)
  } catch (error) {
    console.warn('Failed to load material map viewport', error)
  }
}

const persistViewport = () => {
  if (typeof window === 'undefined') return
  try {
    const viewport: CanvasViewport = {
      x: translateX.value,
      y: translateY.value,
      zoom: zoomLevel.value
    }
    window.localStorage.setItem(viewportStorageKey.value, JSON.stringify(viewport))
  } catch (error) {
    console.warn('Failed to persist material map viewport', error)
  }
}

const schedulePersistViewport = () => {
  if (typeof window === 'undefined') return
  if (viewportPersistFrame) window.cancelAnimationFrame(viewportPersistFrame)
  viewportPersistFrame = window.requestAnimationFrame(() => {
    viewportPersistFrame = 0
    persistViewport()
  })
}

const isCharacterEditing = (nodeId: string) => Boolean(characterEditingMap[nodeId])

const setCharacterEditing = (nodeId: string, editing: boolean) => {
  if (editing) characterEditingMap[nodeId] = true
  else delete characterEditingMap[nodeId]
}

const isMaterialEditing = (nodeId: string) => Boolean(materialEditingMap[nodeId])

const setMaterialEditing = (nodeId: string, editing: boolean) => {
  if (editing) materialEditingMap[nodeId] = true
  else delete materialEditingMap[nodeId]
}

const worldNode = computed<CanvasNode>(() => ({
  id: 'world',
  type: 'world',
  title: '世界观基础',
  value: form.world_raw,
  x: 520,
  y: 40,
  width: 420,
  height: 210
}))

const characterNodes = computed<CanvasNode[]>(() => {
  const cards = splitBlocks(form.character_raw)
  return cards.map((value, index) => ({
    id: `character-${index}`,
    type: 'character',
    title: getCharacterNodeLabel(value, `人物 ${index + 1}`),
    value,
    x: 160 + (index % 3) * 360,
    y: 360 + Math.floor(index / 3) * 260,
    width: 300,
    height: 210
  }))
})

const conflictNodes = computed<CanvasNode[]>(() => {
  const baseY = 680 + Math.ceil(Math.max(characterNodes.value.length, 1) / 3) * 180
  return splitLines(form.conflict_raw).map((value, index) => ({
    id: `conflict-${index}`,
    type: 'conflict',
    title: `冲突 ${index + 1}`,
    value,
    x: 240 + (index % 2) * 520,
    y: baseY + Math.floor(index / 2) * 230,
    width: 420,
    height: 170
  }))
})

const ideaNodes = computed<CanvasNode[]>(() => {
  return splitLines(form.raw_text).map((value, index) => ({
    id: `idea-${index}`,
    type: 'idea',
    title: `灵感 ${index + 1}`,
    value,
    x: 1180,
    y: 82 + index * 170,
    width: 280,
    height: 130
  }))
})

const canvasNodes = computed(() => [
  worldNode.value,
  ...characterNodes.value,
  ...conflictNodes.value,
  ...ideaNodes.value
])

const hasInlineEditing = computed(() => {
  return Object.keys(characterEditingMap).length > 0 || Object.keys(materialEditingMap).length > 0
})

const canvasEdges = computed<CanvasEdge[]>(() => {
  const edges: CanvasEdge[] = []
  characterNodes.value.forEach((node) => {
    edges.push({ id: `world-${node.id}`, from: worldNode.value, to: node })
  })
  conflictNodes.value.forEach((node, index) => {
    const linkedCharacters = characterNodes.value.filter((character) => {
      const characterName = normalizeName(getCharacterName(character.value, character.title))
      return characterName && normalizeName(node.value).includes(characterName)
    })
    const participants = linkedCharacters.length > 0
      ? linkedCharacters
      : characterNodes.value.slice(index % Math.max(characterNodes.value.length, 1), index % Math.max(characterNodes.value.length, 1) + 1)
    participants.forEach((character) => {
      edges.push({ id: `${character.id}-${node.id}`, from: character, to: node })
    })
  })
  return edges
})

const boardSize = computed(() => {
  const maxX = Math.max(...canvasNodes.value.map((node) => getNodePosition(node).x + node.width), 1500)
  const maxY = Math.max(...canvasNodes.value.map((node) => getNodePosition(node).y + node.height), 980)
  return {
    width: maxX + 180,
    height: maxY + 160
  }
})

const zoomStyle = computed(() => ({
  width: `${boardSize.value.width}px`,
  height: `${boardSize.value.height}px`,
  transform: `translate(${translateX.value}px, ${translateY.value}px) scale(${zoomLevel.value})`,
  transformOrigin: '0 0',
  transition: isPanning.value ? 'none' : 'transform 0.16s cubic-bezier(0.4, 0, 0.2, 1)'
}))

watch(
  () => props.modelValue,
  (value) => {
    if (hasInlineEditing.value) return
    Object.assign(form, value)
  },
  { deep: true }
)

watch(
  form,
  () => emit('update:modelValue', { ...form }),
  { deep: true }
)

watch(
  positionStorageKey,
  () => {
    loadNodePositions()
    loadViewport()
  },
  { immediate: true }
)

watch(
  hasInlineEditing,
  (editing) => {
    emit('editingStateChange', editing)
  },
  { immediate: true }
)

const zoomAtPoint = (nextZoom: number, clientX?: number, clientY?: number) => {
  const canvas = canvasRef.value
  if (!canvas) return

  const rect = canvas.getBoundingClientRect()
  const oldZoom = zoomLevel.value
  const zoom = clamp(Number(nextZoom.toFixed(2)), 0.45, 1.8)
  const focusX = clientX ?? rect.left + rect.width / 2
  const focusY = clientY ?? rect.top + rect.height / 2
  const contentX = (focusX - rect.left - translateX.value) / oldZoom
  const contentY = (focusY - rect.top - translateY.value) / oldZoom

  zoomLevel.value = zoom
  translateX.value = focusX - rect.left - contentX * zoom
  translateY.value = focusY - rect.top - contentY * zoom
  schedulePersistViewport()
}

const centerCanvas = () => {
  translateX.value = 80
  translateY.value = 42
  zoomLevel.value = 0.92
  schedulePersistViewport()
}

const handleWheel = (event: WheelEvent) => {
  if ((event.target as HTMLElement)?.closest('.node-interactive') && !event.ctrlKey && !event.metaKey) return
  event.preventDefault()
  if (event.ctrlKey || event.metaKey) {
    zoomAtPoint(zoomLevel.value + (event.deltaY < 0 ? 0.08 : -0.08), event.clientX, event.clientY)
    return
  }
  translateX.value -= event.deltaX
  translateY.value -= event.deltaY
  schedulePersistViewport()
}

const isPanTarget = (target: HTMLElement) => {
  return !target.closest('.canvas-node')
    && !target.closest('.canvas-toolbar')
    && !target.closest('.canvas-createbar')
    && !target.closest('.material-map-page__actions')
}

const handlePointerDown = (event: PointerEvent) => {
  const target = event.target as HTMLElement
  if (event.pointerType === 'touch') {
    touchPoints[event.pointerId] = { x: event.clientX, y: event.clientY }
    const ids = Object.keys(touchPoints).map(Number)
    if (ids.length === 2) {
      const [first, second] = ids.map((id) => touchPoints[id])
      if (!first || !second) return
      const centerX = (first.x + second.x) / 2
      const centerY = (first.y + second.y) / 2
      touchGesture.active = true
      touchGesture.distance = Math.hypot(first.x - second.x, first.y - second.y)
      touchGesture.centerX = centerX
      touchGesture.centerY = centerY
      touchGesture.tx = translateX.value
      touchGesture.ty = translateY.value
      handlePointerUp()
    }
    return
  }
  if (event.button !== 0 || !isPanTarget(target) || nodeDragState.active) return
  isPanning.value = true
  startPos.x = event.clientX
  startPos.y = event.clientY
  startPos.tx = translateX.value
  startPos.ty = translateY.value
  canvasRef.value?.setPointerCapture(event.pointerId)
}

const handlePointerMove = (event: PointerEvent) => {
  if (event.pointerType === 'touch' && touchPoints[event.pointerId]) {
    touchPoints[event.pointerId] = { x: event.clientX, y: event.clientY }
    const ids = Object.keys(touchPoints).map(Number)
    if (touchGesture.active && ids.length === 2) {
      const [first, second] = ids.map((id) => touchPoints[id])
      if (!first || !second) return
      const centerX = (first.x + second.x) / 2
      const centerY = (first.y + second.y) / 2
      const distance = Math.hypot(first.x - second.x, first.y - second.y)
      translateX.value = touchGesture.tx + centerX - touchGesture.centerX
      translateY.value = touchGesture.ty + centerY - touchGesture.centerY
      const delta = (distance - touchGesture.distance) / 220
      zoomAtPoint(zoomLevel.value + delta, centerX, centerY)
      touchGesture.distance = distance
      touchGesture.centerX = centerX
      touchGesture.centerY = centerY
      touchGesture.tx = translateX.value
      touchGesture.ty = translateY.value
      schedulePersistViewport()
      return
    }
  }
  if (!isPanning.value) return
  event.preventDefault()
  pendingPan = event
  if (panFrame) return
  panFrame = window.requestAnimationFrame(() => {
    if (!pendingPan) return
    translateX.value = startPos.tx + pendingPan.clientX - startPos.x
    translateY.value = startPos.ty + pendingPan.clientY - startPos.y
    schedulePersistViewport()
    panFrame = 0
    pendingPan = null
  })
}

const handlePointerUp = (event?: PointerEvent) => {
  isPanning.value = false
  pendingPan = null
  if (event?.pointerType === 'touch') {
    delete touchPoints[event.pointerId]
    if (Object.keys(touchPoints).length < 2) touchGesture.active = false
  }
  if (panFrame) {
    window.cancelAnimationFrame(panFrame)
    panFrame = 0
  }
  if (event && canvasRef.value?.hasPointerCapture(event.pointerId)) {
    canvasRef.value.releasePointerCapture(event.pointerId)
  }
}

const addCharacter = (role: string) => {
  const option = characterRoleOptions.find((item) => item.value === role) || characterRoleOptions[0]!
  form.character_raw = appendWithBlankLine(form.character_raw, `角色类型：${option.value}\n${option.template}`)
  nextTick(() => {
    const lastIndex = Math.max(characterNodes.value.length - 1, 0)
    setCharacterEditing(`character-${lastIndex}`, true)
  })
}

const addConflict = () => {
  if (characterNodes.value.length === 0) {
    ElMessage.warning('请先添加人物，再创建冲突')
    return
  }
  selectedConflictCharacterIds.value = []
  conflictDialogVisible.value = true
}

const createConflict = () => {
  const characters = selectedConflictCharacterIds.value
    .map((id) => characterNodes.value.find((node) => node.id === id))
    .filter((node): node is CanvasNode => Boolean(node))
  if (characters.length === 0) {
    ElMessage.warning('请至少选择一个冲突人物')
    return
  }

  if (characters.length === 1) {
    const character = characters[0]
    if (!character) return
    const characterName = getCharacterName(character.value, character.title)
    form.conflict_raw = appendLine(form.conflict_raw, `${characterName}：自身欲望、弱点或处境引发的内部冲突`)
  } else {
    const characterNames = characters.map((node) => getCharacterName(node.value, node.title))
    form.conflict_raw = appendLine(form.conflict_raw, `${characterNames.join(' ↔ ')}：围绕目标、秘密或利益产生直接冲突`)
  }
  conflictDialogVisible.value = false
  selectedConflictCharacterIds.value = []
}

const addIdea = () => {
  form.raw_text = appendLine(form.raw_text, '新的阶段性灵感：')
  nextTick(() => {
    const lastIndex = Math.max(ideaNodes.value.length - 1, 0)
    setMaterialEditing(`idea-${lastIndex}`, true)
  })
}

const updateCharacter = (index: number, value: string) => {
  const cards = [...characterNodes.value.map((node) => node.value)]
  cards[index] = value
  form.character_raw = cards.filter((item) => item.trim()).join('\n\n')
}

const updateCharacterRole = (node: CanvasNode, role: string) => {
  const index = Number(node.id.split('-')[1] || 0)
  updateCharacter(index, replaceOrPrependLabeledLine(node.value, '角色类型', role))
}

const saveCharacterForm = (node: CanvasNode, state: CharacterFormState) => {
  updateCharacter(Number(node.id.split('-')[1] || 0), buildCharacterValue(state))
  setCharacterEditing(node.id, false)
}

const updateCharacterField = (node: CanvasNode, patch: Partial<CharacterFormState>) => {
  updateCharacter(
    Number(node.id.split('-')[1] || 0),
    buildCharacterValue({ ...parseCharacterForm(node.value), ...patch })
  )
}

const updateConflict = (index: number, value: string) => {
  const cards = splitLines(form.conflict_raw)
  cards[index] = value
  form.conflict_raw = cards.filter((item) => item.trim()).join('\n')
}

const updateIdea = (index: number, value: string) => {
  const cards = [...ideaNodes.value.map((node) => node.value)]
  cards[index] = value
  form.raw_text = cards.filter((item) => item.trim()).join('\n')
}

const updateNodeValue = (node: CanvasNode, value: string) => {
  if (node.type === 'world') form.world_raw = value
  if (node.type === 'character') updateCharacter(Number(node.id.split('-')[1] || 0), value)
  if (node.type === 'conflict') updateConflict(Number(node.id.split('-')[1] || 0), value)
  if (node.type === 'idea') updateIdea(Number(node.id.split('-')[1] || 0), value)
}

const deleteNode = (node: CanvasNode) => {
  const index = Number(node.id.split('-')[1] || 0)
  if (node.type === 'character') {
    form.character_raw = characterNodes.value
      .filter((_, itemIndex) => itemIndex !== index)
      .map((item) => item.value)
      .join('\n\n')
  }
  if (node.type === 'conflict') {
    form.conflict_raw = splitLines(form.conflict_raw)
      .filter((_, itemIndex) => itemIndex !== index)
      .join('\n')
  }
  if (node.type === 'idea') {
    form.raw_text = splitLines(form.raw_text)
      .filter((_, itemIndex) => itemIndex !== index)
      .join('\n')
  }
}

const handleIdeaDragStart = (node: CanvasNode) => {
  draggingIdea.value = node.value
  draggingIdeaIndex.value = Number(node.id.split('-')[1] || -1)
}

const resetIdeaDrag = () => {
  draggingIdea.value = ''
  draggingIdeaIndex.value = -1
  dropActive.value = false
  activeDropCharacterId.value = ''
}

const handleNodeDragOver = (node: CanvasNode) => {
  if (!draggingIdea.value || node.type !== 'character') return
  dropActive.value = true
  activeDropCharacterId.value = node.id
}

const handleMainDrop = () => {
  resetIdeaDrag()
}

const handleCharacterDrop = (node: CanvasNode) => {
  if (!draggingIdea.value || node.type !== 'character') {
    resetIdeaDrag()
    return
  }
  const characterName = getCharacterName(node.value, node.title)
  form.conflict_raw = appendLine(form.conflict_raw, `${characterName} ← 灵感转入：${draggingIdea.value}`)
  if (draggingIdeaIndex.value >= 0) {
    form.raw_text = splitLines(form.raw_text)
      .filter((_, index) => index !== draggingIdeaIndex.value)
      .join('\n')
  }
  resetIdeaDrag()
}

const edgePath = (edge: CanvasEdge) => {
  const fromPosition = getNodePosition(edge.from)
  const toPosition = getNodePosition(edge.to)
  const startX = fromPosition.x + edge.from.width / 2
  const startY = fromPosition.y + edge.from.height / 2
  const endX = toPosition.x + edge.to.width / 2
  const endY = toPosition.y + edge.to.height / 2
  const midY = startY + (endY - startY) * 0.55
  return `M ${startX} ${startY} C ${startX} ${midY}, ${endX} ${midY}, ${endX} ${endY}`
}

const nodeStyle = (node: CanvasNode) => {
  const position = getNodePosition(node)
  return {
    left: `${position.x}px`,
    top: `${position.y}px`,
    width: `${node.width}px`,
    minHeight: `${node.height}px`
  }
}

const handleNodePointerDown = (event: PointerEvent, node: CanvasNode) => {
  const target = event.target as HTMLElement
  if (event.button !== 0 || target.closest('.node-interactive')) return
  nodeDragState.active = true
  nodeDragState.nodeKey = nodePositionKey(node)
  nodeDragState.pointerId = event.pointerId
  nodeDragState.startX = event.clientX
  nodeDragState.startY = event.clientY
  const position = getNodePosition(node)
  nodeDragState.originX = position.x
  nodeDragState.originY = position.y
  ;(event.currentTarget as HTMLElement)?.setPointerCapture(event.pointerId)
}

const handleNodePointerMove = (event: PointerEvent) => {
  if (!nodeDragState.active || nodeDragState.pointerId !== event.pointerId) return
  event.preventDefault()
  const scale = zoomLevel.value || 1
  nodePositionOverrides[nodeDragState.nodeKey] = {
    x: nodeDragState.originX + (event.clientX - nodeDragState.startX) / scale,
    y: nodeDragState.originY + (event.clientY - nodeDragState.startY) / scale
  }
}

const handleNodePointerUp = (event: PointerEvent) => {
  if (!nodeDragState.active || nodeDragState.pointerId !== event.pointerId) return
  nodeDragState.active = false
  nodeDragState.nodeKey = ''
  nodeDragState.pointerId = -1
  ;(event.currentTarget as HTMLElement)?.releasePointerCapture(event.pointerId)
  persistNodePositions()
}

const formatCharacterDisplayRows = (value: string) => {
  const parsed = parseCharacterForm(value)
  return [
    { label: '姓名', value: parsed.name || '未填写' },
    { label: '性格', value: parsed.personality || '未填写' },
    { label: '身份', value: parsed.identity || '未填写' },
    { label: '欲望', value: parsed.desire || '未填写' },
    { label: '弱点', value: parsed.weakness || '未填写' }
  ]
}

const formatMaterialDisplayRows = (value: string) => {
  const lines = String(value || '')
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean)

  if (lines.length === 0) return [{ label: '内容', value: '未填写' }]

  return lines.map((line) => {
    const matched = line.match(/^([^:：]{1,14})[:：]\s*(.*)$/)
    if (!matched) return { label: '内容', value: line }
    return {
      label: matched[1]?.trim() || '内容',
      value: matched[2]?.trim() || '未填写'
    }
  })
}

onMounted(() => {
  nextTick(loadViewport)
})
</script>

<template>
  <section class="material-map-page">
    <div class="material-map-page__header">
      <slot name="workspace-nav" />
      <div class="material-map-page__switch">
        <slot name="workspace-switch" />
      </div>
    </div>

    <div
      ref="canvasRef"
      :class="['mindmap-canvas', { 'is-panning': isPanning, 'is-drop-active': dropActive }]"
      @wheel="handleWheel"
      @pointerdown="handlePointerDown"
      @pointermove="handlePointerMove"
      @pointerup="handlePointerUp"
      @pointercancel="handlePointerUp"
      @pointerleave="handlePointerUp"
      @drop.prevent="handleMainDrop"
    >
      <div class="material-map-page__actions">
        <button class="canvas-createbar__button canvas-createbar__button--save" type="button" :disabled="saving" @click="emit('save')">
          <el-icon><DocumentChecked /></el-icon>
          <span>{{ saving ? '保存中' : '保存素材' }}</span>
        </button>
        <button class="canvas-createbar__button canvas-createbar__button--extract" type="button" @click="emit('extract')">
          <el-icon><MagicStick /></el-icon>
          <span>信息提取</span>
        </button>
        <button class="canvas-createbar__button canvas-createbar__button--outline" type="button" @click="emit('next')">
          <el-icon><Memo /></el-icon>
          <span>生成大纲</span>
        </button>
      </div>

      <div class="canvas-createbar">
        <el-dropdown trigger="click" @command="(role: string | number | object) => addCharacter(String(role))">
          <button class="canvas-createbar__button canvas-createbar__button--character" type="button">
            <el-icon><UserFilled /></el-icon>
            <span>添加人物</span>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item
                v-for="role in characterRoleOptions"
                :key="role.value"
                :command="role.value"
              >
                {{ role.label }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <button class="canvas-createbar__button canvas-createbar__button--conflict" type="button" @click="addConflict">
          <el-icon><Connection /></el-icon>
          <span>添加冲突</span>
        </button>
        <button class="canvas-createbar__button canvas-createbar__button--idea" type="button" @click="addIdea">
          <el-icon><EditPen /></el-icon>
          <span>添加灵感</span>
        </button>
      </div>

      <div class="canvas-toolbar">
        <el-button size="small" :icon="Minus" @click="zoomAtPoint(zoomLevel - 0.1)" />
        <el-button size="small" @click="centerCanvas">{{ Math.round(zoomLevel * 100) }}%</el-button>
        <el-button size="small" :icon="Plus" @click="zoomAtPoint(zoomLevel + 0.1)" />
      </div>

      <div class="mindmap-scroller" :style="zoomStyle">
        <svg class="canvas-edges" :width="boardSize.width" :height="boardSize.height" aria-hidden="true">
          <path
            v-for="edge in canvasEdges"
            :key="edge.id"
            :d="edgePath(edge)"
            :class="{ 'is-dashed': edge.dashed }"
          />
        </svg>

        <article
          v-for="node in canvasNodes"
          :key="node.id"
          :class="[
            'canvas-node',
            `canvas-node--${node.type}`,
            { 'canvas-node--drop-target': node.id === activeDropCharacterId }
          ]"
          :style="nodeStyle(node)"
          :draggable="node.type === 'idea'"
          @dragstart="handleIdeaDragStart(node)"
          @dragend="resetIdeaDrag"
          @pointerdown="handleNodePointerDown($event, node)"
          @pointermove="handleNodePointerMove"
          @pointerup="handleNodePointerUp"
          @pointercancel="handleNodePointerUp"
          @dragover.prevent="handleNodeDragOver(node)"
          @dragleave="activeDropCharacterId = activeDropCharacterId === node.id ? '' : activeDropCharacterId"
          @drop.stop.prevent="handleCharacterDrop(node)"
        >
          <div class="canvas-node__header">
            <div>
              <span v-if="node.type !== 'character'">{{ node.type }}</span>
              <span v-else>{{ getCharacterRole(node.value) }}</span>
              <strong>{{ node.title }}</strong>
            </div>
            <el-button
              v-if="node.type !== 'world'"
              size="small"
              text
              type="danger"
              class="node-interactive"
              @click.stop="deleteNode(node)"
            >
              删除
            </el-button>
          </div>
          <template v-if="node.type === 'character'">
            <div v-if="isCharacterEditing(node.id)" class="character-card-form">
              <el-select
                :model-value="parseCharacterForm(node.value).role"
                class="node-interactive"
                placeholder="角色类型"
                @update:model-value="updateCharacterRole(node, String($event))"
              >
                <el-option
                  v-for="role in characterRoleOptions"
                  :key="role.value"
                  :label="role.label"
                  :value="role.value"
                />
              </el-select>
              <el-input
                class="node-interactive"
                :model-value="parseCharacterForm(node.value).name"
                placeholder="姓名"
                @update:model-value="updateCharacterField(node, { name: String($event) })"
              />
              <el-input
                class="node-interactive"
                :model-value="parseCharacterForm(node.value).personality"
                placeholder="性格"
                @update:model-value="updateCharacterField(node, { personality: String($event) })"
              />
              <el-input
                class="node-interactive"
                :model-value="parseCharacterForm(node.value).identity"
                placeholder="身份"
                @update:model-value="updateCharacterField(node, { identity: String($event) })"
              />
              <el-input
                class="node-interactive"
                :model-value="parseCharacterForm(node.value).desire"
                placeholder="欲望"
                @update:model-value="updateCharacterField(node, { desire: String($event) })"
              />
              <el-input
                class="node-interactive"
                :model-value="parseCharacterForm(node.value).weakness"
                placeholder="弱点"
                @update:model-value="updateCharacterField(node, { weakness: String($event) })"
              />
              <div class="character-card-form__actions">
                <el-button class="node-interactive" size="small" @click.stop="saveCharacterForm(node, parseCharacterForm(node.value))">完成</el-button>
              </div>
            </div>
            <div v-else class="character-card-display">
              <dl class="character-card-display__rows">
                <div
                  v-for="item in formatCharacterDisplayRows(node.value)"
                  :key="item.label"
                  class="character-card-display__row"
                >
                  <dt>{{ item.label }}</dt>
                  <dd>{{ item.value }}</dd>
                </div>
              </dl>
              <div class="character-card-display__actions">
                <el-button class="node-interactive" size="small" text @click.stop="setCharacterEditing(node.id, true)">编辑</el-button>
              </div>
            </div>
          </template>
          <template v-else>
            <div v-if="isMaterialEditing(node.id)" class="material-card-form">
              <el-input
                class="node-interactive"
                :model-value="node.value"
                type="textarea"
                :rows="node.type === 'world' ? 7 : node.type === 'conflict' ? 5 : 5"
                resize="none"
                :placeholder="node.type === 'world' ? '写入世界规则、时代、势力、限制条件...' : '直接在卡片内编辑内容'"
                @update:model-value="updateNodeValue(node, String($event))"
              />
              <div class="material-card-form__actions">
                <el-button class="node-interactive" size="small" @click.stop="setMaterialEditing(node.id, false)">完成</el-button>
              </div>
            </div>
            <div v-else class="material-card-display">
              <dl class="material-card-display__rows">
                <div
                  v-for="(item, index) in formatMaterialDisplayRows(node.value)"
                  :key="`${node.id}-${item.label}-${index}`"
                  class="material-card-display__row"
                >
                  <dt>{{ item.label }}</dt>
                  <dd>{{ item.value }}</dd>
                </div>
              </dl>
              <div class="material-card-display__actions">
                <el-button class="node-interactive" size="small" text @click.stop="setMaterialEditing(node.id, true)">编辑</el-button>
              </div>
            </div>
          </template>
        </article>
      </div>
    </div>

    <el-dialog v-model="conflictDialogVisible" title="选择冲突人物" width="520px">
      <div class="conflict-dialog">
        <el-checkbox-group v-model="selectedConflictCharacterIds" class="conflict-character-list">
          <el-checkbox
            v-for="character in characterNodes"
            :key="character.id"
            :label="character.id"
            border
          >
            {{ character.title }}
          </el-checkbox>
        </el-checkbox-group>
      </div>
      <template #footer>
        <el-button @click="conflictDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="createConflict">生成冲突卡片</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<style scoped>
.material-map-page {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  padding: 24px 32px 0;
  box-sizing: border-box;
}

.material-map-page__header {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 64px;
  margin-bottom: 8px;
}

.material-map-page__header :slotted(.workspace-back) {
  position: absolute;
  left: 0;
}

.material-map-page__switch {
  display: flex;
  justify-content: center;
}

.material-map-page__actions {
  position: absolute;
  top: 16px;
  right: 16px;
  z-index: 22;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
  max-width: min(560px, calc(50% - 130px));
  padding: 10px;
  border: 1px solid rgba(255, 255, 255, 0.64);
  border-radius: 20px;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.9), rgba(239, 246, 255, 0.58)),
    rgba(255, 255, 255, 0.72);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.82),
    0 16px 38px rgba(15, 23, 42, 0.1);
  backdrop-filter: blur(18px) saturate(145%);
  -webkit-backdrop-filter: blur(18px) saturate(145%);
}

.mindmap-canvas {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  border: 1px solid #dbeafe;
  border-radius: 30px 30px 0 0;
  background-color: #f8fafc;
  background-image:
    linear-gradient(#e2e8f0 1px, transparent 1px),
    linear-gradient(90deg, #e2e8f0 1px, transparent 1px);
  background-size: 30px 30px;
  box-shadow: 0 24px 70px rgba(15, 23, 42, 0.08);
  cursor: grab;
  touch-action: none;
  user-select: none;
}

.mindmap-canvas.is-panning {
  cursor: grabbing;
}

.mindmap-canvas.is-drop-active {
  box-shadow: inset 0 0 0 2px rgba(20, 184, 166, 0.35), 0 24px 70px rgba(15, 23, 42, 0.08);
}

.canvas-toolbar {
  position: absolute;
  left: 50%;
  top: 16px;
  z-index: 20;
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
  padding: 10px;
  border: 1px solid rgba(148, 163, 184, 0.28);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 14px 36px rgba(15, 23, 42, 0.08);
  transform: translateX(-50%);
}

.canvas-createbar {
  position: absolute;
  left: 16px;
  top: 16px;
  z-index: 21;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  border: 1px solid rgba(255, 255, 255, 0.64);
  border-radius: 20px;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.9), rgba(240, 253, 250, 0.54)),
    rgba(255, 255, 255, 0.72);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.82),
    0 16px 38px rgba(15, 23, 42, 0.1);
  backdrop-filter: blur(18px) saturate(145%);
  -webkit-backdrop-filter: blur(18px) saturate(145%);
}

.canvas-createbar :deep(.el-dropdown) {
  display: inline-flex;
}

.canvas-createbar__button {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  height: 36px;
  padding: 0 13px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.74);
  color: #334155;
  font-size: 13px;
  font-weight: 850;
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.06);
  cursor: pointer;
  transition: border-color 0.18s ease, color 0.18s ease, transform 0.18s ease, box-shadow 0.18s ease, background 0.18s ease;
}

.canvas-createbar__button:hover {
  transform: translateY(-1px);
}

.canvas-createbar__button:disabled {
  opacity: 0.58;
  cursor: not-allowed;
  transform: none;
}

.canvas-createbar__button--character:hover {
  border-color: rgba(20, 184, 166, 0.45);
  background: #f0fdfa;
  color: #0f766e;
  box-shadow: 0 12px 24px rgba(20, 184, 166, 0.16);
}

.canvas-createbar__button--conflict:hover {
  border-color: rgba(249, 115, 22, 0.42);
  background: #fff7ed;
  color: #ea580c;
  box-shadow: 0 12px 24px rgba(249, 115, 22, 0.14);
}

.canvas-createbar__button--idea:hover {
  border-color: rgba(124, 58, 237, 0.36);
  background: #faf5ff;
  color: #7c3aed;
  box-shadow: 0 12px 24px rgba(124, 58, 237, 0.13);
}

.canvas-createbar__button--save:hover {
  border-color: rgba(37, 99, 235, 0.38);
  background: #eff6ff;
  color: #2563eb;
  box-shadow: 0 12px 24px rgba(37, 99, 235, 0.13);
}

.canvas-createbar__button--extract:hover {
  border-color: rgba(20, 184, 166, 0.45);
  background: #f0fdfa;
  color: #0f766e;
  box-shadow: 0 12px 24px rgba(20, 184, 166, 0.16);
}

.canvas-createbar__button--outline:hover {
  border-color: rgba(249, 115, 22, 0.42);
  background: #fff7ed;
  color: #ea580c;
  box-shadow: 0 12px 24px rgba(249, 115, 22, 0.14);
}

.conflict-dialog {
  display: grid;
  gap: 14px;
}

.conflict-character-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.conflict-character-list :deep(.el-checkbox) {
  height: auto;
  margin: 0;
  padding: 10px 12px;
  white-space: normal;
}

.mindmap-scroller {
  position: absolute;
  left: 0;
  top: 0;
  will-change: transform;
}

.canvas-edges {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.canvas-edges path {
  fill: none;
  stroke: rgba(15, 118, 110, 0.28);
  stroke-width: 3;
  stroke-linecap: round;
}

.canvas-edges path.is-dashed {
  stroke: rgba(124, 58, 237, 0.26);
  stroke-dasharray: 10 10;
}

.canvas-node {
  position: absolute;
  z-index: 3;
  padding: 16px;
  border: 1px solid rgba(226, 232, 240, 0.95);
  border-radius: 22px;
  background: rgba(255, 255, 255, 0.94);
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.1);
  cursor: default;
  user-select: text;
}

.canvas-node--world {
  border-color: rgba(37, 99, 235, 0.32);
  background: linear-gradient(180deg, #eff6ff 0%, #ffffff 100%);
}

.canvas-node--character {
  border-color: rgba(20, 184, 166, 0.32);
  background: linear-gradient(180deg, #f0fdfa 0%, #ffffff 100%);
}

.canvas-node--conflict {
  border-color: rgba(249, 115, 22, 0.32);
  background: linear-gradient(180deg, #fff7ed 0%, #ffffff 100%);
}

.canvas-node--idea {
  border-color: rgba(124, 58, 237, 0.28);
  background: linear-gradient(180deg, #faf5ff 0%, #ffffff 100%);
  cursor: grab;
}

.canvas-node--drop-target {
  border-color: rgba(20, 184, 166, 0.85);
  box-shadow: 0 0 0 4px rgba(20, 184, 166, 0.14), 0 22px 56px rgba(15, 23, 42, 0.16);
}

.canvas-node__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 10px;
  cursor: grab;
}

.canvas-node__header span,
.role-chip,
.canvas-node__header strong {
  display: block;
}

.canvas-node__header span,
.role-chip {
  width: fit-content;
  padding: 0;
  border: 0;
  background: transparent;
  color: #0f766e;
  font-size: 11px;
  font-weight: 900;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  cursor: pointer;
}

.canvas-node__header strong {
  margin-top: 4px;
  color: #0f172a;
  font-size: 18px;
}

.character-card-form {
  display: grid;
  gap: 10px;
}

.character-card-form__actions,
.material-card-form__actions {
  display: flex;
  justify-content: flex-end;
}

.material-card-form {
  display: grid;
  gap: 10px;
}

.character-card-display,
.material-card-display {
  display: flex;
  min-height: 132px;
  flex-direction: column;
  justify-content: space-between;
  gap: 12px;
}

.material-card-display {
  min-height: 112px;
}

.character-card-display__rows,
.material-card-display__rows {
  display: grid;
  gap: 8px;
  margin: 0;
}

.character-card-display__row,
.material-card-display__row {
  display: grid;
  grid-template-columns: 52px minmax(0, 1fr);
  gap: 8px;
}

.material-card-display__row {
  grid-template-columns: minmax(52px, max-content) minmax(0, 1fr);
}

.character-card-display__row dt,
.material-card-display__row dt {
  color: #0f766e;
  font-size: 12px;
  font-weight: 800;
}

.material-card-display__row dt {
  max-width: 92px;
  word-break: break-word;
}

.character-card-display__row dd,
.material-card-display__row dd {
  margin: 0;
  color: #0f172a;
  font-size: 13px;
  line-height: 1.45;
  word-break: break-word;
  white-space: pre-wrap;
}

.character-card-display__actions,
.material-card-display__actions {
  display: flex;
  justify-content: flex-end;
}

.canvas-node--world .canvas-node__header span {
  color: #2563eb;
}

.canvas-node--conflict .canvas-node__header span {
  color: #ea580c;
}

.canvas-node--idea .canvas-node__header span {
  color: #7c3aed;
}

@media (max-width: 900px) {
  .material-map-page {
    padding: 16px 12px 0;
  }

  .material-map-page__header {
    align-items: flex-start;
    flex-direction: column;
    min-height: 0;
    gap: 12px;
  }

  .material-map-page__header :slotted(.workspace-back) {
    position: static;
  }

  .material-map-page__actions {
    position: absolute;
    top: 76px;
    left: 12px;
    right: 12px;
    max-width: none;
    justify-content: flex-start;
  }

  .canvas-toolbar {
    top: 16px;
    right: 12px;
    left: auto;
    max-width: none;
    transform: none;
  }

  .canvas-createbar {
    right: 12px;
    flex-wrap: wrap;
  }
}
</style>
