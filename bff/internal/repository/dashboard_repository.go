package repository

import "todo-api/internal/domain"

type DashboardRepository interface {
	// FindByUser はユーザーが保存したガジェットの並びを返す。まだ保存していなければ nil を返す
	FindByUser(userID int) ([]domain.DashboardGadget, error)
	// Save はユーザーのガジェットの並びを保存する（前に保存したものは置き換える）
	Save(userID int, gadgets []domain.DashboardGadget) error
}
