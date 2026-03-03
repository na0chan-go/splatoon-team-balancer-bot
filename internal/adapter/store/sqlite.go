package store

import botstore "github.com/na0chan-go/splatoon-team-balancer-bot/internal/store"

func NewSQLiteStore(path string) (*botstore.SQLiteStore, error) {
	return botstore.NewSQLiteStore(path)
}
