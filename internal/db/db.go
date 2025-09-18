package db

import (
	"database/sql"
	"time"

	"github.com/Veysel440/go-notes-api/internal/config"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	mysqlDrv "github.com/golang-migrate/migrate/v4/database/mysql"
	file "github.com/golang-migrate/migrate/v4/source/file"
)

func OpenAndMigrate(cfg config.Config) (*sql.DB, func(), error) {
	sqlDB, err := sql.Open("mysql", cfg.DBDsn)
	if err != nil {
		return nil, nil, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, nil, err
	}

	src, err := file.New("file://migrations")
	if err != nil {
		sqlDB.Close()
		return nil, nil, err
	}
	drv, err := mysqlDrv.WithInstance(sqlDB, &mysqlDrv.Config{})
	if err != nil {
		sqlDB.Close()
		return nil, nil, err
	}
	m, err := migrate.NewWithInstance("file", src, "mysql", drv)
	if err != nil {
		sqlDB.Close()
		return nil, nil, err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		sqlDB.Close()
		return nil, nil, err
	}
	return sqlDB, func() {}, nil
}
