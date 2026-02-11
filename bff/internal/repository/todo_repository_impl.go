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

	var todos []domain.Todo
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

		todo := domain.Todo{
			ID:       id,
			Title:    record[1],
			Done:     done,
			Deadline: record[3],
			Status:   status,
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

	var todos []domain.Todo
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

		todo := domain.Todo{
			ID:       id,
			Title:    record[1],
			Done:     done,
			Deadline: record[3],
			Status:   status,
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

	writer.Write([]string{"ID", "Title", "Done", "Deadline", "Status"})

	for _, todo := range todos {
		writer.Write([]string{
			strconv.Itoa(todo.ID),
			todo.Title,
			strconv.FormatBool(todo.Done),
			todo.Deadline,
			string(todo.Status),
		})
	}

	return nil
}
