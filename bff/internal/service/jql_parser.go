package service

import (
	"fmt"
	"strings"
	"unicode"
)

// JQL（Jira Query Language）の簡易版の字句解析・構文解析。
// 例: project = KAN AND status != Done AND assignee = currentUser() ORDER BY priority DESC
//
//	query   := [or] [ORDER BY order {"," order}]
//	or      := and {OR and}
//	and     := not {AND not}
//	not     := NOT not | "(" or ")" | clause
//	clause  := field ( op value | [NOT] IN list | IS [NOT] (EMPTY | NULL) )
//	op      := "=" | "!=" | "~" | ">" | ">=" | "<" | "<="
//	value   := word | string | function "(" ")" | EMPTY | NULL
//	list    := "(" value {"," value} ")"
//	order   := field [ASC | DESC]
//
// キーワード（AND・OR など）は大文字・小文字を区別しない。値に空白や記号を含むときは "…" か '…' で囲む

// JQLError は JQL の誤り。Pos はクエリの先頭からの文字数（0 から。ルーン単位）
type JQLError struct {
	Pos int
	Msg string
}

func (e *JQLError) Error() string {
	return fmt.Sprintf("%d 文字目: %s", e.Pos+1, e.Msg)
}

func jqlErrorf(pos int, format string, args ...any) *JQLError {
	return &JQLError{Pos: pos, Msg: fmt.Sprintf(format, args...)}
}

type jqlTokenKind int

const (
	tokEOF jqlTokenKind = iota
	tokWord
	tokString
	tokOp
	tokLParen
	tokRParen
	tokComma
)

type jqlToken struct {
	kind jqlTokenKind
	text string // 文字列は囲みを外してエスケープを戻した中身
	pos  int
}

// isKeyword は引用符で囲んでいない語が、大文字・小文字を区別せずキーワード kw かを返す
func (t jqlToken) isKeyword(kw string) bool {
	return t.kind == tokWord && strings.EqualFold(t.text, kw)
}

// describe はエラーメッセージに出すトークンの表し方
func (t jqlToken) describe() string {
	switch t.kind {
	case tokEOF:
		return "クエリの終わり"
	case tokString:
		return fmt.Sprintf("「\"%s\"」", t.text)
	default:
		return fmt.Sprintf("「%s」", t.text)
	}
}

var jqlKeywords = []string{"AND", "OR", "NOT", "IN", "IS", "EMPTY", "NULL", "ORDER", "BY", "ASC", "DESC"}

func isJQLKeyword(word string) bool {
	for _, kw := range jqlKeywords {
		if strings.EqualFold(word, kw) {
			return true
		}
	}
	return false
}

// isWordRune は引用符なしの語に使える文字か。空白と JQL の記号以外はすべて使える（KAN-12・2026-10-01・-7d・日本語など）
func isWordRune(r rune) bool {
	return !unicode.IsSpace(r) && !strings.ContainsRune(`=!~<>(),"'`, r)
}

// tokenizeJQL はクエリをトークンに分ける。最後は必ず tokEOF
func tokenizeJQL(query string) ([]jqlToken, error) {
	runes := []rune(query)
	tokens := []jqlToken{}
	for i := 0; i < len(runes); {
		r := runes[i]
		switch {
		case unicode.IsSpace(r):
			i++
		case r == '(':
			tokens = append(tokens, jqlToken{tokLParen, "(", i})
			i++
		case r == ')':
			tokens = append(tokens, jqlToken{tokRParen, ")", i})
			i++
		case r == ',':
			tokens = append(tokens, jqlToken{tokComma, ",", i})
			i++
		case r == '=' || r == '~':
			tokens = append(tokens, jqlToken{tokOp, string(r), i})
			i++
		case r == '!':
			if i+1 < len(runes) && runes[i+1] == '=' {
				tokens = append(tokens, jqlToken{tokOp, "!=", i})
				i += 2
			} else {
				return nil, jqlErrorf(i, "「!」の後には「=」が必要です")
			}
		case r == '<' || r == '>':
			if i+1 < len(runes) && runes[i+1] == '=' {
				tokens = append(tokens, jqlToken{tokOp, string(r) + "=", i})
				i += 2
			} else {
				tokens = append(tokens, jqlToken{tokOp, string(r), i})
				i++
			}
		case r == '"' || r == '\'':
			start := i
			var b strings.Builder
			i++
			closed := false
			for i < len(runes) {
				c := runes[i]
				if c == '\\' && i+1 < len(runes) {
					b.WriteRune(runes[i+1])
					i += 2
					continue
				}
				if c == r {
					closed = true
					i++
					break
				}
				b.WriteRune(c)
				i++
			}
			if !closed {
				return nil, jqlErrorf(start, "引用符 %c が閉じられていません", r)
			}
			tokens = append(tokens, jqlToken{tokString, b.String(), start})
		default:
			start := i
			for i < len(runes) && isWordRune(runes[i]) {
				i++
			}
			tokens = append(tokens, jqlToken{tokWord, string(runes[start:i]), start})
		}
	}
	return append(tokens, jqlToken{tokEOF, "", len(runes)}), nil
}

// 構文木

type jqlNodeKind int

const (
	jqlAnd jqlNodeKind = iota
	jqlOr
	jqlNot
	jqlClauseNode
)

type jqlNode struct {
	kind     jqlNodeKind
	children []*jqlNode
	clause   *jqlClause
}

// 条件の演算子（構文上の形。= EMPTY は is empty にまとめる）
const (
	jqlOpEq         = "="
	jqlOpNe         = "!="
	jqlOpContains   = "~"
	jqlOpGt         = ">"
	jqlOpGe         = ">="
	jqlOpLt         = "<"
	jqlOpLe         = "<="
	jqlOpIn         = "in"
	jqlOpNotIn      = "not in"
	jqlOpIsEmpty    = "is empty"
	jqlOpIsNotEmpty = "is not empty"
)

type jqlClause struct {
	field    string
	fieldPos int
	op       string
	opPos    int
	values   []jqlValue
}

type jqlValueKind int

const (
	jqlValueText jqlValueKind = iota
	jqlValueFunc
	jqlValueEmpty
)

type jqlValue struct {
	kind   jqlValueKind
	text   string // 語・文字列の中身、または関数名
	quoted bool
	pos    int
}

type jqlOrderItem struct {
	field string
	desc  bool
	pos   int
}

type jqlQuery struct {
	where *jqlNode // nil なら絞り込まない
	order []jqlOrderItem
}

type jqlParser struct {
	tokens []jqlToken
	i      int
}

func (p *jqlParser) peek() jqlToken { return p.tokens[p.i] }

func (p *jqlParser) next() jqlToken {
	t := p.tokens[p.i]
	if t.kind != tokEOF {
		p.i++
	}
	return t
}

// parseJQL はクエリを構文木にする。誤りは位置付きの *JQLError
func parseJQL(query string) (*jqlQuery, error) {
	tokens, err := tokenizeJQL(query)
	if err != nil {
		return nil, err
	}
	p := &jqlParser{tokens: tokens}
	q := &jqlQuery{}
	if t := p.peek(); t.kind != tokEOF && !t.isKeyword("ORDER") {
		if q.where, err = p.parseOr(); err != nil {
			return nil, err
		}
	}
	if p.peek().isKeyword("ORDER") {
		if q.order, err = p.parseOrderBy(); err != nil {
			return nil, err
		}
	}
	if t := p.peek(); t.kind != tokEOF {
		if t.kind == tokRParen {
			return nil, jqlErrorf(t.pos, "対応する「(」がない「)」があります")
		}
		return nil, jqlErrorf(t.pos, "%sの前に AND か OR が必要です", t.describe())
	}
	return q, nil
}

func (p *jqlParser) parseOr() (*jqlNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().isKeyword("OR") {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &jqlNode{kind: jqlOr, children: []*jqlNode{left, right}}
	}
	return left, nil
}

func (p *jqlParser) parseAnd() (*jqlNode, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.peek().isKeyword("AND") {
		p.next()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &jqlNode{kind: jqlAnd, children: []*jqlNode{left, right}}
	}
	return left, nil
}

func (p *jqlParser) parseNot() (*jqlNode, error) {
	t := p.peek()
	if t.isKeyword("NOT") {
		p.next()
		child, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &jqlNode{kind: jqlNot, children: []*jqlNode{child}}, nil
	}
	if t.kind == tokLParen {
		p.next()
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if closing := p.peek(); closing.kind != tokRParen {
			return nil, jqlErrorf(closing.pos, "%sの前に「)」が必要です（%d 文字目の「(」が閉じられていません）", closing.describe(), t.pos+1)
		}
		p.next()
		return inner, nil
	}
	return p.parseClause()
}

func (p *jqlParser) parseField() (string, int, error) {
	t := p.peek()
	switch {
	case t.kind == tokString || (t.kind == tokWord && !isJQLKeyword(t.text)):
		p.next()
		return t.text, t.pos, nil
	case t.kind == tokEOF:
		return "", 0, jqlErrorf(t.pos, "項目名が必要です（例: status = Done）")
	default:
		return "", 0, jqlErrorf(t.pos, "%sの位置には項目名が必要です", t.describe())
	}
}

func (p *jqlParser) parseClause() (*jqlNode, error) {
	field, fieldPos, err := p.parseField()
	if err != nil {
		return nil, err
	}
	c := &jqlClause{field: field, fieldPos: fieldPos}
	t := p.next()
	c.opPos = t.pos
	switch {
	case t.kind == tokOp:
		c.op = t.text
		v, err := p.parseValue(false)
		if err != nil {
			return nil, err
		}
		// = EMPTY・!= EMPTY は IS EMPTY・IS NOT EMPTY と同じ
		if v.kind == jqlValueEmpty {
			switch c.op {
			case jqlOpEq:
				c.op = jqlOpIsEmpty
			case jqlOpNe:
				c.op = jqlOpIsNotEmpty
			default:
				return nil, jqlErrorf(v.pos, "EMPTY は =・!=・IS・IN と組み合わせて使ってください")
			}
		} else {
			c.values = []jqlValue{v}
		}
	case t.isKeyword("IN"):
		c.op = jqlOpIn
		if c.values, err = p.parseList(); err != nil {
			return nil, err
		}
	case t.isKeyword("NOT"):
		if in := p.next(); !in.isKeyword("IN") {
			return nil, jqlErrorf(in.pos, "NOT の後には IN が必要です（例: status NOT IN (Done)）")
		}
		c.op = jqlOpNotIn
		if c.values, err = p.parseList(); err != nil {
			return nil, err
		}
	case t.isKeyword("IS"):
		c.op = jqlOpIsEmpty
		if p.peek().isKeyword("NOT") {
			p.next()
			c.op = jqlOpIsNotEmpty
		}
		if e := p.next(); !e.isKeyword("EMPTY") && !e.isKeyword("NULL") {
			return nil, jqlErrorf(e.pos, "IS の後には EMPTY か NOT EMPTY が必要です")
		}
	case t.kind == tokEOF:
		return nil, jqlErrorf(t.pos, "「%s」の後に演算子（= != ~ > >= < <= IN NOT IN IS）が必要です", field)
	default:
		return nil, jqlErrorf(t.pos, "%sの位置には演算子（= != ~ > >= < <= IN NOT IN IS）が必要です", t.describe())
	}
	return &jqlNode{kind: jqlClauseNode, clause: c}, nil
}

// parseValue は値を 1 つ読む。inList なら IN の括弧の中
func (p *jqlParser) parseValue(inList bool) (jqlValue, error) {
	t := p.next()
	switch t.kind {
	case tokString:
		return jqlValue{kind: jqlValueText, text: t.text, quoted: true, pos: t.pos}, nil
	case tokWord:
		if t.isKeyword("EMPTY") || t.isKeyword("NULL") {
			return jqlValue{kind: jqlValueEmpty, pos: t.pos}, nil
		}
		if isJQLKeyword(t.text) {
			return jqlValue{}, jqlErrorf(t.pos, "%sの位置には値が必要です（キーワードを値にするときは \"…\" で囲んでください）", t.describe())
		}
		// 名前の直後の ( は関数の呼び出し（引数には対応しない）
		if p.peek().kind == tokLParen {
			p.next()
			if closing := p.next(); closing.kind != tokRParen {
				return jqlValue{}, jqlErrorf(closing.pos, "関数 %s() の引数には対応していません", t.text)
			}
			return jqlValue{kind: jqlValueFunc, text: t.text, pos: t.pos}, nil
		}
		return jqlValue{kind: jqlValueText, text: t.text, pos: t.pos}, nil
	case tokLParen:
		if !inList {
			return jqlValue{}, jqlErrorf(t.pos, "複数の値を指定するときは IN を使ってください（例: status IN (Done, Open)）")
		}
	case tokEOF:
		return jqlValue{}, jqlErrorf(t.pos, "値が必要です")
	}
	return jqlValue{}, jqlErrorf(t.pos, "%sの位置には値が必要です", t.describe())
}

func (p *jqlParser) parseList() ([]jqlValue, error) {
	// IN の後の関数は、値の一覧を返すものとして括弧なしでも書ける（Jira と同じ。例: issuekey IN issueHistory()）
	if t := p.peek(); t.kind == tokWord && p.tokens[p.i+1].kind == tokLParen {
		v, err := p.parseValue(false)
		if err != nil {
			return nil, err
		}
		return []jqlValue{v}, nil
	}
	open := p.next()
	if open.kind != tokLParen {
		return nil, jqlErrorf(open.pos, "IN の後には値の一覧が必要です（例: IN (Done, Open)）")
	}
	values := []jqlValue{}
	for {
		v, err := p.parseValue(true)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
		t := p.next()
		if t.kind == tokRParen {
			return values, nil
		}
		if t.kind != tokComma {
			return nil, jqlErrorf(t.pos, "%sの位置には「,」か「)」が必要です（%d 文字目の「(」が閉じられていません）", t.describe(), open.pos+1)
		}
	}
}

func (p *jqlParser) parseOrderBy() ([]jqlOrderItem, error) {
	p.next() // ORDER
	if by := p.next(); !by.isKeyword("BY") {
		return nil, jqlErrorf(by.pos, "ORDER の後には BY が必要です")
	}
	items := []jqlOrderItem{}
	for {
		field, pos, err := p.parseField()
		if err != nil {
			return nil, err
		}
		item := jqlOrderItem{field: field, pos: pos}
		if t := p.peek(); t.isKeyword("ASC") || t.isKeyword("DESC") {
			p.next()
			item.desc = t.isKeyword("DESC")
		}
		items = append(items, item)
		if p.peek().kind != tokComma {
			return items, nil
		}
		p.next()
	}
}
