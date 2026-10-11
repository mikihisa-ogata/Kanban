package service

import (
	"errors"
	"fmt"
	"strings"

	"todo-api/internal/domain"
)

var ErrStatusNameDuplicate = errors.New("同じ名前のステータスが既にあります")

var ErrInvalidStatusCategory = errors.New("カテゴリは ToDo・InProgress・Done のいずれかにしてください")

var ErrUnassignedWorkflow = errors.New("スペース未割り当ての課題のワークフローは変更できません")

var ErrStatusInUse = errors.New("このステータスの課題があります。移し先のステータスを選んでください")

var ErrInvalidMoveTarget = errors.New("移し先は、同じスペースの別のステータスを選んでください")

var ErrLastStatus = errors.New("ワークフローの最後のステータスは削除できません")

var ErrInvalidStatusOrder = errors.New("並び順には、スペースのステータスをすべて 1 回ずつ指定してください")

// CreateStatus はスペースのワークフローの末尾にステータスを追加し、追加したステータスを返す
func (s *spaceService) CreateStatus(spaceID int, name string, category string) (domain.WorkflowStatus, error) {
	statuses, err := s.editableStatuses(spaceID)
	if err != nil {
		return domain.WorkflowStatus{}, err
	}
	name, err = domain.NormalizeStatusName(name)
	if err != nil {
		return domain.WorkflowStatus{}, err
	}
	if !domain.ValidStatusCategory(domain.StatusCategory(category)) {
		return domain.WorkflowStatus{}, ErrInvalidStatusCategory
	}
	if hasStatusName(statuses, name, 0) {
		return domain.WorkflowStatus{}, ErrStatusNameDuplicate
	}
	id, err := s.statuses.repo.Create(domain.WorkflowStatus{SpaceID: spaceID, Name: name, Category: domain.StatusCategory(category)})
	if err != nil {
		return domain.WorkflowStatus{}, err
	}
	return s.findStatus(id)
}

// RenameStatus はステータスの名前を変える。カテゴリは変えられない（課題の完了の扱いが変わるため）
func (s *spaceService) RenameStatus(id int, name string) (domain.WorkflowStatus, error) {
	status, err := s.findStatus(id)
	if err != nil {
		return domain.WorkflowStatus{}, err
	}
	statuses, err := s.editableStatuses(status.SpaceID)
	if err != nil {
		return domain.WorkflowStatus{}, err
	}
	name, err = domain.NormalizeStatusName(name)
	if err != nil {
		return domain.WorkflowStatus{}, err
	}
	if hasStatusName(statuses, name, id) {
		return domain.WorkflowStatus{}, ErrStatusNameDuplicate
	}
	if err := s.statuses.repo.Rename(id, name); err != nil {
		return domain.WorkflowStatus{}, err
	}
	return s.findStatus(id)
}

// ReorderStatuses はスペースのステータスを ids の順（ボードの列の順）に並べ直す
func (s *spaceService) ReorderStatuses(spaceID int, ids []int) error {
	statuses, err := s.editableStatuses(spaceID)
	if err != nil {
		return err
	}
	if len(ids) != len(statuses) {
		return ErrInvalidStatusOrder
	}
	seen := map[int]bool{}
	for _, id := range ids {
		if seen[id] || !containsStatus(statuses, id) {
			return ErrInvalidStatusOrder
		}
		seen[id] = true
	}
	return s.statuses.repo.Reorder(spaceID, ids)
}

// DeleteStatus はステータスを削除する。そのステータスの課題があれば moveToID のステータス（同じスペースの別のステータス）へ移してから消す。
// 課題があるのに moveToID が 0 なら ErrStatusInUse を返す
func (s *spaceService) DeleteStatus(id int, moveToID int) error {
	status, err := s.findStatus(id)
	if err != nil {
		return err
	}
	statuses, err := s.editableStatuses(status.SpaceID)
	if err != nil {
		return err
	}
	if len(statuses) <= 1 {
		return ErrLastStatus
	}
	todos, err := s.todoRepo.FindAll()
	if err != nil {
		return err
	}
	using := []domain.Todo{}
	for _, todo := range todos {
		if todo.StatusID == id {
			using = append(using, todo)
		}
	}
	if len(using) > 0 {
		if moveToID == 0 {
			return ErrStatusInUse
		}
		if moveToID == id || !containsStatus(statuses, moveToID) {
			return ErrInvalidMoveTarget
		}
		target, err := s.findStatus(moveToID)
		if err != nil {
			return err
		}
		names, err := s.history.names()
		if err != nil {
			return err
		}
		for _, todo := range using {
			moved := todo
			applyStatus(&moved, target)
			if err := updateWithHistory(s.todoRepo, s.history, s.statuses, names, todo, moved); err != nil {
				return fmt.Errorf("failed to move todo %d to status %d: %w", todo.ID, moveToID, err)
			}
		}
	}
	return s.statuses.repo.Delete(id)
}

// editableStatuses は変更できるワークフロー（存在するスペースのもの）のステータスを並び順に返す
func (s *spaceService) editableStatuses(spaceID int) ([]domain.WorkflowStatus, error) {
	if spaceID == 0 {
		return nil, ErrUnassignedWorkflow
	}
	exists, err := spaceExists(s.spaceRepo, spaceID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrSpaceNotFound
	}
	return s.statuses.statusesOf(spaceID)
}

func (s *spaceService) findStatus(id int) (domain.WorkflowStatus, error) {
	status, err := s.statuses.find(id)
	if err != nil {
		return domain.WorkflowStatus{}, err
	}
	if status == nil {
		return domain.WorkflowStatus{}, ErrStatusNotFound
	}
	return *status, nil
}

// hasStatusName は exceptID 以外に同じ名前（大文字・小文字は区別しない）のステータスがあるかを返す
func hasStatusName(statuses []domain.WorkflowStatus, name string, exceptID int) bool {
	for _, s := range statuses {
		if s.ID != exceptID && strings.EqualFold(s.Name, name) {
			return true
		}
	}
	return false
}

func containsStatus(statuses []domain.WorkflowStatus, id int) bool {
	for _, s := range statuses {
		if s.ID == id {
			return true
		}
	}
	return false
}
