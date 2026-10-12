package repository

import "todo-api/internal/domain"

type VersionRepository interface {
	// FindAll はすべてのバージョンを、スペース・作成順に返す
	FindAll() ([]domain.Version, error)
	// FindByID はバージョンを返す。存在しなければ nil を返す
	FindByID(id int) (*domain.Version, error)
	// Create はバージョンを作成し、その ID を返す
	Create(version domain.Version) (int, error)
	// Update はバージョンの名前・説明・状態・日付を変える。存在しなければ not found のエラーを返す
	Update(version domain.Version) error
	// Release はバージョンをリリース済みにしてリリース日を releaseDate にする。moveIDs の課題からはこのバージョンを外し、
	// 代わりに moveTo のバージョンを付ける。まとめて 1 つのトランザクションで行う
	Release(id int, releaseDate string, moveIDs []int, moveTo int) error
	// Delete はバージョンと、課題へのその付与を削除する。存在しなければ not found のエラーを返す
	Delete(id int) error
	// SetTodoVersions は課題の修正バージョンを versionIDs に置き換える
	SetTodoVersions(todoID int, versionIDs []int) error
}
