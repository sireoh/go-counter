package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Counter struct {
	ID         int
	Name       string
	Count      int
	LastActive time.Time
}

var (
	db      *sql.DB
	userLoc *time.Location
)

func main() {
	// 1. Load Timezone with safety fallback
	tz := os.Getenv("TIMEZONE")
	if tz == "" {
		tz = "America/Vancouver"
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		log.Printf("Error loading timezone %s: %v. Using UTC.", tz, err)
		userLoc = time.UTC // Never leave this nil
	} else {
		userLoc = loc
	}

	// 2. Database Directory Setup
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/app/data/app.db"
	}
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal("DB Open Error:", err)
	}

	// 3. Schema & Seed
	db.Exec(`CREATE TABLE IF NOT EXISTS counters (
		id INTEGER PRIMARY KEY, 
		name TEXT, 
		count INTEGER DEFAULT 0, 
		last_active DATETIME,
		current_month TEXT
	)`)
	seedData()

	// 4. Routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		Page(getCounters()).Render(r.Context(), w)
	})

	http.HandleFunc("/update/", handleUpdate)

	log.Printf("GOTH App running on :8080 (TZ: %s)", userLoc.String())
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getCounters() []Counter {
	nowStr := time.Now().In(userLoc).Format("01-2006")
	rows, err := db.Query("SELECT id, name, count, last_active FROM counters WHERE current_month = ?", nowStr)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var counters []Counter
	for rows.Next() {
		var c Counter
		rows.Scan(&c.ID, &c.Name, &c.Count, &c.LastActive)
		c.LastActive = c.LastActive.In(userLoc)
		counters = append(counters, c)
	}
	return counters
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	id := r.URL.Path[len("/update/"):]
	dir := r.URL.Query().Get("dir")

	change := 1
	if dir == "down" {
		change = -1
	}

	now := time.Now().In(userLoc)
	db.Exec("UPDATE counters SET count = count + ?, last_active = ? WHERE id = ?", change, now, id)

	var c Counter
	db.QueryRow("SELECT id, name, count, last_active FROM counters WHERE id = ?", id).Scan(&c.ID, &c.Name, &c.Count, &c.LastActive)
	c.LastActive = c.LastActive.In(userLoc)

	Card(c).Render(r.Context(), w)
}

func seedData() {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM counters").Scan(&count)
	if count == 0 {
		now := time.Now().In(userLoc)
		month := now.Format("01-2006")
		db.Exec("INSERT INTO counters (id, name, count, last_active, current_month) VALUES (0, 'vc', 0, ?, ?)", now, month)
		db.Exec("INSERT INTO counters (id, name, count, last_active, current_month) VALUES (1, 'Aries', 0, ?, ?)", now, month)
	}
}
