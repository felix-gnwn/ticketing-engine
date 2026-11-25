package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/felix-gnwn/ticketing-engine/internal/middleware"
	"github.com/felix-gnwn/ticketing-engine/internal/usecase"
)

type AssetUseCase interface {
	RequestUploadURL(ctx context.Context, userID uuid.UUID, filename string, contentType string) (*usecase.UploadResponse, error)
	ConfirmUpload(ctx context.Context, assetID uuid.UUID) error
}

type AssetHandler struct {
	assetUC AssetUseCase
}

func NewAssetHandler(assetUC AssetUseCase) *AssetHandler {
	return &AssetHandler{
		assetUC: assetUC,
	}
}

type uploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

// RequestUploadURL godoc
// @Summary		Request S3 Presigned Upload URL
// @Description	Generates a presigned S3 URL for uploading Govt. ID card
// @Tags		Assets
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param		request body uploadRequest true "Upload Request Payload"
// @Success		200 {object} JSONResponse
// @Failure		400 {object} ErrorResponse
// @Failure		401 {object} ErrorResponse
// @Failure		500 {object} ErrorResponse
// @Router		/assets/upload-url [post]
func (h *AssetHandler) RequestUploadURL(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userIDStr == "" {
		writeError(w, http.StatusUnauthorized, errors.New("User ID not found in context"))
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("Invalid User ID format"))
		return
	}

	var req uploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("Invalid request payload"))
		return
	}

	if req.Filename == "" || req.ContentType == "" {
		writeError(w, http.StatusBadRequest, errors.New("filename and content_type are required"))
		return
	}

	resp, err := h.assetUC.RequestUploadURL(r.Context(), userID, req.Filename, req.ContentType)
	if err != nil {
		if strings.Contains(err.Error(), "invalid file extension") {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		writeError(w, http.StatusInternalServerError, errors.New("Failed to generate upload URL"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Presigned URL generated successfully",
		"data":    resp,
	})
}

type confirmRequest struct {
	AssetID string `json:"asset_id"`
}

// ConfirmUpload godoc
// @Summary		Confirm S3 Asset Upload Status
// @Description	Updates asset metadata status from PENDING to UPLOADED
// @Tags		Assets
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param		request body confirmRequest true "Confirm Asset Payload"
// @Success		200 {object} JSONResponse
// @Failure		400 {object} ErrorResponse
// @Failure		500 {object} ErrorResponse
// @Router		/assets/confirm [put]
func (h *AssetHandler) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
	var req confirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("Invalid request payload"))
		return
	}

	assetID, err := uuid.Parse(req.AssetID)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("Invalid Asset ID format"))
		return
	}

	err = h.assetUC.ConfirmUpload(r.Context(), assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("Failed to confirm upload"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Asset upload confirmed successfully",
	})
}
