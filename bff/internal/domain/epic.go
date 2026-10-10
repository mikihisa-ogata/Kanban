package domain

import "time"

type Epic struct {
	ID        int
	Title     string
	SpaceID   int       // 0 の場合はスペース未割り当て
	CreatedAt time.Time // 作成日時（UTC）
	UpdatedAt time.Time // 最終更新日時（UTC）
}
