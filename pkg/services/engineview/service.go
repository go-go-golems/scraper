package engineview

import (
	"context"

	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

type Service struct {
	engineDB string
}

func NewService(engineDB string) *Service {
	return &Service{engineDB: engineDB}
}

func (s *Service) EngineStatus(ctx context.Context) (*sqlitestore.EngineStatus, error) {
	return sqlitestore.Inspect(ctx, s.engineDB)
}
