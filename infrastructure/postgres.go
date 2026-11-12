package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	"github.com/nemdull/sf-meter-api/domain"
	"github.com/nemdull/sf-meter-api/usecase"
)

// PostgresTechRepo は、PostgreSQL に保存する usecase.TechRepository の実装。
type PostgresTechRepo struct {
	db *sql.DB
}

// NewPostgresTechRepo は接続を開き、Ping でつながることを確かめてから返す。
func NewPostgresTechRepo(ctx context.Context, dsn string) (*PostgresTechRepo, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to open: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres: failed to connect: %w", err)
	}
	return &PostgresTechRepo{db: db}, nil
}

// Ping は DB につながるかを確かめる。/health が使う。
func (r *PostgresTechRepo) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

// Close は接続を閉じる。
func (r *PostgresTechRepo) Close() error {
	return r.db.Close()
}

// orderBy は並び順を SQL にする。値をそのまま埋め込まず、決めた 4 通りだけを返す。
// 名前順は、メモリ実装（文字コード順）と結果をそろえるために COLLATE "C" を付けている。
func orderBy(key usecase.SortKey) string {
	switch key {
	case usecase.SortByScoreAsc:
		return "score ASC, id ASC"
	case usecase.SortByScoreDesc:
		return "score DESC, id ASC"
	case usecase.SortByName:
		return `name COLLATE "C" ASC, id ASC`
	default:
		return "id ASC"
	}
}

// likePattern は、キーワードを「含む」検索の形にする。% と _ は文字として探せるようにエスケープする。
func likePattern(keyword string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(keyword)
	return "%" + escaped + "%"
}

// Find は条件に合う技術と、絞り込み後の総件数を返す。
func (r *PostgresTechRepo) Find(ctx context.Context, query usecase.TechQuery) ([]domain.Tech, int, error) {
	where := "score BETWEEN $1 AND $2"
	args := []any{query.MinScore, query.MaxScore}
	if query.Keyword != "" {
		args = append(args, likePattern(query.Keyword))
		where += fmt.Sprintf(` AND (name ILIKE $%d ESCAPE '\' OR description ILIKE $%d ESCAPE '\')`, len(args), len(args))
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM techs WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres Find count: %w", err)
	}

	statement := "SELECT id, name, description, score FROM techs WHERE " + where + " ORDER BY " + orderBy(query.Sort)
	if query.Limit > 0 {
		args = append(args, query.Limit)
		statement += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if query.Offset > 0 {
		args = append(args, query.Offset)
		statement += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := r.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres Find: %w", err)
	}
	defer rows.Close()

	techs := make([]domain.Tech, 0)
	for rows.Next() {
		var tech domain.Tech
		if err := rows.Scan(&tech.ID, &tech.Name, &tech.Description, &tech.Score); err != nil {
			return nil, 0, fmt.Errorf("postgres Find scan: %w", err)
		}
		techs = append(techs, tech)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres Find rows: %w", err)
	}
	return techs, total, nil
}

// FindByID は指定 ID の技術を返す。無ければ domain.ErrNotFound。
func (r *PostgresTechRepo) FindByID(ctx context.Context, id int) (domain.Tech, error) {
	var tech domain.Tech
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, description, score FROM techs WHERE id = $1`, id,
	).Scan(&tech.ID, &tech.Name, &tech.Description, &tech.Score)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Tech{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Tech{}, fmt.Errorf("postgres FindByID: %w", err)
	}
	return tech, nil
}

// Create は新しい行を足し、割り当てられた ID を入れて返す。
func (r *PostgresTechRepo) Create(ctx context.Context, tech domain.Tech) (domain.Tech, error) {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO techs (name, description, score) VALUES ($1, $2, $3) RETURNING id`,
		tech.Name, tech.Description, tech.Score,
	).Scan(&tech.ID)
	if err != nil {
		return domain.Tech{}, fmt.Errorf("postgres Create: %w", err)
	}
	return tech, nil
}

// Update は tech.ID の行を置き換える。無ければ domain.ErrNotFound。
func (r *PostgresTechRepo) Update(ctx context.Context, tech domain.Tech) (domain.Tech, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE techs SET name = $1, description = $2, score = $3, updated_at = NOW() WHERE id = $4`,
		tech.Name, tech.Description, tech.Score, tech.ID,
	)
	if err != nil {
		return domain.Tech{}, fmt.Errorf("postgres Update: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return domain.Tech{}, fmt.Errorf("postgres Update rows: %w", err)
	}
	if affected == 0 {
		return domain.Tech{}, domain.ErrNotFound
	}
	return tech, nil
}

// Delete は指定 ID の行を消す。無ければ domain.ErrNotFound。
func (r *PostgresTechRepo) Delete(ctx context.Context, id int) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM techs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("postgres Delete: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres Delete rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Reset は、テーブルを空にして ID の採番を 1 に戻し、渡した技術を順に入れ直す。
// 結合テストが毎回同じ状態から始められるようにするためのもので、アプリ本体からは呼ばない。
func (r *PostgresTechRepo) Reset(ctx context.Context, seeds ...domain.Tech) error {
	if _, err := r.db.ExecContext(ctx, `TRUNCATE TABLE techs RESTART IDENTITY`); err != nil {
		return fmt.Errorf("postgres Reset: %w", err)
	}
	for _, tech := range seeds {
		if _, err := r.Create(ctx, tech); err != nil {
			return err
		}
	}
	return nil
}
