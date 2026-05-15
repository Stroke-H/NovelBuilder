<script setup lang="ts">
import type { NovelProject } from '@/api/novelWriter'

defineProps<{
  projects: NovelProject[]
  selectedId: string
  loading: boolean
}>()

const emit = defineEmits<{
  select: [project: NovelProject]
  create: []
  settings: []
  export: [project: NovelProject]
  delete: [project: NovelProject]
}>()

const stageLabelMap: Record<string, string> = {
  material_ready: '素材整理中',
  info_extracted: '事实库已生成',
  outline_ready: '大纲已生成',
  style_ready: '文风画像已生成',
  chapter_drafted: '章节创作中',
  chapter_audited: '审计修订中',
  chapter_approved: '章节已确认'
}

const formatStage = (stage: string) => stageLabelMap[stage] || '编辑中'
</script>

<template>
  <section class="project-list">
    <div class="project-list__header">
      <div>
        <p class="project-list__eyebrow">Novel Generater</p>
        <h2 class="project-list__title">小说创作入口</h2>
        <p class="project-list__desc">选择一本正在编辑或已保存的小说，继续进入素材图谱与文风生成流程。</p>
      </div>
      <div class="project-list__header-actions">
        <el-button @click="emit('settings')">设置</el-button>
        <el-button type="primary" @click="emit('create')">新建小说</el-button>
      </div>
    </div>

    <el-skeleton v-if="loading" :rows="4" animated />
    <el-empty v-else-if="projects.length === 0" description="暂无小说项目">
      <el-button type="primary" @click="emit('create')">创建第一本小说</el-button>
    </el-empty>
    <div v-else class="project-list__items">
      <article v-for="project in projects" :key="project.id" class="project-card-shell">
        <div
          :class="['project-card', { 'project-card--active': project.id === selectedId }]"
          role="button"
          tabindex="0"
          @click="emit('select', project)"
          @keydown.enter.prevent="emit('select', project)"
          @keydown.space.prevent="emit('select', project)"
        >
          <div class="project-card__hero">
            <div class="project-card__cover">
              <div class="project-card__cover-content">
                <span>{{ project.genre || '小说' }}</span>
                <strong>{{ project.title }}</strong>
                <small>{{ project.outline?.chapters?.length || 0 }} 章大纲</small>
              </div>
            </div>
            <div class="project-card__body">
              <strong class="project-card__title">{{ project.title }}</strong>
              <p class="project-card__subtitle">{{ project.genre || '未设置题材' }} · {{ project.chapters?.length || 0 }} 章正文</p>
              <div class="project-card__stats">
                <span>{{ formatStage(project.current_stage || 'material_ready') }}</span>
                <span>{{ project.outline?.chapters?.length || 0 }} 个大纲章节</span>
              </div>
              <p class="project-card__summary">
                更新于 {{ project.updated_at || project.created_at }}，继续进入素材图谱、文风生成与章节创作流程。
              </p>
              <div class="project-card__cta">
                <span>继续编辑</span>
              </div>
            </div>
          </div>
          <div class="project-card__footer">
            <span class="project-card__stage">
              <i class="project-card__stage-dot" />
              {{ formatStage(project.current_stage || 'material_ready') }}
            </span>
            <div class="project-card__actions">
              <button class="project-card__action project-card__action--soft" type="button" @click.stop="emit('export', project)">
                导出 Markdown
              </button>
              <button class="project-card__action project-card__action--danger" type="button" @click.stop="emit('delete', project)">
                删除
              </button>
            </div>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>

<style scoped>
.project-list {
  width: 100%;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 28px;
  padding: 26px;
  min-height: 620px;
  box-shadow: 0 20px 60px rgba(15, 23, 42, 0.06);
}

.project-list__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 18px;
}

.project-list__header-actions {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}

.project-list__eyebrow {
  margin: 0 0 4px;
  color: #0f766e;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.project-list__title {
  margin: 0;
  color: #0f172a;
  font-size: 28px;
}

.project-list__desc {
  margin: 8px 0 0;
  color: #64748b;
  line-height: 1.7;
}

.project-list__items {
  display: grid;
  grid-template-columns: repeat(3, minmax(280px, 32%));
  align-items: start;
  justify-content: space-between;
  gap: 28px 2%;
}

.project-card-shell {
  min-height: 291px;
}

.project-card {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
  min-height: 291px;
  overflow: visible;
  padding: 0;
  text-align: left;
  border: 0;
  border-radius: 10px;
  background: #eef2f6;
  box-shadow: none;
  cursor: pointer;
  transition: 0.2s ease;
  color: #64748b;
  outline: none;
}

.project-card:hover,
.project-card--active,
.project-card:focus-visible {
  transform: translateY(-1px);
  box-shadow: 0 18px 40px rgba(15, 23, 42, 0.12);
}

.project-card__hero {
  position: relative;
  display: flex;
  min-height: 220px;
  padding: 28px 28px 0 28px;
  background: #f6a3ab;
  border-radius: 10px 10px 0 0;
  overflow: visible;
}

.project-card-shell:nth-child(5n + 2) .project-card__hero {
  background: #9edce8;
}

.project-card-shell:nth-child(5n + 3) .project-card__hero {
  background: #e7b7d4;
}

.project-card-shell:nth-child(5n + 4) .project-card__hero {
  background: #f7ca96;
}

.project-card-shell:nth-child(5n) .project-card__hero {
  background: #c8b3e3;
}

.project-card__hero::after {
  content: none;
}

.project-card__cover {
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
  transition: 0.3s ease;
}

.project-card__cover::before {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(90deg, rgba(255, 255, 255, 0.12), transparent 18%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.28), transparent 38%, rgba(15, 23, 42, 0.18));
  content: "";
}

.project-card:hover .project-card__cover,
.project-card--active .project-card__cover,
.project-card:focus-visible .project-card__cover {
  transform: scale(1.035);
}

.project-card__cover-content {
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

.project-card__cover-content span,
.project-card__cover-content small {
  position: absolute;
  left: 14px;
  right: 14px;
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.project-card__cover-content span {
  top: 16px;
}

.project-card__cover-content small {
  bottom: 16px;
}

.project-card__cover-content strong {
  display: -webkit-box;
  overflow: hidden;
  margin: 0;
  font-size: 24px;
  line-height: 1.15;
  letter-spacing: 0;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 5;
}

.project-card__body {
  display: flex;
  min-width: 0;
  max-width: 300px;
  flex-direction: column;
  overflow: hidden;
  padding: 0 0 0 22px;
  color: #ffffff;
}

.project-card__stage {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: 0 0 auto;
  color: #64748b;
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
}

.project-card__stage-dot {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: #14b8a6;
  box-shadow: 0 0 0 3px rgba(20, 184, 166, 0.12);
}

.project-card__title {
  display: block;
  margin: 0 0 3px;
  color: #ffffff;
  font-size: 20px;
  font-weight: 800;
  line-height: 1.2;
}

.project-card__subtitle {
  margin: 0 0 12px;
  color: #ffffff;
  font-size: 13px;
  line-height: 1.5;
}

.project-card__stats {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 18px;
  color: #ffffff;
  font-size: 13px;
  line-height: 1.1;
}

.project-card__stats span {
  position: relative;
}

.project-card__stats span + span::before {
  content: none;
}

.project-card__summary {
  display: -webkit-box;
  overflow: hidden;
  margin: 0 0 22px;
  color: #ffffff;
  font-size: 13px;
  line-height: 1.35;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.project-card__cta {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 29px;
  width: min(168px, 100%);
  border-radius: 999px;
  background: #ffffff;
  color: inherit;
  font-size: 14px;
  font-weight: 800;
}

.project-card__cta span {
  color: inherit;
}

.project-card-shell:nth-child(5n + 1) .project-card__cta {
  color: #e9939d;
}

.project-card-shell:nth-child(5n + 2) .project-card__cta {
  color: #72bac8;
}

.project-card-shell:nth-child(5n + 3) .project-card__cta {
  color: #cf97b9;
}

.project-card-shell:nth-child(5n + 4) .project-card__cta {
  color: #efaf64;
}

.project-card-shell:nth-child(5n) .project-card__cta {
  color: #a27bc8;
}

.project-card__footer {
  display: flex;
  align-items: center;
  flex-wrap: nowrap;
  justify-content: flex-start;
  gap: 8px;
  min-height: 71px;
  padding: 12px 16px 10px 210px;
  background: #eef2f6;
  border-radius: 0 0 10px 10px;
}

.project-card__actions {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: nowrap;
  justify-content: flex-end;
  margin-left: auto;
  gap: 8px;
}

.project-card__action {
  min-height: 30px;
  padding: 0 10px;
  border: 0;
  border-radius: 999px;
  background: #eef2f7;
  color: #5b6b84;
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
  transition: 0.2s ease;
}

.project-card__action:hover {
  transform: translateY(-1px);
}

.project-card__action--soft {
  background: #ffffff;
  color: #5b6b84;
  box-shadow: inset 0 0 0 1px #d6deea;
}

.project-card__action--danger {
  background: #fff1f2;
  color: #e16b75;
  box-shadow: inset 0 0 0 1px #f5c7cd;
}

@media (max-width: 960px) {
  .project-list__items {
    grid-template-columns: 1fr;
    justify-content: stretch;
  }
}

@media (max-width: 760px) {
  .project-list__header {
    align-items: flex-start;
    flex-direction: column;
  }

  .project-list__items {
    gap: 24px;
  }

  .project-card {
    min-height: 0;
  }

  .project-card-shell {
    min-height: 0;
  }

  .project-card__hero {
    min-height: 0;
    flex-direction: column;
    padding: 18px 18px 0;
  }

  .project-card__hero::after {
    content: none;
  }

  .project-card__cover {
    width: 160px;
    height: 240px;
    margin: 8px auto -48px;
  }

  .project-card__cover-content strong {
    font-size: 30px;
  }

  .project-card__body {
    max-width: none;
    padding: 68px 0 24px;
  }

  .project-card__title {
    font-size: 28px;
  }

  .project-card__subtitle,
  .project-card__stats,
  .project-card__summary {
    font-size: 16px;
  }

  .project-card__stats {
    gap: 14px;
    margin-bottom: 20px;
  }

  .project-card__summary {
    margin-bottom: 24px;
  }

  .project-card__cta {
    min-height: 52px;
    width: 100%;
    font-size: 18px;
  }

  .project-card__footer {
    align-items: flex-start;
    flex-direction: column;
    min-height: 0;
    padding: 20px 18px;
  }
}
</style>
