package repository

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"sync"

	"todo-api/internal/domain"
)

const epicCSVFilePath = "epics.csv"

type epicRepository struct {
	mu sync.RWMutex
}

func NewEpicRepository() EpicRepository {
	return &epicRepository{}
}

func (r *epicRepository) FindAll() ([]domain.Epic, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	epics, err := r.readCSV()
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.Epic{}, nil
		}
		return nil, err
	}

	return epics, nil
}

func (r *epicRepository) Create(epic domain.Epic) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	epics, err := r.readCSV()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	maxID := 0
	for _, e := range epics {
		if e.ID > maxID {
			maxID = e.ID
		}
	}
	epic.ID = maxID + 1

	epics = append(epics, epic)
	return r.writeCSV(epics)
}

func (r *epicRepository) Delete(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	epics, err := r.readCSV()
	if err != nil {
		return err
	}

	for i, epic := range epics {
		if epic.ID == id {
			epics = append(epics[:i], epics[i+1:]...)
			return r.writeCSV(epics)
		}
	}

	return fmt.Errorf("epic with ID %d not found", id)
}

func (r *epicRepository) readCSV() ([]domain.Epic, error) {
	file, err := os.Open(epicCSVFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	epics := []domain.Epic{}
	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) < 2 {
			continue
		}

		id, _ := strconv.Atoi(record[0])
		epics = append(epics, domain.Epic{
			ID:    id,
			Title: record[1],
		})
	}

	return epics, nil
}

func (r *epicRepository) writeCSV(epics []domain.Epic) error {
	file, err := os.Create(epicCSVFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"ID", "Title"})

	for _, epic := range epics {
		writer.Write([]string{
			strconv.Itoa(epic.ID),
			epic.Title,
		})
	}

	return nil
}
