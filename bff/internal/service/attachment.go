package service

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var ErrAttachmentNotFound = errors.New("添付ファイルが見つかりません")

var ErrAttachmentTooLarge = fmt.Errorf("添付ファイルは 1 つ %dMB までです", domain.AttachmentMaxSize>>20)

var ErrAttachmentEmpty = errors.New("空のファイルは添付できません")

type AttachmentService interface {
	// GetAttachments は課題の添付ファイルを添付順に返す。課題がなければ ErrIssueNotFound を返す
	GetAttachments(todoID int) ([]domain.Attachment, error)
	// AddAttachment は課題にファイルを添付し、添付ファイルの情報を返す。size は content のバイト数。
	// 課題がなければ ErrIssueNotFound、上限を超えれば ErrAttachmentTooLarge を返す
	AddAttachment(todoID int, fileName string, size int64, content io.Reader) (domain.Attachment, error)
	// OpenAttachment は添付ファイルの情報と、ファイル本体を返す。呼び出し側で Close する
	OpenAttachment(id int) (domain.Attachment, io.ReadCloser, error)
	// DeleteAttachment は添付ファイルを削除する。なければ ErrAttachmentNotFound を返す
	DeleteAttachment(id int) error
}

type attachmentService struct {
	attachmentRepo repository.AttachmentRepository
	todoRepo       repository.TodoRepository
}

func NewAttachmentService(a repository.AttachmentRepository, t repository.TodoRepository) AttachmentService {
	return &attachmentService{attachmentRepo: a, todoRepo: t}
}

func (s *attachmentService) GetAttachments(todoID int) ([]domain.Attachment, error) {
	if err := requireTodo(s.todoRepo, todoID); err != nil {
		return nil, err
	}
	return s.attachmentRepo.FindByTodo(todoID)
}

func (s *attachmentService) AddAttachment(todoID int, fileName string, size int64, content io.Reader) (domain.Attachment, error) {
	if size > domain.AttachmentMaxSize {
		return domain.Attachment{}, ErrAttachmentTooLarge
	}
	if size <= 0 {
		return domain.Attachment{}, ErrAttachmentEmpty
	}
	if err := requireTodo(s.todoRepo, todoID); err != nil {
		return domain.Attachment{}, err
	}

	// 種類はクライアントの申告ではなく中身の先頭から判定する（画像として表示してよいかの判断に使うため）
	head := make([]byte, 512)
	n, err := io.ReadFull(content, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return domain.Attachment{}, err
	}
	head = head[:n]
	attachment := domain.Attachment{
		TodoID:      todoID,
		FileName:    domain.NormalizeAttachmentName(fileName),
		ContentType: http.DetectContentType(head),
	}
	// 上限を超えた分は保存しない（size の申告より実際の中身が大きい場合に備える）
	body := io.LimitReader(io.MultiReader(bytes.NewReader(head), content), domain.AttachmentMaxSize)
	id, err := s.attachmentRepo.Create(attachment, body)
	if err != nil {
		return domain.Attachment{}, err
	}
	return s.findAttachment(id)
}

func (s *attachmentService) OpenAttachment(id int) (domain.Attachment, io.ReadCloser, error) {
	attachment, err := s.findAttachment(id)
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	file, err := s.attachmentRepo.Open(attachment)
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	return attachment, file, nil
}

func (s *attachmentService) DeleteAttachment(id int) error {
	if _, err := s.findAttachment(id); err != nil {
		return err
	}
	return s.attachmentRepo.Delete(id)
}

func (s *attachmentService) findAttachment(id int) (domain.Attachment, error) {
	attachment, err := s.attachmentRepo.FindByID(id)
	if err != nil {
		return domain.Attachment{}, err
	}
	if attachment == nil {
		return domain.Attachment{}, ErrAttachmentNotFound
	}
	return *attachment, nil
}

// requireTodo は課題が存在することを確かめる。なければ ErrIssueNotFound を返す
func requireTodo(repo repository.TodoRepository, todoID int) error {
	todos, err := repo.FindAll()
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
