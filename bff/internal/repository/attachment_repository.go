package repository

import (
	"io"

	"todo-api/internal/domain"
)

type AttachmentRepository interface {
	// FindByTodo は課題の添付ファイルを添付順（古い順）に返す
	FindByTodo(todoID int) ([]domain.Attachment, error)
	// FindByID は添付ファイルの情報を返す。存在しなければ nil を返す
	FindByID(id int) (*domain.Attachment, error)
	// Create は content を保存先のディレクトリにファイルとして保存し、情報を記録する。
	// 保存名・サイズ・添付日時はここで決める。作成した添付ファイルの ID を返す
	Create(attachment domain.Attachment, content io.Reader) (int, error)
	// Open はファイル本体を読むために開く。呼び出し側で Close する
	Open(attachment domain.Attachment) (io.ReadCloser, error)
	// Delete は添付ファイルの情報とファイル本体を削除する。存在しなければ not found のエラーを返す
	Delete(id int) error
	// RemoveFiles は添付ファイルの本体だけを削除する（情報は課題の削除で消したあと）。もうないファイルは無視する
	RemoveFiles(attachments []domain.Attachment) error
}
