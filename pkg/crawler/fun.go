package crawler

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

var rxFunName = regexp.MustCompile(`^[a-z$][a-zA-Z]{0,15}`)

type parseState int

const (
	stateNone parseState = iota
	stateKey
	stateStr
	stateExp
	stateStd
)

type exitParseInfo struct {
	kind parseState
	isK  bool
}

func PowerfulFind(s *goquery.Selection, q string) *goquery.Selection {
	rxSelectPseudoEq := regexp.MustCompile(`:eq\(\d+\)`)
	if rxSelectPseudoEq.MatchString(q) {
		rs := rxSelectPseudoEq.FindAllStringIndex(q, -1)
		sel := s
		for _, r := range rs {
			iStr := q[r[0]+4 : r[1]-1]
			i64, _ := strconv.ParseInt(iStr, 10, 32)
			i := int(i64)
			sq := q[:r[0]]
			q = strings.TrimSpace(q[r[1]:])
			sel = sel.Find(sq).Eq(i)
		}
		if q != "" {
			sel = sel.Find(q)
		}
		return sel
	}
	return s.Find(q)
}

type Fun struct {
	Name   string
	Raw    string
	Params []string

	Document  *goquery.Document
	Selection *goquery.Selection
	Result    string

	PrevFun *Fun
	NextFun *Fun
}

func (f *Fun) InitSelector() error {
	if len(f.Params) > 0 {
		f.Selection = PowerfulFind(f.Selection, f.Params[0])
	}
	return nil
}

func (f *Fun) Invoke() (string, error) {
	var err error
	switch f.Name {
	case "$":
		err = f.invokeDollar()
	case "attr":
		f.Result, err = f.invokeAttr()
	case "text":
		f.Result = f.invokeText()
	case "html":
		f.Result, err = f.invokeHtml()
	case "outerHTML":
		f.Result, err = f.invokeOuterHtml()
	case "style":
		f.Result, err = f.invokeStyle()
	case "href":
		f.Result, err = f.invokeHref()
	case "src":
		f.Result, err = f.invokeSrc()
	case "class":
		f.Result, err = f.invokeClass()
	case "id":
		f.Result, err = f.invokeId()
	case "expand":
		f.Result, err = f.invokeExpand()
	case "match":
		f.Result, err = f.invokeMatch()
	}
	if err != nil {
		return "", err
	}
	if f.NextFun != nil {
		return f.NextFun.Invoke()
	}
	return f.Result, nil
}

func (f *Fun) invokeDollar() error {
	return f.InitSelector()
}

func (f *Fun) invokeAttr() (string, error) {
	v, _ := f.PrevFun.Selection.Attr(f.Params[0])
	return v, nil
}

func (f *Fun) invokeText() string {
	return f.PrevFun.Selection.Text()
}

func (f *Fun) invokeHtml() (string, error) {
	return f.PrevFun.Selection.Html()
}

func (f *Fun) invokeOuterHtml() (string, error) {
	return goquery.OuterHtml(f.PrevFun.Selection)
}

func (f *Fun) invokeStyle() (string, error) {
	v, _ := f.PrevFun.Selection.Attr("style")
	return v, nil
}

func (f *Fun) invokeHref() (string, error) {
	v, _ := f.PrevFun.Selection.Attr("href")
	return v, nil
}

func (f *Fun) invokeSrc() (string, error) {
	v, _ := f.PrevFun.Selection.Attr("src")
	return v, nil
}

func (f *Fun) invokeClass() (string, error) {
	v, _ := f.PrevFun.Selection.Attr("class")
	return v, nil
}

func (f *Fun) invokeId() (string, error) {
	v, _ := f.PrevFun.Selection.Attr("id")
	return v, nil
}

func (f *Fun) invokeExpand() (string, error) {
	rx, err := regexp.Compile(f.Params[0])
	if err != nil {
		return "", err
	}
	src := f.PrevFun.Result
	var dst []byte
	m := rx.FindStringSubmatchIndex(src)
	s := rx.ExpandString(dst, f.Params[1], src, m)
	return string(s), nil
}

func (f *Fun) invokeMatch() (string, error) {
	rx, err := regexp.Compile(f.Params[0])
	if err != nil {
		return "", err
	}
	rs := rx.FindAllStringSubmatch(f.PrevFun.Result, -1)
	if len(rs) > 0 && len(rs[0]) > 1 {
		return rs[0][1], nil
	}
	return "", nil
}

func (f *Fun) Append(s string) (*Fun, *Fun) {
	f.NextFun = ParseFun(f.Selection, s)
	f.NextFun.PrevFun = f
	return f, f.NextFun
}

func ParseFun(sel *goquery.Selection, str string) *Fun {
	fun := new(Fun)
	fun.Raw = str
	fun.Selection = sel

	sa := rxFunName.FindAllString(str, -1)
	fun.Name = sa[0]
	ls := str[len(sa[0]):]
	var ps []string
	p, pl := parseParams(ls)
	for i := 0; ; i++ {
		if v, ok := p["$"+strconv.Itoa(i)]; ok {
			ps = append(ps, v)
		} else {
			break
		}
	}
	if len(ps) > 0 {
		fun.Params = ps
	}
	ls = ls[pl+1:]
	if ls != "" {
		ls = ls[1:]
		fun.Append(ls)
	}

	return fun
}

func charAtOffset(s string, i, o int) rune {
	oi := i + o
	if oi >= 0 && oi < len(s) {
		return rune(s[oi])
	}
	return 0
}

func charSkipWhitespace(s string, i, o int) rune {
	if i+o < 0 || i >= len(s) {
		return 0
	}
	if o < 0 {
		j := i
		for j >= 0 && o != 0 {
			j--
			if !unicode.IsSpace(rune(s[j])) {
				o++
			}
		}
		return rune(s[j])
	} else if o > 0 {
		j := i
		for j < len(s)-1 && o != 0 {
			j++
			if !unicode.IsSpace(rune(s[j])) {
				o--
			}
		}
		return rune(s[j])
	}
	return rune(s[i])
}

func isEntryDelim(co rune) bool {
	return co == '(' || co == ','
}

func isStdPrefix(co1, c rune) bool {
	return (co1 == '=' || co1 == ',' || co1 == '(') && !unicode.IsSpace(c) && c != '"' && c != '`'
}

func isStrExpPrefix(co rune) bool {
	return co == '=' || co == ',' || co == '('
}

func enterParseState(s string, i int, c rune, inExp, inStr, inStd bool) parseState {
	if inExp || inStr || inStd {
		return stateNone
	}
	co1 := charSkipWhitespace(s, i, -1)
	if isEntryDelim(co1) && (unicode.IsLetter(c) || c == '@') {
		return stateKey
	}
	if isStdPrefix(co1, c) {
		return stateStd
	}
	co2 := charSkipWhitespace(s, i, -2)
	if isStrExpPrefix(co2) {
		switch co1 {
		case '"':
			return stateStr
		case '`':
			return stateExp
		}
	}
	return stateNone
}

func exitParseState(s string, i int, c rune, inKey, inStr, inExp, inStd bool) exitParseInfo {
	if c == '\\' {
		return exitParseInfo{kind: stateNone}
	}
	co1 := charSkipWhitespace(s, i, 1)
	cso1 := charAtOffset(s, i, 1)
	if inKey && (co1 == ',' || co1 == ')' || co1 == '=') {
		return exitParseInfo{kind: stateKey, isK: co1 != ','}
	}
	if inStr && cso1 == '"' {
		return exitParseInfo{kind: stateStr}
	}
	if inExp && cso1 == '`' {
		return exitParseInfo{kind: stateExp}
	}
	if inStd && (co1 == ',' || co1 == ')') {
		return exitParseInfo{kind: stateStd}
	}
	return exitParseInfo{kind: stateNone}
}

func isInAnyParseState(inKey, inStr, inExp, inStd bool) bool {
	return inKey || inStr || inExp || inStd
}

func isEndParen(inExp, inStd, inStr, inKey bool, c rune) bool {
	return !inExp && !inStd && !inStr && !inKey && c == ')'
}

func applyEnterParseState(s string, i int, c rune, inKey, inStr, inExp, inStd *bool) {
	switch enterParseState(s, i, c, *inExp, *inStr, *inStd) {
	case stateKey:
		*inKey = true
	case stateStr:
		*inStr = true
	case stateExp:
		*inExp = true
	case stateStd:
		*inStd = true
	}
}

func applyExitParseState(s string, i int, c rune, inKey, inStr, inExp, inStd *bool, sb *bytes.Buffer, pK *string, kvMap map[string]string, pIsK *bool, insertVal func(string)) {
	if c == '\\' {
		return
	}
	info := exitParseState(s, i, c, *inKey, *inStr, *inExp, *inStd)
	switch info.kind {
	case stateKey:
		*inKey = false
		*pK = strings.TrimSpace(sb.String())
		kvMap[*pK] = ""
		if info.isK {
			*pIsK = true
		}
		sb.Reset()
	case stateStr:
		*inStr = false
		v := strings.TrimSpace(sb.String())
		v = strings.ReplaceAll(v, `\\`, `\`)
		insertVal(v)
		sb.Reset()
	case stateExp:
		*inExp = false
		insertVal(strings.TrimSpace(sb.String()))
		sb.Reset()
	case stateStd:
		*inStd = false
		insertVal(strings.TrimSpace(sb.String()))
		sb.Reset()
	}
}

// start with "(", will return params map and end pos.
// all params string type:
// (key1 = 0, key2 = "str_exam\"ple", key3 = `exp_\`example\n`)
// (key1 = 0, key2, key3)
// (key1, key2, key3)
// ("str_exam\"ple", /exp_\/example\n/, 2)
// source: https://github.com/wspl/creeper/blob/eb1753da1c54ade30e8e6ee82e1923b4473dbc13/town.go
func parseParams(s string) (map[string]string, int) {
	endPos := -1

	kvMap := map[string]string{}
	pK := ""
	pIsK := false

	var sb bytes.Buffer

	inKey := false
	inStr := false
	inExp := false
	inStd := false

	noKeyIndex := 0
	insertVal := func(v string) {
		if pIsK {
			kvMap[pK] = strings.TrimSpace(sb.String())
		} else {
			kvMap["$"+strconv.Itoa(noKeyIndex)] = v
			noKeyIndex++
		}
		pIsK = false
	}

	for i, c := range s {
		if i == 0 && c != '(' {
			return nil, -1
		}

		applyEnterParseState(s, i, c, &inKey, &inStr, &inExp, &inStd)

		if isInAnyParseState(inKey, inStr, inExp, inStd) {
			_, _ = sb.WriteRune(c)
		}

		if isEndParen(inExp, inStd, inStr, inKey, c) {
			endPos = i
		}

		applyExitParseState(s, i, c, &inKey, &inStr, &inExp, &inStd, &sb, &pK, kvMap, &pIsK, insertVal)

		if endPos > -1 {
			break
		}
	}

	return kvMap, endPos
}
