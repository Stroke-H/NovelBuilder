package services

import (
	"encoding/json"
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

func RegisterNovelWriterSettingsRoutes(novels *gin.RouterGroup) {
	novels.GET("/settings", GetNovelWriterSettingsHandler)
	novels.PUT("/settings", UpdateNovelWriterSettingsHandler)
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
