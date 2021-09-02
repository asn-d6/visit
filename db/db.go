package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
)

const (
	databaseName = "./foo.db" // XXX rename...
)

type Database struct {
	db *sql.DB
}

// Open a database file and return its driver. If the file can't be found make
// a new one
func InitDatabase() *Database {
	// Check if the db needs to be initialized (there is probably a better way)
	need_to_initialize := false
	if _, err := os.Stat(databaseName); os.IsNotExist(err) {
		need_to_initialize = true
	}

	// Open the db file (or make a new one if needed)
	db, err := sql.Open("sqlite3", databaseName)
	if err != nil {
		panic(err)
	}

	if need_to_initialize {
		// Initialize if needed
		_, err = db.Exec("CREATE TABLE validator_state (validator_idx INTEGER, epoch INTEGER, distance INT)")
		if err != nil {
			panic(err)
		}
	}

	return &Database{
		db: db,
	}
}

// Register an attestation by 'validator_idx' at 'epoch'
func (db *Database) RegisterAttestation(validator_idx int, epoch int, distance int) {
	// XXX ewww this db.db thing is dirty
	_, err := db.db.Exec("INSERT INTO validator_state(validator_idx, epoch, distance) VALUES(?, ?, ?)", validator_idx, epoch, distance)
	if err != nil {
		panic(err)
	}
}

// Cursed function XXX
func (db *Database) QueryAttestations() {
	rows, err := db.db.Query("SELECT validator_idx, epoch, distance FROM validator_state")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var epoch int
		var distance int
		err = rows.Scan(&id, &epoch, &distance)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(id, epoch, distance)
	}
}

func (db *Database) Close() {
	db.db.Close()
}
