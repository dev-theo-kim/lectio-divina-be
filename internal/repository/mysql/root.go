// Package mysql provides the MySQL database for the service.
package mysql

import (
	"github.com/dev-theo-kim/lectio-divina-be/internal/lib/log"
	"github.com/dev-theo-kim/lectio-divina-be/internal/service/config"
	"github.com/upper/db/v4"
)

type Mysql struct {
	LectioDivina *LectioDivina
	log          log.Logger
}

func New(cfg config.Config) *Mysql {
	s := &Mysql{
		log: log.New("module", "mysql"),
	}

	db.LC().SetLevel(db.LogLevelPanic)

	s.LectioDivina = NewLectioDivina(cfg.MySQL["lectio_divina"])

	return s
}
