package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type User struct {
	ID      int     `db:"id"`
	Name    string  `db:"name"`
	Email   string  `db:"email"`
	Balance float64 `db:"balance"`
}

func openDB() (*sqlx.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable"
	}
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func InsertUser(db *sqlx.DB, user User) error {
	_, err := db.NamedExec(`INSERT INTO users (name, email, balance) VALUES (:name, :email, :balance) ON CONFLICT (email) DO NOTHING`, user)
	return err
}

func GetAllUsers(db *sqlx.DB) ([]User, error) {
	var users []User
	err := db.Select(&users, `SELECT id, name, email, balance FROM users ORDER BY id`)
	return users, err
}

func GetUserByID(db *sqlx.DB, id int) (User, error) {
	var u User
	err := db.Get(&u, `SELECT id, name, email, balance FROM users WHERE id=$1`, id)
	return u, err
}

func TransferBalance(db *sqlx.DB, fromID int, toID int, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	var from User
	err = tx.Get(&from, `SELECT id, name, email, balance FROM users WHERE id=$1 FOR UPDATE`, fromID)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sender not found: %w", err)
	}
	var to User
	err = tx.Get(&to, `SELECT id, name, email, balance FROM users WHERE id=$1 FOR UPDATE`, toID)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("receiver not found: %w", err)
	}
	if from.Balance < amount {
		_ = tx.Rollback()
		return fmt.Errorf("insufficient funds: have %.2f need %.2f", from.Balance, amount)
	}
	_, err = tx.Exec(`UPDATE users SET balance = balance - $1 WHERE id = $2`, amount, fromID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	_, err = tx.Exec(`UPDATE users SET balance = balance + $1 WHERE id = $2`, amount, toID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return err
	}
	return nil
}

func main() {
	db, err := openDB()
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()
	fmt.Println("connected")
	u1 := User{Name: "Alice", Email: "alice@example.com", Balance: 100.0}
	u2 := User{Name: "Bob", Email: "bob@example.com", Balance: 50.0}
	_ = InsertUser(db, u1)
	_ = InsertUser(db, u2)
	users, err := GetAllUsers(db)
	if err != nil {
		log.Fatalf("select all: %v", err)
	}
	fmt.Println("before transfer:")
	for _, u := range users {
		fmt.Printf("%d %s %.2f\n", u.ID, u.Name, u.Balance)
	}
	if len(users) >= 2 {
		if err := TransferBalance(db, users[0].ID, users[1].ID, 30.0); err != nil {
			log.Fatalf("transfer: %v", err)
		}
	}
	fmt.Println("after transfer:")
	updated, err := GetAllUsers(db)
	if err != nil {
		log.Fatalf("select all: %v", err)
	}
	for _, u := range updated {
		fmt.Printf("%d %s %.2f\n", u.ID, u.Name, u.Balance)
	}
}
