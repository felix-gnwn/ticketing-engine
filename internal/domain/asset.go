package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AssetStatus string

const (
	AssetStatusPending	AssetStatus = "PENDING"
	AssetStatusUploaded	AssetStatus = "UPLOADED"
	AssetStatusFailed	AssetStatus = "FAILED"
)

// Asset represents metadata file that will be saved in PostgreSQL
type Asset struct {
	ID			uuid.UUID	`json:"id"`
	UserID		uuid.UUID	`json:"user_id"`
	ObjectKey	string		`json:"object_key"` // ex: "idcard/09b611.jpg"
	Status		AssetStatus	`json:"status"`
	CreatedAt	time.Time	`json:"created_at"`
	UpdatedAt	time.Time	`json:"updated_at"`
}

type AssetRepository interface {
	CreateAsset(ctx context.Context, asset *Asset) error
	UpdateAssetStatus(ctx context.Context, id uuid.UUID, status AssetStatus) error
	GetAssetByID(ctx context.Context, id uuid.UUID) (*Asset, error)
}

type StorageRepository interface {
	GeneratePresignedURL(ctx context.Context, objectKey string, contentType string) (string, error)
}