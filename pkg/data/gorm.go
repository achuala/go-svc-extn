package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Data .
type Data struct {
	db *gorm.DB
}

type Transaction interface {
	InTx(context.Context, func(ctx context.Context) error) error
}

type GormOptions struct {
	SkipDefaultTransaction bool
	MaxIdleConns           int
	MaxOpenConns           int
	ConnMaxIdleTime        time.Duration
	ConnMaxLifetime        time.Duration
}

type contextTxKey struct{}

// Execute the database actions in a transaction
func (d *Data) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ctx = context.WithValue(ctx, contextTxKey{}, tx)
		return fn(ctx)
	})
}

// DB Get the database connection
func (d *Data) DB(ctx context.Context) *gorm.DB {
	tx, ok := ctx.Value(contextTxKey{}).(*gorm.DB)
	if ok {
		return tx
	}
	return d.db
}

// NewTransaction .
func NewTransaction(d *Data) Transaction {
	return d
}

// NewData .
func NewData(db *gorm.DB, logger log.Logger) (*Data, func(), error) {
	d := &Data{
		db: db,
	}
	return d, func() {
	}, nil
}

// NewDB gorm Connecting to a Database
func NewGorm(dsn string) (*gorm.DB, error) {
	opts := GormOptions{
		SkipDefaultTransaction: true,
		MaxIdleConns:           1,
		MaxOpenConns:           10,
		ConnMaxIdleTime:        15 * time.Minute,
		ConnMaxLifetime:        8 * time.Hour,
	}
	return NewGormWithOptions(dsn, log.DefaultLogger, opts)
}

func NewGormWithOptions(dsn string, logger log.Logger, opts GormOptions) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{SkipDefaultTransaction: opts.SkipDefaultTransaction,
		DisableAutomaticPing: true,
		Logger:               NewGormLogger(logger)})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
	sqlDB.SetConnMaxIdleTime(opts.ConnMaxIdleTime)
	sqlDB.SetConnMaxLifetime(opts.ConnMaxLifetime)
	return db, nil
}

// Paginate Pagination
func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}

		switch {
		case pageSize > 100:
			pageSize = 100
		case pageSize <= 0:
			pageSize = 10
		}

		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}
