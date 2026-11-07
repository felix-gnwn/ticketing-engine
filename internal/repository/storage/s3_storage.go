package storage

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type s3StorageRepo struct {
	presignClient *s3.PresignClient
	bucketName    string
}

func NewS3StorageRepository(client *s3.Client, bucketName string) domain.StorageRepository {
	return &s3StorageRepo{
		presignClient: s3.NewPresignClient(client),
		bucketName:    bucketName,
	}
}

func (r *s3StorageRepo) GeneratePresignedURL(ctx context.Context, objectKey string, contentType string) (string, error) {
	putInput := &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType), // ex: image/jpeg
	}

	req, err := r.presignClient.PresignPutObject(ctx, putInput, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		return "", err
	}

	return req.URL, nil
}
