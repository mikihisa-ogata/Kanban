package repository

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"sync"

	"todo-api/internal/domain"
)

const spaceCSVFilePath = "spaces.csv"

type spaceRepository struct {
	mu sync.RWMutex
}

func NewSpaceRepository() SpaceRepository {
	return &spaceRepository{}
}

func (r *spaceRepository) FindAll() ([]domain.Space, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	spaces, err := r.readCSV()
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.Space{}, nil
		}
		return nil, err
	}

	return spaces, nil
}

func (r *spaceRepository) Create(space domain.Space) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	spaces, err := r.readCSV()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	maxID := 0
	for _, e := range spaces {
		if e.ID > maxID {
			maxID = e.ID
		}
	}
	space.ID = maxID + 1

	spaces = append(spaces, space)
	return r.writeCSV(spaces)
}

func (r *spaceRepository) Delete(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	spaces, err := r.readCSV()
	if err != nil {
		return err
	}

	for i, space := range spaces {
		if space.ID == id {
			spaces = append(spaces[:i], spaces[i+1:]...)
			return r.writeCSV(spaces)
		}
	}

	return fmt.Errorf("space with ID %d not found", id)
}

func (r *spaceRepository) readCSV() ([]domain.Space, error) {
	file, err := os.Open(spaceCSVFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	spaces := []domain.Space{}
	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) < 2 {
			continue
		}

		id, _ := strconv.Atoi(record[0])
		spaces = append(spaces, domain.Space{
			ID:    id,
			Title: record[1],
		})
	}

	return spaces, nil
}

func (r *spaceRepository) writeCSV(spaces []domain.Space) error {
	file, err := os.Create(spaceCSVFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"ID", "Title"})

	for _, space := range spaces {
		writer.Write([]string{
			strconv.Itoa(space.ID),
			space.Title,
		})
	}

	return nil
}
