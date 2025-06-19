package gcs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func ConnectGoogleCloudStorage() (*storage.Client, error) {
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
	}

	client, err := storage.NewClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Cloud Storage client: %w", err)
	}

	return client, nil
}

type GCSService struct {
	client     *storage.Client
	bucketName string
}

func NewGCSService(bucketName, credentialsPath string) (*GCSService, error) {
	ctx := context.Background()
	
	var client *storage.Client
	var err error
	
	if credentialsPath != "" {
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(credentialsPath))
	} else {
		// Use default credentials (útil para ambiente de produção)
		client, err = storage.NewClient(ctx)
	}
	
	if err != nil {
		return nil, fmt.Errorf("erro ao criar cliente GCS: %v", err)
	}

	return &GCSService{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// UploadFile faz upload de um arquivo para o GCS
func (gcs *GCSService) UploadFile(ctx context.Context, filePath, objectName string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("erro ao abrir arquivo: %v", err)
	}
	defer file.Close()

	bucket := gcs.client.Bucket(gcs.bucketName)
	obj := bucket.Object(objectName)
	
	writer := obj.NewWriter(ctx)
	writer.ContentType = getContentType(filePath)
	
	if _, err := io.Copy(writer, file); err != nil {
		return "", fmt.Errorf("erro ao fazer upload: %v", err)
	}
	
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("erro ao fechar writer: %v", err)
	}

	// Retornar URL público
	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", gcs.bucketName, objectName)
	return publicURL, nil
}

// UploadFromReader faz upload de dados de um io.Reader
func (gcs *GCSService) UploadFromReader(ctx context.Context, reader io.Reader, objectName, contentType string) (string, error) {
	bucket := gcs.client.Bucket(gcs.bucketName)
	obj := bucket.Object(objectName)
	
	writer := obj.NewWriter(ctx)
	writer.ContentType = contentType
	
	if _, err := io.Copy(writer, reader); err != nil {
		return "", fmt.Errorf("erro ao fazer upload: %v", err)
	}
	
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("erro ao fechar writer: %v", err)
	}

	// Retornar URL público
	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", gcs.bucketName, objectName)
	return publicURL, nil
}

// CreateSongFolder cria uma estrutura de pastas para uma música
func (gcs *GCSService) CreateSongFolder(ctx context.Context, songFolder string) error {
	// No GCS, não precisamos criar pastas explicitamente
	// Elas são criadas automaticamente quando fazemos upload de arquivos
	return nil
}

// UploadSongFiles faz upload de todos os arquivos de uma música
func (gcs *GCSService) UploadSongFiles(ctx context.Context, localSongFolder, gcsFolder string) (map[string]string, error) {
	uploadedFiles := make(map[string]string)
	
	err := filepath.Walk(localSongFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			return nil
		}
		
		// Calcular o nome do objeto no GCS
		relPath, err := filepath.Rel(localSongFolder, path)
		if err != nil {
			return err
		}
		
		objectName := filepath.Join(gcsFolder, relPath)
		// Normalizar para usar / no GCS
		objectName = filepath.ToSlash(objectName)
		
		// Fazer upload
		publicURL, err := gcs.UploadFile(ctx, path, objectName)
		if err != nil {
			return fmt.Errorf("erro ao fazer upload de %s: %v", path, err)
		}
		
		uploadedFiles[relPath] = publicURL
		return nil
	})
	
	return uploadedFiles, err
}

// GenerateSignedURL gera uma URL assinada para acesso temporário
func (gcs *GCSService) GenerateSignedURL(objectName string, expiration time.Duration) (string, error) {
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(expiration),
	}
	
	// Usar a função global SignedURL
	url, err := storage.SignedURL(gcs.bucketName, objectName, opts)
	if err != nil {
		return "", fmt.Errorf("erro ao gerar URL assinada: %v", err)
	}
	
	return url, nil
}

// Close fecha a conexão com o GCS
func (gcs *GCSService) Close() error {
	return gcs.client.Close()
}

// getContentType determina o content type baseado na extensão do arquivo
func getContentType(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".json":
		return "application/json"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}
