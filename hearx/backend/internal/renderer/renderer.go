package renderer

import (
	"context"
	"fmt"
	"strings"

	"github.com/renlulu/hearx/backend/internal/llm"
	"github.com/renlulu/hearx/backend/internal/model"
)

type Renderer struct {
	llm llm.Provider
}

func New(llmProvider llm.Provider) *Renderer {
	return &Renderer{llm: llmProvider}
}

func (r *Renderer) RenderMarkdown(segments []model.TranscriptSegment, translation string) string {
	var sb strings.Builder
	sb.WriteString("# HearX 翻译文稿\n\n")
	sb.WriteString("## 中文翻译\n\n")
	sb.WriteString(translation)
	sb.WriteString("\n\n---\n\n")
	sb.WriteString("## 原文转写\n\n")
	for _, seg := range segments {
		fmt.Fprintf(&sb, "[%s] %s\n\n", formatTimestamp(seg.Start), seg.Text)
	}
	return sb.String()
}

func (r *Renderer) RenderSRT(segments []model.TranscriptSegment) string {
	var sb strings.Builder
	for i, seg := range segments {
		fmt.Fprintf(&sb, "%d\n", i+1)
		fmt.Fprintf(&sb, "%s --> %s\n", formatSRTTime(seg.Start), formatSRTTime(seg.End))
		fmt.Fprintf(&sb, "%s\n\n", seg.Text)
	}
	return sb.String()
}

func (r *Renderer) GenerateSummary(ctx context.Context, translation string) (string, error) {
	prompt := "请用中文为以下翻译文稿生成一段简洁的摘要（3-5句话），抓住核心要点：\n\n" + translation
	return r.llm.Chat(ctx, "你是一个专业的中文内容摘要助手。", prompt)
}

func formatTimestamp(seconds float64) string {
	m := int(seconds) / 60
	s := int(seconds) % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func formatSRTTime(seconds float64) string {
	h := int(seconds) / 3600
	m := (int(seconds) % 3600) / 60
	s := int(seconds) % 60
	ms := int((seconds - float64(int(seconds))) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}
