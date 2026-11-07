package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type assetRepository struct {
	db *pgxpool.Pool
}

func NewAssetRepository(db *pgxpool.Pool) domain.AssetRepository {
	return &assetRepository{
		db: db,
	}
}

func (r *assetRepository) CreateAsset(ctx context.Context, asset *domain.Asset) error {
	query := `
		INSERT INTO
			assets (
				id,
				user_id,
				object_key,
				status,
				created_at,
				updated_at
			)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6
		)
	`

	_, err := r.db.Exec(ctx, query,
		asset.ID,
		asset.UserID.String(),
		asset.ObjectKey,
		asset.Status,
		asset.CreatedAt,
		asset.UpdatedAt,
	)

	return err
}

func (r *assetRepository) UpdateAssetStatus(ctx context.Context, id uuid.UUID, status domain.AssetStatus) error {
	query := `
		UPDATE
			assets
		SET
			status = $1,
			updated_at = CURRENT_TIMESTAMP
		WHERE
			id = $2
	`

	commandTag, err := r.db.Exec(ctx, query, status, id)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return errors.New("asset not found")
	}

	return nil
}

func (r *assetRepository) GetAssetByID(ctx context.Context, id uuid.UUID) (*domain.Asset, error) {
	query := `
		SELECT
			id,
			user_id,
			object_key,
			status,
			created_at,
			updated_at
		FROM
			assets
		WHERE
			id = $1
	`

	var asset domain.Asset
	var userIDStr string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&asset.ID,
		&userIDStr,
		&asset.ObjectKey,
		&asset.Status,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("asset not found")
		}
		return nil, err
	}

	parsedUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user_id format in database")
	}
	asset.UserID = parsedUserID

	return &asset, nil
}
