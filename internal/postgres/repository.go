package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

func Migrate(config Config) error {
	dialect := "postgres"

	db, err := sql.Open(dialect, config.DbUrl())
	if err != nil {
		return err
	}
	defer func(pg *sql.DB) { _ = pg.Close() }(db)

	if err = goose.SetDialect(dialect); err != nil {
		return err
	}

	goose.SetTableName(config.Schema + ".goose_db_version")
	return goose.Up(db, config.MigrationsDir)
}

type Database interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

type Repository struct {
	db Database
}

func NewRepository(config Config) (*Repository, error) {
	conf, err := pgxpool.ParseConfig(config.DbUrl())
	if err != nil {
		return nil, err
	}

	if conf.ConnConfig.RuntimeParams == nil {
		conf.ConnConfig.RuntimeParams = make(map[string]string)
	}

	conf.ConnConfig.RuntimeParams["search_path"] = config.Schema

	conf.MaxConns = int32(6)
	conf.MinConns = int32(0)
	conf.MaxConnLifetime = time.Hour
	conf.MaxConnIdleTime = time.Minute * 30
	conf.HealthCheckPeriod = time.Minute
	conf.ConnConfig.ConnectTimeout = time.Second * 10

	pool, err := pgxpool.NewWithConfig(context.Background(), conf)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return &Repository{pool}, nil
}

func WithTx(tx pgx.Tx) *Repository {
	return &Repository{
		db: tx,
	}
}

func (repo *Repository) StoreAsset(ctx context.Context, asset *Asset) (*Asset, error) {
	if asset == nil {
		return nil, errors.New("asset is nil")
	}

	created := new(Asset)

	err := repo.db.QueryRow(
		ctx,
		storeAssetQuery,
		asset.Name,
		asset.Address,
		asset.IntervalSeconds,
		asset.TimeoutSeconds,
	).Scan(
		&created.ID,
		&created.Name,
		&created.Address,
		&created.IntervalSeconds,
		&created.TimeoutSeconds,
		&created.CreatedAt,
		&created.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("store asset: asset with given address already exists")
	}
	if err != nil {
		return nil, fmt.Errorf("store asset: %w", err)
	}

	return created, nil
}

// ---------------------------------------------------------------------------------------------------------------------

func (repo *Repository) ProcessProbe(ctx context.Context, asset *Asset, probe *ProbeResult) (*ProbeProcessingResult, error) {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("process probe: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txRepo := WithTx(tx)

	storedProbe, err := txRepo.StoreProbe(ctx, probe)
	if err != nil {
		return nil, fmt.Errorf("process probe: %w", err)
	}

	openIncident, err := txRepo.GetOpenIncidentByAsset(ctx, asset.ID)
	if err != nil {
		return nil, fmt.Errorf("process probe: %w", err)
	}

	result := &ProbeProcessingResult{Probe: storedProbe}

	switch {
	/*
		The probe has failed, we must create an incident and
		a notification to be sent.
	*/
	case !storedProbe.Success:
		if openIncident != nil {
			result.Incident = openIncident
			break
		}

		severity := "CRITICAL"
		incident := &Incident{
			Severity:  &severity,
			Summary:   nil,
			StartedAt: storedProbe.CreatedAt,
			AssetID:   asset.ID,
		}
		if summary := ToDetails(DOWN, storedProbe); summary != "" {
			incident.Summary = &summary
		}

		createdIncident, createErr := txRepo.StoreIncident(ctx, incident)
		if createErr != nil {
			return nil, createErr
		}

		result.Incident = createdIncident
		result.Notification = &Notification{
			AssetID:   asset.ID,
			AssetName: asset.Name,
			Event:     DOWN,
			Timestamp: storedProbe.CreatedAt.Time,
			Details:   ToDetails(DOWN, storedProbe),
		}

		break
	/*
		The probe was successful and there was an open incident
		which has to be resolved and a notification to be sent.
	*/
	case openIncident != nil:
		resolvedIncident, resolveErr := txRepo.ResolveIncident(ctx, openIncident.ID, storedProbe.CreatedAt.Time)
		if resolveErr != nil {
			return nil, fmt.Errorf("process probe: %w", resolveErr)
		}

		result.Incident = resolvedIncident
		result.Notification = &Notification{
			AssetID:   asset.ID,
			AssetName: asset.Name,
			Event:     RECOVERY,
			Timestamp: storedProbe.CreatedAt.Time,
			Details:   ToDetails(RECOVERY, probe),
		}

		break
	/*
		The probe was successful and there wasn't any incident
		to be resolved. Do nothing.
	*/
	default:

	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("process probe: commit tx: %w", err)
	}

	return result, nil
}

// ---------------------------------------------------------------------------------------------------------------------

func (repo *Repository) StoreProbe(ctx context.Context, probe *ProbeResult) (*ProbeResult, error) {
	if probe == nil {
		return nil, errors.New("probe is nil")
	}

	created := new(ProbeResult)

	err := repo.db.QueryRow(
		ctx,
		storeProbeQuery,
		probe.Success,
		probe.Code,
		probe.ErrMessage,
		probe.AssetID,
	).Scan(
		&created.ID,
		&created.Success,
		&created.Code,
		&created.ErrMessage,
		&created.CreatedAt,
		&created.AssetID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("store probe: no rows affected")
	}
	if err != nil {
		return nil, fmt.Errorf("store probe: %w", err)
	}

	return created, nil
}

func (repo *Repository) StoreIncident(ctx context.Context, incident *Incident) (*Incident, error) {
	if incident == nil {
		return nil, errors.New("incident is nil")
	}

	created := new(Incident)

	startedAt := time.Now().UTC()
	if incident.StartedAt.Valid {
		startedAt = incident.StartedAt.Time
	}

	var endedAt any
	if incident.EndedAt.Valid {
		endedAt = incident.EndedAt.Time
	}

	err := repo.db.QueryRow(
		ctx,
		storeIncidentQuery,
		incident.Severity,
		incident.Summary,
		startedAt,
		endedAt,
		incident.AssetID,
	).Scan(
		&created.ID,
		&created.Severity,
		&created.Summary,
		&created.StartedAt,
		&created.EndedAt,
		&created.AssetID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("store incident: no rows affected")
	}
	if err != nil {
		return nil, fmt.Errorf("store incident: %w", err)
	}

	return created, nil
}

func (repo *Repository) GetOpenIncidentByAsset(ctx context.Context, assetID string) (*Incident, error) {
	if assetID == "" {
		return nil, errors.New("asset id is empty")
	}

	incident := new(Incident)

	err := repo.db.QueryRow(
		ctx,
		getOpenIncidentByAssetQuery,
		assetID,
	).Scan(
		&incident.ID,
		&incident.Severity,
		&incident.Summary,
		&incident.StartedAt,
		&incident.EndedAt,
		&incident.AssetID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get open incident: %w", err)
	}

	return incident, nil
}

func (repo *Repository) ResolveIncident(ctx context.Context, incidentID string, endedAt time.Time) (*Incident, error) {
	if incidentID == "" {
		return nil, errors.New("incident id is empty")
	}

	resolved := new(Incident)

	err := repo.db.QueryRow(
		ctx,
		resolveIncidentQuery,
		incidentID,
		endedAt.UTC(),
	).Scan(
		&resolved.ID,
		&resolved.Severity,
		&resolved.Summary,
		&resolved.StartedAt,
		&resolved.EndedAt,
		&resolved.AssetID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("resolve incident: incident not found")
	}
	if err != nil {
		return nil, fmt.Errorf("resolve incident: %w", err)
	}

	return resolved, nil
}

// ---------------------------------------------------------------------------------------------------------------------

func (repo *Repository) CleanupProbes(ctx context.Context, olderThanDays int) (int64, error) {
	ts := time.Now().UTC().Add(-time.Duration(olderThanDays) * 24 * time.Hour)
	ct, err := repo.db.Exec(ctx, cleanupProbesQuery, ts)
	if err != nil {
		return 0, fmt.Errorf("cleanup probes: %w", err)
	}
	return ct.RowsAffected(), nil
}
