package ai

import (
	"regexp"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// Whisper hallucinates subtitle credits during silence/music passages.
var artifactPattern = regexp.MustCompile(`(?i)(untertitel im auftrag|untertitelung des zdf|untertitel der amara|amara\.org|copyright wdr|swr 20\d\d)`)

const maxWordRepeats = 4

// collapseRepetition reduces runs of 5+ identical words ("oh, oh, oh, ...")
// to a single occurrence. Go's RE2 has no backreferences, so this walks the
// tokens directly; comparison ignores case and trailing punctuation.
func collapseRepetition(text string) string {
	words := strings.Fields(text)
	if len(words) <= maxWordRepeats {
		return text
	}

	normalize := func(w string) string {
		return strings.ToLower(strings.TrimRight(w, ",.!?"))
	}

	var out []string
	runStart := 0

	flush := func(end int) {
		runLen := end - runStart
		if runLen > maxWordRepeats {
			out = append(out, words[end-1])
		} else {
			out = append(out, words[runStart:end]...)
		}
	}

	for i := 1; i <= len(words); i++ {
		if i == len(words) || normalize(words[i]) != normalize(words[runStart]) {
			flush(i)
			runStart = i
		}
	}

	if len(out) == len(words) {
		return text
	}

	leading := ""
	if strings.HasPrefix(text, " ") {
		leading = " "
	}

	return leading + strings.Join(out, " ")
}

// CleanTranscription strips Whisper hallucination artifacts: credit lines
// invented during silence are dropped entirely, word-repetition loops are
// collapsed, and the full text is rebuilt from the surviving segments.
func CleanTranscription(resp openai.AudioResponse) openai.AudioResponse {
	if len(resp.Segments) == 0 {
		resp.Text = collapseRepetition(resp.Text)
		return resp
	}

	kept := resp.Segments[:0]
	var texts []string

	for _, segment := range resp.Segments {
		if artifactPattern.MatchString(segment.Text) {
			continue
		}

		segment.Text = collapseRepetition(segment.Text)
		kept = append(kept, segment)
		texts = append(texts, strings.TrimSpace(segment.Text))
	}

	resp.Segments = kept
	resp.Text = strings.Join(texts, " ")

	return resp
}
