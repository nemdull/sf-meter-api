package infrastructure

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/nemdull/sf-meter-api/domain"
)

// PostgresTechRepo is a PostgreSQL implementation of domain.TechRepository.
type PostgresTechRepo struct {
	db *sql.DB
}

// NewPostgresTechRepo opens a connection and verifies it with Ping.
func NewPostgresTechRepo(dsn string) (*PostgresTechRepo, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to open: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("postgres: failed to connect: %w", err)
	}
	return &PostgresTechRepo{db: db}, nil
}

// FindAll returns all techs.
func (r *PostgresTechRepo) FindAll() ([]domain.Tech, error) {
	rows, err := r.db.Query(
		`SELECT id, name, description, score FROM techs ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("postgres FindAll: %w", err)
	}
	defer rows.Close()

	var techs []domain.Tech
	for rows.Next() {
		var t domain.Tech
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Score); err != nil {
			return nil, fmt.Errorf("postgres FindAll scan: %w", err)
		}
		techs = append(techs, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres FindAll rows: %w", err)
	}
	return techs, nil
}

// FindByID returns the tech with the given ID, or nil if not found.
func (r *PostgresTechRepo) FindByID(id int) (*domain.Tech, error) {
	row := r.db.QueryRow(
		`SELECT id, name, description, score FROM techs WHERE id = $1`,
		id,
	)
	var t domain.Tech
	if err := row.Scan(&t.ID, &t.Name, &t.Description, &t.Score); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres FindByID: %w", err)
	}
	return &t, nil
}

// Create inserts a new tech and returns the assigned ID.
func (r *PostgresTechRepo) Create(t domain.Tech) (int, error) {
	var id int
	err := r.db.QueryRow(
		`INSERT INTO techs (name, description, score) VALUES ($1, $2, $3) RETURNING id`,
		t.Name, t.Description, t.Score,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("postgres Create: %w", err)
	}
	return id, nil
}

// Update modifies an existing tech.
func (r *PostgresTechRepo) Update(t domain.Tech) error {
	result, err := r.db.Exec(
		`UPDATE techs SET name=$1, description=$2, score=$3 WHERE id=$4`,
		t.Name, t.Description, t.Score, t.ID,
	)
	if err != nil {
		return fmt.Errorf("postgres Update: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("postgres Update: tech %d not found", t.ID)
	}
	return nil
}

// Delete removes a tech by ID.
func (r *PostgresTechRepo) Delete(id int) error {
	_, err := r.db.Exec(`DELETE FROM techs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("postgres Delete: %w", err)
	}
	return nil
}
