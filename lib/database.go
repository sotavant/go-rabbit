package lib

import (
	"database/sql"
)

type Database struct {
	Config database
}

func (d *Database) Open(common Common) (*sql.DB) {
	user := d.Config.User
	pass := d.Config.Pass
	database := d.Config.Database

	db, err := sql.Open("mysql", user + ":" + pass + "@/" + database)
	common.FailOnError(err, "Opening connection with database")

	return db
}