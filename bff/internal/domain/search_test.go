package domain

import "testing"

func TestIssueSearchNormalize(t *testing.T) {
	q := IssueSearch{}
	if err := q.Normalize(); err != nil {
		t.Fatal(err)
	}
	if q.Sort != IssueSortCreated || !q.Desc || q.Page != 1 || q.PerPage != DefaultIssuePerPage {
		t.Errorf("defaults = %+v", q)
	}
	q = IssueSearch{Sort: IssueSortKey, Page: 3, PerPage: 10}
	if err := q.Normalize(); err != nil || q.Desc || q.Page != 3 || q.PerPage != 10 {
		t.Errorf("explicit = %+v, %v", q, err)
	}
	for _, bad := range []IssueSearch{{Sort: "title"}, {Page: -1}, {PerPage: 101}, {PerPage: -5}} {
		if err := bad.Normalize(); err == nil {
			t.Errorf("Normalize(%+v) should fail", bad)
		}
	}
	if err := (&IssueSearch{Types: []IssueType{"Epic"}}).Normalize(); err != ErrInvalidIssueSearchType {
		t.Errorf("invalid type error = %v", err)
	}
}
