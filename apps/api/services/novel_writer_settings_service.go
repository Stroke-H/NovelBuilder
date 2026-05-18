package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	feishumodel "novel-generater-api/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type NovelStyleTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	UpdatedAt   string `json:"updated_at"`
}

type NovelWriterGeneralSettings struct {
	MaxChapters                        int `json:"max_chapters"`
	MaxChapterWords                    int `json:"max_chapter_words"`
	DBMaxOpenConns                     int `json:"db_max_open_conns"`
	DBMaxIdleConns                     int `json:"db_max_idle_conns"`
	DBConnMaxLifetimeMinutes           int `json:"db_conn_max_lifetime_minutes"`
	DBTimeoutSeconds                   int `json:"db_timeout_seconds"`
	FrontendRequestTimeoutSeconds      int `json:"frontend_request_timeout_seconds"`
	AIRequestTimeoutSeconds            int `json:"ai_request_timeout_seconds"`
	AILongRequestTimeoutSeconds        int `json:"ai_long_request_timeout_seconds"`
	DefaultAIBackendTimeoutSeconds     int `json:"default_ai_backend_timeout_seconds"`
	ChapterAIBackendTimeoutSeconds     int `json:"chapter_ai_backend_timeout_seconds"`
	FullReviewAIBackendTimeoutSeconds  int `json:"full_review_ai_backend_timeout_seconds"`
	StyleTemplateChapterTimeoutSeconds int `json:"style_template_chapter_timeout_seconds"`
	StyleTemplateSummaryTimeoutSeconds int `json:"style_template_summary_timeout_seconds"`
	ChapterMaxTokens                   int `json:"chapter_max_tokens"`
	StyleReferenceSampleRunes          int `json:"style_reference_sample_runes"`
	AuditContentMaxRunes               int `json:"audit_content_max_runes"`
	RevisionContentMaxRunes            int `json:"revision_content_max_runes"`
	FullReviewPayloadMaxRunes          int `json:"full_review_payload_max_runes"`
	StyleTemplateChapterRunes          int `json:"style_template_chapter_runes"`
	StyleTemplateObservationsMaxRunes  int `json:"style_template_observations_max_runes"`
	MaterialRawMaxRunes                int `json:"material_raw_max_runes"`
	MaterialCharacterMaxRunes          int `json:"material_character_max_runes"`
	MaterialWorldMaxRunes              int `json:"material_world_max_runes"`
	MaterialConflictMaxRunes           int `json:"material_conflict_max_runes"`
	PromptCardLimit                    int `json:"prompt_card_limit"`
	PromptCardNameMaxRunes             int `json:"prompt_card_name_max_runes"`
	PromptCardDescriptionMaxRunes      int `json:"prompt_card_description_max_runes"`
	PromptQuestionMaxRunes             int `json:"prompt_question_max_runes"`
	OutlineInitialBatchSize            int `json:"outline_initial_batch_size"`
	OutlineBatchSize                   int `json:"outline_batch_size"`
	OutlineSmallBatchMaxTokens         int `json:"outline_small_batch_max_tokens"`
	OutlineMediumBatchMaxTokens        int `json:"outline_medium_batch_max_tokens"`
	OutlineLargeBatchMaxTokens         int `json:"outline_large_batch_max_tokens"`
	BatchRetryAttempts                 int `json:"batch_retry_attempts"`
	OutlineWaitTimeoutMinutes          int `json:"outline_wait_timeout_minutes"`
	RuntimePollingIntervalMs           int `json:"runtime_polling_interval_ms"`
	OutlinePollingIntervalMs           int `json:"outline_polling_interval_ms"`
	FinishedTaskRetentionMinutes       int `json:"finished_task_retention_minutes"`
	StyleTemplateRetryAttempts         int `json:"style_template_retry_attempts"`
}

type NovelWriterLocalSettings struct {
	General        NovelWriterGeneralSettings `json:"general"`
	StyleTemplates []NovelStyleTemplate       `json:"style_templates"`
}

type NovelWriterSettingsPayload struct {
	AIConfig       feishumodel.AIConfig       `json:"ai_config"`
	General        NovelWriterGeneralSettings `json:"general"`
	StyleTemplates []NovelStyleTemplate       `json:"style_templates"`
}

const (
	defaultNovelMaxChapters     = 200
	defaultNovelMaxChapterWords = 80000
)

type generateStyleTemplateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SourceText  string `json:"source_text"`
}

func RegisterNovelWriterSettingsRoutes(novels *gin.RouterGroup) {
	novels.GET("/settings", GetNovelWriterSettingsHandler)
	novels.PUT("/settings", UpdateNovelWriterSettingsHandler)
	novels.POST("/settings/style-templates/generate", GenerateStyleTemplateHandler)
}

func GetNovelWriterSettingsHandler(c *gin.Context) {
	settings, err := loadNovelWriterSettingsPayload()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, settings)
}

func UpdateNovelWriterSettingsHandler(c *gin.Context) {
	var req NovelWriterSettingsPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.AIConfig.ProviderGroups = normalizeAIProviderGroups(req.AIConfig.ProviderGroups)
	req.AIConfig.Providers = normalizeAIProviders(req.AIConfig.Providers)
	req.General = normalizeNovelWriterGeneralSettings(req.General)
	req.StyleTemplates = normalizeStyleTemplates(req.StyleTemplates)

	if err := feishumodel.SaveAIConfig(&req.AIConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := saveNovelWriterLocalSettings(NovelWriterLocalSettings{
		General:        req.General,
		StyleTemplates: req.StyleTemplates,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, req)
}

func GenerateStyleTemplateHandler(c *gin.Context) {
	var req generateStyleTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.SourceText = strings.TrimSpace(req.SourceText)
	if req.SourceText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参考小说内容不能为空"})
		return
	}

	chapterSegments := splitTextIntoChapters(req.SourceText)
	if len(chapterSegments) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未识别到可用的参考内容"})
		return
	}

	settingsProject := NovelProject{
		ID:    "NOVEL-WRITER-SETTINGS",
		Title: "创作设置",
	}
	taskCtx, task := beginNovelRuntimeTask(settingsProject, "style_template_generate", "文风模版生成 0/"+fmt.Sprintf("%d", len(chapterSegments))+" 章")

	go func() {
		if err := generateStyleTemplateFromNovel(taskCtx, task, req, chapterSegments); err != nil {
			if isContextCanceledError(err) {
				task.complete("cancelled", "文风模版生成已终止", "")
				return
			}
			task.complete("failed", "文风模版生成失败", err.Error())
			return
		}
		task.complete("completed", "文风模版生成完成", "")
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"ok":      true,
		"task_id": task.task.ID,
	})
}

func loadNovelWriterSettingsPayload() (NovelWriterSettingsPayload, error) {
	localSettings, err := loadNovelWriterLocalSettings()
	if err != nil {
		return NovelWriterSettingsPayload{}, err
	}
	config := feishumodel.LoadAIConfig()
	if config == nil {
		config = &feishumodel.AIConfig{}
	}
	config.ProviderGroups = normalizeAIProviderGroups(config.ProviderGroups)
	config.Providers = normalizeAIProviders(config.Providers)
	if localSettings.StyleTemplates == nil {
		localSettings.StyleTemplates = []NovelStyleTemplate{}
	}

	return NovelWriterSettingsPayload{
		AIConfig:       *config,
		General:        localSettings.General,
		StyleTemplates: localSettings.StyleTemplates,
	}, nil
}

func normalizeNovelWriterGeneralSettings(settings NovelWriterGeneralSettings) NovelWriterGeneralSettings {
	settings.MaxChapters = normalizeIntSetting(settings.MaxChapters, defaultNovelMaxChapters, 1, 1000)
	settings.MaxChapterWords = normalizeIntSetting(settings.MaxChapterWords, defaultNovelMaxChapterWords, 1200, 200000)
	settings.DBMaxOpenConns = normalizeIntSetting(settings.DBMaxOpenConns, 10, 1, 100)
	settings.DBMaxIdleConns = normalizeIntSetting(settings.DBMaxIdleConns, 5, 1, settings.DBMaxOpenConns)
	settings.DBConnMaxLifetimeMinutes = normalizeIntSetting(settings.DBConnMaxLifetimeMinutes, 30, 1, 240)
	settings.DBTimeoutSeconds = normalizeIntSetting(settings.DBTimeoutSeconds, 5, 1, 60)
	settings.FrontendRequestTimeoutSeconds = normalizeIntSetting(settings.FrontendRequestTimeoutSeconds, 15, 5, 120)
	settings.AIRequestTimeoutSeconds = normalizeIntSetting(settings.AIRequestTimeoutSeconds, 120, 30, 600)
	settings.AILongRequestTimeoutSeconds = normalizeIntSetting(settings.AILongRequestTimeoutSeconds, 300, 60, 1800)
	settings.DefaultAIBackendTimeoutSeconds = normalizeIntSetting(settings.DefaultAIBackendTimeoutSeconds, 110, 30, 600)
	settings.ChapterAIBackendTimeoutSeconds = normalizeIntSetting(settings.ChapterAIBackendTimeoutSeconds, 240, 60, 1800)
	settings.FullReviewAIBackendTimeoutSeconds = normalizeIntSetting(settings.FullReviewAIBackendTimeoutSeconds, 240, 60, 1800)
	settings.StyleTemplateChapterTimeoutSeconds = normalizeIntSetting(settings.StyleTemplateChapterTimeoutSeconds, 120, 30, 900)
	settings.StyleTemplateSummaryTimeoutSeconds = normalizeIntSetting(settings.StyleTemplateSummaryTimeoutSeconds, 180, 30, 1200)
	settings.ChapterMaxTokens = normalizeIntSetting(settings.ChapterMaxTokens, 120000, 9000, 200000)
	settings.StyleReferenceSampleRunes = normalizeIntSetting(settings.StyleReferenceSampleRunes, 12000, 3000, 60000)
	settings.AuditContentMaxRunes = normalizeIntSetting(settings.AuditContentMaxRunes, 16000, 3000, 80000)
	settings.RevisionContentMaxRunes = normalizeIntSetting(settings.RevisionContentMaxRunes, 16000, 3000, 80000)
	settings.FullReviewPayloadMaxRunes = normalizeIntSetting(settings.FullReviewPayloadMaxRunes, 220000, 20000, 500000)
	settings.StyleTemplateChapterRunes = normalizeIntSetting(settings.StyleTemplateChapterRunes, 7000, 2000, 30000)
	settings.StyleTemplateObservationsMaxRunes = normalizeIntSetting(settings.StyleTemplateObservationsMaxRunes, 70000, 10000, 200000)
	settings.MaterialRawMaxRunes = normalizeIntSetting(settings.MaterialRawMaxRunes, 4000, 1000, 20000)
	settings.MaterialCharacterMaxRunes = normalizeIntSetting(settings.MaterialCharacterMaxRunes, 6000, 1000, 30000)
	settings.MaterialWorldMaxRunes = normalizeIntSetting(settings.MaterialWorldMaxRunes, 5000, 1000, 30000)
	settings.MaterialConflictMaxRunes = normalizeIntSetting(settings.MaterialConflictMaxRunes, 5000, 1000, 30000)
	settings.PromptCardLimit = normalizeIntSetting(settings.PromptCardLimit, 40, 5, 200)
	settings.PromptCardNameMaxRunes = normalizeIntSetting(settings.PromptCardNameMaxRunes, 120, 20, 500)
	settings.PromptCardDescriptionMaxRunes = normalizeIntSetting(settings.PromptCardDescriptionMaxRunes, 500, 80, 3000)
	settings.PromptQuestionMaxRunes = normalizeIntSetting(settings.PromptQuestionMaxRunes, 300, 80, 2000)
	settings.OutlineInitialBatchSize = normalizeIntSetting(settings.OutlineInitialBatchSize, 1, 1, 10)
	settings.OutlineBatchSize = normalizeIntSetting(settings.OutlineBatchSize, 5, 1, 20)
	settings.OutlineSmallBatchMaxTokens = normalizeIntSetting(settings.OutlineSmallBatchMaxTokens, 9000, 3000, 60000)
	settings.OutlineMediumBatchMaxTokens = normalizeIntSetting(settings.OutlineMediumBatchMaxTokens, 18000, 6000, 90000)
	settings.OutlineLargeBatchMaxTokens = normalizeIntSetting(settings.OutlineLargeBatchMaxTokens, 30000, 9000, 120000)
	settings.BatchRetryAttempts = normalizeIntSetting(settings.BatchRetryAttempts, 3, 1, 10)
	settings.OutlineWaitTimeoutMinutes = normalizeIntSetting(settings.OutlineWaitTimeoutMinutes, 15, 1, 120)
	settings.RuntimePollingIntervalMs = normalizeIntSetting(settings.RuntimePollingIntervalMs, 1500, 500, 10000)
	settings.OutlinePollingIntervalMs = normalizeIntSetting(settings.OutlinePollingIntervalMs, 5000, 1000, 30000)
	settings.FinishedTaskRetentionMinutes = normalizeIntSetting(settings.FinishedTaskRetentionMinutes, 10, 1, 120)
	settings.StyleTemplateRetryAttempts = normalizeIntSetting(settings.StyleTemplateRetryAttempts, 3, 1, 10)
	return settings
}

func normalizeIntSetting(value int, fallback int, minValue int, maxValue int) int {
	if value <= 0 {
		value = fallback
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func loadNovelWriterGeneralSettings() NovelWriterGeneralSettings {
	settings, err := loadNovelWriterLocalSettings()
	if err != nil {
		return normalizeNovelWriterGeneralSettings(NovelWriterGeneralSettings{})
	}
	return normalizeNovelWriterGeneralSettings(settings.General)
}

func normalizeAIProviderGroups(groups []feishumodel.AIProviderGroupConfig) []feishumodel.AIProviderGroupConfig {
	if len(groups) == 0 {
		return []feishumodel.AIProviderGroupConfig{}
	}
	result := make([]feishumodel.AIProviderGroupConfig, 0, len(groups))
	for _, group := range groups {
		group.Name = strings.TrimSpace(group.Name)
		group.APIKey = strings.TrimSpace(group.APIKey)
		group.BaseURL = strings.TrimSpace(group.BaseURL)
		models := make([]feishumodel.AIProviderModelConfig, 0, len(group.Models))
		for _, model := range group.Models {
			name := strings.TrimSpace(model.Name)
			targetModel := strings.TrimSpace(model.Model)
			if name == "" && targetModel == "" {
				continue
			}
			models = append(models, feishumodel.AIProviderModelConfig{
				Name:       name,
				Model:      targetModel,
				Capability: strings.TrimSpace(model.Capability),
			})
		}
		group.Models = models
		if group.Name == "" && group.APIKey == "" && group.BaseURL == "" && len(group.Models) == 0 {
			continue
		}
		result = append(result, group)
	}
	return result
}

func normalizeAIProviders(providers []feishumodel.AIProviderConfig) []feishumodel.AIProviderConfig {
	if len(providers) == 0 {
		return []feishumodel.AIProviderConfig{}
	}
	result := make([]feishumodel.AIProviderConfig, 0, len(providers))
	for _, provider := range providers {
		provider.Name = strings.TrimSpace(provider.Name)
		provider.APIKey = strings.TrimSpace(provider.APIKey)
		provider.BaseURL = strings.TrimSpace(provider.BaseURL)
		provider.Model = strings.TrimSpace(provider.Model)
		provider.Capability = strings.TrimSpace(provider.Capability)
		if provider.Name == "" && provider.APIKey == "" && provider.BaseURL == "" && provider.Model == "" {
			continue
		}
		result = append(result, provider)
	}
	return result
}

func normalizeStyleTemplates(templates []NovelStyleTemplate) []NovelStyleTemplate {
	if len(templates) == 0 {
		return []NovelStyleTemplate{}
	}
	result := make([]NovelStyleTemplate, 0, len(templates))
	for _, item := range templates {
		name := strings.TrimSpace(item.Name)
		content := strings.TrimSpace(item.Content)
		description := strings.TrimSpace(item.Description)
		if name == "" && content == "" && description == "" {
			continue
		}
		updatedAt := strings.TrimSpace(item.UpdatedAt)
		if updatedAt == "" {
			updatedAt = time.Now().Format("2006-01-02 15:04:05")
		}
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = "TPL-" + uuid.NewString()
		}
		result = append(result, NovelStyleTemplate{
			ID:          id,
			Name:        firstNonEmpty(name, "未命名模版"),
			Description: description,
			Content:     content,
			UpdatedAt:   updatedAt,
		})
	}
	return result
}

func novelWriterSettingsPath() string {
	path := os.Getenv("NOVEL_GENERATER_WRITER_SETTINGS_FILE")
	if path == "" {
		path = "data/novel_writer_settings.json"
	}
	return path
}

func loadNovelWriterLocalSettings() (NovelWriterLocalSettings, error) {
	path := novelWriterSettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NovelWriterLocalSettings{
				General:        normalizeNovelWriterGeneralSettings(NovelWriterGeneralSettings{}),
				StyleTemplates: []NovelStyleTemplate{},
			}, nil
		}
		return NovelWriterLocalSettings{}, err
	}
	var settings NovelWriterLocalSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return NovelWriterLocalSettings{}, err
	}
	settings.General = normalizeNovelWriterGeneralSettings(settings.General)
	settings.StyleTemplates = normalizeStyleTemplates(settings.StyleTemplates)
	return settings, nil
}

func saveNovelWriterLocalSettings(settings NovelWriterLocalSettings) error {
	settings.General = normalizeNovelWriterGeneralSettings(settings.General)
	settings.StyleTemplates = normalizeStyleTemplates(settings.StyleTemplates)
	path := novelWriterSettingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

type chapterStyleObservation struct {
	ChapterTitle string   `json:"chapter_title"`
	Summary      string   `json:"summary"`
	Narration    string   `json:"narration"`
	Sentence     string   `json:"sentence"`
	Dialogue     string   `json:"dialogue"`
	Rhythm       string   `json:"rhythm"`
	Techniques   []string `json:"techniques"`
	AvoidRules   []string `json:"avoid_rules"`
}

func generateStyleTemplateFromNovel(
	ctx context.Context,
	task *novelRuntimeTaskEntry,
	req generateStyleTemplateRequest,
	chapterSegments []textChapterSegment,
) error {
	general := loadNovelWriterGeneralSettings()
	observations := make([]chapterStyleObservation, 0, len(chapterSegments))
	system := "你是小说文风分析师。你只提炼抽象文风规律，不复述具体情节，不模仿原句。请只输出 JSON。"

	for index, chapter := range chapterSegments {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		task.update(fmt.Sprintf("文风模版生成 %d/%d 章", index+1, len(chapterSegments)), "running")
		user := fmt.Sprintf(`请分析这一章的文风特征，输出结构化结论。
要求：
1. 不要总结剧情细节，只关注写法。
2. narration / sentence / dialogue / rhythm 要描述抽象规律。
3. techniques / avoid_rules 各输出 3-6 条短句。

输出 JSON schema:
{"chapter_title":"","summary":"","narration":"","sentence":"","dialogue":"","rhythm":"","techniques":[""],"avoid_rules":[""]}

章节标题：%s
章节正文：%s`, chapter.Title, truncateForAI(chapter.Content, general.StyleTemplateChapterRunes))

		var observation chapterStyleObservation
		if err := retryStyleTemplateAI(ctx, func() error {
			return callNovelAIJSONWithTimeoutContext(ctx, system, user, &observation, 0, time.Duration(general.StyleTemplateChapterTimeoutSeconds)*time.Second)
		}, general.StyleTemplateRetryAttempts); err != nil {
			task.update(fmt.Sprintf("文风模版生成 %d/%d 章（已跳过失败章节）", index+1, len(chapterSegments)), "running")
			continue
		}
		if strings.TrimSpace(observation.ChapterTitle) == "" {
			observation.ChapterTitle = chapter.Title
		}
		observations = append(observations, observation)
	}

	if len(observations) == 0 {
		return fmt.Errorf("所有章节文风提炼均失败，未生成可汇总内容")
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	task.update("文风模版汇总中", "running")
	content := ""
	result := NovelStyleTemplate{}
	observationsJSON := mustJSON(observations)
	if len([]rune(observationsJSON)) <= general.StyleTemplateObservationsMaxRunes {
		aggregateSystem := "你是资深小说文风总编。请把分章文风观察汇总成一份可复用的高质量文风模版，只输出 JSON。"
		aggregateUser := fmt.Sprintf(`请基于整本参考小说的逐章文风观察，生成一份精华版文风模版。
要求：
1. 输出内容必须是抽象风格规则，不要保留原作专属设定、情节、人物名称或原句。
2. description 用一句话概括该模版适合的风格。
3. content 要像可直接给写作模型使用的文风参考，结构清晰，覆盖叙述、句式、对话、节奏、爽点/钩子、禁忌项。

输出 JSON schema:
{"name":"","description":"","content":""}

用户预设名称：%s
用户预设描述：%s
逐章文风观察：%s`, req.Name, req.Description, observationsJSON)

		if err := callNovelAIJSONWithTimeoutContext(ctx, aggregateSystem, aggregateUser, &result, 0, time.Duration(general.StyleTemplateSummaryTimeoutSeconds)*time.Second); err == nil {
			content = strings.TrimSpace(result.Content)
		} else {
			task.update("文风模版汇总失败，正在使用逐章观察兜底", "running")
		}
	} else {
		task.update("逐章观察过长，正在使用本地压缩汇总", "running")
	}

	if content == "" {
		content = buildFallbackStyleTemplateContent(observations)
	}
	if content == "" {
		return fmt.Errorf("文风模版生成结果为空")
	}

	description := firstNonEmpty(
		strings.TrimSpace(req.Description),
		strings.TrimSpace(result.Description),
		buildFallbackStyleTemplateDescription(observations),
		"由整本参考小说逐章提炼生成",
	)

	settings, err := loadNovelWriterSettingsPayload()
	if err != nil {
		return err
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	template := NovelStyleTemplate{
		ID:          "TPL-" + uuid.NewString(),
		Name:        firstNonEmpty(strings.TrimSpace(req.Name), strings.TrimSpace(result.Name), "整本参考小说文风模版"),
		Description: description,
		Content:     content,
		UpdatedAt:   now,
	}
	settings.StyleTemplates = append([]NovelStyleTemplate{template}, normalizeStyleTemplates(settings.StyleTemplates)...)
	return saveNovelWriterLocalSettings(NovelWriterLocalSettings{
		General:        settings.General,
		StyleTemplates: settings.StyleTemplates,
	})
}

func buildFallbackStyleTemplateDescription(observations []chapterStyleObservation) string {
	if len(observations) == 0 {
		return ""
	}
	summaries := make([]string, 0, minInt(len(observations), 2))
	for _, observation := range observations {
		summary := strings.TrimSpace(observation.Summary)
		if summary == "" {
			continue
		}
		summaries = append(summaries, summary)
		if len(summaries) >= 2 {
			break
		}
	}
	if len(summaries) == 0 {
		return "由整本参考小说逐章提炼生成"
	}
	return strings.Join(summaries, "；")
}

func retryStyleTemplateAI(ctx context.Context, action func() error, maxAttempts int) error {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := action(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return lastErr
}

func buildFallbackStyleTemplateContent(observations []chapterStyleObservation) string {
	if len(observations) == 0 {
		return ""
	}

	joinTop := func(values []string, limit int) string {
		trimmed := make([]string, 0, limit)
		seen := map[string]bool{}
		for _, value := range values {
			text := strings.TrimSpace(value)
			if text == "" || seen[text] {
				continue
			}
			seen[text] = true
			trimmed = append(trimmed, text)
			if len(trimmed) >= limit {
				break
			}
		}
		return strings.Join(trimmed, "；")
	}

	collectBullets := func(values []string, limit int) []string {
		result := make([]string, 0, limit)
		seen := map[string]bool{}
		for _, value := range values {
			text := strings.TrimSpace(value)
			if text == "" || seen[text] {
				continue
			}
			seen[text] = true
			result = append(result, text)
			if len(result) >= limit {
				break
			}
		}
		return result
	}

	narrations := make([]string, 0, len(observations))
	sentences := make([]string, 0, len(observations))
	dialogues := make([]string, 0, len(observations))
	rhythms := make([]string, 0, len(observations))
	techniques := make([]string, 0, len(observations)*3)
	avoidRules := make([]string, 0, len(observations)*3)
	summaries := make([]string, 0, len(observations))

	for _, observation := range observations {
		narrations = append(narrations, observation.Narration)
		sentences = append(sentences, observation.Sentence)
		dialogues = append(dialogues, observation.Dialogue)
		rhythms = append(rhythms, observation.Rhythm)
		techniques = append(techniques, observation.Techniques...)
		avoidRules = append(avoidRules, observation.AvoidRules...)
		summaries = append(summaries, observation.Summary)
	}

	sections := []string{
		"# 文风总述\n" + firstNonEmpty(joinTop(summaries, 3), "整体风格以稳定叙事、明确节奏和清晰人物表达为主。"),
		"# 叙述规则\n" + firstNonEmpty(joinTop(narrations, 4), "保持稳定叙述视角，避免跳脱和信息堆砌。"),
		"# 句式规则\n" + firstNonEmpty(joinTop(sentences, 4), "句式长短交替，保证阅读流动感与重点句落点。"),
		"# 对话规则\n" + firstNonEmpty(joinTop(dialogues, 4), "对话服务人物性格与情节推进，避免无效对白。"),
		"# 节奏规则\n" + firstNonEmpty(joinTop(rhythms, 4), "推进与停顿交替，关键节点需要明确钩子和情绪抬升。"),
	}

	techniqueBullets := collectBullets(techniques, 8)
	if len(techniqueBullets) > 0 {
		sections = append(sections, "# 建议模仿\n- "+strings.Join(techniqueBullets, "\n- "))
	}

	avoidBullets := collectBullets(avoidRules, 8)
	if len(avoidBullets) > 0 {
		sections = append(sections, "# 必须避免\n- "+strings.Join(avoidBullets, "\n- "))
	}

	return strings.TrimSpace(strings.Join(sections, "\n\n"))
}
