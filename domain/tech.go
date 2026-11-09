package domain

import "fmt"

// Tech represents a fictional technology evaluated by the SF-Meter.
type Tech struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Score       int    `json:"score"` // 0–100: likelihood of real-world realization
}

// Validate checks domain invariants.
// Score must be between 0 and 100 inclusive.
// Name must not be empty.
func (t Tech) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("tech: name must not be empty")
	}
	if t.Score < 0 || t.Score > 100 {
		return fmt.Errorf("tech: score %d is out of range [0, 100]", t.Score)
	}
	return nil
}

// TechRepository is the persistence interface.
// Infrastructure layer must implement this; domain layer must not know about DB.
type TechRepository interface {
	FindAll() ([]Tech, error)
	FindByID(id int) (*Tech, error)
}
