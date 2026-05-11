package compliance_api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"backend/document/parser"
	"backend/global"
	"backend/models"
	"backend/models/res"

	"github.com/cloudwego/hertz/pkg/app"
)

const (
	tenderFileRole   = "tender"
	responseFileRole = "response"
	tenderMaxSize    = 20 * 1024 * 1024 // 20MB
	responseMaxSize  = 50 * 1024 * 1024 // 50MB
)

// UploadComplianceFile 处理合规文件上传与解析
func (api *API) UploadComplianceFile(c context.Context, ctx *app.RequestContext) {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		res.FailWithMessage("获取上传文件失败: "+err.Error(), c, ctx)
		return
	}

	role := strings.ToLower(strings.TrimSpace(string(ctx.FormValue("role"))))
	if role == "" {
		res.FailWithMessage("缺少文件角色参数 role", c, ctx)
		return
	}
	if role != tenderFileRole && role != responseFileRole {
		res.FailWithMessage("文件角色仅支持 tender 或 response", c, ctx)
		return
	}

	if fileHeader.Size <= 0 {
		res.FailWithMessage("文件内容为空", c, ctx)
		return
	}

	maxSize := responseMaxSize
	if role == tenderFileRole {
		maxSize = tenderMaxSize
	}
	if fileHeader.Size > int64(maxSize) {
		res.FailWithMessage(fmt.Sprintf("文件大小超出限制，%s 文件最大 %dMB", role, maxSize/1024/1024), c, ctx)
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	switch ext {
	case ".pdf", ".docx", ".doc":
	default:
		res.FailWithMessage("仅支持 PDF、DOCX、DOC 格式", c, ctx)
		return
	}

	uploadDir := filepath.Join("uploads", "compliance", "sources")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		res.FailWithMessage("创建上传目录失败: "+err.Error(), c, ctx)
		return
	}

	timestamp := time.Now().UnixNano()
	sanitizedName := sanitizeFileName(fileHeader.Filename)
	fileName := fmt.Sprintf("%d_%s", timestamp, sanitizedName)
	savePath := filepath.Join(uploadDir, fileName)

	if err := ctx.SaveUploadedFile(fileHeader, savePath); err != nil {
		res.FailWithMessage("保存文件失败: "+err.Error(), c, ctx)
		return
	}

	parseFactory := parser.NewParserFactory()
	docParser, err := parseFactory.GetParser(ext)
	if err != nil {
		_ = os.Remove(savePath)
		res.FailWithMessage("获取解析器失败: "+err.Error(), c, ctx)
		return
	}

	start := time.Now()
	parseResult, err := docParser.ParseWithMetadata(savePath)

	parseLog := &models.DocumentParseLog{
		FilePath:  savePath,
		FileName:  fileHeader.Filename,
		FileType:  strings.TrimPrefix(ext, "."),
		FileSize:  fileHeader.Size,
		ParseTime: int(time.Since(start).Milliseconds()),
		Success:   err == nil,
	}

	if err != nil {
		parseLog.ErrorMsg = err.Error()
		if _, insertErr := global.DB.Insert(parseLog); insertErr != nil && global.Log != nil {
			global.Log.Error("保存解析日志失败", "error", insertErr)
		}
		_ = os.Remove(savePath)
		res.FailWithMessage("解析文件失败: "+err.Error(), c, ctx)
		return
	}

	if parseResult != nil {
		parseLog.ParseMethod = parseResult.ParseMethod
		parseLog.QualityScore = parseResult.QualityScore
		if parseResult.Metadata != nil {
			parseLog.TextLength = parseResult.Metadata.TextLength
			parseLog.TableCount = parseResult.Metadata.TableCount
		}
	}

	if _, insertErr := global.DB.Insert(parseLog); insertErr != nil {
		res.FailWithMessage("保存解析日志失败: "+insertErr.Error(), c, ctx)
		return
	}

	preview := ""
	if parseResult != nil {
		preview = buildContentPreview(parseResult.Content)
	}

	pageCount := 0
	textLength := 0
	if parseResult != nil && parseResult.Metadata != nil {
		pageCount = parseResult.Metadata.PageCount
		textLength = parseResult.Metadata.TextLength
	}

	response := UploadComplianceFileResponse{
		FileID:         strconv.FormatUint(parseLog.ID, 10),
		FileRole:       role,
		FileName:       fileHeader.Filename,
		FilePath:       savePath,
		FileSize:       fileHeader.Size,
		FileType:       strings.TrimPrefix(ext, "."),
		ParseMethod:    parseLog.ParseMethod,
		QualityScore:   parseLog.QualityScore,
		TextLength:     textLength,
		PageCount:      pageCount,
		UploadedAt:     time.Now().Format(time.RFC3339),
		ContentPreview: preview,
	}

	res.OkWithData(response, c, ctx)
}

func sanitizeFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	base = strings.ReplaceAll(base, " ", "_")
	base = strings.ReplaceAll(base, "..", "")
	if base == "" {
		return "file"
	}
	return base
}

func buildContentPreview(content string) string {
	if content == "" {
		return ""
	}
	const maxPreviewRunes = 500
	runeSlice := []rune(strings.TrimSpace(content))
	if len(runeSlice) <= maxPreviewRunes {
		return string(runeSlice)
	}
	return string(runeSlice[:maxPreviewRunes]) + "..."
}



