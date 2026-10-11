package domain

// LinkType は課題リンクの種類（Jira の Blocks / Relates / Duplicates / Clones）
type LinkType string

const (
	LinkTypeBlocks     LinkType = "Blocks"
	LinkTypeRelates    LinkType = "Relates"
	LinkTypeDuplicates LinkType = "Duplicates"
	LinkTypeClones     LinkType = "Clones"
)

// linkRelation はリンクの種類と、リンク元（outward）・リンク先（inward）から見た名前
type linkRelation struct {
	linkType LinkType
	outward  string
	inward   string
}

// linkRelations はリンクの種類を表示順に並べたもの。Relates は向きがないので両側とも同じ名前になる
var linkRelations = []linkRelation{
	{LinkTypeBlocks, "blocks", "is blocked by"},
	{LinkTypeRelates, "relates to", "relates to"},
	{LinkTypeDuplicates, "duplicates", "is duplicated by"},
	{LinkTypeClones, "clones", "is cloned by"},
}

// RelationBlockedBy は「ブロックされている」側から見たリンクの名前
const RelationBlockedBy = "is blocked by"

// Link は保存されている課題リンク。SourceID の課題が TargetID の課題を Type する（例: Source blocks Target）
type Link struct {
	ID       int
	Type     LinkType
	SourceID int
	TargetID int
}

// IssueLink はある課題から見たリンク。API の出力専用
type IssueLink struct {
	ID       int
	Relation string      // この課題から見た名前（例: "is blocked by"）
	Issue    LinkedIssue // リンク相手の課題
}

// LinkedIssue はリンク相手の課題の概要
type LinkedIssue struct {
	ID     int
	Key    string
	Title  string
	Status Status // 互換用（Todo.Status と同じ）
	// StatusName・StatusCategory は相手の課題のステータスの名前とカテゴリ
	StatusName     string
	StatusCategory StatusCategory
	IssueType      IssueType
	Priority       Priority
}

// ParseRelation は課題から見たリンクの名前（例: "is blocked by"）を、リンクの種類と向きに変える。
// outward が true ならその課題がリンク元（Source）になる。Relates は常に outward
func ParseRelation(relation string) (linkType LinkType, outward bool, ok bool) {
	for _, r := range linkRelations {
		if relation == r.outward {
			return r.linkType, true, true
		}
		if relation == r.inward {
			return r.linkType, false, true
		}
	}
	return "", false, false
}

// RelationFrom は todoID の課題から見たリンクの名前を返す
func (l Link) RelationFrom(todoID int) string {
	for _, r := range linkRelations {
		if r.linkType != l.Type {
			continue
		}
		if l.SourceID == todoID {
			return r.outward
		}
		return r.inward
	}
	return string(l.Type)
}

// SameAs は 2 つのリンクが同じ課題の組を同じ意味でつなぐかを返す。Relates は向きを区別しない
func (l Link) SameAs(other Link) bool {
	if l.Type != other.Type {
		return false
	}
	if l.SourceID == other.SourceID && l.TargetID == other.TargetID {
		return true
	}
	return l.Type == LinkTypeRelates && l.SourceID == other.TargetID && l.TargetID == other.SourceID
}

// RelationOrder は表示順でのリンクの名前の位置を返す（outward の名前を inward の名前より先にする）
func RelationOrder(relation string) int {
	for i, r := range linkRelations {
		if relation == r.outward {
			return i * 2
		}
		if relation == r.inward {
			return i*2 + 1
		}
	}
	return len(linkRelations) * 2
}
