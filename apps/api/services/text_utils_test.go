package services

import (
	"strings"
	"testing"
)

func TestSampleTextForAIShortText(t *testing.T) {
	input := "这是一个很短的参考文本。"
	output := sampleTextForAI(input, 12000)
	if output != input {
		t.Fatalf("expected short text to remain unchanged, got %q", output)
	}
}

func TestSampleTextForAILongText(t *testing.T) {
	input := strings.Repeat("第一段风格内容。", 1200) +
		strings.Repeat("第二段中部内容。", 1200) +
		strings.Repeat("第三段结尾内容。", 1200)

	output := sampleTextForAI(input, 12000)
	if !strings.Contains(output, "【开篇片段】") {
		t.Fatalf("expected sampled output to contain head label")
	}
	if !strings.Contains(output, "【中段片段】") {
		t.Fatalf("expected sampled output to contain middle label")
	}
	if !strings.Contains(output, "【后段片段】") {
		t.Fatalf("expected sampled output to contain tail label")
	}
	if !strings.Contains(output, "第一段风格内容") {
		t.Fatalf("expected sampled output to include head content")
	}
	if !strings.Contains(output, "第二段中部内容") {
		t.Fatalf("expected sampled output to include middle content")
	}
	if !strings.Contains(output, "第三段结尾内容") {
		t.Fatalf("expected sampled output to include tail content")
	}
}
