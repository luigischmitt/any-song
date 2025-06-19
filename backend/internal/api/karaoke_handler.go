package api

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type KaraokeProcessRequest struct {
	SongTitle  string `json:"song_title"`
	ArtistName string `json:"artist_name"`
}

type KaraokeProcessResponse struct {
	Success          bool   `json:"success"`
	Message          string `json:"message"`
	JobID            string `json:"job_id"`
	OriginalAudioURL string `json:"original_audio_url"`
	BackingTrackURL  string `json:"backing_track_url,omitempty"`
	LyricsURL        string `json:"lyrics_url,omitempty"`
	Status           string `json:"status"`
	ProcessingTime   string `json:"processing_time"`
}

func (api *API) ProcessKaraokeHandler(c *fiber.Ctx) error {
	start := time.Now()
	
	// Verificar autenticação
	user, exists := GetUserFromContext(c)
	if !exists {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Usuário não autenticado",
		})
	}

	// Obter arquivo do upload
	file, err := c.FormFile("audio_file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Arquivo de áudio não encontrado",
		})
	}

	// Validar tipo de arquivo
	if !strings.Contains(file.Header.Get("Content-Type"), "audio") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Tipo de arquivo inválido. Apenas arquivos de áudio são aceitos.",
		})
	}

	// Gerar ID único para o job
	jobID := uuid.New().String()
	
	// Obter dados do formulário
	songTitle := c.FormValue("song_title")
	artistName := c.FormValue("artist_name")
	
	if songTitle == "" {
		songTitle = strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
	}

	log.Printf("Iniciando processamento de karaokê para: %s - %s (User: %s)", artistName, songTitle, user.Email)

	// Abrir arquivo
	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Erro ao abrir arquivo",
		})
	}
	defer src.Close()

	// Ler dados do arquivo
	fileData, err := io.ReadAll(src)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Erro ao ler arquivo",
		})
	}

	// Criar nome único para o arquivo
	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("%s_%s_%s%s", user.UID, timestamp, 
		strings.ReplaceAll(songTitle, " ", "_"), filepath.Ext(file.Filename))

	// Upload para Google Cloud Storage
	log.Printf("Fazendo upload para GCS: %s", fileName)
	bucketName := os.Getenv("GCS_BUCKET_NAME")
	if bucketName == "" {
		bucketName = "anysong-audios"
	}

	originalAudioURL, err := api.GCSService.UploadFile(bucketName, fileName, fileData)
	if err != nil {
		log.Printf("Erro no upload para GCS: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Erro ao fazer upload do arquivo",
		})
	}

	log.Printf("Upload para GCS concluído: %s", originalAudioURL)

	// Processar com MusicAI em background
	go func() {
		api.processMusicAI(jobID, originalAudioURL, songTitle, artistName, user.UID)
	}()

	response := KaraokeProcessResponse{
		Success:          true,
		Message:          "Processamento iniciado com sucesso",
		JobID:            jobID,
		OriginalAudioURL: originalAudioURL,
		Status:           "processing",
		ProcessingTime:   time.Since(start).String(),
	}

	return c.JSON(response)
}

func (api *API) processMusicAI(jobID, audioURL, songTitle, artistName, userID string) {
	log.Printf("Iniciando processamento MusicAI para job %s", jobID)
	
	// Salvar arquivo temporariamente
	tempDir := "/tmp/karaoke_processing"
	os.MkdirAll(tempDir, 0755)
	
	tempFile := filepath.Join(tempDir, fmt.Sprintf("%s.mp3", jobID))
	
	// Executar script Python para MusicAI
	pythonScript := fmt.Sprintf(`
import sys
import os
sys.path.append('/app/utils')
from backing_track import gen_backing_track
import requests

# Download do arquivo
audio_url = '%s'
temp_file = '%s'

try:
    response = requests.get(audio_url)
    with open(temp_file, 'wb') as f:
        f.write(response.content)
    
    # Processar com MusicAI
    backing_track_url = gen_backing_track(temp_file)
    
    if backing_track_url:
        print(f"SUCCESS:{backing_track_url}")
    else:
        print("ERROR:Failed to generate backing track")
        
except Exception as e:
    print(f"ERROR:{str(e)}")
    
finally:
    # Limpar arquivo temporário
    if os.path.exists(temp_file):
        os.remove(temp_file)
`, audioURL, tempFile)

	cmd := exec.Command("python3", "-c", pythonScript)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		log.Printf("Erro no processamento MusicAI para job %s: %v", jobID, err)
		api.updateJobStatus(jobID, "failed", "", string(output))
		return
	}

	outputStr := string(output)
	log.Printf("Saída do MusicAI para job %s: %s", jobID, outputStr)

	if strings.HasPrefix(outputStr, "SUCCESS:") {
		backingTrackURL := strings.TrimPrefix(outputStr, "SUCCESS:")
		backingTrackURL = strings.TrimSpace(backingTrackURL)
		log.Printf("Backing track gerado com sucesso para job %s: %s", jobID, backingTrackURL)
		api.updateJobStatus(jobID, "completed", backingTrackURL, "")
	} else {
		log.Printf("Falha no processamento MusicAI para job %s: %s", jobID, outputStr)
		api.updateJobStatus(jobID, "failed", "", outputStr)
	}
}

func (api *API) updateJobStatus(jobID, status, backingTrackURL, errorMessage string) {
	// Aqui você pode salvar o status no Redis ou banco de dados
	// Por enquanto, vamos apenas logar
	log.Printf("Job %s atualizado: status=%s, backing_track=%s, error=%s", 
		jobID, status, backingTrackURL, errorMessage)
	
	// Salvar no Redis para consulta posterior
	jobData := map[string]interface{}{
		"status":            status,
		"backing_track_url": backingTrackURL,
		"error_message":     errorMessage,
		"updated_at":        time.Now().Unix(),
	}
	
	// Converter para JSON e salvar no Redis
	key := fmt.Sprintf("karaoke_job:%s", jobID)
	api.CacheService.Set(key, jobData, 24*time.Hour) // Manter por 24 horas
}

func (api *API) GetKaraokeJobStatusHandler(c *fiber.Ctx) error {
	jobID := c.Params("job_id")
	if jobID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Job ID é obrigatório",
		})
	}

	// Buscar status no Redis
	key := fmt.Sprintf("karaoke_job:%s", jobID)
	
	var jobData map[string]interface{}
	found, err := api.CacheService.Get(key, &jobData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Erro ao buscar status do job",
		})
	}

	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Job não encontrado",
		})
	}

	return c.JSON(fiber.Map{
		"job_id": jobID,
		"status": jobData["status"],
		"backing_track_url": jobData["backing_track_url"],
		"error_message": jobData["error_message"],
		"updated_at": jobData["updated_at"],
	})
} 