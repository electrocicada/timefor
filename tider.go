package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const timeoutForProlonging = 600

func main() {
	newCmd := flag.NewFlagSet("new", flag.ExitOnError)
	newShift := newCmd.String("shift", "", "start time shift like 10m, 1h10m, etc.")
	updateCmd := flag.NewFlagSet("update", flag.ExitOnError)
	updateFinish := updateCmd.Bool("finish", false, "finish current activity")
	usage := fmt.Sprintf("expected %#v or %#v sub-command", newCmd.Name(), updateCmd.Name())

	if len(os.Args) < 2 {
		log.Fatalln(usage)
	}

	switch os.Args[1] {
	case newCmd.Name():
		_ = newCmd.Parse(os.Args[2:])
		if len(newCmd.Args()) < 1 {
			log.Fatalln("expected not empty name argument")
		}
		New(newCmd.Args()[0], *newShift)
	case updateCmd.Name():
		_ = updateCmd.Parse(os.Args[2:])
		Update(*updateFinish)
	default:
		log.Fatalln(usage)
	}
}

func connectDb() *sqlx.DB {
	db, err := sqlx.Open("sqlite3", "log.db")
	if err != nil {
		log.Fatalf("cannot open SQLite database: %v", err)
	}

	var exists bool
	err = db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type="table" AND name="log"`).Scan(&exists)
	if err != nil {
		log.Fatal(err)
	} else if !exists {
		initDb(db)
	}
	return db
}

func initDb(db *sqlx.DB) {
	_, err := db.Exec(`
		CREATE TABLE log(
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			started INTEGER NOT NULL,
			elapsed INTEGER NOT NULL DEFAULT 0,
			updated INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			UNIQUE (name, started)
		);

		CREATE TRIGGER on_insert_started INSERT ON log
		FOR EACH ROW
		BEGIN
			SELECT RAISE(ABORT, 'started must be latest')
			WHERE NEW.started <= (SELECT MAX(started + elapsed) FROM log);
		END;

		CREATE TRIGGER on_update_updated UPDATE ON log
		FOR EACH ROW
		BEGIN
			UPDATE log SET updated=strftime('%s', 'now') WHERE id = NEW.id;
		END;

		CREATE VIEW log_pretty AS
		SELECT
			id,
			name,
			date(started, 'unixepoch', 'localtime') start_date,
			time(started, 'unixepoch', 'localtime') start_time,
			elapsed,
			elapsed / 60 elapsed_minutes,
			datetime(updated, 'unixepoch', 'localtime') updated_ts
		FROM log;
	`)
	if err != nil {
		log.Fatalf("cannot initiate SQLite database: %v", err)
	}
}

// New activity with name and optional time shift
func New(name string, shift string) {
	db := connectDb()
	defer db.Close()

	shiftSeconds := 0
	if shift != "" {
		shiftDuration, err := time.ParseDuration(shift)
		if err != nil {
			log.Fatalf("wrong shift format: %v", err)
		}
		shiftSeconds = int(shiftDuration.Seconds())
	}
	_, err := db.NamedExec(`
		INSERT INTO log (name, started) VALUES (:name, strftime('%s', 'now') - :shiftSeconds)
	`, map[string]interface{}{
		"name":         name,
		"shiftSeconds": shiftSeconds,
	})
	if err != nil {
		log.Fatalf("cannot insert new activity into database: %v", err)
	}
}

// Update or finish current activity
func Update(finish bool) {
	db := connectDb()
	defer db.Close()

	res, err := db.NamedExec(`
		WITH current AS (
			SELECT id
			FROM log
			WHERE strftime('%s', 'now') - updated < :timeoutForProlonging AND elapsed = 0
			ORDER BY id DESC
			LIMIT 1
		)
		UPDATE log SET
			elapsed=(CASE WHEN :shouldBeFinished THEN strftime('%s', 'now') - started ELSE 0 END),
			updated=strftime('%s', 'now')
		WHERE id IN (SELECT id FROM current)
	`, map[string]interface{}{
		"timeoutForProlonging": timeoutForProlonging,
		"shouldBeFinished":     finish,
	})
	if err != nil {
		log.Fatalf("cannot update current activity: %v", err)
	}
	rowCnt, err := res.RowsAffected()
	if err != nil {
		log.Fatalf("cannot update current activity: %v", err)
	}
	if rowCnt == 0 {
		log.Fatalf("no current activity")
	}
}
