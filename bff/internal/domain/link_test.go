package domain

import "testing"

func TestParseRelation(t *testing.T) {
	tests := []struct {
		relation string
		linkType LinkType
		outward  bool
		ok       bool
	}{
		{"blocks", LinkTypeBlocks, true, true},
		{"is blocked by", LinkTypeBlocks, false, true},
		{"relates to", LinkTypeRelates, true, true},
		{"duplicates", LinkTypeDuplicates, true, true},
		{"is duplicated by", LinkTypeDuplicates, false, true},
		{"clones", LinkTypeClones, true, true},
		{"is cloned by", LinkTypeClones, false, true},
		{"Blocks", "", false, false},
		{"", "", false, false},
	}
	for _, tt := range tests {
		linkType, outward, ok := ParseRelation(tt.relation)
		if linkType != tt.linkType || outward != tt.outward || ok != tt.ok {
			t.Errorf("ParseRelation(%q) = %q, %v, %v", tt.relation, linkType, outward, ok)
		}
	}
}

func TestLink_RelationFromAndSameAs(t *testing.T) {
	blocks := Link{Type: LinkTypeBlocks, SourceID: 1, TargetID: 2}
	if got := blocks.RelationFrom(1); got != "blocks" {
		t.Errorf("RelationFrom(source) = %q", got)
	}
	if got := blocks.RelationFrom(2); got != "is blocked by" {
		t.Errorf("RelationFrom(target) = %q", got)
	}
	if blocks.SameAs(Link{Type: LinkTypeBlocks, SourceID: 2, TargetID: 1}) {
		t.Error("blocks in the opposite direction should be a different link")
	}
	if !blocks.SameAs(Link{Type: LinkTypeBlocks, SourceID: 1, TargetID: 2}) {
		t.Error("same blocks link should be the same")
	}
	relates := Link{Type: LinkTypeRelates, SourceID: 1, TargetID: 2}
	if !relates.SameAs(Link{Type: LinkTypeRelates, SourceID: 2, TargetID: 1}) {
		t.Error("relates to should ignore direction")
	}
	if relates.RelationFrom(2) != "relates to" {
		t.Errorf("relates RelationFrom(target) = %q", relates.RelationFrom(2))
	}
}
