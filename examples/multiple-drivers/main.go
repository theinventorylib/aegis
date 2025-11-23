package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/db"

	// Example 1: Import lib/pq for PostgreSQL
	_ "github.com/lib/pq"
	// Example 2: Alternatively, you can use pgx
	// _ "github.com/jackc/pgx/v5/stdlib"
	// Example 3: Or use MySQL
	// _ "github.com/go-sql-driver/mysql"
	// Example 4: Or even SQLite for testing
	// _ "github.com/mattn/go-sqlite3"
)

func main() {
	fmt.Println("Aegis Database-Agnostic Example")
	fmt.Println("================================")

	// Example 1: PostgreSQL with lib/pq
	postgresExample()

	// Example 2: PostgreSQL with pgx (commented out)
	// pgxExample()

	// Example 3: MySQL (commented out)
	// mysqlExample()

	// Example 4: SQLite (commented out)
	// sqliteExample()
}

// Example 1: Using lib/pq driver for PostgreSQL
func postgresExample() {
	fmt.Println("Example 1: PostgreSQL with lib/pq driver")
	fmt.Println("-----------------------------------------")

	connString := "postgres://user:password@localhost:5432/aegis_db?sslmode=disable"

	// Standard Go database/sql setup
	sqlDB, err := sql.Open("postgres", connString)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer sqlDB.Close()

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		log.Printf("Failed to ping database (this is expected if DB isn't running): %v\n\n", err)
		return
	}

	// Initialize Aegis with the database connection
	auth, err := aegis.New(
		config.WithDB(sqlDB, db.PostgreSQL),
		config.WithJWTSecret([]byte("your-secret-key")),
		config.WithAPIMode(true), // Skip CSRF for this example
	)
	if err != nil {
		log.Fatal("Failed to initialize Aegis:", err)
	}

	fmt.Printf("✓ Successfully initialized Aegis with PostgreSQL (lib/pq)\n")
	fmt.Printf("✓ Database provider: %T\n\n", auth.GetDB())
}

// Example 2: Using pgx driver for PostgreSQL
// Uncomment the pgx import and this function to try it
/*
func pgxExample() {
	fmt.Println("Example 2: PostgreSQL with pgx driver")
	fmt.Println("--------------------------------------")

	connString := "postgres://user:password@localhost:5432/aegis_db?sslmode=disable"

	// Using pgx driver - same code, different driver!
	sqlDB, err := sql.Open("pgx", connString)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		log.Printf("Failed to ping database: %v\n\n", err)
		return
	}

	// Same Aegis initialization - it's driver-agnostic!
	auth, err := aegis.New(
		config.WithDB(sqlDB, db.PostgreSQL),
		config.WithJWTSecret([]byte("your-secret-key")),
		config.WithAPIMode(true),
	)
	if err != nil {
		log.Fatal("Failed to initialize Aegis:", err)
	}

	fmt.Printf("✓ Successfully initialized Aegis with PostgreSQL (pgx)\n\n")
}
*/

// Example 3: Using MySQL
// Uncomment the mysql import and this function to try it
/*
func mysqlExample() {
	fmt.Println("Example 3: MySQL")
	fmt.Println("----------------")

	connString := "user:password@tcp(127.0.0.1:3306)/aegis_db?parseTime=true"

	sqlDB, err := sql.Open("mysql", connString)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		log.Printf("Failed to ping database: %v\n\n", err)
		return
	}

	// Just change the dialect - everything else stays the same!
	auth, err := aegis.New(
		config.WithDB(sqlDB, db.MySQL),
		config.WithJWTSecret([]byte("your-secret-key")),
		config.WithAPIMode(true),
	)
	if err != nil {
		log.Fatal("Failed to initialize Aegis:", err)
	}

	fmt.Printf("✓ Successfully initialized Aegis with MySQL\n\n")
}
*/

// Example 4: Using SQLite (great for testing!)
// Uncomment the sqlite import and this function to try it
/*
func sqliteExample() {
	fmt.Println("Example 4: SQLite (in-memory)")
	fmt.Println("-----------------------------")

	// Use in-memory database for testing
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer sqlDB.Close()

	// SQLite doesn't need Ping for in-memory databases

	auth, err := aegis.New(
		config.WithDB(sqlDB, db.SQLite),
		config.WithJWTSecret([]byte("your-secret-key")),
		config.WithAPIMode(true),
	)
	if err != nil {
		log.Fatal("Failed to initialize Aegis:", err)
	}

	fmt.Printf("✓ Successfully initialized Aegis with SQLite (in-memory)\n")
	fmt.Printf("✓ Perfect for unit tests!\n\n")
}
*/
