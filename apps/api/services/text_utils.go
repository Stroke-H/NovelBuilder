package services

import "strings"

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func truncateForAI(text string, maxRunes int) string {
	runes := []rune(text)
	if maxRunes <= 0 || len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes]) + "\n\n内容过长，已截断。"
}

func sampleTextForAI(text string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(text))
	if maxRunes <= 0 || len(runes) <= maxRunes {
		return string(runes)
	}
	if maxRunes < 600 {
		return truncateForAI(string(runes), maxRunes)
	}

	type segment struct {
		label string
		start int
		end   int
	}

	labels := []string{"【开篇片段】\n", "【中段片段】\n", "【后段片段】\n"}
	separator := "\n\n"
	note := "\n\n内容过长，以上为开篇、中段、后段抽样片段。"

	overhead := len([]rune(note)) + len([]rune(separator))*2
	for _, label := range labels {
		overhead += len([]rune(label))
	}
	budget := maxRunes - overhead
	if budget < 300 {
		return truncateForAI(string(runes), maxRunes)
	}

	perSegment := budget / 3
	if perSegment < 100 {
		return truncateForAI(string(runes), maxRunes)
	}

	total := len(runes)
	head := segment{label: labels[0], start: 0, end: minInt(perSegment, total)}
	midStart := maxInt(0, total/2-perSegment/2)
	middle := segment{label: labels[1], start: midStart, end: minInt(midStart+perSegment, total)}
	tailStart := maxInt(0, total-perSegment)
	tail := segment{label: labels[2], start: tailStart, end: total}

	parts := []string{
		head.label + strings.TrimSpace(string(runes[head.start:head.end])),
		middle.label + strings.TrimSpace(string(runes[middle.start:middle.end])),
		tail.label + strings.TrimSpace(string(runes[tail.start:tail.end])),
	}

	result := strings.Join(parts, separator) + note
	resultRunes := []rune(result)
	if len(resultRunes) <= maxRunes {
		return result
	}
	return string(resultRunes[:maxRunes])
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
