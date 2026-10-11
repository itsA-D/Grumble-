package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"grumble/internal/store/postgres/queries"
)

// errTxAlreadyOpen guards against nested units of work, which are always bugs.
var errTxAlreadyOpen = errors.New("postgres: transaction already open")

// TxManager runs a unit of work in a single transaction (ARCHITECTURE.md
// section 7). Services never manage pgx.Tx directly: they call InTx and then
// the repository methods they hold, which transparently route through the
// transaction that InTx stored in the context. Nesting InTx is an error.
type TxManager struct {
	pool *pgxpool.Pool
}

// NewTxManager builds a manager on the shared pool.
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// InTx commits the result of fn, or rolls back when fn returns an error. The
// derived context carries the transaction; repository methods called with it
// operate inside the same transaction. The original ctx is only used to
// control the commit/rollback round trip.
func (m *TxManager) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txKey{}) != nil {
		return errTxAlreadyOpen
	}
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	txCtx := context.WithValue(ctx, txKey{}, tx)

	if err := fn(txCtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return nil
}

// handle returns the transaction from the context when a unit of work is
// active, otherwise the pool: query execution follows the caller's scope.
func handle(ctx context.Context, pool *pgxpool.Pool) queries.DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

type txKey struct{}
