package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"

	"url-shortener/internal/storage"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err := db.Prepare(`
	CREATE TABLE IF NOT EXISTS url(
		id INTEGER PRIMARY KEY,
		alias TEXT NOT NULL UNIQUE,
		url TEXT NOT NULL);
	CREATE INDEX IF NOT EXISTS idx_alias ON url(alias);
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func SaveGeneratingAlias(storagePath string) error {
	const op = "storage.sqlite.SaveGeneratingAlias"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	stmt, err := db.Prepare(`
	CREATE TABLE IF NOT EXISTS alias_value(
		id INTEGER PRIMARY KEY,
		value INT NOT NULL,
		name TEXT NOT NULL UNIQUE);
	CREATE INDEX IF NOT EXISTS idx_name ON alias_value(name);
	`)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	stmt, err = db.Prepare(`
	INSERT INTO alias_value(value, name) 
		VALUES(1, 'AliasLength'),
			  (0, 'PointerOne'),
			  (0, 'PointerTwo'),
			  (0, 'PointerThree'),
			  (0, 'PointerFour');
	`)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (s *Storage) SaveURL(urlToSave string, alias string) (int64, error) {
	const op = "storage.sqlite.SaveURL"

	stmt, err := s.db.Prepare("INSERT INTO url(url, alias) VALUES(?, ?)")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.Exec(urlToSave, alias)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExists)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.sqlite.GetURL"

	stmt, err := s.db.Prepare("SELECT url FROM url WHERE alias = ?")
	if err != nil {
		return "", fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var resURL string

	err = stmt.QueryRow(alias).Scan(&resURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", storage.ErrURLNotFound
		}

		return "", fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return resURL, nil
}

func (s *Storage) UpdateURL(id int, newURL string) (string, error) {
	const op = "storage.sqlite.UpdateURL"

	// Getting current URL
	var resURL string
	err := s.db.QueryRow("SELECT url FROM url WHERE id = ?", id).Scan(&resURL)
	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrURLNotFound
	} else if err != nil {
		return "", fmt.Errorf("%s: select statement: %w", op, err)
	}

	// Checking, is URL changed
	if resURL == newURL {
		return "Same url, nothing changed", nil
	}

	// Updating URL in storage
	res, err := s.db.Exec("UPDATE url SET url = ? WHERE id = ?", newURL, id)
	if err != nil {
		return "", fmt.Errorf("%s: update statement: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("%s: failed to get rows affected: %w", op, err)
	}
	if rowsAffected == 0 {
		return "No such ID", storage.ErrURLNotFound
	}

	return resURL, nil
}

func (s *Storage) DeleteURL(id int) (string, error) {
	const op = "storage.sqlite.DeleteURL"

	// Getting URL before Deleting
	var resURL string
	err := s.db.QueryRow("SELECT url FROM url WHERE id = ?", id).Scan(&resURL)
	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrURLNotFound
	} else if err != nil {
		return "", fmt.Errorf("%s: select statement: %w", op, err)
	}

	// Deleting URL
	res, err := s.db.Exec("DELETE FROM url WHERE id = ?", id)
	if err != nil {
		return "", fmt.Errorf("%s: delete statement: %w", op, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return "", storage.ErrURLNotFound
	}

	return resURL, nil
}
