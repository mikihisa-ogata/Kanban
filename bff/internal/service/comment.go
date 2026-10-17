package service

import (
	"errors"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var ErrCommentNotFound = errors.New("コメントが見つかりません")

var ErrCommentForbidden = errors.New("ほかのユーザーのコメントは編集・削除できません")

type CommentService interface {
	// GetComments は課題のコメントを投稿順（古い順）に返す。課題がなければ ErrIssueNotFound を返す
	GetComments(todoID int) ([]domain.Comment, error)
	// CreateComment は課題に authorID のユーザーとしてコメントを投稿し、作成したコメントを返す。課題がなければ ErrIssueNotFound を返す。
	// 投稿者は自動でウォッチャーになる。メンションされたユーザーには「メンションされました」、ほかのウォッチャーには「コメントしました」と通知する
	CreateComment(todoID int, body string, authorID int) (domain.Comment, error)
	// UpdateComment はコメントの本文を変え、変更後のコメントを返す。なければ ErrCommentNotFound、
	// actorID のユーザーが投稿者でなければ ErrCommentForbidden を返す（投稿者のないコメントは誰でも編集できる）。
	// 編集で新しくメンションしたユーザーには、actorID のユーザーからの通知を作る
	UpdateComment(id int, body string, actorID int) (domain.Comment, error)
	// DeleteComment はコメントを削除する。なければ ErrCommentNotFound、
	// actorID のユーザーが投稿者でなければ ErrCommentForbidden を返す（投稿者のないコメントは誰でも削除できる）
	DeleteComment(id int, actorID int) error
}

type commentService struct {
	commentRepo   repository.CommentRepository
	todoRepo      repository.TodoRepository
	watchers      repository.WatcherRepository
	notifications repository.NotificationRepository
	users         repository.UserRepository
}

func NewCommentService(c repository.CommentRepository, t repository.TodoRepository, w repository.WatcherRepository, n repository.NotificationRepository, u repository.UserRepository) CommentService {
	return &commentService{commentRepo: c, todoRepo: t, watchers: w, notifications: n, users: u}
}

func (s *commentService) GetComments(todoID int) ([]domain.Comment, error) {
	if err := requireTodo(s.todoRepo, todoID); err != nil {
		return nil, err
	}
	return s.commentRepo.FindByTodo(todoID)
}

func (s *commentService) CreateComment(todoID int, body string, authorID int) (domain.Comment, error) {
	if err := domain.ValidateCommentBody(body); err != nil {
		return domain.Comment{}, err
	}
	if err := requireTodo(s.todoRepo, todoID); err != nil {
		return domain.Comment{}, err
	}
	id, err := s.commentRepo.Create(domain.Comment{TodoID: todoID, Body: body, AuthorID: authorID})
	if err != nil {
		return domain.Comment{}, err
	}
	if err := s.watchers.Add(todoID, authorID); err != nil {
		return domain.Comment{}, err
	}
	// メンションされたウォッチャーには、同じコメントの通知を重ねて送らない
	mentioned, err := notifyMentions(s.users, s.notifications, todoID, authorID, domain.MentionedUserIDs(body), domain.Excerpt(body))
	if err != nil {
		return domain.Comment{}, err
	}
	if err := notifyWatchers(s.watchers, s.notifications, todoID, authorID, domain.NotificationCommented, domain.Excerpt(body), mentioned); err != nil {
		return domain.Comment{}, err
	}
	return s.findComment(id)
}

func (s *commentService) UpdateComment(id int, body string, actorID int) (domain.Comment, error) {
	if err := domain.ValidateCommentBody(body); err != nil {
		return domain.Comment{}, err
	}
	current, err := s.findComment(id)
	if err != nil {
		return domain.Comment{}, err
	}
	if !current.CanBeChangedBy(actorID) {
		return domain.Comment{}, ErrCommentForbidden
	}
	if err := s.commentRepo.Update(id, body); err != nil {
		return domain.Comment{}, err
	}
	newly := domain.NewlyMentionedUserIDs(current.Body, body)
	if _, err := notifyMentions(s.users, s.notifications, current.TodoID, actorID, newly, domain.Excerpt(body)); err != nil {
		return domain.Comment{}, err
	}
	return s.findComment(id)
}

func (s *commentService) DeleteComment(id int, actorID int) error {
	current, err := s.findComment(id)
	if err != nil {
		return err
	}
	if !current.CanBeChangedBy(actorID) {
		return ErrCommentForbidden
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
