package repository

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"todo-api/internal/domain"
)

// ImportResult は CSV から取り込んだ件数
type ImportResult struct {
	Todos  int
	Epics  int
	Spaces int
}

// ImportCSV は dir の todos.csv・epics.csv・spaces.csv を MySQL に取り込む（CSV 保存からの移行用）。
// ID とタスクの並び順（CSV の行の順番）はそのまま保つ。ファイルがなければ 0 件として扱う。
// 二重に取り込まないよう、取り込み先のテーブルが1つでも空でなければ何もせずにエラーを返す。
// CSV ファイルは読むだけで変更しない
func ImportCSV(db *sql.DB, dir string) (ImportResult, error) {
	todos, err := readTodosCSV(filepath.Join(dir, "todos.csv"))
	if err != nil {
		return ImportResult{}, err
	}
	epics, err := readEpicsCSV(filepath.Join(dir, "epics.csv"))
	if err != nil {
		return ImportResult{}, err
	}
	spaces, err := readSpacesCSV(filepath.Join(dir, "spaces.csv"))
	if err != nil {
		return ImportResult{}, err
	}

	tx, err := db.Begin()
	if err != nil {
		return ImportResult{}, err
	}
	defer tx.Rollback()

	for _, table := range []string{"todos", "epics", "spaces"} {
		var count int
		if err := tx.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			return ImportResult{}, err
		}
		if count > 0 {
			return ImportResult{}, fmt.Errorf("table %s is not empty (%d rows): already imported?", table, count)
		}
	}

	for _, space := range spaces {
		if _, err := tx.Exec(`INSERT INTO spaces (id, title) VALUES (?, ?)`, space.ID, space.Title); err != nil {
			return ImportResult{}, fmt.Errorf("failed to import space %d: %w", space.ID, err)
		}
	}
	for _, epic := range epics {
		if _, err := tx.Exec(`INSERT INTO epics (id, title, space_id) VALUES (?, ?, ?)`,
			epic.ID, epic.Title, epic.SpaceID); err != nil {
			return ImportResult{}, fmt.Errorf("failed to import epic %d: %w", epic.ID, err)
		}
	}
	for i, todo := range todos {
		deadline, err := toNullDeadline(todo.Deadline)
		if err != nil {
			return ImportResult{}, fmt.Errorf("failed to import todo %d: %w", todo.ID, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO todos (id, title, done, deadline, status, epic_id, description, space_id, position)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			todo.ID, todo.Title, todo.Done, deadline, string(todo.Status),
			todo.EpicID, todo.Description, todo.SpaceID, i+1); err != nil {
			return ImportResult{}, fmt.Errorf("failed to import todo %d: %w", todo.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return ImportResult{}, err
	}
	return ImportResult{Todos: len(todos), Epics: len(epics), Spaces: len(spaces)}, nil
}

// readCSVRecords は CSV のヘッダー行を除いた行を返す。ファイルがなければ空を返す
func readCSVRecords(path string) ([][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// 旧形式のファイルは行ごとに列数が異なることがあるため、列数を検査しない
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	if len(records) == 0 {
		return nil, nil
	}
	return records[1:], nil
}

// readTodosCSV は todos.csv を読む。Status・EpicID・Description・SpaceID の列がない旧形式にも対応する
func readTodosCSV(path string) ([]domain.Todo, error) {
	records, err := readCSVRecords(path)
	if err != nil {
		return nil, err
	}

	todos := []domain.Todo{}
	for _, record := range records {
		if len(record) < 4 {
			continue
		}

		id, _ := strconv.Atoi(record[0])
		done, _ := strconv.ParseBool(record[2])

		status := domain.StatusOpen
		if len(record) >= 5 {
			status = domain.Status(record[4])
		}

		epicID := 0
		if len(record) >= 6 {
			epicID, _ = strconv.Atoi(record[5])
		}

		description := ""
		if len(record) >= 7 {
			description = record[6]
		}

		spaceID := 0
		if len(record) >= 8 {
			spaceID, _ = strconv.Atoi(record[7])
		}

		todos = append(todos, domain.Todo{
			ID:          id,
			Title:       record[1],
			Done:        done,
			Deadline:    record[3],
			Status:      status,
			EpicID:      epicID,
			Description: description,
			SpaceID:     spaceID,
		})
	}
	return todos, nil
}

// readEpicsCSV は epics.csv を読む。SpaceID の列がない旧形式にも対応する
func readEpicsCSV(path string) ([]domain.Epic, error) {
	records, err := readCSVRecords(path)
	if err != nil {
		return nil, err
	}

	epics := []domain.Epic{}
	for _, record := range records {
		if len(record) < 2 {
			continue
		}

		id, _ := strconv.Atoi(record[0])
		spaceID := 0
		if len(record) >= 3 {
			spaceID, _ = strconv.Atoi(record[2])
		}
		epics = append(epics, domain.Epic{ID: id, Title: record[1], SpaceID: spaceID})
	}
	return epics, nil
}

// readSpacesCSV は spaces.csv を読む
func readSpacesCSV(path string) ([]domain.Space, error) {
	records, err := readCSVRecords(path)
	if err != nil {
		return nil, err
	}

	spaces := []domain.Space{}
	for _, record := range records {
		if len(record) < 2 {
			continue
		}

		id, _ := strconv.Atoi(record[0])
		spaces = append(spaces, domain.Space{ID: id, Title: record[1]})
	}
	return spaces, nil
}
