package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Veysel440/go-notes-api/internal/config"
	dbx "github.com/Veysel440/go-notes-api/internal/db"
	"github.com/Veysel440/go-notes-api/internal/repos"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("kullanım: seed-admin <email>")
	}
	email := os.Args[1]

	cfg := config.Load()
	db, closeFn, err := dbx.OpenAndMigrate(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer closeFn()

	users := repos.Users{DB: db}
	roles := repos.Roles{DB: db}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	u, err := users.FindByEmail(ctx, email)
	if err != nil {
		log.Fatalf("kullanıcı yok: %v", err)
	}

	if err := roles.Assign(ctx, u.ID, "admin"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("ok: admin atandı →", email)
}
