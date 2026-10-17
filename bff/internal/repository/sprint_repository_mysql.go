package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"todo-api/internal/domain"
)

type sprintMySQLRepository struct {
	db *sql.DB
}

func NewSprintMySQLRepository(db *sql.DB) SprintRepository {
	return &sprintMySQLRepository{db: db}
}

// selectSprints はスプリントを読む SELECT。完了したときに記録した件数とポイントも読む（scanSprint と列を合わせる）
const selectSprints = `SELECT s.id, s.space_id, s.name, s.goal, s.state, s.start_date, s.end_date, s.started_at, s.completed_at, s.created_at,
		        (SELECT COUNT(*) FROM sprint_issues i WHERE i.sprint_id = s.id AND i.completed),
		        (SELECT COUNT(*) FROM sprint_issues i WHERE i.sprint_id = s.id AND NOT i.completed),
		        (SELECT COALESCE(SUM(i.story_points), 0) FROM sprint_issues i WHERE i.sprint_id = s.id AND i.completed)
		 FROM sprints s`

func (r *sprintMySQLRepository) FindAll() ([]domain.Sprint, error) {
	rows, err := r.db.Query(selectSprints + ` ORDER BY s.space_id, s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sprints := []domain.Sprint{}
	for rows.Next() {
		sprint, err := scanSprint(rows)
		if err != nil {
			return nil, err
		}
		sprints = append(sprints, sprint)
	}
	return sprints, rows.Err()
}

func (r *sprintMySQLRepository) FindByID(id int) (*domain.Sprint, error) {
	sprint, err := scanSprint(r.db.QueryRow(selectSprints+` WHERE s.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sprint, nil
}

func scanSprint(row interface{ Scan(dest ...any) error }) (domain.Sprint, error) {
	var s domain.Sprint
	var state string
	var start, end, startedAt, completedAt sql.NullTime
	if err := row.Scan(&s.ID, &s.SpaceID, &s.Name, &s.Goal, &state, &start, &end, &startedAt, &completedAt, &s.CreatedAt,
		&s.CompletedIssues, &s.IncompleteIssues, &s.CompletedPoints); err != nil {
		return domain.Sprint{}, err
	}
	s.State = domain.SprintState(state)
	if start.Valid {
		s.StartDate = start.Time.Format(deadlineLayout)
	}
	if end.Valid {
		s.EndDate = end.Time.Format(deadlineLayout)
	}
	if startedAt.Valid {
		t := startedAt.Time
		s.StartedAt = &t
	}
	if completedAt.Valid {
		t := completedAt.Time
		s.CompletedAt = &t
	}
	return s, nil
}

func (r *sprintMySQLRepository) Create(sprint domain.Sprint) (int, error) {
	start, err := toNullDeadline(sprint.StartDate)
	if err != nil {
		return 0, err
	}
	end, err := toNullDeadline(sprint.EndDate)
	if err != nil {
		return 0, err
	}
	result, err := r.db.Exec(
		`INSERT INTO sprints (space_id, name, goal, state, start_date, end_date, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sprint.SpaceID, sprint.Name, sprint.Goal, string(domain.SprintStateFuture), start, end, currentTime())
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *sprintMySQLRepository) Update(sprint domain.Sprint) error {
	start, err := toNullDeadline(sprint.StartDate)
	if err != nil {
		return err
	}
	end, err := toNullDeadline(sprint.EndDate)
	if err != nil {
		return err
	}
	result, err := r.db.Exec(
		`UPDATE sprints SET name = ?, goal = ?, state = ?, start_date = ?, end_date = ?, started_at = IF(? AND started_at IS NULL, ?, started_at) WHERE id = ?`,
		sprint.Name, sprint.Goal, string(sprint.State), start, end, sprint.State == domain.SprintStateActive, currentTime(), sprint.ID)
	if err != nil {
		return err
	}
	return requireAffected(result, "sprint", sprint.ID)
}

func (r *sprintMySQLRepository) Complete(id int, issues []domain.SprintIssue, moveIDs []int, moveTo int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, issue := range issues {
		if _, err := tx.Exec(
			`INSERT INTO sprint_issues (sprint_id, todo_id, story_points, completed) VALUES (?, ?, ?, ?)`,
			id, issue.TodoID, issue.StoryPoints, issue.Completed); err != nil {
			return err
		}
	}
	for _, todoID := range moveIDs {
		if _, err := tx.Exec(`UPDATE todos SET sprint_id = ? WHERE id = ? OR parent_id = ?`, moveTo, todoID, todoID); err != nil {
			return err
		}
	}
	result, err := tx.Exec(`UPDATE sprints SET state = ?, completed_at = ? WHERE id = ?`, string(domain.SprintStateClosed), currentTime(), id)
	if err != nil {
		return err
	}
	if err := requireAffected(result, "sprint", id); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *sprintMySQLRepository) Delete(id int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE todos SET sprint_id = 0 WHERE sprint_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM sprint_issues WHERE sprint_id = ?`, id); err != nil {
		return err
	}
	result, err := tx.Exec(`DELETE FROM sprints WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if err := requireAffected(result, "sprint", id); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *sprintMySQLRepository) SetTodoSprint(todoID int, sprintID int) error {
	result, err := r.db.Exec(`UPDATE todos SET sprint_id = ? WHERE id = ? OR parent_id = ?`, sprintID, todoID, todoID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("todo with ID %d not found", todoID)
	}
	return nil
}

func (r *sprintMySQLRepository) FindIssues(sprintID int) ([]domain.SprintIssue, error) {
	rows, err := r.db.Query(
		`SELECT sprint_id, todo_id, story_points, completed FROM sprint_issues WHERE sprint_id = ? ORDER BY todo_id`, sprintID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	issues := []domain.SprintIssue{}
	for rows.Next() {
		var issue domain.SprintIssue
		var points sql.NullFloat64
		if err := rows.Scan(&issue.SprintID, &issue.TodoID, &points, &issue.Completed); err != nil {
			return nil, err
		}
		if points.Valid {
			issue.StoryPoints = &points.Float64
		}
		issues = append(issues, issue)
	}
	return issues, rows.Err()
}
