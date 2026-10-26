package repository

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TxManager adalah satu-satunya tempat transaction dibuka dan ditutup.
type TxManager struct {
	pool        *pgxpool.Pool
	maxAttempts int
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool, maxAttempts: 3}
}

// WithTx menjalankan fn di dalam satu transaction.
// PERBAIKAN BUG #3: Parameter fn menerima DBTX (bukan pgx.Tx) sesuai komentar & pemanggilan di service
func (m *TxManager) WithTx(ctx context.Context, fn func(tx DBTX) error) error {
	var err error
	for attempt := 1; attempt <= m.maxAttempts; attempt++ {
		err = m.runOnce(ctx, fn)
		if !isRetryable(err) {
			return err
		}

		// Jeda singkat yang sedikit acak
		wait := time.Duration(attempt*20+rand.IntN(30)) * time.Millisecond
		select {
		case <-ctx.Done():
			return err
		case <-time.After(wait):
		}
	}
	return err
}

// PERBAIKAN BUG #3: Parameter fn menerima DBTX
func (m *TxManager) runOnce(ctx context.Context, fn func(tx DBTX) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("memulai transaction: %w", err)
	}
	// PERBAIKAN BUG #4: Rollback SELALU dijadwalkan agar koneksi tidak bocor saat gagal
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func isRetryable(err error) bool {
	code := pgErrorCode(err)
	return code == codeSerializationFailure || code == codeDeadlockDetected
}