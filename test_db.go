package main

import (
    "fmt"
    "github.com/jackc/pgx/v5"
    "errors"
)

func main() {
    err := fmt.Errorf("wrapped: %w", pgx.ErrNoRows)
    fmt.Println(err == pgx.ErrNoRows)
    fmt.Println(errors.Is(err, pgx.ErrNoRows))
}
