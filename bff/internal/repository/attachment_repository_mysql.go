package repository

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"todo-api/internal/domain"
)

// attachmentMySQLRepository は添付ファイルの情報を MySQL に、ファイル本体を dir に保存する
type attachmentMySQLRepository struct {
	db  *sql.DB
	dir string
}

func NewAttachmentMySQLRepository(db *sql.DB, dir string) AttachmentRepository {
	return &attachmentMySQLRepository{db: db, dir: dir}
}

const selectAttachments = `SELECT id, todo_id, file_name, content_type, size, storage_name, created_at FROM attachments`

func (r *attachmentMySQLRepository) FindByTodo(todoID int) ([]domain.Attachment, error) {
	rows, err := r.db.Query(selectAttachments+` WHERE todo_id = ? ORDER BY id`, todoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attachments := []domain.Attachment{}
	for rows.Next() {
		attachment, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, attachment)
	}
	return attachments, rows.Err()
}

func (r *attachmentMySQLRepository) FindByID(id int) (*domain.Attachment, error) {
	attachment, err := scanAttachment(r.db.QueryRow(selectAttachments+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

func scanAttachment(row interface{ Scan(dest ...any) error }) (domain.Attachment, error) {
	var a domain.Attachment
	err := row.Scan(&a.ID, &a.TodoID, &a.FileName, &a.ContentType, &a.Size, &a.StorageName, &a.CreatedAt)
	return a, err
}

// Create はファイルを一時ファイルに書いてから保存名へ移し、情報を記録する。記録に失敗したらファイルを消す
func (r *attachmentMySQLRepository) Create(attachment domain.Attachment, content io.Reader) (int, error) {
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		return 0, fmt.Errorf("failed to create attachments directory: %w", err)
	}
	storageName, err := randomStorageName()
	if err != nil {
		return 0, err
	}
	tmp, err := os.CreateTemp(r.dir, ".upload-*")
	if err != nil {
		return 0, err
	}
	size, err := io.Copy(tmp, content)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(tmp.Name())
		return 0, err
	}
	path := filepath.Join(r.dir, storageName)
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return 0, err
	}

	result, err := r.db.Exec(
		`INSERT INTO attachments (todo_id, file_name, content_type, size, storage_name, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		attachment.TodoID, attachment.FileName, attachment.ContentType, size, storageName, currentTime())
	if err != nil {
		os.Remove(path)
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *attachmentMySQLRepository) Open(attachment domain.Attachment) (io.ReadCloser, error) {
	return os.Open(filepath.Join(r.dir, attachment.StorageName))
}

// Delete は情報を消してからファイル本体を消す。ファイルが既にない場合はそのまま成功にする
func (r *attachmentMySQLRepository) Delete(id int) error {
	attachment, err := r.FindByID(id)
	if err != nil {
		return err
	}
	if attachment == nil {
		return fmt.Errorf("attachment with ID %d not found", id)
	}
	result, err := r.db.Exec(`DELETE FROM attachments WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if err := requireAffected(result, "attachment", id); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(r.dir, attachment.StorageName)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (r *attachmentMySQLRepository) RemoveFiles(attachments []domain.Attachment) error {
	for _, attachment := range attachments {
		if err := os.Remove(filepath.Join(r.dir, attachment.StorageName)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

// randomStorageName は推測できない保存名（32 文字の 16 進数）を返す
func randomStorageName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
