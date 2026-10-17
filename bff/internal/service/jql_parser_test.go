package service

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// formatJQL は構文木を読みやすい文字列にする（テスト用）。値は種類ごとに 語・"文字列"・関数()・EMPTY と書く
func formatJQL(q *jqlQuery) string {
	var b strings.Builder
	if q.where != nil {
		b.WriteString(formatJQLNode(q.where))
	}
	for i, o := range q.order {
		if i == 0 {
			if b.Len() > 0 {
				b.WriteString(" ")
			}
			b.WriteString("ORDER BY ")
		} else {
			b.WriteString(", ")
		}
		b.WriteString(o.field)
		if o.desc {
			b.WriteString(" DESC")
		}
	}
	return b.String()
}

func formatJQLNode(n *jqlNode) string {
	switch n.kind {
	case jqlAnd, jqlOr:
		op := map[jqlNodeKind]string{jqlAnd: "AND", jqlOr: "OR"}[n.kind]
		return fmt.Sprintf("(%s %s %s)", formatJQLNode(n.children[0]), op, formatJQLNode(n.children[1]))
	case jqlNot:
		return fmt.Sprintf("NOT %s", formatJQLNode(n.children[0]))
	}
	c := n.clause
	values := []string{}
	for _, v := range c.values {
		switch {
		case v.kind == jqlValueFunc:
			values = append(values, v.text+"()")
		case v.kind == jqlValueEmpty:
			values = append(values, "EMPTY")
		case v.quoted:
			values = append(values, `"`+v.text+`"`)
		default:
			values = append(values, v.text)
		}
	}
	switch c.op {
	case jqlOpIn, jqlOpNotIn:
		return fmt.Sprintf("[%s %s (%s)]", c.field, c.op, strings.Join(values, ", "))
	case jqlOpIsEmpty, jqlOpIsNotEmpty:
		return fmt.Sprintf("[%s %s]", c.field, c.op)
	}
	return fmt.Sprintf("[%s %s %s]", c.field, c.op, values[0])
}

func TestParseJQL(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{"", ""},
		{"   ", ""},
		{"status = Done", "[status = Done]"},
		{"project = KAN AND status != Done AND assignee = currentUser() ORDER BY priority DESC",
			"(([project = KAN] AND [status != Done]) AND [assignee = currentUser()]) ORDER BY priority DESC"},
		// キーワードは大文字・小文字を区別しない
		{"status = Done and type = Bug or not priority = High", "(([status = Done] AND [type = Bug]) OR NOT [priority = High])"},
		// AND は OR より強く結び付き、括弧で変えられる
		{"a = 1 OR b = 2 AND c = 3", "([a = 1] OR ([b = 2] AND [c = 3]))"},
		{"(a = 1 OR b = 2) AND c = 3", "(([a = 1] OR [b = 2]) AND [c = 3])"},
		{"NOT (a = 1 OR b = 2)", "NOT ([a = 1] OR [b = 2])"},
		{"NOT NOT a = 1", "NOT NOT [a = 1]"},
		{"((a = 1))", "[a = 1]"},
		// 演算子
		{"a != 1", "[a != 1]"},
		{"text ~ ログイン", "[text ~ ログイン]"},
		{"created > -7d", "[created > -7d]"},
		{"created >= 2026-10-01", "[created >= 2026-10-01]"},
		{"created<now()", "[created < now()]"},
		{"due <= \"2026-10-01\"", `[due <= "2026-10-01"]`},
		{"status IN (Done, \"In Progress\", 'To Do')", `[status in (Done, "In Progress", "To Do")]`},
		{"status not in (Done)", "[status not in (Done)]"},
		{"assignee IN (EMPTY, currentUser())", "[assignee in (EMPTY, currentUser())]"},
		{"issuekey IN issueHistory()", "[issuekey in (issueHistory())]"},
		{"sprint not in openSprints() AND a = 1", "([sprint not in (openSprints())] AND [a = 1])"},
		{"assignee IS EMPTY", "[assignee is empty]"},
		{"assignee is not null", "[assignee is not empty]"},
		// = EMPTY・!= EMPTY は IS EMPTY・IS NOT EMPTY と同じ
		{"due = EMPTY", "[due is empty]"},
		{"due != null", "[due is not empty]"},
		// 記号の前後の空白は要らない
		{"status=Done AND(type=Bug)", "([status = Done] AND [type = Bug])"},
		// 引用符の中のエスケープと、キーワードを値にする
		{`text ~ "say \"hi\""`, `[text ~ "say "hi""]`},
		{`status = "AND"`, `[status = "AND"]`},
		{`"epic link" = 3`, "[epic link = 3]"},
		{"key = KAN-12", "[key = KAN-12]"},
		// ORDER BY だけ・複数の列
		{"ORDER BY created", "ORDER BY created"},
		{"status = Done order by priority desc, created asc, key", "[status = Done] ORDER BY priority DESC, created, key"},
	}
	for _, tt := range tests {
		q, err := parseJQL(tt.query)
		if err != nil {
			t.Errorf("parseJQL(%q) error = %v", tt.query, err)
			continue
		}
		if got := formatJQL(q); got != tt.want {
			t.Errorf("parseJQL(%q) = %s, want %s", tt.query, got, tt.want)
		}
	}
}

func TestParseJQL_Errors(t *testing.T) {
	tests := []struct {
		query string
		pos   int    // 誤りの位置（0 から）
		msg   string // メッセージに含まれる文字列
	}{
		{"status", 6, "演算子"},
		{"status Done", 7, "演算子"},
		{"status =", 8, "値が必要"},
		{"status = Done AND", 17, "項目名"},
		{"status = Done OR", 16, "項目名"},
		{"AND status = Done", 0, "項目名"},
		{"status = Done type = Bug", 14, "AND か OR"},
		{"(status = Done", 14, "「)」が必要"},
		{"status = Done)", 13, "「(」がない「)」"},
		{"()", 1, "項目名"},
		{"status = \"Done", 9, "閉じられていません"},
		{"status = 'Done", 9, "閉じられていません"},
		{"status ! Done", 7, "「=」が必要"},
		{"status IN Done", 10, "値の一覧"},
		{"issuekey IN issueHistory(1)", 25, "引数"},
		{"status IN ()", 11, "値が必要"},
		{"status IN (Done", 15, "「,」か「)」"},
		{"status IN (Done Open)", 16, "「,」か「)」"},
		{"status IN (Done,)", 16, "値が必要"},
		{"status NOT Done", 11, "NOT の後には IN"},
		{"status IS Done", 10, "EMPTY"},
		{"status IS NOT", 13, "EMPTY"},
		{"status = (Done)", 9, "IN を使って"},
		{"status = AND", 9, "キーワード"},
		{"due > EMPTY", 6, "EMPTY は"},
		{"assignee = currentUser(1)", 23, "引数"},
		{"status = = Done", 9, "値が必要"},
		{"ORDER created", 6, "BY"},
		{"ORDER BY", 8, "項目名"},
		{"ORDER BY created,", 17, "項目名"},
		{"ORDER BY created DESC status", 22, "AND か OR"},
		// 位置はバイトではなく文字で数える
		{"text ~ 日本語 type", 11, "AND か OR"},
	}
	for _, tt := range tests {
		_, err := parseJQL(tt.query)
		var jqlErr *JQLError
		if !errors.As(err, &jqlErr) {
			t.Errorf("parseJQL(%q) error = %v, want *JQLError", tt.query, err)
			continue
		}
		if jqlErr.Pos != tt.pos || !strings.Contains(jqlErr.Msg, tt.msg) {
			t.Errorf("parseJQL(%q) error = %d: %s, want %d: …%s…", tt.query, jqlErr.Pos, jqlErr.Msg, tt.pos, tt.msg)
		}
	}
	// メッセージには 1 から数えた位置が付く
	_, err := parseJQL("status")
	if err == nil || !strings.HasPrefix(err.Error(), "7 文字目: ") {
		t.Errorf("error message = %v", err)
	}
}
