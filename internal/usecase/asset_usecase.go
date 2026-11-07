package usecase

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type UploadResponse struct {
	AssetID      uuid.UUID `json:"asset_id"`
	PresignedURL string    `json:"presigned_url"`
}

type assetUseCase struct {
	assetRepo   domain.AssetRepository
	storageRepo domain.StorageRepository
}

func NewAssetUseCase(ar domain.AssetRepository, sr domain.StorageRepository) *assetUseCase {
	return &assetUseCase{
		assetRepo:   ar,
		storageRepo: sr,
	}
}

func (u *assetUseCase) RequestUploadURL(ctx context.Context, userID uuid.UUID, filename string, contentType string) (*UploadResponse, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return nil, fmt.Errorf("invalid file extension")
	}

	objectKey := fmt.Sprintf("idcard/%s%s", uuid.New().String(), ext)

	assetID := uuid.New()
	newAsset := &domain.Asset{
		ID:        assetID,
		UserID:    userID,
		ObjectKey: objectKey,
		Status:    domain.AssetStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := u.assetRepo.CreateAsset(ctx, newAsset)
	if err != nil {
		return nil, fmt.Errorf("failed to save asset metadata: %w", err)
	}

	presignedURL, err := u.storageRepo.GeneratePresignedURL(ctx, objectKey, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return &UploadResponse{
		AssetID:      assetID,
		PresignedURL: presignedURL,
	}, nil
}

func (u *assetUseCase) ConfirmUpload(ctx context.Context, assetID uuid.UUID) error {
	err := u.assetRepo.UpdateAssetStatus(ctx, assetID, domain.AssetStatusUploaded)
	if err != nil {
		return fmt.Errorf("failed to confirm asset upload: %w", err)
	}

	return nil
}
