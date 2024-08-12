package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Data .
type Data struct {
	db  *gorm.DB
	log *log.Helper
}

type Transaction interface {
	InTx(context.Context, func(ctx context.Context) error) error
}

type contextTxKey struct{}

func (d *Data) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ctx = context.WithValue(ctx, contextTxKey{}, tx)
		return fn(ctx)
	})
}

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
	l := log.NewHelper(log.With(logger, "module", "gorm"))
	d := &Data{
		db:  db,
		log: l,
	}
	return d, func() {
	}, nil
}

// NewDB gorm Connecting to a Database
func NewDB(dsn string, logger log.Logger) *gorm.DB {
	log := log.NewHelper(log.With(logger, "module", "gorm"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{SkipDefaultTransaction: true, PrepareStmt: true})
	if err != nil {
		log.Fatalf("failed opening connection to database: %v", err)
	}

	return db
}

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
