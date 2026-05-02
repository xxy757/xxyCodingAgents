package learning

import (
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// SearchOptions 定义学习检索参数。
type SearchOptions struct {
	ProjectSlug string
	Phase       string
	QueryText   string
	Limit       int
}

// Searcher 基于 JSONL 存储做轻量检索。
type Searcher struct {
	store *Store
}

// NewSearcher 创建 learnings 检索器。
func NewSearcher(rootDir string) *Searcher {
	return &Searcher{
		store: NewStore(rootDir),
	}
}

// SearchInsights 返回可直接注入 Prompt Layer 3 的经验文本。
func (s *Searcher) SearchInsights(opts SearchOptions) ([]string, error) {
	if s == nil || s.store == nil {
		return nil, nil
	}
	if opts.Limit <= 0 {
		opts.Limit = 5
	}

	entries, err := s.store.ReadAll(opts.ProjectSlug)
	if err != nil || len(entries) == 0 {
		return nil, err
	}

	phase := normalizePhase(opts.Phase)
	tokens := tokenizeQuery(opts.QueryText)

	type scored struct {
		entry     Entry
		score     int
		createdAt time.Time
	}

	bestByKey := make(map[string]scored)
	for _, e := range entries {
		score := scoreEntry(e, phase, tokens)
		if score <= 0 {
			continue
		}
		key := dedupKey(e)
		c := scored{
			entry:     e,
			score:     score,
			createdAt: parseTS(e.TS),
		}
		if existing, ok := bestByKey[key]; !ok || c.score > existing.score || c.createdAt.After(existing.createdAt) {
			bestByKey[key] = c
		}
	}

	if len(bestByKey) == 0 {
		return nil, nil
	}

	all := make([]scored, 0, len(bestByKey))
	for _, v := range bestByKey {
		all = append(all, v)
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].score != all[j].score {
			return all[i].score > all[j].score
		}
		return all[i].createdAt.After(all[j].createdAt)
	})

	if len(all) > opts.Limit {
		all = all[:opts.Limit]
	}
	out := make([]string, 0, len(all))
	for _, v := range all {
		out = append(out, formatInsight(v.entry))
	}
	return out, nil
}

func scoreEntry(e Entry, phase string, tokens []string) int {
	score := 0

	skill := normalizePhase(e.Skill)
	if phase != "" {
		if skill == phase {
			score += 5
		} else if alias, ok := phaseAlias[phase]; ok && skill == alias {
			score += 4
		}
	}

	blob := strings.ToLower(strings.Join([]string{
		e.Key,
		e.Insight,
		strings.Join(e.Files, " "),
	}, "\n"))
	for _, token := range tokens {
		if strings.Contains(blob, token) {
			score += 2
		}
	}
	score += e.Confidence / 3
	return score
}

func formatInsight(e Entry) string {
	meta := make([]string, 0, 3)
	if e.Skill != "" {
		meta = append(meta, e.Skill)
	}
	if e.Type != "" {
		meta = append(meta, e.Type)
	}
	if e.Confidence > 0 {
		meta = append(meta, "c="+strconv.Itoa(e.Confidence))
	}
	prefix := ""
	if len(meta) > 0 {
		prefix = "[" + strings.Join(meta, "/") + "] "
	}
	if e.Key != "" {
		return prefix + e.Key + ": " + e.Insight
	}
	return prefix + e.Insight
}

func dedupKey(e Entry) string {
	return strings.ToLower(strings.TrimSpace(e.Key) + "|" + strings.TrimSpace(e.Insight))
}

func parseTS(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

func normalizePhase(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func tokenizeQuery(v string) []string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return nil
	}
	parts := strings.FieldsFunc(v, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	set := make(map[string]struct{})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if len(p) < 3 {
			continue
		}
		if _, ok := set[p]; ok {
			continue
		}
		set[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

var phaseAlias = map[string]string{
	"browser-qa":  "qa",
	"code-review": "review",
	"postmortem":  "retro",
}
