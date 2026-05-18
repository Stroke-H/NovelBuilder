package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	feishumodel "novel-generater-api/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openai "github.com/sashabaranov/go-openai"
)

type NovelProject struct {
	ID             string             `json:"id"`
	Title          string             `json:"title"`
	Genre          string             `json:"genre"`
	TargetWords    int                `json:"target_words"`
	TargetChapters int                `json:"target_chapters"`
	Status         string             `json:"status"`
	CurrentStage   string             `json:"current_stage"`
	CreatedBy      string             `json:"created_by"`
	CreatedAt      string             `json:"created_at"`
	UpdatedAt      string             `json:"updated_at"`
	Materials      NovelMaterials     `json:"materials"`
	Extracted      NovelExtractedInfo `json:"extracted"`
	Outline        NovelOutline       `json:"outline"`
	StyleProfile   NovelStyleProfile  `json:"style_profile"`
	Chapters       []NovelChapter     `json:"chapters"`
	Memory         NovelMemory        `json:"memory"`
	FullReview     NovelFullReview    `json:"full_review"`
}

type NovelMaterials struct {
	RawText      string `json:"raw_text"`
	CharacterRaw string `json:"character_raw"`
	WorldRaw     string `json:"world_raw"`
	ConflictRaw  string `json:"conflict_raw"`
	ReferenceRaw string `json:"reference_raw"`
}

type NovelExtractedInfo struct {
	Characters    []NovelInfoCard `json:"characters"`
	WorldRules    []NovelInfoCard `json:"world_rules"`
	Conflicts     []NovelInfoCard `json:"conflicts"`
	KeyEvents     []NovelInfoCard `json:"key_events"`
	OpenQuestions []string        `json:"open_questions"`
}

type NovelInfoCard struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type NovelOutline struct {
	Logline           string                `json:"logline"`
	Acts              []NovelInfoCard       `json:"acts"`
	Chapters          []NovelChapterOutline `json:"chapters"`
	GenerationStatus  string                `json:"generation_status"`
	TargetChapters    int                   `json:"target_chapters"`
	GeneratedChapters int                   `json:"generated_chapters"`
	BatchSize         int                   `json:"batch_size"`
	GenerationError   string                `json:"generation_error"`
}

type NovelChapterOutline struct {
	ID           string              `json:"id"`
	Title        string              `json:"title"`
	Goal         string              `json:"goal"`
	Conflict     string              `json:"conflict"`
	Hook         string              `json:"hook"`
	Summary      string              `json:"summary"`
	BeforeState  NovelChapterState   `json:"before_state"`
	AfterState   NovelChapterState   `json:"after_state"`
	MustHappen   []string            `json:"must_happen"`
	TensionCurve []NovelTensionPoint `json:"tension_curve"`
	KeyScenes    []string            `json:"key_scenes"`
	NewHooks     []string            `json:"new_hooks"`
}

type NovelChapterState struct {
	Characters   []NovelCharacterState `json:"characters"`
	PlotHooks    []string              `json:"plot_hooks"`
	PlotAdvances []string              `json:"plot_advances"`
}

type NovelCharacterState struct {
	Name     string `json:"name"`
	State    string `json:"state"`
	Location string `json:"location"`
}

type NovelTensionPoint struct {
	Position int    `json:"position"`
	Value    int    `json:"value"`
	Note     string `json:"note"`
}

type NovelStyleProfile struct {
	Summary    string   `json:"summary"`
	Narration  string   `json:"narration"`
	Sentence   string   `json:"sentence"`
	Dialogue   string   `json:"dialogue"`
	Rhythm     string   `json:"rhythm"`
	DoRules    []string `json:"do_rules"`
	AvoidRules []string `json:"avoid_rules"`
}

type NovelChapter struct {
	ID              string                `json:"id"`
	OutlineID       string                `json:"outline_id"`
	Title           string                `json:"title"`
	Status          string                `json:"status"`
	Content         string                `json:"content"`
	Summary         string                `json:"summary"`
	Audit           NovelAuditReport      `json:"audit"`
	Versions        []NovelChapterVersion `json:"versions"`
	ActiveVersionID string                `json:"active_version_id"`
	CreatedAt       string                `json:"created_at"`
	UpdatedAt       string                `json:"updated_at"`
}

type NovelChapterVersion struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"created_at"`
}

type NovelAuditReport struct {
	TotalScore     int              `json:"total_score"`
	AIFlavorScore  int              `json:"ai_flavor_score"`
	CharacterScore int              `json:"character_score"`
	LogicScore     int              `json:"logic_score"`
	StyleScore     int              `json:"style_score"`
	Issues         []NovelAuditItem `json:"issues"`
	RevisionAdvice string           `json:"revision_advice"`
}

type NovelAuditItem struct {
	Severity   string `json:"severity"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Suggestion string `json:"suggestion"`
}

type NovelFullReview struct {
	TotalScore                int                    `json:"total_score"`
	CoherenceScore            int                    `json:"coherence_score"`
	LogicReasonabilityScore   int                    `json:"logic_reasonability_score"`
	CharacterConsistencyScore int                    `json:"character_consistency_score"`
	TriggerReasonabilityScore int                    `json:"trigger_reasonability_score"`
	Summary                   string                 `json:"summary"`
	Issues                    []NovelFullReviewIssue `json:"issues"`
	RevisionAdvice            string                 `json:"revision_advice"`
	ReviewedAt                string                 `json:"reviewed_at"`
	AppliedAt                 string                 `json:"applied_at"`
}

type NovelFullReviewIssue struct {
	Severity     string `json:"severity"`
	Dimension    string `json:"dimension"`
	ChapterID    string `json:"chapter_id"`
	ChapterTitle string `json:"chapter_title"`
	Title        string `json:"title"`
	Detail       string `json:"detail"`
	Suggestion   string `json:"suggestion"`
}

type NovelMemory struct {
	ChapterSummaries []NovelInfoCard `json:"chapter_summaries"`
	CharacterStates  []NovelInfoCard `json:"character_states"`
	OpenHooks        []NovelInfoCard `json:"open_hooks"`
	Timeline         []NovelInfoCard `json:"timeline"`
}

type novelCreateRequest struct {
	Title          string         `json:"title"`
	Genre          string         `json:"genre"`
	TargetWords    int            `json:"target_words"`
	TargetChapters int            `json:"target_chapters"`
	Materials      NovelMaterials `json:"materials"`
}

type novelGenerateChapterRequest struct {
	OutlineID string `json:"outline_id"`
}

func init() {
	sqlJSONTables["novel_projects"] = sqlJSONTable{
		Table:      "novel_projects",
		PrimaryKey: "id",
		Columns: []sqlJSONColumn{
			{Column: "id", Field: "id"},
			{Column: "title", Field: "title"},
			{Column: "genre", Field: "genre"},
			{Column: "target_words", Field: "target_words"},
			{Column: "target_chapters", Field: "target_chapters"},
			{Column: "status", Field: "status"},
			{Column: "current_stage", Field: "current_stage"},
			{Column: "created_by", Field: "created_by"},
			{Column: "created_at", Field: "created_at"},
			{Column: "updated_at", Field: "updated_at"},
			{Column: "materials", Field: "materials", JSON: true},
			{Column: "extracted", Field: "extracted", JSON: true},
			{Column: "outline", Field: "outline", JSON: true},
			{Column: "style_profile", Field: "style_profile", JSON: true},
			{Column: "chapters", Field: "chapters", JSON: true},
			{Column: "memory", Field: "memory", JSON: true},
			{Column: "full_review", Field: "full_review", JSON: true},
		},
	}
}

func EnsureNovelWriterTables() error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	db, _, err := DatabaseManager.DB(ctx)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS novel_projects (
			id VARCHAR(64) PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			genre VARCHAR(120) NULL,
			target_words INT NULL,
			target_chapters INT NULL,
			status VARCHAR(40) NULL,
			current_stage VARCHAR(80) NULL,
			created_by VARCHAR(120) NULL,
			created_at VARCHAR(40) NULL,
			updated_at VARCHAR(40) NULL,
			materials JSON NULL,
			extracted JSON NULL,
			outline JSON NULL,
			style_profile JSON NULL,
			chapters JSON NULL,
			memory JSON NULL,
			full_review JSON NULL,
			raw_json JSON NOT NULL,
			migrated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	return ensureNovelProjectsFullReviewColumn(ctx, db)
}

func ensureNovelProjectsFullReviewColumn(ctx context.Context, db *sql.DB) error {
	var columnName string
	err := db.QueryRowContext(ctx, `
		SELECT COLUMN_NAME
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
			AND TABLE_NAME = 'novel_projects'
			AND COLUMN_NAME = 'full_review'
		LIMIT 1
	`).Scan(&columnName)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	_, err = db.ExecContext(ctx, "ALTER TABLE novel_projects ADD COLUMN full_review JSON NULL AFTER memory")
	return err
}

func RegisterNovelWriterRoutes(api *gin.RouterGroup) {
	novels := api.Group("/novel-writer")
	novels.Use(func(c *gin.Context) {
		if novelWriterAuthDisabled() {
			c.Next()
			return
		}
		_, err := CurrentUserFromRequest(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid token"})
			c.Abort()
			return
		}
		c.Next()
	})

	RegisterNovelWriterSettingsRoutes(novels)
	novels.GET("/projects", ListNovelProjectsHandler)
	novels.POST("/projects", CreateNovelProjectHandler)
	novels.GET("/projects/:id", GetNovelProjectHandler)
	novels.PUT("/projects/:id", UpdateNovelProjectHandler)
	novels.DELETE("/projects/:id", DeleteNovelProjectHandler)
	novels.POST("/projects/:id/delete", DeleteNovelProjectHandler)
	novels.POST("/projects/:id/extract", ExtractNovelInfoHandler)
	novels.POST("/projects/:id/outline", PlanNovelOutlineHandler)
	novels.POST("/projects/:id/style", AnalyzeNovelStyleHandler)
	novels.POST("/projects/:id/chapters/generate", GenerateNovelChapterHandler)
	novels.POST("/projects/:id/chapters/:chapterId/audit", AuditNovelChapterHandler)
	novels.POST("/projects/:id/chapters/:chapterId/revise", ReviseNovelChapterHandler)
	novels.POST("/projects/:id/chapters/:chapterId/versions/:versionId/adopt", AdoptNovelChapterVersionHandler)
	novels.POST("/projects/:id/chapters/:chapterId/approve", ApproveNovelChapterHandler)
	novels.POST("/projects/:id/full-review", ReviewNovelFullQualityHandler)
	novels.POST("/projects/:id/full-review/revise", ReviseNovelByFullReviewHandler)
	novels.GET("/tasks", ListNovelRuntimeTasksHandler)
	novels.POST("/tasks/:taskId/cancel", CancelNovelRuntimeTaskHandler)
}

func novelWriterAuthDisabled() bool {
	value := strings.TrimSpace(os.Getenv("NOVEL_GENERATER_AUTH_DISABLED"))
	return value == "" || value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}

func currentNovelWriterUser(c *gin.Context) *User {
	user, err := CurrentUserFromRequest(c)
	if err == nil && user != nil {
		return user
	}
	if !novelWriterAuthDisabled() {
		return nil
	}
	username := strings.TrimSpace(os.Getenv("NOVEL_GENERATER_DEFAULT_USERNAME"))
	if username == "" {
		username = "local-writer"
	}
	return &User{
		ID:       "local-writer",
		Username: username,
		Nickname: "本地创作者",
	}
}

func ListNovelProjectsHandler(c *gin.Context) {
	projects, err := listNovelProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, projects)
}

func CreateNovelProjectHandler(c *gin.Context) {
	user := currentNovelWriterUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid token"})
		return
	}
	var req novelCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	targetWords, targetChapters := clampNovelProjectTargets(req.TargetWords, req.TargetChapters)

	now := time.Now().Format("2006-01-02 15:04:05")
	project := NovelProject{
		ID:             "NOVEL-" + uuid.NewString(),
		Title:          strings.TrimSpace(req.Title),
		Genre:          strings.TrimSpace(req.Genre),
		TargetWords:    targetWords,
		TargetChapters: targetChapters,
		Status:         "draft",
		CurrentStage:   "material_ready",
		CreatedBy:      user.Username,
		CreatedAt:      now,
		UpdatedAt:      now,
		Materials:      req.Materials,
		Chapters:       []NovelChapter{},
	}
	if err := saveNovelProject(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, project)
}

func GetNovelProjectHandler(c *gin.Context) {
	project, err := getNovelProject(c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if sqlErrNoRows(err) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, project)
}

func UpdateNovelProjectHandler(c *gin.Context) {
	project, err := getNovelProject(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	var req NovelProject
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	project.Title = firstNonEmpty(strings.TrimSpace(req.Title), project.Title)
	project.Genre = strings.TrimSpace(req.Genre)
	project.TargetWords, project.TargetChapters = clampNovelProjectTargets(req.TargetWords, req.TargetChapters)
	project.Materials = req.Materials
	project.Extracted = req.Extracted
	if shouldKeepPersistedOutline(project.Outline, req.Outline) {
		req.Outline = project.Outline
	}
	project.Outline = req.Outline
	project.StyleProfile = req.StyleProfile
	chaptersChanged := mustJSON(project.Chapters) != mustJSON(req.Chapters)
	project.Chapters = req.Chapters
	project.Memory = req.Memory
	project.FullReview = req.FullReview
	if chaptersChanged {
		clearNovelFullReview(&project)
	}
	project.Status = firstNonEmpty(strings.TrimSpace(req.Status), project.Status)
	project.CurrentStage = firstNonEmpty(strings.TrimSpace(req.CurrentStage), project.CurrentStage)
	project.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	if err := saveNovelProject(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, project)
}

func shouldKeepPersistedOutline(persisted NovelOutline, incoming NovelOutline) bool {
	if len(persisted.Chapters) <= len(incoming.Chapters) {
		return false
	}
	return persisted.GenerationStatus == "generating" || persisted.GenerationStatus == "ready"
}

func clampNovelProjectTargets(targetWords int, targetChapters int) (int, int) {
	general := loadNovelWriterGeneralSettings()
	if targetChapters > general.MaxChapters {
		targetChapters = general.MaxChapters
	}
	if targetWords > 0 && targetChapters > 0 {
		maxTotalWords := targetChapters * general.MaxChapterWords
		if targetWords > maxTotalWords {
			targetWords = maxTotalWords
		}
	}
	return targetWords, targetChapters
}

func clampNovelAuditScore(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func normalizeNovelAuditReport(report *NovelAuditReport) {
	values := []int{
		report.TotalScore,
		report.AIFlavorScore,
		report.CharacterScore,
		report.LogicScore,
		report.StyleScore,
	}
	maxScore := 0
	for _, value := range values {
		if value > maxScore {
			maxScore = value
		}
	}
	legacyTenScale := maxScore > 0 && maxScore <= 10
	scale := 1
	if legacyTenScale {
		scale = 10
	}

	report.AIFlavorScore = clampNovelAuditScore(report.AIFlavorScore * scale)
	report.CharacterScore = clampNovelAuditScore(report.CharacterScore * scale)
	report.LogicScore = clampNovelAuditScore(report.LogicScore * scale)
	report.StyleScore = clampNovelAuditScore(report.StyleScore * scale)
	report.TotalScore = clampNovelAuditScore(report.TotalScore * scale)

	if report.TotalScore == 0 {
		report.TotalScore = clampNovelAuditScore((report.AIFlavorScore + report.CharacterScore + report.LogicScore + report.StyleScore + 2) / 4)
	}
}

func normalizeNovelFullReviewReport(report *NovelFullReview) {
	values := []int{
		report.TotalScore,
		report.CoherenceScore,
		report.LogicReasonabilityScore,
		report.CharacterConsistencyScore,
		report.TriggerReasonabilityScore,
	}
	maxScore := 0
	for _, value := range values {
		if value > maxScore {
			maxScore = value
		}
	}
	legacyTenScale := maxScore > 0 && maxScore <= 10
	scale := 1
	if legacyTenScale {
		scale = 10
	}

	report.TotalScore = clampNovelAuditScore(report.TotalScore * scale)
	report.CoherenceScore = clampNovelAuditScore(report.CoherenceScore * scale)
	report.LogicReasonabilityScore = clampNovelAuditScore(report.LogicReasonabilityScore * scale)
	report.CharacterConsistencyScore = clampNovelAuditScore(report.CharacterConsistencyScore * scale)
	report.TriggerReasonabilityScore = clampNovelAuditScore(report.TriggerReasonabilityScore * scale)

	if report.TotalScore == 0 {
		report.TotalScore = clampNovelAuditScore((report.CoherenceScore + report.LogicReasonabilityScore + report.CharacterConsistencyScore + report.TriggerReasonabilityScore + 2) / 4)
	}

	for index := range report.Issues {
		report.Issues[index].Severity = normalizeNovelSeverity(report.Issues[index].Severity)
		report.Issues[index].Dimension = strings.TrimSpace(report.Issues[index].Dimension)
		report.Issues[index].ChapterID = strings.TrimSpace(report.Issues[index].ChapterID)
		report.Issues[index].ChapterTitle = strings.TrimSpace(report.Issues[index].ChapterTitle)
		report.Issues[index].Title = strings.TrimSpace(report.Issues[index].Title)
		report.Issues[index].Detail = strings.TrimSpace(report.Issues[index].Detail)
		report.Issues[index].Suggestion = strings.TrimSpace(report.Issues[index].Suggestion)
	}
}

func normalizeNovelSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high", "medium", "low":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "medium"
	}
}

func clearNovelFullReview(project *NovelProject) {
	if project == nil {
		return
	}
	project.FullReview = NovelFullReview{}
}

func DeleteNovelProjectHandler(c *gin.Context) {
	if err := deleteNovelProject(c.Param("id")); err != nil {
		status := http.StatusInternalServerError
		if sqlErrNoRows(err) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func ListNovelRuntimeTasksHandler(c *gin.Context) {
	c.JSON(http.StatusOK, listNovelRuntimeTasks())
}

func CancelNovelRuntimeTaskHandler(c *gin.Context) {
	if err := cancelNovelRuntimeTask(strings.TrimSpace(c.Param("taskId"))); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func ExtractNovelInfoHandler(c *gin.Context) {
	project, err := getNovelProject(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	taskCtx, task := beginNovelRuntimeTask(project, "extract", "信息提取")
	defer task.finish()
	system := "你是严谨的小说设定编辑。请只输出 JSON，不要输出 Markdown。"
	user := fmt.Sprintf(`从以下素材中提取小说创作事实库。
要求：
1. 不要虚构素材中没有的信息，缺失内容放入 open_questions。
2. characters/world_rules/conflicts/key_events 都用 name + description。
3. 输出 JSON schema:
{"characters":[{"name":"","description":""}],"world_rules":[{"name":"","description":""}],"conflicts":[{"name":"","description":""}],"key_events":[{"name":"","description":""}],"open_questions":[""]}

素材：
人物：%s
世界观：%s
冲突事件：%s
其他素材：%s`, project.Materials.CharacterRaw, project.Materials.WorldRaw, project.Materials.ConflictRaw, project.Materials.RawText)

	var extracted NovelExtractedInfo
	if err := callNovelAIJSONContext(taskCtx, system, user, &extracted); err != nil {
		if isContextCanceledError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "信息提取任务已终止"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	project.Extracted = extracted
	project.CurrentStage = "info_extracted"
	project.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	if err := saveNovelProjectPreservingGeneratedOutline(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, project)
}

func PlanNovelOutlineHandler(c *gin.Context) {
	project, err := getNovelProject(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	taskCtx, task := beginNovelRuntimeTask(project, "outline", "大纲生成")
	chapterCount := project.TargetChapters
	general := loadNovelWriterGeneralSettings()
	if chapterCount > general.MaxChapters {
		chapterCount = general.MaxChapters
	}
	if chapterCount <= 0 {
		chapterCount = minInt(8, general.MaxChapters)
	}
	batchSize := general.OutlineInitialBatchSize
	if chapterCount < batchSize {
		batchSize = chapterCount
	}
	outline, err := generateNovelOutlineBatch(taskCtx, project, 1, batchSize, chapterCount, NovelOutline{})
	if err != nil {
		task.finish()
		if isContextCanceledError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "大纲生成任务已终止"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	outline.TargetChapters = chapterCount
	outline.GeneratedChapters = len(outline.Chapters)
	outline.BatchSize = general.OutlineBatchSize
	outline.GenerationStatus = "ready"
	outline.GenerationError = ""
	if len(outline.Chapters) < chapterCount {
		outline.GenerationStatus = "generating"
	}
	project.Outline = outline
	// Regenerating the outline starts a new chapter plan, so stale chapter drafts,
	// audits, and memory derived from the previous outline must be discarded.
	project.Chapters = []NovelChapter{}
	project.Memory = NovelMemory{}
	clearNovelFullReview(&project)
	project.CurrentStage = "outline_ready"
	project.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	if err := saveNovelProject(project); err != nil {
		task.finish()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if outline.GenerationStatus == "generating" {
		go func() {
			defer task.finish()
			continueNovelOutlineGeneration(taskCtx, project.ID)
		}()
	} else {
		task.finish()
	}
	c.JSON(http.StatusOK, project)
}

func AnalyzeNovelStyleHandler(c *gin.Context) {
	project, err := getNovelProject(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	taskCtx, task := beginNovelRuntimeTask(project, "style", "文风画像")
	defer task.finish()
	general := loadNovelWriterGeneralSettings()
	system := "你是文风分析师。你只能提炼抽象风格规则，不能复刻原文句子、设定、人物或情节。请只输出 JSON。"
	referenceText := strings.TrimSpace(project.Materials.ReferenceRaw)
	referenceInstruction := "用户未提供文风参考文本。请结合小说题材、事实库和商业网文可读性，生成一套原创、自然、不带明显 AI 味的默认文风画像。"
	if referenceText != "" {
		referenceInstruction = fmt.Sprintf("参考文本（若全文较长，已按开篇、中段、后段抽样）：\n%s", sampleTextForAI(referenceText, general.StyleReferenceSampleRunes))
	}
	user := fmt.Sprintf(`分析参考文本的整体文风，用于后续原创小说的抽象风格指导。
输出 JSON schema:
{"summary":"","narration":"","sentence":"","dialogue":"","rhythm":"","do_rules":[""],"avoid_rules":[""]}

小说：%s
题材：%s
事实库：%s
%s`, project.Title, project.Genre, mustJSON(project.Extracted), referenceInstruction)

	var profile NovelStyleProfile
	if err := callNovelAIJSONContext(taskCtx, system, user, &profile); err != nil {
		if isContextCanceledError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "文风画像任务已终止"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	project.StyleProfile = profile
	project.CurrentStage = "style_ready"
	project.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	if err := saveNovelProjectPreservingGeneratedOutline(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, project)
}

func GenerateNovelChapterHandler(c *gin.Context) {
	project, err := getNovelProject(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	var req novelGenerateChapterRequest
	_ = c.ShouldBindJSON(&req)
	outline := findNovelOutline(project, req.OutlineID)
	if outline.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "outline chapter is required"})
		return
	}
	taskCtx, task := beginNovelRuntimeTask(project, "chapter_generate", fmt.Sprintf("正文生成：%s", outline.Title))
	defer task.finish()
	targetWords := novelChapterTargetWords(project)
	previousContext := previousNovelChapterContext(project, outline.ID, 3)
	system := "你是小说写手。请原创生成正文，不要复刻参考文本的句子、人物、世界观或情节。输出 JSON。"
	user := fmt.Sprintf(`请根据上下文生成一个章节草稿。
%s

要求：
1. 正文 content 以中文小说正文形式输出。
2. summary 概括本章发生的关键事实。
3. 风格只参考 style_profile 的抽象规则，不复制参考文本。
4. 本章目标字数约 %d 字；如果上下文不足，也要优先保证完整场景和章节钩子。
5. 必须覆盖章节规格中的 must_happen，并让结尾承接 hook/new_hooks。
6. 输出 JSON schema: {"title":"","content":"","summary":"","after_state":{"characters":[{"name":"","state":"","location":""}],"plot_hooks":[""],"plot_advances":[""]},"new_hooks":[""]}

小说：%s
章节规格：%s
前三章上下文：%s
事实库：%s
文风画像：%s
长期记忆：%s`, novelWritingSkillGuide, targetWords, project.Title, mustJSON(outline), previousContext, mustJSON(project.Extracted), mustJSON(project.StyleProfile), mustJSON(project.Memory))

	var generated struct {
		Title      string            `json:"title"`
		Content    string            `json:"content"`
		Summary    string            `json:"summary"`
		AfterState NovelChapterState `json:"after_state"`
		NewHooks   []string          `json:"new_hooks"`
	}
	general := loadNovelWriterGeneralSettings()
	if err := callNovelAIJSONWithTimeoutContext(taskCtx, system, user, &generated, novelChapterMaxTokens(targetWords), time.Duration(general.ChapterAIBackendTimeoutSeconds)*time.Second); err != nil {
		if isContextCanceledError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "正文生成任务已终止"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	chapter := NovelChapter{
		ID:        "CH-" + uuid.NewString(),
		OutlineID: outline.ID,
		Title:     firstNonEmpty(generated.Title, outline.Title),
		Status:    "draft",
		Content:   generated.Content,
		Summary:   generated.Summary,
		CreatedAt: now,
		UpdatedAt: now,
	}
	chapter.Versions = append(chapter.Versions, NovelChapterVersion{
		ID:        "VER-" + uuid.NewString(),
		Type:      "draft",
		Content:   chapter.Content,
		Reason:    "AI 初稿",
		CreatedAt: now,
	})
	chapter.ActiveVersionID = chapter.Versions[len(chapter.Versions)-1].ID
	project.Chapters = append(project.Chapters, chapter)
	clearNovelFullReview(&project)
	project.CurrentStage = "chapter_drafting"
	project.UpdatedAt = now
	if err := saveNovelProjectPreservingGeneratedOutline(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, project)
}

func AuditNovelChapterHandler(c *gin.Context) {
	project, chapter, index, ok := getNovelProjectChapter(c)
	if !ok {
		return
	}
	taskCtx, task := beginNovelRuntimeTask(project, "chapter_audit", fmt.Sprintf("章节审计：%s", chapter.Title))
	defer task.finish()
	general := loadNovelWriterGeneralSettings()
	system := "你是小说审计员，从 AI 味、人物一致性、剧情漏洞、文风贴合度审计章节。请只输出 JSON。"
	user := fmt.Sprintf(`审计以下章节。
%s

评分要求：
1. total_score、ai_flavor_score、character_score、logic_score、style_score 全部使用 0-100 的整数。
2. 分数越高表示质量越好，AI 味分数越高表示越自然、越不像 AI 生成。
3. total_score 是综合分，必须与四项维度保持同一量纲，不要使用 10 分制。

输出 JSON schema:
{"total_score":0,"ai_flavor_score":0,"character_score":0,"logic_score":0,"style_score":0,"issues":[{"severity":"high|medium|low","title":"","detail":"","suggestion":""}],"revision_advice":""}

事实库：%s
文风画像：%s
章节标题：%s
章节正文：%s`, novelAuditSkillGuide, mustJSON(project.Extracted), mustJSON(project.StyleProfile), chapter.Title, truncateForAI(chapter.Content, general.AuditContentMaxRunes))

	var report NovelAuditReport
	if err := callNovelAIJSONContext(taskCtx, system, user, &report); err != nil {
		if isContextCanceledError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "章节审计任务已终止"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	normalizeNovelAuditReport(&report)
	chapter.Audit = report
	chapter.Status = "reviewing"
	chapter.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	project.Chapters[index] = chapter
	project.CurrentStage = "chapter_auditing"
	project.UpdatedAt = chapter.UpdatedAt
	if err := saveNovelProjectPreservingGeneratedOutline(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, project)
}

func ReviseNovelChapterHandler(c *gin.Context) {
	project, chapter, index, ok := getNovelProjectChapter(c)
	if !ok {
		return
	}
	taskCtx, task := beginNovelRuntimeTask(project, "chapter_revise", fmt.Sprintf("审计修订：%s", chapter.Title))
	defer task.finish()
	general := loadNovelWriterGeneralSettings()
	system := "你是小说修订者。根据审计意见修复问题，保留原章节优点。请只输出 JSON。"
	user := fmt.Sprintf(`根据审计意见修订章节。
输出 JSON schema: {"content":"","summary":""}

审计意见：%s
原正文：%s`, mustJSON(chapter.Audit), truncateForAI(chapter.Content, general.RevisionContentMaxRunes))

	var revised struct {
		Content string `json:"content"`
		Summary string `json:"summary"`
	}
	if err := callNovelAIJSONContext(taskCtx, system, user, &revised); err != nil {
		if isContextCanceledError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "审计修订任务已终止"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	chapter.Status = "revision_needed"
	chapter.UpdatedAt = now
	chapter.Versions = append(chapter.Versions, NovelChapterVersion{
		ID:        "VER-" + uuid.NewString(),
		Type:      "revision",
		Content:   revised.Content,
		Reason:    "根据审计意见智能修订",
		CreatedAt: now,
	})
	project.Chapters[index] = chapter
	project.CurrentStage = "chapter_revising"
	project.UpdatedAt = now
	if err := saveNovelProjectPreservingGeneratedOutline(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, project)
}

func AdoptNovelChapterVersionHandler(c *gin.Context) {
	project, chapter, index, ok := getNovelProjectChapter(c)
	if !ok {
		return
	}

	versionID := strings.TrimSpace(c.Param("versionId"))
	if versionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version id is required"})
		return
	}

	for _, version := range chapter.Versions {
		if version.ID != versionID {
			continue
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		chapter.Content = version.Content
		chapter.ActiveVersionID = version.ID
		chapter.UpdatedAt = now
		project.Chapters[index] = chapter
		clearNovelFullReview(&project)
		project.UpdatedAt = now
		if err := saveNovelProjectPreservingGeneratedOutline(project); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, project)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "chapter version not found"})
}

func ApproveNovelChapterHandler(c *gin.Context) {
	project, chapter, index, ok := getNovelProjectChapter(c)
	if !ok {
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	chapter.Status = "approved"
	chapter.UpdatedAt = now
	project.Chapters[index] = chapter
	project.Memory.ChapterSummaries = upsertNovelInfoCard(project.Memory.ChapterSummaries, chapter.Title, chapter.Summary)
	project.CurrentStage = "chapter_approved"
	project.UpdatedAt = now
	if err := saveNovelProjectPreservingGeneratedOutline(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, project)
}

func ReviewNovelFullQualityHandler(c *gin.Context) {
	project, err := getNovelProject(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	chapters, missingTitles := novelContentChaptersInOutlineOrder(project)
	if len(chapters) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先生成正文后再进行全文质量核验"})
		return
	}
	if len(missingTitles) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("以下章节尚未生成正文，暂无法进行全文质量核验：%s", strings.Join(missingTitles, "、"))})
		return
	}

	taskCtx, task := beginNovelRuntimeTask(project, "full_review", "全文质量核验")
	defer task.finish()
	general := loadNovelWriterGeneralSettings()
	chapterPayload := truncateForAI(mustJSON(buildNovelFullReviewPayload(chapters)), general.FullReviewPayloadMaxRunes)

	system := "你是中文长篇小说总审校，负责从全本层面检查逻辑合理性、剧情连贯性、角色设定一致性、事件触发合理性。请只输出 JSON。"
	user := fmt.Sprintf(`请对这部小说做全文质量核验，重点检查：
1. 各章节之间的因果链是否完整，是否存在跳跃、断裂或前后矛盾。
2. 人物动机、性格、能力、关系是否前后一致。
3. 关键事件的触发是否自然，是否存在突兀推进、强行转折或设定失效。
4. 全本是否存在伏笔遗失、世界观规则冲突、时间线问题、阶段推进失衡。

评分要求：
1. total_score、coherence_score、logic_reasonability_score、character_consistency_score、trigger_reasonability_score 全部使用 0-100 的整数。
2. 分数越高表示质量越好。
3. issues 按严重程度输出 high|medium|low。
4. chapter_id / chapter_title 如果能定位到具体章节就填写，若是全局问题可留空。

输出 JSON schema:
{"total_score":0,"coherence_score":0,"logic_reasonability_score":0,"character_consistency_score":0,"trigger_reasonability_score":0,"summary":"","issues":[{"severity":"high|medium|low","dimension":"","chapter_id":"","chapter_title":"","title":"","detail":"","suggestion":""}],"revision_advice":""}

小说标题：%s
事实库：%s
文风画像：%s
全书大纲：%s
章节正文：%s`, project.Title, mustJSON(project.Extracted), mustJSON(project.StyleProfile), mustJSON(project.Outline.Chapters), chapterPayload)

	var report NovelFullReview
	if err := callNovelAIJSONWithTimeoutContext(taskCtx, system, user, &report, 0, time.Duration(general.FullReviewAIBackendTimeoutSeconds)*time.Second); err != nil {
		if isContextCanceledError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "全文质量核验任务已终止"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	normalizeNovelFullReviewReport(&report)
	report.ReviewedAt = now
	report.AppliedAt = ""
	project.FullReview = report
	project.CurrentStage = "full_reviewed"
	project.UpdatedAt = now
	if err := saveNovelProjectPreservingGeneratedOutline(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, project)
}

func ReviseNovelByFullReviewHandler(c *gin.Context) {
	project, err := getNovelProject(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if project.FullReview.ReviewedAt == "" && len(project.FullReview.Issues) == 0 && strings.TrimSpace(project.FullReview.RevisionAdvice) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先完成全文质量核验"})
		return
	}

	chapters, missingTitles := novelContentChaptersInOutlineOrder(project)
	if len(chapters) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先生成正文后再进行全文修订"})
		return
	}
	if len(missingTitles) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("以下章节尚未生成正文，暂无法进行全文修订：%s", strings.Join(missingTitles, "、"))})
		return
	}

	taskCtx, task := beginNovelRuntimeTask(project, "full_review_revise", "按核验结果修订全文")
	defer task.finish()
	general := loadNovelWriterGeneralSettings()
	chapterPayload := truncateForAI(mustJSON(buildNovelFullReviewPayload(chapters)), general.FullReviewPayloadMaxRunes)

	system := "你是中文长篇小说总编修订者。请严格按照全文核验意见修订小说全文，只输出 JSON。"
	user := fmt.Sprintf(`请根据全文质量核验结果，对这部小说做全书级修订。
要求：
1. 仅修改确有必要的章节，尽量保留已有优点和原有文风。
2. 输出 chapters 数组时，只返回实际需要改动的章节。
3. chapter_id 必须与输入保持一致。
4. content 输出修订后的完整章节正文，summary 输出修订后的章节摘要。

输出 JSON schema:
{"chapters":[{"chapter_id":"","title":"","content":"","summary":"","reason":""}]}

全文核验结果：%s
全书大纲：%s
章节正文：%s`, mustJSON(project.FullReview), mustJSON(project.Outline.Chapters), chapterPayload)

	var revised struct {
		Chapters []struct {
			ChapterID string `json:"chapter_id"`
			Title     string `json:"title"`
			Content   string `json:"content"`
			Summary   string `json:"summary"`
			Reason    string `json:"reason"`
		} `json:"chapters"`
	}
	if err := callNovelAIJSONWithTimeoutContext(taskCtx, system, user, &revised, 0, time.Duration(general.FullReviewAIBackendTimeoutSeconds)*time.Second); err != nil {
		if isContextCanceledError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "全文修订任务已终止"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	revisedByID := map[string]struct {
		Title   string
		Content string
		Summary string
		Reason  string
	}{}
	for _, item := range revised.Chapters {
		chapterID := strings.TrimSpace(item.ChapterID)
		content := strings.TrimSpace(item.Content)
		if chapterID == "" || content == "" {
			continue
		}
		revisedByID[chapterID] = struct {
			Title   string
			Content string
			Summary string
			Reason  string
		}{
			Title:   strings.TrimSpace(item.Title),
			Content: content,
			Summary: strings.TrimSpace(item.Summary),
			Reason:  strings.TrimSpace(item.Reason),
		}
	}

	for index, chapter := range project.Chapters {
		revisedChapter, ok := revisedByID[chapter.ID]
		if !ok {
			continue
		}
		version := NovelChapterVersion{
			ID:        "VER-" + uuid.NewString(),
			Type:      "full_review_revision",
			Content:   revisedChapter.Content,
			Reason:    firstNonEmpty(revisedChapter.Reason, "根据全文质量核验结果修订"),
			CreatedAt: now,
		}
		chapter.Title = firstNonEmpty(revisedChapter.Title, chapter.Title)
		chapter.Content = revisedChapter.Content
		chapter.Summary = firstNonEmpty(revisedChapter.Summary, chapter.Summary)
		chapter.Status = "draft"
		chapter.Audit = NovelAuditReport{}
		chapter.ActiveVersionID = version.ID
		chapter.UpdatedAt = now
		chapter.Versions = append(chapter.Versions, version)
		project.Chapters[index] = chapter
	}

	project.FullReview.AppliedAt = now
	project.CurrentStage = "full_review_revised"
	project.UpdatedAt = now
	if err := saveNovelProjectPreservingGeneratedOutline(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, project)
}

func listNovelProjects() ([]NovelProject, error) {
	if err := EnsureNovelWriterTables(); err != nil {
		return nil, err
	}
	projects, err := sqlListJSON[NovelProject]("novel_projects", "")
	if err != nil {
		return nil, err
	}
	sort.SliceStable(projects, func(i, j int) bool {
		return projects[i].UpdatedAt > projects[j].UpdatedAt
	})
	return projects, nil
}

func getNovelProject(id string) (NovelProject, error) {
	if err := EnsureNovelWriterTables(); err != nil {
		return NovelProject{}, err
	}
	return sqlGetJSONByID[NovelProject]("novel_projects", id)
}

func saveNovelProject(project NovelProject) error {
	if err := EnsureNovelWriterTables(); err != nil {
		return err
	}
	return sqlUpsertJSON("novel_projects", project)
}

func saveNovelProjectPreservingGeneratedOutline(project NovelProject) error {
	latest, err := getNovelProject(project.ID)
	if err == nil && shouldKeepPersistedOutline(latest.Outline, project.Outline) {
		project.Outline = latest.Outline
	}
	return saveNovelProject(project)
}

func saveNovelProjectOutline(projectID string, outline NovelOutline) error {
	latest, err := getNovelProject(projectID)
	if err != nil {
		return err
	}
	if len(latest.Outline.Chapters) > len(outline.Chapters) {
		outline.Chapters = mergeNovelOutlineChapters(latest.Outline.Chapters, outline.Chapters)
		outline.GeneratedChapters = len(outline.Chapters)
	}
	latest.Outline = outline
	latest.CurrentStage = "outline_ready"
	latest.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	return saveNovelProject(latest)
}

func deleteNovelProject(id string) error {
	if err := EnsureNovelWriterTables(); err != nil {
		return err
	}
	if _, err := getNovelProject(id); err != nil {
		return err
	}
	return sqlDeleteJSON("novel_projects", id)
}

func getNovelProjectChapter(c *gin.Context) (NovelProject, NovelChapter, int, bool) {
	project, err := getNovelProject(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return NovelProject{}, NovelChapter{}, -1, false
	}
	chapterID := c.Param("chapterId")
	for index, chapter := range project.Chapters {
		if chapter.ID == chapterID {
			return project, chapter, index, true
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "chapter not found"})
	return NovelProject{}, NovelChapter{}, -1, false
}

func findNovelOutline(project NovelProject, outlineID string) NovelChapterOutline {
	if outlineID != "" {
		for _, item := range project.Outline.Chapters {
			if item.ID == outlineID {
				return item
			}
		}
	}
	for _, item := range project.Outline.Chapters {
		if !hasChapterForOutline(project.Chapters, item.ID) {
			return item
		}
	}
	if len(project.Outline.Chapters) > 0 {
		return project.Outline.Chapters[0]
	}
	return NovelChapterOutline{}
}

func hasChapterForOutline(chapters []NovelChapter, outlineID string) bool {
	for _, chapter := range chapters {
		if chapter.OutlineID == outlineID {
			return true
		}
	}
	return false
}

func generateNovelOutlineBatchAdaptive(ctx context.Context, project NovelProject, startChapter int, totalChapters int, existing NovelOutline, preferredBatchSize int) (NovelOutline, int, error) {
	var lastErr error
	general := loadNovelWriterGeneralSettings()
	for _, batchSize := range novelOutlineBatchCandidates(preferredBatchSize, totalChapters-startChapter+1, general.OutlineBatchSize) {
		if ctx.Err() != nil {
			return NovelOutline{}, 0, ctx.Err()
		}
		endChapter := startChapter + batchSize - 1
		if endChapter > totalChapters {
			endChapter = totalChapters
		}
		outline, err := generateNovelOutlineBatch(ctx, project, startChapter, endChapter, totalChapters, existing)
		if err == nil {
			return outline, batchSize, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no outline batch size is available")
	}
	return NovelOutline{}, 0, fmt.Errorf("outline batch starting at chapter %d failed after adaptive retries: %w", startChapter, lastErr)
}

func novelOutlineBatchCandidates(preferredBatchSize int, remainingChapters int, fallbackBatchSize int) []int {
	raw := []int{preferredBatchSize, fallbackBatchSize, 5, 3, 1}
	candidates := make([]int, 0, len(raw))
	seen := map[int]bool{}
	for _, size := range raw {
		if size <= 0 {
			continue
		}
		if size > remainingChapters {
			size = remainingChapters
		}
		if size <= 0 || seen[size] {
			continue
		}
		seen[size] = true
		candidates = append(candidates, size)
	}
	return candidates
}

func generateNovelOutlineBatch(ctx context.Context, project NovelProject, startChapter int, endChapter int, totalChapters int, existing NovelOutline) (NovelOutline, error) {
	system := "你是商业小说结构规划师。请只输出 JSON，不要输出 Markdown。"
	existingForPrompt := compactNovelOutlineForPrompt(existing)
	user := fmt.Sprintf(`基于事实库生成小说大纲和第 %d-%d 章的完整章节规格。
%s

要求：
1. 本次只输出第 %d-%d 章，必须严格生成 %d 个 chapters。
2. 每章必须有 title、goal、conflict、hook、summary、before_state、after_state、must_happen、tension_curve、key_scenes、new_hooks。
3. before_state/after_state 要包含 characters、plot_hooks、plot_advances；tension_curve 至少包含 position 0、50、100 三个点。
4. acts 是三幕式或卷结构。已有 logline/acts 时保持一致，不要重写主方向。
5. 不要虚构事实库之外的硬设定；可以在不冲突的前提下补充剧情规划。
6. 输出 JSON schema:
{"logline":"","acts":[{"name":"","description":""}],"chapters":[{"title":"","goal":"","conflict":"","hook":"","summary":"","before_state":{"characters":[{"name":"","state":"","location":""}],"plot_hooks":[""],"plot_advances":[""]},"after_state":{"characters":[{"name":"","state":"","location":""}],"plot_hooks":[""],"plot_advances":[""]},"must_happen":[""],"tension_curve":[{"position":0,"value":3,"note":""},{"position":50,"value":8,"note":""},{"position":100,"value":5,"note":""}],"key_scenes":[""],"new_hooks":[""]}]}

小说：%s
题材：%s
目标总章节数：%d
当前素材：%s
事实库：%s
文风画像：%s
已有大纲摘要：%s`, startChapter, endChapter, novelOutlineSkillGuide, startChapter, endChapter, endChapter-startChapter+1, project.Title, project.Genre, totalChapters, mustJSON(compactNovelMaterialsForOutline(project.Materials)), mustJSON(compactNovelExtractedForPrompt(project.Extracted)), mustJSON(project.StyleProfile), mustJSON(existingForPrompt))

	var batch NovelOutline
	if err := callNovelAIJSONWithMaxTokensContext(ctx, system, user, &batch, novelOutlineMaxTokens(endChapter-startChapter+1)); err != nil {
		return NovelOutline{}, err
	}
	if len(batch.Chapters) == 0 {
		return NovelOutline{}, fmt.Errorf("outline batch %d-%d returned no chapters", startChapter, endChapter)
	}
	if existing.Logline == "" {
		existing.Logline = batch.Logline
	}
	if len(existing.Acts) == 0 {
		existing.Acts = batch.Acts
	}
	for index := range batch.Chapters {
		globalIndex := startChapter + index
		batch.Chapters[index].ID = fmt.Sprintf("outline-%02d", globalIndex)
	}
	existing.Chapters = mergeNovelOutlineChapters(existing.Chapters, batch.Chapters)
	return existing, nil
}

func compactNovelMaterialsForOutline(materials NovelMaterials) NovelMaterials {
	general := loadNovelWriterGeneralSettings()
	return NovelMaterials{
		RawText:      truncateForAI(materials.RawText, general.MaterialRawMaxRunes),
		CharacterRaw: truncateForAI(materials.CharacterRaw, general.MaterialCharacterMaxRunes),
		WorldRaw:     truncateForAI(materials.WorldRaw, general.MaterialWorldMaxRunes),
		ConflictRaw:  truncateForAI(materials.ConflictRaw, general.MaterialConflictMaxRunes),
		ReferenceRaw: "",
	}
}

func compactNovelExtractedForPrompt(extracted NovelExtractedInfo) NovelExtractedInfo {
	general := loadNovelWriterGeneralSettings()
	return NovelExtractedInfo{
		Characters:    compactNovelInfoCardsForPrompt(extracted.Characters, general.PromptCardLimit),
		WorldRules:    compactNovelInfoCardsForPrompt(extracted.WorldRules, general.PromptCardLimit),
		Conflicts:     compactNovelInfoCardsForPrompt(extracted.Conflicts, general.PromptCardLimit),
		KeyEvents:     compactNovelInfoCardsForPrompt(extracted.KeyEvents, general.PromptCardLimit),
		OpenQuestions: compactNovelStringsForPrompt(extracted.OpenQuestions, general.PromptCardLimit, general.PromptQuestionMaxRunes),
	}
}

func compactNovelInfoCardsForPrompt(items []NovelInfoCard, limit int) []NovelInfoCard {
	if limit <= 0 || len(items) <= limit {
		limit = len(items)
	}
	result := make([]NovelInfoCard, 0, limit)
	general := loadNovelWriterGeneralSettings()
	for _, item := range items[:limit] {
		result = append(result, NovelInfoCard{
			Name:        truncateForAI(item.Name, general.PromptCardNameMaxRunes),
			Description: truncateForAI(item.Description, general.PromptCardDescriptionMaxRunes),
		})
	}
	return result
}

func compactNovelStringsForPrompt(items []string, limit int, maxRunes int) []string {
	if limit <= 0 || len(items) <= limit {
		limit = len(items)
	}
	result := make([]string, 0, limit)
	for _, item := range items[:limit] {
		result = append(result, truncateForAI(item, maxRunes))
	}
	return result
}

func compactNovelOutlineForPrompt(outline NovelOutline) NovelOutline {
	compact := NovelOutline{
		Logline: outline.Logline,
		Acts:    outline.Acts,
	}
	compact.Chapters = make([]NovelChapterOutline, 0, len(outline.Chapters))
	for _, chapter := range outline.Chapters {
		compact.Chapters = append(compact.Chapters, NovelChapterOutline{
			ID:       chapter.ID,
			Title:    chapter.Title,
			Goal:     chapter.Goal,
			Conflict: chapter.Conflict,
			Hook:     chapter.Hook,
			Summary:  chapter.Summary,
			NewHooks: chapter.NewHooks,
		})
	}
	return compact
}

func mergeNovelOutlineChapters(current []NovelChapterOutline, incoming []NovelChapterOutline) []NovelChapterOutline {
	byID := make(map[string]NovelChapterOutline, len(current)+len(incoming))
	order := make([]string, 0, len(current)+len(incoming))
	for _, chapter := range current {
		if chapter.ID == "" {
			continue
		}
		if _, exists := byID[chapter.ID]; !exists {
			order = append(order, chapter.ID)
		}
		byID[chapter.ID] = chapter
	}
	for _, chapter := range incoming {
		if chapter.ID == "" {
			continue
		}
		if _, exists := byID[chapter.ID]; !exists {
			order = append(order, chapter.ID)
		}
		byID[chapter.ID] = chapter
	}
	sort.SliceStable(order, func(i, j int) bool {
		return order[i] < order[j]
	})
	result := make([]NovelChapterOutline, 0, len(order))
	for _, id := range order {
		result = append(result, byID[id])
	}
	return result
}

func continueNovelOutlineGeneration(ctx context.Context, projectID string) {
	for {
		if ctx.Err() != nil {
			project, err := getNovelProject(projectID)
			if err == nil {
				project.Outline.GenerationStatus = "cancelled"
				project.Outline.GenerationError = "大纲生成任务已终止"
				project.Outline.GeneratedChapters = len(project.Outline.Chapters)
				_ = saveNovelProjectOutline(project.ID, project.Outline)
			}
			return
		}
		project, err := getNovelProject(projectID)
		if err != nil {
			return
		}
		target := project.Outline.TargetChapters
		if target <= 0 {
			target = project.TargetChapters
		}
		if target <= 0 || len(project.Outline.Chapters) >= target {
			project.Outline.GenerationStatus = "ready"
			project.Outline.GeneratedChapters = len(project.Outline.Chapters)
			_ = saveNovelProjectOutline(project.ID, project.Outline)
			return
		}
		batchSize := project.Outline.BatchSize
		if batchSize <= 0 {
			batchSize = loadNovelWriterGeneralSettings().OutlineBatchSize
		}
		start := len(project.Outline.Chapters) + 1
		end := start + batchSize - 1
		if end > target {
			end = target
		}
		previousCount := len(project.Outline.Chapters)
		outline, usedBatchSize, err := generateNovelOutlineBatchAdaptive(ctx, project, start, target, project.Outline, end-start+1)
		if err != nil {
			if isContextCanceledError(err) {
				project.Outline.GenerationStatus = "cancelled"
				project.Outline.GenerationError = "大纲生成任务已终止"
				project.Outline.GeneratedChapters = len(project.Outline.Chapters)
				_ = saveNovelProjectOutline(project.ID, project.Outline)
				return
			}
			project.Outline.GenerationStatus = "failed"
			project.Outline.GenerationError = err.Error()
			project.Outline.GeneratedChapters = len(project.Outline.Chapters)
			_ = saveNovelProjectOutline(project.ID, project.Outline)
			return
		}
		if len(outline.Chapters) <= previousCount {
			project.Outline.GenerationStatus = "failed"
			project.Outline.GenerationError = fmt.Sprintf("outline batch %d-%d did not add new chapters", start, end)
			project.Outline.GeneratedChapters = len(project.Outline.Chapters)
			_ = saveNovelProjectOutline(project.ID, project.Outline)
			return
		}
		outline.TargetChapters = target
		outline.BatchSize = usedBatchSize
		outline.GeneratedChapters = len(outline.Chapters)
		outline.GenerationStatus = "generating"
		outline.GenerationError = ""
		if len(outline.Chapters) >= target {
			outline.GenerationStatus = "ready"
		}
		if err := saveNovelProjectOutline(project.ID, outline); err != nil {
			return
		}
		if outline.GenerationStatus == "ready" {
			return
		}
	}
}

func novelChapterTargetWords(project NovelProject) int {
	maxChapterWords := loadNovelWriterGeneralSettings().MaxChapterWords
	if project.TargetWords > 0 && project.TargetChapters > 0 {
		average := project.TargetWords / project.TargetChapters
		if average < 1200 {
			return 1200
		}
		if average > maxChapterWords {
			return maxChapterWords
		}
		return average
	}
	return 3000
}

func novelChapterMaxTokens(targetWords int) int {
	maxTokensLimit := loadNovelWriterGeneralSettings().ChapterMaxTokens
	if targetWords <= 0 {
		return 9000
	}
	maxTokens := targetWords * 2
	if maxTokens < 9000 {
		return 9000
	}
	if maxTokens > maxTokensLimit {
		return maxTokensLimit
	}
	return maxTokens
}

func previousNovelChapterContext(project NovelProject, outlineID string, limit int) string {
	if limit <= 0 {
		limit = 3
	}
	outlineOrder := map[string]int{}
	for index, outline := range project.Outline.Chapters {
		outlineOrder[outline.ID] = index
	}
	currentOrder, hasCurrentOrder := outlineOrder[outlineID]

	type chapterContext struct {
		Order   int    `json:"order"`
		Title   string `json:"title"`
		Status  string `json:"status"`
		Summary string `json:"summary"`
	}

	items := make([]chapterContext, 0, len(project.Chapters))
	for index, chapter := range project.Chapters {
		order, ok := outlineOrder[chapter.OutlineID]
		if !ok {
			order = index
		}
		if hasCurrentOrder && order >= currentOrder {
			continue
		}
		items = append(items, chapterContext{
			Order:   order,
			Title:   chapter.Title,
			Status:  chapter.Status,
			Summary: firstNonEmpty(strings.TrimSpace(chapter.Summary), truncateForAI(chapter.Content, 600)),
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Order > items[j].Order
	})
	if len(items) > limit {
		items = items[:limit]
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Order < items[j].Order
	})
	return mustJSON(items)
}

type novelFullReviewChapterPayload struct {
	ChapterID string `json:"chapter_id"`
	OutlineID string `json:"outline_id"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	Content   string `json:"content"`
}

func novelContentChaptersInOutlineOrder(project NovelProject) ([]NovelChapter, []string) {
	byOutlineID := make(map[string]NovelChapter, len(project.Chapters))
	for _, chapter := range project.Chapters {
		byOutlineID[chapter.OutlineID] = chapter
	}

	ordered := make([]NovelChapter, 0, len(project.Chapters))
	missingTitles := make([]string, 0)
	usedChapterIDs := map[string]bool{}

	for _, outline := range project.Outline.Chapters {
		chapter, ok := byOutlineID[outline.ID]
		if !ok || strings.TrimSpace(chapter.Content) == "" {
			missingTitles = append(missingTitles, firstNonEmpty(strings.TrimSpace(outline.Title), "未命名章节"))
			continue
		}
		ordered = append(ordered, chapter)
		usedChapterIDs[chapter.ID] = true
	}

	for _, chapter := range project.Chapters {
		if usedChapterIDs[chapter.ID] || strings.TrimSpace(chapter.Content) == "" {
			continue
		}
		ordered = append(ordered, chapter)
	}

	return ordered, missingTitles
}

func buildNovelFullReviewPayload(chapters []NovelChapter) []novelFullReviewChapterPayload {
	result := make([]novelFullReviewChapterPayload, 0, len(chapters))
	for _, chapter := range chapters {
		result = append(result, novelFullReviewChapterPayload{
			ChapterID: chapter.ID,
			OutlineID: chapter.OutlineID,
			Title:     chapter.Title,
			Summary:   chapter.Summary,
			Content:   chapter.Content,
		})
	}
	return result
}

func callNovelAIJSON(systemPrompt string, userPrompt string, target any) error {
	return callNovelAIJSONContext(context.Background(), systemPrompt, userPrompt, target)
}

func callNovelAIJSONContext(ctx context.Context, systemPrompt string, userPrompt string, target any) error {
	return callNovelAIJSONWithMaxTokensContext(ctx, systemPrompt, userPrompt, target, 0)
}

func callNovelAIJSONWithMaxTokens(systemPrompt string, userPrompt string, target any, maxTokens int) error {
	return callNovelAIJSONWithMaxTokensContext(context.Background(), systemPrompt, userPrompt, target, maxTokens)
}

func callNovelAIJSONWithTimeoutContext(ctx context.Context, systemPrompt string, userPrompt string, target any, maxTokens int, requestTimeout time.Duration) error {
	provider, err := selectNovelProvider()
	if err != nil {
		return err
	}
	metrics := novelAIRequestMetrics(provider, systemPrompt, userPrompt, maxTokens)
	if requestTimeout <= 0 {
		requestTimeout = time.Duration(loadNovelWriterGeneralSettings().DefaultAIBackendTimeoutSeconds) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	clientConfig := openai.DefaultConfig(provider.APIKey)
	clientConfig.BaseURL = provider.BaseURL
	client := openai.NewClientWithConfig(clientConfig)
	req := openai.ChatCompletionRequest{
		Model: provider.Model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.7,
	}
	if maxTokens > 0 {
		req.MaxTokens = maxTokens
	}
	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return fmt.Errorf("%w; %s", err, metrics)
	}
	if len(resp.Choices) == 0 {
		return fmt.Errorf("novel AI provider returned no choices; %s", metrics)
	}
	if err := decodeNovelJSON(resp.Choices[0].Message.Content, target); err != nil {
		return fmt.Errorf("%w; %s; output_runes=%d", err, metrics, len([]rune(resp.Choices[0].Message.Content)))
	}
	return nil
}

func callNovelAIJSONWithMaxTokensContext(ctx context.Context, systemPrompt string, userPrompt string, target any, maxTokens int) error {
	return callNovelAIJSONWithTimeoutContext(ctx, systemPrompt, userPrompt, target, maxTokens, time.Duration(loadNovelWriterGeneralSettings().DefaultAIBackendTimeoutSeconds)*time.Second)
}

func isContextCanceledError(err error) bool {
	return err == context.Canceled || (err != nil && strings.Contains(strings.ToLower(err.Error()), "context canceled"))
}

func novelOutlineMaxTokens(chapterCount int) int {
	general := loadNovelWriterGeneralSettings()
	if chapterCount <= 1 {
		return general.OutlineSmallBatchMaxTokens
	}
	if chapterCount <= 3 {
		return general.OutlineMediumBatchMaxTokens
	}
	return general.OutlineLargeBatchMaxTokens
}

func novelAIRequestMetrics(provider feishumodel.AIProviderConfig, systemPrompt string, userPrompt string, maxTokens int) string {
	contextTokens := novelModelContextTokens(provider.Model)
	if contextTokens <= 0 {
		return fmt.Sprintf(
			"provider=%s model=%s capability=%s prompt_runes=%d system_runes=%d user_runes=%d max_tokens=%d context_tokens=unknown",
			provider.Name,
			provider.Model,
			provider.Capability,
			len([]rune(systemPrompt))+len([]rune(userPrompt)),
			len([]rune(systemPrompt)),
			len([]rune(userPrompt)),
			maxTokens,
		)
	}
	return fmt.Sprintf(
		"provider=%s model=%s capability=%s prompt_runes=%d system_runes=%d user_runes=%d max_tokens=%d context_tokens=%d",
		provider.Name,
		provider.Model,
		provider.Capability,
		len([]rune(systemPrompt))+len([]rune(userPrompt)),
		len([]rune(systemPrompt)),
		len([]rune(userPrompt)),
		maxTokens,
		contextTokens,
	)
}

func novelModelContextTokens(model string) int {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "mimo-v2-pro", "xiaomi/mimo-v2-pro":
		return 1048576
	default:
		return 0
	}
}

func selectNovelProvider() (feishumodel.AIProviderConfig, error) {
	config := feishumodel.LoadAIConfig()
	for _, capability := range []string{"novel_writing", "casual_chat_pro", "casual_chat", "chat"} {
		providers := config.EffectiveProvidersByCapability(capability)
		if len(providers) > 0 {
			return providers[0], nil
		}
	}
	return feishumodel.AIProviderConfig{}, fmt.Errorf("novel writing AI provider is not configured")
}

func decodeNovelJSON(content string, target any) error {
	cleaned := strings.TrimSpace(content)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)
	if err := json.Unmarshal([]byte(cleaned), target); err == nil {
		return nil
	} else if cleaned == "" {
		return fmt.Errorf("AI output is empty")
	}
	re := regexp.MustCompile(`(?s)\{.*\}`)
	match := re.FindString(cleaned)
	if match == "" {
		return fmt.Errorf("AI output is not valid JSON: no JSON object found, output length=%d", len([]rune(cleaned)))
	}
	if err := json.Unmarshal([]byte(match), target); err != nil {
		return fmt.Errorf("AI output is not valid JSON: %w, output length=%d", err, len([]rune(cleaned)))
	}
	return nil
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func upsertNovelInfoCard(items []NovelInfoCard, name string, description string) []NovelInfoCard {
	for index, item := range items {
		if item.Name == name {
			items[index].Description = description
			return items
		}
	}
	return append(items, NovelInfoCard{Name: name, Description: description})
}
