package mysql

import (
	"github.com/dev-theo-kim/lectio-divina-be/internal/lib/log"
	"github.com/dev-theo-kim/lectio-divina-be/internal/service/config"
	"github.com/upper/db/v4"
	"github.com/upper/db/v4/adapter/mysql"
)

type LectioDivina struct {
	sess db.Session
	log  log.Logger
}

func NewLectioDivina(cfg config.MySQL) *LectioDivina {
	c := &LectioDivina{
		log: log.New("module", "lectio_divina"),
	}

	connURL := mysql.ConnectionURL{
		Host:     cfg.Host,
		Database: cfg.DB,
		User:     cfg.User,
		Password: cfg.Pass,
	}

	var err error
	if c.sess, err = mysql.Open(connURL); err != nil {
		c.log.Crit("Failed to open MySQL connection", "error", err)
	}

	return c
}
