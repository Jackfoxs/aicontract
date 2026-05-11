package splitter

import (
	"hash/fnv"
	"regexp"
	"strings"
	"unicode"
)

// RuleUnit 表示按规范原子化后的最小规则单元
type RuleUnit struct {
	Title       string   // 规则标题或头部（如：第3条 医疗器械定义）
	Text        string   // 规则正文文本
	SectionPath string   // 章节路径（如：第3章>3.1>3.1.2）
	RuleID      string   // 规则ID（可由标题/序号派生）
	Page        int      // 所在页码（没有则为0）
	CharStart   int      // 在原文字符串中的rune起始位置
	CharEnd     int      // 在原文字符串中的rune结束位置（含）
	Anchors     []string // 锚点短语（标题、编号等）
	Aliases     []string // 别名/常见俗称
}

var (
	// 章节：第十章、第三章 标题
	chapterRe = regexp.MustCompile(`(?m)^\s*第[一二三四五六七八九十百千零〇0-9]+章[^\n]*`)
	// 条款：第12条、第一条 标题
	articleRe = regexp.MustCompile(`(?m)^\s*第[一二三四五六七八九十百千零〇0-9]+条[^\n]*`)
	// 数字分级：1.、1.1、2.3.4 节标题
	dottedRe = regexp.MustCompile(`(?m)^\s*(?:[0-9]{1,2}\.){1,4}[^\n]*`)
	// 阿拉伯数字枚举：1、 1. 1) (1) （1）等
	enumArabicRe = regexp.MustCompile(`(?m)^\s*(?:\(?[0-9]{1,3}\)?[、\.．)]|（[0-9]{1,3}）)\s*[^\n]*`)
	// 中文数字枚举：一、 （一） 等
	enumChineseRe = regexp.MustCompile(`(?m)^\s*(?:[（(]?[一二三四五六七八九十百千零〇]+[)）]|[一二三四五六七八九十百千零〇]+、)\s*[^\n]*`)
	// 前缀抽取用
	articlePrefixRe = regexp.MustCompile(`^第[一二三四五六七八九十百千零〇0-9]+条`)
	chapterPrefixRe = regexp.MustCompile(`^第[一二三四五六七八九十百千零〇0-9]+章`)
	dottedPrefixRe  = regexp.MustCompile(`^(?:[0-9]{1,2}\.){1,3}[0-9]{1,2}`)
	// 任意标题统一匹配（章/条/数字分级/枚举项）
	anyHeaderRe = regexp.MustCompile(`(?m)^\s*(?:第[一二三四五六七八九十百千零〇0-9]+章[^\n]*|第[一二三四五六七八九十百千零〇0-9]+条[^\n]*|(?:[0-9]{1,2}\.){1,4}[^\n]*|(?:\(?[0-9]{1,3}\)?[、\.．)]|（[0-9]{1,3}）)\s*[^\n]*|(?:[（(]?[一二三四五六七八九十百千零〇]+[)）]|[一二三四五六七八九十百千零〇]+、)\s*[^\n]*)`)
	// 前缀提取：用于 SectionPath
	enumArabicPrefixRe  = regexp.MustCompile(`^(?:\(?[0-9]{1,3}\)?[、\.．)]|（[0-9]{1,3}）)`)
	enumChinesePrefixRe = regexp.MustCompile(`^(?:[（(]?[一二三四五六七八九十百千零〇]+[)）]|[一二三四五六七八九十百千零〇]+、)`)
)

// HasRuleHeaders 判断文本中是否存在可用于规则级切分的“章/条/编号/枚举”标题
func HasRuleHeaders(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}
	return anyHeaderRe.MatchString(trimmed)
}

// SplitRules 将文本按“条/款/项/编号”进行启发式规则级切分
// 优先级：条款 > 数字分级 > 回退为自然段聚合
func SplitRules(content string) []RuleUnit {
	return SplitRulesWithPage(content, 0)
}

// SplitRulesWithPage 同上，但附带页码信息
func SplitRulesWithPage(content string, page int) []RuleUnit {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return []RuleUnit{}
	}

	// 统一用“任意标题”切分，避免不同类型标题被吞并
	units := sliceByAnyHeader(trimmed, anyHeaderRe, page)
	if len(units) > 0 {
		return units
	}
	// 兜底：按空行聚合为片段
	return fallbackParagraphs(trimmed, page)
}

// SplitArticles 仅按“第X条”进行切分
func SplitArticles(content string) []RuleUnit {
	return SplitArticlesWithPage(content, 0)
}

// SplitArticlesWithPage 仅按“第X条”进行切分，并附带页码信息
func SplitArticlesWithPage(content string, page int) []RuleUnit {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return []RuleUnit{}
	}
	units := sliceByHeader(trimmed, articleRe, page)
	if len(units) > 0 {
		return units
	}
	return []RuleUnit{}
}

// sliceByHeader 基于标题正则切片，产生 RuleUnit
func sliceByHeader(s string, re *regexp.Regexp, page int) []RuleUnit {
	idx := re.FindAllStringIndex(s, -1)
	if len(idx) == 0 {
		return nil
	}

	var result []RuleUnit
	runeOffsets := buildRunePrefix(s)

	for i := 0; i < len(idx); i++ {
		start := idx[i][0]
		end := len(s)
		if i+1 < len(idx) {
			end = idx[i+1][0]
		}
		rawHeader := strings.TrimSpace(s[idx[i][0]:idx[i][1]])
		body := strings.TrimSpace(s[idx[i][1]:end])
		// 截断正文到下一个标题前
		if j := re.FindStringIndex(body); j != nil && j[0] > 0 {
			body = strings.TrimSpace(body[:j[0]])
		}

		startRune := runeOffsetAtByte(runeOffsets, start)
		endRune := runeOffsetAtByte(runeOffsets, start+len(rawHeader)+len(body))

		// 标题仅保留核心编号（如：第八十三条），内容保留完整（含标题）
		title := coreTitleFromHeader(rawHeader)
		headerRemainder := strings.TrimSpace(strings.TrimPrefix(rawHeader, title))
		primaryLine := title
		if headerRemainder != "" {
			primaryLine = strings.TrimSpace(title + " " + headerRemainder)
		}
		text := primaryLine
		if body != "" {
			text = strings.TrimSpace(primaryLine + "\n" + body)
		}
		if text == "" {
			continue
		}

		anchors := []string{rawHeader, compactSpaces(rawHeader), title, compactSpaces(title)}
		unit := RuleUnit{
			Title:       title,
			Text:        text,
			SectionPath: deriveSectionPath(title),
			RuleID:      deriveRuleID(title),
			Page:        page,
			CharStart:   startRune,
			CharEnd:     endRune,
			Anchors:     uniqueStrings(anchors),
			Aliases:     []string{},
		}
		result = append(result, unit)
	}

	return result
}

// sliceByAnyHeader 使用统一标题正则并切到下一任意标题，防止不同类型标题跨越合并
func sliceByAnyHeader(s string, re *regexp.Regexp, page int) []RuleUnit {
	idx := re.FindAllStringIndex(s, -1)
	if len(idx) == 0 {
		return nil
	}
	var result []RuleUnit
	runeOffsets := buildRunePrefix(s)
	for i := 0; i < len(idx); i++ {
		start := idx[i][0]
		end := len(s)
		if i+1 < len(idx) {
			end = idx[i+1][0]
		}
		// 标题行到换行符
		seg := s[start:end]
		lineEnd := strings.IndexByte(seg, '\n')
		header := strings.TrimSpace(func() string {
			if lineEnd >= 0 {
				return seg[:lineEnd]
			}
			return seg
		}())
		bodyStart := start + len(header)
		if bodyStart < end && s[bodyStart] == '\n' {
			bodyStart++
		}
		body := strings.TrimSpace(s[bodyStart:end])

		startRune := runeOffsetAtByte(runeOffsets, start)
		endRune := runeOffsetAtByte(runeOffsets, end)
		anchors := []string{header, compactSpaces(header)}
		// 标题仅保留核心编号（如：第八十三条）
		core := coreTitleFromHeader(header)
		headerRemainder := strings.TrimSpace(strings.TrimPrefix(header, core))
		primaryLine := core
		if headerRemainder != "" {
			primaryLine = strings.TrimSpace(core + " " + headerRemainder)
		}
		combined := primaryLine
		if body != "" {
			combined = strings.TrimSpace(primaryLine + "\n" + body)
		}
		result = append(result, RuleUnit{
			Title:       core,
			Text:        combined,
			SectionPath: deriveSectionPath(header),
			RuleID:      deriveRuleID(header),
			Page:        page,
			CharStart:   startRune,
			CharEnd:     endRune,
			Anchors:     uniqueStrings(append(anchors, core, compactSpaces(core))),
			Aliases:     []string{},
		})
	}
	return result
}

// fallbackParagraphs 按空行聚合
func fallbackParagraphs(s string, page int) []RuleUnit {
	paras := strings.Split(s, "\n\n")
	var res []RuleUnit
	offset := 0
	runeOffsets := buildRunePrefix(s)
	for _, p := range paras {
		p = strings.TrimSpace(p)
		if p == "" {
			offset += 2 // 跳过空行
			continue
		}
		startByte := strings.Index(s[offset:], p)
		if startByte < 0 {
			startByte = 0
		}
		startByte += offset
		endByte := startByte + len(p)
		startRune := runeOffsetAtByte(runeOffsets, startByte)
		endRune := runeOffsetAtByte(runeOffsets, endByte)

		title := firstLine(p)
		anchors := []string{title, compactSpaces(title)}
		res = append(res, RuleUnit{
			Title:       title,
			Text:        p,
			SectionPath: "",
			RuleID:      deriveRuleID(title),
			Page:        page,
			CharStart:   startRune,
			CharEnd:     endRune,
			Anchors:     uniqueStrings(anchors),
			Aliases:     []string{},
		})
		offset = endByte
	}
	return res
}

// Fingerprint 生成归一化指纹（FNV-1a 64）
func Fingerprint(s string) string {
	n := normalize(s)
	// 控制长度以稳定指纹
	if len(n) > 2048 {
		n = n[:2048]
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(n))
	return toHex(h.Sum(nil))
}

func toHex(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexdigits[v>>4]
		out[i*2+1] = hexdigits[v&0x0f]
	}
	return string(out)
}

// normalize 统一大小写、去空白与标点
func normalize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		if unicode.IsPunct(r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func compactSpaces(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	if len(s) > 80 {
		return strings.TrimSpace(s[:80])
	}
	return strings.TrimSpace(s)
}

func deriveRuleID(title string) string {
	// 粗略以标题指纹为ID，稳定可追踪
	return Fingerprint(title)
}

func deriveSectionPath(title string) string {
	t := strings.TrimSpace(title)
	if m := articlePrefixRe.FindString(t); m != "" {
		return m
	}
	if m := chapterPrefixRe.FindString(t); m != "" {
		return m
	}
	if m := dottedPrefixRe.FindString(t); m != "" {
		return m
	}
	if m := enumArabicPrefixRe.FindString(t); m != "" {
		return m
	}
	if m := enumChinesePrefixRe.FindString(t); m != "" {
		return m
	}
	// 回退：压缩空白后的标题
	return compactSpaces(t)
}

// coreTitleFromHeader 提取标题的核心编号/标识，如：
// "第八十三条 ……" -> "第八十三条"
// "第三章 总则" -> "第三章"
// "1.2.3 标题" -> "1.2.3"
// "（一） 内容" -> "（一）"
func coreTitleFromHeader(header string) string {
	h := strings.TrimSpace(header)
	if m := articlePrefixRe.FindString(h); m != "" {
		return m
	}
	if m := chapterPrefixRe.FindString(h); m != "" {
		return m
	}
	if m := dottedPrefixRe.FindString(h); m != "" {
		return m
	}
	if m := enumArabicPrefixRe.FindString(h); m != "" {
		return strings.TrimSpace(m)
	}
	if m := enumChinesePrefixRe.FindString(h); m != "" {
		return strings.TrimSpace(m)
	}
	// 回退：取第一行的第一个词组
	line := firstLine(h)
	// 到下一个空格/制表符为止
	for i, r := range line {
		if unicode.IsSpace(r) {
			return strings.TrimSpace(line[:i])
		}
	}
	return strings.TrimSpace(line)
}

// runeOffsetAtByte 将byte下标转换为rune下标
func runeOffsetAtByte(prefix []int, byteIdx int) int {
	// prefix[k] = 字节k处的rune计数，超界则返回末尾
	if byteIdx < 0 {
		return 0
	}
	if byteIdx >= len(prefix) {
		return prefix[len(prefix)-1]
	}
	return prefix[byteIdx]
}

// buildRunePrefix 构建字节位置到rune数量的前缀表
func buildRunePrefix(s string) []int {
	// prefix长度= len(s)+1，prefix[i] 表示 到i字节前的rune计数
	// 为了简化索引，这里构建 len(s) 长度的数组，值为“到该字节为止的rune计数”
	prefix := make([]int, len(s)+1)
	count := 0
	for i := 0; i < len(s); {
		r, size := rune(s[i]), 1
		if r >= 0x80 { // 多字节
			r, size = utf8DecodeRuneInString(s[i:])
		}
		for k := 0; k < size; k++ {
			prefix[i+k] = count
		}
		i += size
		count++
	}
	if len(prefix) > 0 {
		prefix[len(prefix)-1] = count
	}
	return prefix
}

func utf8DecodeRuneInString(s string) (r rune, size int) {
	// 手写一个小封装，避免额外依赖
	for i := 1; i <= 4 && i <= len(s); i++ {
		rr := []rune(s[:i])
		if len(rr) == 1 {
			return rr[0], i
		}
	}
	rr := []rune(s)
	if len(rr) == 0 {
		return rune(0), 1
	}
	return rr[0], len(string(rr[0]))
}

func uniqueStrings(in []string) []string {
	m := make(map[string]struct{}, len(in))
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := m[s]; ok {
			continue
		}
		m[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
