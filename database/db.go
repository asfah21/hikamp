package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

// DB is the global database connection
var DB *sql.DB

// Init initializes the database connection and creates tables
func Init() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		host := getEnv("DB_HOST", "localhost")
		port := getEnv("DB_PORT", "5432")
		user := getEnv("DB_USER", "postgres")
		password := getEnv("DB_PASSWORD", "postgres")
		dbname := getEnv("DB_NAME", "hikamp")

		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbname)
	}

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Printf("Warning: Database not available: %v", err)
		log.Println("Running with limited functionality")
		return
	}

	log.Println("Database connected successfully")
	createTables()
}

// createTables creates all required tables
func createTables() {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(100) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			name VARCHAR(200),
			email VARCHAR(200),
			role VARCHAR(50) DEFAULT 'admin',
			enabled BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS devices (
			id SERIAL PRIMARY KEY,
			name VARCHAR(200) NOT NULL,
			ip_address VARCHAR(100) NOT NULL,
			port INTEGER DEFAULT 80,
			username VARCHAR(100),
			password VARCHAR(255),
			location VARCHAR(200),
			status VARCHAR(50) DEFAULT 'offline',
			firmware VARCHAR(100),
			last_sync TIMESTAMP,
			enabled BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS audio_files (
			id SERIAL PRIMARY KEY,
			name VARCHAR(200) NOT NULL,
			category VARCHAR(50) DEFAULT 'Custom',
			duration INTEGER DEFAULT 0,
			file_size BIGINT DEFAULT 0,
			sample_rate INTEGER DEFAULT 0,
			file_path VARCHAR(500),
			device_id INTEGER REFERENCES devices(id) ON DELETE SET NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS broadcast_schedules (
			id SERIAL PRIMARY KEY,
			name VARCHAR(200) NOT NULL,
			audio_id INTEGER REFERENCES audio_files(id) ON DELETE CASCADE,
			device_id INTEGER REFERENCES devices(id) ON DELETE CASCADE,
			schedule_type VARCHAR(50) NOT NULL,
			begin_time VARCHAR(50),
			end_time VARCHAR(50),
			volume INTEGER DEFAULT 50,
			enabled BOOLEAN DEFAULT true,
			day_of_week INTEGER,
			specific_date VARCHAR(20),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS broadcast_logs (
			id SERIAL PRIMARY KEY,
			time VARCHAR(50),
			device_id INTEGER REFERENCES devices(id) ON DELETE SET NULL,
			device_name VARCHAR(200),
			audio_id INTEGER REFERENCES audio_files(id) ON DELETE SET NULL,
			audio_name VARCHAR(200),
			result VARCHAR(50),
			duration INTEGER DEFAULT 0,
			status VARCHAR(50),
			error_message TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			id SERIAL PRIMARY KEY,
			key VARCHAR(100) UNIQUE NOT NULL,
			value TEXT,
			description VARCHAR(500),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS prayer_locations (
			id SERIAL PRIMARY KEY,
			latitude DOUBLE PRECISION,
			longitude DOUBLE PRECISION,
			timezone VARCHAR(50),
			method INTEGER DEFAULT 1,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS prayer_times (
			id SERIAL PRIMARY KEY,
			date DATE NOT NULL,
			fajr VARCHAR(20),
			dhuhr VARCHAR(20),
			asr VARCHAR(20),
			maghrib VARCHAR(20),
			isha VARCHAR(20),
			location_id INTEGER REFERENCES prayer_locations(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(date, location_id)
		)`,
		`CREATE TABLE IF NOT EXISTS prayer_broadcast_configs (
			id SERIAL PRIMARY KEY,
			prayer VARCHAR(20) UNIQUE NOT NULL,
			audio_id INTEGER REFERENCES audio_files(id) ON DELETE SET NULL,
			device_id INTEGER REFERENCES devices(id) ON DELETE SET NULL,
			volume INTEGER DEFAULT 50,
			enabled BOOLEAN DEFAULT false
		)`,
	}

	for _, table := range tables {
		_, err := DB.Exec(table)
		if err != nil {
			log.Printf("Failed to create table: %v", err)
		}
	}

	// Seed default admin user if not exists
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if count == 0 {
		DB.Exec(`INSERT INTO users (username, password, name, email, role, enabled) 
			VALUES ('admin', 'admin123', 'Administrator', 'admin@hikamp.com', 'admin', true)`)
		log.Println("Default admin user created")
	}

	// Seed default settings
	defaultSettings := map[string]string{
		"company_name": "Hikvision Broadcast",
		"timezone":     "Asia/Makassar",

		"default_volume": "50",
		"auto_sync":      "false",
		"dark_mode":      "false",
	}
	for key, value := range defaultSettings {
		var sCount int
		DB.QueryRow("SELECT COUNT(*) FROM settings WHERE key = $1", key).Scan(&sCount)
		if sCount == 0 {
			DB.Exec("INSERT INTO settings (key, value, description) VALUES ($1, $2, $3)", key, value, "")
		}
	}

	// Seed default prayer broadcast configs if not exists
	prayers := []string{"fajr", "dhuhr", "asr", "maghrib", "isha"}
	for _, p := range prayers {
		var pCount int
		DB.QueryRow("SELECT COUNT(*) FROM prayer_broadcast_configs WHERE prayer = $1", p).Scan(&pCount)
		if pCount == 0 {
			DB.Exec("INSERT INTO prayer_broadcast_configs (prayer, volume, enabled) VALUES ($1, 50, false)", p)
		}
	}

	// Migration: add sample_rate column to audio_files if it doesn't exist
	DB.Exec(`ALTER TABLE audio_files ADD COLUMN IF NOT EXISTS sample_rate INTEGER DEFAULT 0`)

	// Migration: add duration_str column to audio_files if it doesn't exist
	DB.Exec(`ALTER TABLE audio_files ADD COLUMN IF NOT EXISTS duration_str VARCHAR(20) DEFAULT ''`)

	log.Println("Database tables initialized")
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
