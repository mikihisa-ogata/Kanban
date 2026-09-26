package repository

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"sync"

	"todo-api/internal/domain"
)

const csvFilePath = "todos.csv"

type todoRepository struct {
	mu sync.RWMutex
}

func NewTodoRepository() TodoRepository {
	return &todoRepository{}
}

func (r *todoRepository) FindAll() ([]domain.Todo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	file, err := os.Open(csvFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.Todo{}, nil
		}
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	todos := []domain.Todo{}
	for i, record := range records {
		if i == 0 {
			continue
		}
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

		todo := domain.Todo{
			ID:          id,
			Title:       record[1],
			Done:        done,
			Deadline:    record[3],
			Status:      status,
			EpicID:      epicID,
			Description: description,
			SpaceID:     spaceID,
		}
		todos = append(todos, todo)
	}

	return todos, nil
}

func (r *todoRepository) Create(todo domain.Todo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	todos, err := r.readCSV()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	maxID := 0
	for _, t := range todos {
		if t.ID > maxID {
			maxID = t.ID
		}
	}
	todo.ID = maxID + 1

	todos = append(todos, todo)
	return r.writeCSV(todos)
}

func (r *todoRepository) Update(id int, todo domain.Todo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	todos, err := r.readCSV()
	if err != nil {
		return err
	}

	for i, t := range todos {
		if t.ID == id {
			todo.ID = id
			todos[i] = todo
			return r.writeCSV(todos)
		}
	}

	return fmt.Errorf("todo with ID %d not found", id)
}

func (r *todoRepository) Delete(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	todos, err := r.readCSV()
	if err != nil {
		return err
	}

	for i, todo := range todos {
		if todo.ID == id {
			todos = append(todos[:i], todos[i+1:]...)
			return r.writeCSV(todos)
		}
	}

	return fmt.Errorf("todo with ID %d not found", id)
}

// Move は CSV の行を並べ替える。タスクの表示順は CSV の行の順番に従う
func (r *todoRepository) Move(id int, beforeID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	todos, err := r.readCSV()
	if err != nil {
		return err
	}

	index := -1
	for i, t := range todos {
		if t.ID == id {
			index = i
			break
		}
	}
	if index == -1 {
		return fmt.Errorf("todo with ID %d not found", id)
	}
	if beforeID == id {
		return nil
	}

	moved := todos[index]
	rest := append(todos[:index:index], todos[index+1:]...)

	insertAt := len(rest)
	if beforeID != 0 {
		insertAt = -1
		for i, t := range rest {
			if t.ID == beforeID {
				insertAt = i
				break
			}
		}
		if insertAt == -1 {
			return fmt.Errorf("todo with ID %d not found", beforeID)
		}
	}

	reordered := make([]domain.Todo, 0, len(todos))
	reordered = append(reordered, rest[:insertAt]...)
	reordered = append(reordered, moved)
	reordered = append(reordered, rest[insertAt:]...)
	return r.writeCSV(reordered)
}

func (r *todoRepository) readCSV() ([]domain.Todo, error) {
	file, err := os.Open(csvFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	todos := []domain.Todo{}
	for i, record := range records {
		if i == 0 {
			continue
		}
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

		todo := domain.Todo{
			ID:          id,
			Title:       record[1],
			Done:        done,
			Deadline:    record[3],
			Status:      status,
			EpicID:      epicID,
			Description: description,
			SpaceID:     spaceID,
		}
		todos = append(todos, todo)
	}

	return todos, nil
}

func (r *todoRepository) writeCSV(todos []domain.Todo) error {
	file, err := os.Create(csvFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"ID", "Title", "Done", "Deadline", "Status", "EpicID", "Description", "SpaceID"})

	for _, todo := range todos {
		writer.Write([]string{
			strconv.Itoa(todo.ID),
			todo.Title,
			strconv.FormatBool(todo.Done),
			todo.Deadline,
			string(todo.Status),
			strconv.Itoa(todo.EpicID),
			todo.Description,
			strconv.Itoa(todo.SpaceID),
		})
	}

	return nil
}
