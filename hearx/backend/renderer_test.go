package backend_test

import (
	"strings"
	"testing"

	"github.com/renlulu/hearx/backend/internal/model"
	"github.com/renlulu/hearx/backend/internal/renderer"
)

func TestRenderMarkdown(t *testing.T) {
	r := renderer.New(nil) // LLM not needed for markdown
	segments := []model.TranscriptSegment{
		{Start: 0, End: 5.5, Text: "Hello world"},
		{Start: 5.5, End: 10, Text: "This is a test"},
	}
	translation := "你好世界\n这是一个测试"

	md := r.RenderMarkdown(segments, translation)

	if !strings.Contains(md, "# HearX 翻译文稿") {
		t.Error("markdown should contain title")
	}
	if !strings.Contains(md, "你好世界") {
		t.Error("markdown should contain translation")
	}
	if !strings.Contains(md, "Hello world") {
		t.Error("markdown should contain original text")
	}
	if !strings.Contains(md, "[00:00]") {
		t.Error("markdown should contain timestamps")
	}
}

func TestRenderSRT(t *testing.T) {
	r := renderer.New(nil)
	segments := []model.TranscriptSegment{
		{Start: 0, End: 5.5, Text: "Hello world"},
		{Start: 5.5, End: 10.123, Text: "This is a test"},
	}

	srt := r.RenderSRT(segments)

	if !strings.Contains(srt, "1\n") {
		t.Error("SRT should contain sequence number")
	}
	if !strings.Contains(srt, "00:00:00,000 --> 00:00:05,500") {
		t.Error("SRT should contain properly formatted timestamps")
	}
	if !strings.Contains(srt, "Hello world") {
		t.Error("SRT should contain text")
	}
	if !strings.Contains(srt, "00:00:10,") {
		t.Error("SRT should handle second segment end time")
	}
}
