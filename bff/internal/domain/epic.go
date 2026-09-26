package domain

type Epic struct {
	ID      int
	Title   string
	SpaceID int // 0 の場合はスペース未割り当て
}
