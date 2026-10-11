package service

import (
	"errors"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var ErrCommentNotFound = errors.New("コメントが見つかりません")

type CommentService interface {
	// GetComments は課題のコメントを投稿順（古い順）に返す。課題がなければ ErrIssueNotFound を返す
	GetComments(todoID int) ([]domain.Comment, error)
	// CreateComment は課題にコメントを投稿し、作成したコメントを返す。課題がなければ ErrIssueNotFound を返す
	CreateComment(todoID int, body string) (domain.Comment, error)
	// UpdateComment はコメントの本文を変え、変更後のコメントを返す。なければ ErrCommentNotFound を返す
	UpdateComment(id int, body string) (domain.Comment, error)
	// DeleteComment はコメントを削除する。なければ ErrCommentNotFound を返す
	DeleteComment(id int) error
}

type commentService struct {
	commentRepo repository.CommentRepository
	todoRepo    repository.TodoRepository
}

func NewCommentService(c repository.CommentRepository, t repository.TodoRepository) CommentService {
	return &commentService{commentRepo: c, todoRepo: t}
}

func (s *commentService) GetComments(todoID int) ([]domain.Comment, error) {
	if err := s.requireTodo(todoID); err != nil {
		return nil, err
	}
	return s.commentRepo.FindByTodo(todoID)
}

func (s *commentService) CreateComment(todoID int, body string) (domain.Comment, error) {
	if err := domain.ValidateCommentBody(body); err != nil {
		return domain.Comment{}, err
	}
	if err := s.requireTodo(todoID); err != nil {
		return domain.Comment{}, err
	}
	id, err := s.commentRepo.Create(domain.Comment{TodoID: todoID, Body: body})
	if err != nil {
		return domain.Comment{}, err
	}
	return s.findComment(id)
}

func (s *commentService) UpdateComment(id int, body string) (domain.Comment, error) {
	if err := domain.ValidateCommentBody(body); err != nil {
		return domain.Comment{}, err
	}
	if _, err := s.findComment(id); err != nil {
		return domain.Comment{}, err
	}
	if err := s.commentRepo.Update(id, body); err != nil {
		return domain.Comment{}, err
	}
	return s.findComment(id)
}

func (s *commentService) DeleteComment(id int) error {
	if _, err := s.findComment(id); err != nil {
		return err
	}
	return s.commentRepo.Delete(id)
}

// findComment はコメントを返す。なければ ErrCommentNotFound を返す
func (s *commentService) findComment(id int) (domain.Comment, error) {
	comment, err := s.commentRepo.FindByID(id)
	if err != nil {
		return domain.Comment{}, err
	}
	if comment == nil {
		return domain.Comment{}, ErrCommentNotFound
	}
	return *comment, nil
}

// requireTodo は課題が存在することを確かめる。なければ ErrIssueNotFound を返す
func (s *commentService) requireTodo(todoID int) error {
	todos, err := s.todoRepo.FindAll()
	if err != nil {
		return err
	}
	for _, todo := range todos {
		if todo.ID == todoID {
			return nil
		}
	}
	return ErrIssueNotFound
}
