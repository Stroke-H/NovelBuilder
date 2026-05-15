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

type NovelWriterLocalSettings struct {
	StyleTemplates []NovelStyleTemplate `json:"style_templates"`
}

type NovelWriterSettingsPayload struct {
	AIConfig       feishumodel.AIConfig `json:"ai_config"`
	StyleTemplates []NovelStyleTemplate `json:"style_templates"`
}

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
	req.StyleTemplates = normalizeStyleTemplates(req.StyleTemplates)

	if err := feishumodel.SaveAIConfig(&req.AIConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := saveNovelWriterLocalSettings(NovelWriterLocalSettings{StyleTemplates: req.StyleTemplates}); err != nil {
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
		defer task.finish()
		if err := generateStyleTemplateFromNovel(taskCtx, task, req, chapterSegments); err != nil {
			if isContextCanceledError(err) {
				return
			}
			task.update("文风模版生成失败", "cancelling")
		}
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
		StyleTemplates: localSettings.StyleTemplates,
	}, nil
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
			return NovelWriterLocalSettings{}, nil
		}
		return NovelWriterLocalSettings{}, err
	}
	var settings NovelWriterLocalSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return NovelWriterLocalSettings{}, err
	}
	settings.StyleTemplates = normalizeStyleTemplates(settings.StyleTemplates)
	return settings, nil
}

func saveNovelWriterLocalSettings(settings NovelWriterLocalSettings) error {
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
章节正文：%s`, chapter.Title, truncateForAI(chapter.Content, 7000))

		var observation chapterStyleObservation
		if err := callNovelAIJSONWithTimeoutContext(ctx, system, user, &observation, 0, 120*time.Second); err != nil {
			return err
		}
		if strings.TrimSpace(observation.ChapterTitle) == "" {
			observation.ChapterTitle = chapter.Title
		}
		observations = append(observations, observation)
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	task.update("文风模版汇总中", "running")
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
逐章文风观察：%s`, req.Name, req.Description, mustJSON(observations))

	var result NovelStyleTemplate
	if err := callNovelAIJSONWithTimeoutContext(ctx, aggregateSystem, aggregateUser, &result, 0, 180*time.Second); err != nil {
		return err
	}

	settings, err := loadNovelWriterSettingsPayload()
	if err != nil {
		return err
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	template := NovelStyleTemplate{
		ID:          "TPL-" + uuid.NewString(),
		Name:        firstNonEmpty(strings.TrimSpace(req.Name), strings.TrimSpace(result.Name), "整本参考小说文风模版"),
		Description: firstNonEmpty(strings.TrimSpace(req.Description), strings.TrimSpace(result.Description), "由整本参考小说逐章提炼生成"),
		Content:     strings.TrimSpace(result.Content),
		UpdatedAt:   now,
	}
	settings.StyleTemplates = append([]NovelStyleTemplate{template}, normalizeStyleTemplates(settings.StyleTemplates)...)
	return saveNovelWriterLocalSettings(NovelWriterLocalSettings{StyleTemplates: settings.StyleTemplates})
}
