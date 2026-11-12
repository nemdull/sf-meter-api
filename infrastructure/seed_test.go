package infrastructure_test

import (
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/nemdull/sf-meter-api/domain"
	"github.com/nemdull/sf-meter-api/infrastructure"
)

// db/init.sql の INSERT の 1 行分: ('名前', '説明', 数値)
var seedRow = regexp.MustCompile(`\(\s*'([^']*)',\s*'([^']*)',\s*(\d+)\s*\)`)

// メモリ実装の初期データ（seed.go）と、PostgreSQL の初期データ（db/init.sql）が同じであること。
// 片方だけ直して、保存先によって中身が違う、という状態になるのを止める
func TestSeedTechs_MatchesInitSQL(t *testing.T) {
	sql, err := os.ReadFile("../db/init.sql")
	if err != nil {
		t.Fatal(err)
	}

	var fromSQL []domain.Tech
	for _, match := range seedRow.FindAllStringSubmatch(string(sql), -1) {
		score, err := strconv.Atoi(match[3])
		if err != nil {
			t.Fatal(err)
		}
		fromSQL = append(fromSQL, domain.Tech{Name: match[1], Description: match[2], Score: score})
	}

	seeds := infrastructure.SeedTechs()
	if len(fromSQL) != len(seeds) {
		t.Fatalf("件数が違う: init.sql は %d 件、seed.go は %d 件", len(fromSQL), len(seeds))
	}
	for i := range seeds {
		if seeds[i] != fromSQL[i] {
			t.Errorf("%d 件目が違う:\n seed.go  %+v\n init.sql %+v", i+1, seeds[i], fromSQL[i])
		}
	}
}

func TestSeedTechs_AllValid(t *testing.T) {
	for _, tech := range infrastructure.SeedTechs() {
		if err := tech.Validate(); err != nil {
			t.Errorf("%s: %v", tech.Name, err)
		}
	}
}
