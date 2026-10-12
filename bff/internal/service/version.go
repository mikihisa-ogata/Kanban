package service

import (
	"errors"
	"sort"
	"strings"
	"time"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var (
	ErrVersionNotFound        = errors.New("指定されたバージョンが存在しません")
	ErrUnassignedVersion      = errors.New("スペース未割り当ての課題にはバージョンを作れません")
	ErrVersionNameDuplicate   = errors.New("同じ名前のバージョンが既にあります")
	ErrVersionAlreadyReleased = errors.New("リリースできるのは未リリースのバージョンだけです")
	ErrInvalidVersionMoveTo   = errors.New("未完了の課題の移し先は、同じスペースの別の未リリースのバージョンを選んでください")
	ErrVersionOtherSpace      = errors.New("課題と同じスペースのバージョンを選んでください")
	ErrVersionArchived        = errors.New("アーカイブしたバージョンは課題に付けられません")
	ErrVersionTodoNotFound    = errors.New("指定された課題が存在しません")
)

type VersionService interface {
	// GetVersions はすべてのバージョンを、スペース・作成順に返す
	GetVersions() ([]domain.Version, error)
	// CreateVersion はスペースに未リリースのバージョンを作る
	CreateVersion(spaceID int, name, description, startDate, releaseDate string) (domain.Version, error)
	// UpdateVersion はバージョンの名前・説明・日付・状態を変える
	UpdateVersion(id int, name, description, startDate, releaseDate string, status string) (domain.Version, error)
	// ReleaseVersion は未リリースのバージョンをリリース済みにする。releaseDate が空なら、設定済みのリリース日か今日にする。
	// moveTo（同じスペースの別の未リリースのバージョン）が 0 でなければ、完了していない課題をこのバージョンからそちらへ付け替える。
	// 0 なら完了していない課題もこのバージョンのまま残す（Jira の「無視してリリース」）
	ReleaseVersion(id int, releaseDate string, moveTo int) (domain.Version, error)
	// DeleteVersion はバージョンを削除する。課題からも外れる
	DeleteVersion(id int) error
	// SetTodoVersions は課題の修正バージョンを versionIDs（課題と同じスペースのバージョン）に置き換える
	SetTodoVersions(todoID int, versionIDs []int) error
}

type versionService struct {
	repo      repository.VersionRepository
	todoRepo  repository.TodoRepository
	spaceRepo repository.SpaceRepository
}

func NewVersionService(r repository.VersionRepository, t repository.TodoRepository, s repository.SpaceRepository) VersionService {
	return &versionService{repo: r, todoRepo: t, spaceRepo: s}
}

func (s *versionService) GetVersions() ([]domain.Version, error) {
	return s.repo.FindAll()
}

func (s *versionService) CreateVersion(spaceID int, name, description, startDate, releaseDate string) (domain.Version, error) {
	if spaceID == 0 {
		return domain.Version{}, ErrUnassignedVersion
	}
	exists, err := spaceExists(s.spaceRepo, spaceID)
	if err != nil {
		return domain.Version{}, err
	}
	if !exists {
		return domain.Version{}, ErrSpaceNotFound
	}
	version := domain.Version{SpaceID: spaceID, Status: domain.VersionUnreleased}
	if err := s.applyFields(&version, name, description, startDate, releaseDate); err != nil {
		return domain.Version{}, err
	}
	id, err := s.repo.Create(version)
	if err != nil {
		return domain.Version{}, err
	}
	return s.find(id)
}

func (s *versionService) UpdateVersion(id int, name, description, startDate, releaseDate string, status string) (domain.Version, error) {
	version, err := s.find(id)
	if err != nil {
		return domain.Version{}, err
	}
	if !domain.ValidVersionStatus(domain.VersionStatus(status)) {
		return domain.Version{}, domain.ErrInvalidVersionStatus
	}
	if err := s.applyFields(&version, name, description, startDate, releaseDate); err != nil {
		return domain.Version{}, err
	}
	version.Status = domain.VersionStatus(status)
	if err := s.repo.Update(version); err != nil {
		return domain.Version{}, err
	}
	return s.find(id)
}

func (s *versionService) ReleaseVersion(id int, releaseDate string, moveTo int) (domain.Version, error) {
	version, err := s.find(id)
	if err != nil {
		return domain.Version{}, err
	}
	if version.Status != domain.VersionUnreleased {
		return domain.Version{}, ErrVersionAlreadyReleased
	}
	if releaseDate == "" {
		releaseDate = version.ReleaseDate
	}
	if releaseDate == "" {
		releaseDate = time.Now().Format("2006-01-02")
	}
	if err := domain.ValidateVersionDates(version.StartDate, releaseDate); err != nil {
		return domain.Version{}, err
	}
	if moveTo != 0 {
		target, err := s.repo.FindByID(moveTo)
		if err != nil {
			return domain.Version{}, err
		}
		if target == nil || target.ID == id || target.SpaceID != version.SpaceID || target.Status != domain.VersionUnreleased {
			return domain.Version{}, ErrInvalidVersionMoveTo
		}
	}

	todos, err := s.todoRepo.FindAll()
	if err != nil {
		return domain.Version{}, err
	}
	moveIDs := []int{}
	for _, todo := range todos {
		if moveTo != 0 && !todo.Done && containsInt(todo.FixVersionIDs, id) {
			moveIDs = append(moveIDs, todo.ID)
		}
	}
	if err := s.repo.Release(id, releaseDate, moveIDs, moveTo); err != nil {
		return domain.Version{}, err
	}
	return s.find(id)
}

func (s *versionService) DeleteVersion(id int) error {
	if _, err := s.find(id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func (s *versionService) SetTodoVersions(todoID int, versionIDs []int) error {
	todos, err := s.todoRepo.FindAll()
	if err != nil {
		return err
	}
	var todo *domain.Todo
	for i := range todos {
		if todos[i].ID == todoID {
			todo = &todos[i]
		}
	}
	if todo == nil {
		return ErrVersionTodoNotFound
	}
	ids := []int{}
	for _, id := range versionIDs {
		if containsInt(ids, id) {
			continue
		}
		version, err := s.find(id)
		if err != nil {
			return err
		}
		if version.SpaceID != todo.SpaceID {
			return ErrVersionOtherSpace
		}
		// 付いたままのアーカイブ済みのバージョンは残せるが、新しくは付けられない
		if version.Status == domain.VersionArchived && !containsInt(todo.FixVersionIDs, id) {
			return ErrVersionArchived
		}
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return s.repo.SetTodoVersions(todoID, ids)
}

func (s *versionService) find(id int) (domain.Version, error) {
	version, err := s.repo.FindByID(id)
	if err != nil {
		return domain.Version{}, err
	}
	if version == nil {
		return domain.Version{}, ErrVersionNotFound
	}
	return *version, nil
}

// applyFields は名前・説明・日付を検証して version に入れる。同じスペースに同じ名前（大文字・小文字は区別しない）があれば ErrVersionNameDuplicate
func (s *versionService) applyFields(version *domain.Version, name, description, startDate, releaseDate string) error {
	name, err := domain.NormalizeVersionName(name)
	if err != nil {
		return err
	}
	description, err = domain.NormalizeVersionDescription(description)
	if err != nil {
		return err
	}
	if err := domain.ValidateVersionDates(startDate, releaseDate); err != nil {
		return err
	}
	versions, err := s.repo.FindAll()
	if err != nil {
		return err
	}
	for _, v := range versions {
		if v.SpaceID == version.SpaceID && v.ID != version.ID && strings.EqualFold(v.Name, name) {
			return ErrVersionNameDuplicate
		}
	}
	version.Name, version.Description, version.StartDate, version.ReleaseDate = name, description, startDate, releaseDate
	return nil
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
