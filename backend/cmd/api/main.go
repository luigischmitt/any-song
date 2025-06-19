package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/josevitorrodriguess/any-song/backend/internal/storage/gcs"
)

// Estruturas para autenticação
type SignInRequest struct {
	IDToken string `json:"idToken"`
}

type SignInResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	User    *User  `json:"user,omitempty"`
}

type User struct {
	FirebaseUID    string `json:"firebase_uid"`
	Email          string `json:"email"`
	Name           string `json:"name"`
	ProfilePicture string `json:"profile_picture"`
}

// Estruturas para o processamento de karaokê
type ProcessRequest struct {
	AudioFile  string `json:"audio_file" form:"audio_file"`
	SongTitle  string `json:"song_title" form:"song_title"`
	ArtistName string `json:"artist_name" form:"artist_name"`
}

type ProcessResponse struct {
	Success bool   `json:"success"`
	JobID   string `json:"job_id"`
	Message string `json:"message"`
}

type StatusResponse struct {
	Status       string `json:"status"`
	Progress     int    `json:"progress"`
	Message      string `json:"message"`
	ErrorMessage string `json:"error_message,omitempty"`
	BackingTrackURL string `json:"backing_track_url,omitempty"`
	OriginalURL     string `json:"original_url,omitempty"`
	GCSFiles        map[string]string `json:"gcs_files,omitempty"`
}

type KaraokeResult struct {
	Success    bool                   `json:"success"`
	AudioName  string                 `json:"audio_name"`
	KaraokeInfo map[string]interface{} `json:"karaoke_info"`
	Files      map[string]string      `json:"files"`
	Error      string                 `json:"error,omitempty"`
}

// Map para armazenar status dos jobs (em produção usar Redis/DB)
var jobStatuses = make(map[string]*JobStatus)

type JobStatus struct {
	Status       string    `json:"status"`
	Progress     int       `json:"progress"`
	Message      string    `json:"message"`
	ErrorMessage string    `json:"error_message,omitempty"`
	StartTime    time.Time `json:"start_time"`
	Files        map[string]string `json:"files,omitempty"`
	GCSFiles     map[string]string `json:"gcs_files,omitempty"`
}

// Map para armazenar usuários (em produção usar database)
var users = make(map[string]*User)

// Serviço GCS global
var gcsService *gcs.GCSService

func main() {
	// Carregar variáveis de ambiente
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ Arquivo .env não encontrado, usando variáveis do sistema")
	}

	// Inicializar GCS
	bucketName := os.Getenv("GCS_BUCKET_NAME")
	credentialsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	
	if bucketName != "" {
		var err error
		gcsService, err = gcs.NewGCSService(bucketName, credentialsPath)
		if err != nil {
			log.Printf("⚠️ Erro ao inicializar GCS: %v", err)
			log.Println("📁 Continuando com armazenamento local")
		} else {
			log.Printf("☁️ GCS inicializado com bucket: %s", bucketName)
		}
	} else {
		log.Println("📁 GCS não configurado, usando armazenamento local")
	}

	r := gin.Default()

	// Configurar CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000", "http://localhost:3001"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"*"}
	config.AllowCredentials = true
	r.Use(cors.New(config))

	// Middleware de logs
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Servir arquivos estáticos (apenas se não estiver usando GCS)
	if gcsService == nil {
		r.Static("/output", "./output")
	}

	// Rotas de autenticação
	r.POST("/signin", signInHandler)
	r.GET("/me", authMiddleware(), getMeHandler)

	// Rotas da API de karaokê
	api := r.Group("/karaoke")
	api.Use(authMiddleware())
	{
		api.POST("/process", processKaraokeHandler)
		api.GET("/status/:job_id", getJobStatusHandler)
	}

	// Outras rotas...
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Println("🎵 AnySong API iniciada na porta 8000")
	log.Fatal(r.Run(":8000"))
}

// Middleware de autenticação simples
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token de autorização necessário"})
			c.Abort()
			return
		}

		// Extrair token (formato: "Bearer <token>")
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Formato de token inválido"})
			c.Abort()
			return
		}

		token := authHeader[7:]
		
		// Verificar se o token existe nos usuários (simulação)
		var foundUser *User
		for _, user := range users {
			if token == user.FirebaseUID { // Simplificação: usando UID como token
				foundUser = user
				break
			}
		}

		if foundUser == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
			c.Abort()
			return
		}

		c.Set("user", foundUser)
		c.Next()
	}
}

func signInHandler(c *gin.Context) {
	var req SignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
		return
	}

	// Simulação de verificação do token Firebase
	// Em produção, você verificaria o token com o Firebase Admin SDK
	log.Printf("📝 Recebido token de login: %s...", req.IDToken[:min(20, len(req.IDToken))])

	// Criar/encontrar usuário (simulação)
	userID := "user_" + uuid.New().String()[:8]
	user := &User{
		FirebaseUID:    req.IDToken, // Simplificação: usando o token como UID
		Email:          "user@example.com",
		Name:           "Usuário Teste",
		ProfilePicture: "https://via.placeholder.com/150",
	}

	// Salvar usuário
	users[userID] = user

	log.Printf("✅ Login realizado com sucesso para usuário: %s", user.Email)

	c.JSON(http.StatusOK, SignInResponse{
		Success: true,
		Message: "Login realizado com sucesso",
		User:    user,
	})
}

func getMeHandler(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user":    user,
	})
}

func processKaraokeHandler(c *gin.Context) {
	// Gerar ID único para o job
	jobID := uuid.New().String()

	// Inicializar status do job
	jobStatuses[jobID] = &JobStatus{
		Status:    "processing",
		Progress:  0,
		Message:   "Iniciando processamento...",
		StartTime: time.Now(),
	}

	// Processar arquivo de upload
	file, header, err := c.Request.FormFile("audio_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Erro ao fazer upload do arquivo: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// Criar diretório de upload se não existir
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Erro ao criar diretório de upload: " + err.Error(),
		})
		return
	}

	// Salvar arquivo temporariamente
	audioPath := filepath.Join(uploadDir, fmt.Sprintf("%s_%s", jobID, header.Filename))
	dst, err := os.Create(audioPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Erro ao salvar arquivo: " + err.Error(),
		})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Erro ao copiar arquivo: " + err.Error(),
		})
		return
	}

	log.Printf("📁 Arquivo salvo: %s (%.2f MB)", audioPath, float64(header.Size)/1024/1024)

	// Processar em background
	go processKaraokeAsync(jobID, audioPath)

	// Retornar resposta imediata
	c.JSON(http.StatusOK, ProcessResponse{
		Success: true,
		JobID:   jobID,
		Message: "Processamento iniciado",
	})
}

func processKaraokeAsync(jobID, audioPath string) {
	jobStatuses[jobID].Status = "processing"
	jobStatuses[jobID].Progress = 10
	jobStatuses[jobID].Message = "Iniciando processamento da música..."

	// Usar o novo script process_song.py com flag --gcs se disponível
	args := []string{"process_song.py", audioPath}
	if gcsService != nil {
		args = append(args, "--gcs")
	}
	
	cmd := exec.Command("python3", args...)
	cmd.Dir = "."

	// Capturar stdout e stderr separadamente
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("❌ Erro ao criar pipe stdout: %v", err)
		jobStatuses[jobID].Status = "failed"
		jobStatuses[jobID].ErrorMessage = fmt.Sprintf("Erro interno: %v", err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("❌ Erro ao criar pipe stderr: %v", err)
		jobStatuses[jobID].Status = "failed"
		jobStatuses[jobID].ErrorMessage = fmt.Sprintf("Erro interno: %v", err)
		return
	}

	// Iniciar o comando
	if err := cmd.Start(); err != nil {
		log.Printf("❌ Erro ao iniciar processo Python: %v", err)
		jobStatuses[jobID].Status = "failed"
		jobStatuses[jobID].ErrorMessage = fmt.Sprintf("Erro ao iniciar processamento: %v", err)
		return
	}

	// Ler stderr em uma goroutine para logs
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("🐍 Python: %s", scanner.Text())
		}
	}()

	// Ler stdout para o JSON
	stdoutBytes, err := io.ReadAll(stdout)
	if err != nil {
		log.Printf("❌ Erro ao ler stdout: %v", err)
		jobStatuses[jobID].Status = "failed"
		jobStatuses[jobID].ErrorMessage = "Erro ao ler resultado do processamento"
		return
	}

	// Aguardar o processo terminar
	if err := cmd.Wait(); err != nil {
		log.Printf("❌ Processo Python terminou com erro: %v", err)
		jobStatuses[jobID].Status = "failed"
		jobStatuses[jobID].ErrorMessage = fmt.Sprintf("Erro no processamento: %v", err)
		return
	}

	// Parse do resultado JSON
	var result map[string]interface{}
	if err := json.Unmarshal(stdoutBytes, &result); err != nil {
		log.Printf("❌ Erro ao fazer parse do JSON: %v", err)
		log.Printf("📄 Output raw: %s", string(stdoutBytes))
		jobStatuses[jobID].Status = "failed"
		jobStatuses[jobID].ErrorMessage = "Erro ao processar resultado"
		return
	}

	// Verificar se houve erro
	if errorMsg, exists := result["error"]; exists {
		log.Printf("❌ Erro no processamento Python: %v", errorMsg)
		jobStatuses[jobID].Status = "failed"
		jobStatuses[jobID].ErrorMessage = fmt.Sprintf("Erro: %v", errorMsg)
		return
	}

	// Atualizar status para upload no GCS se disponível
	if gcsService != nil {
		jobStatuses[jobID].Progress = 70
		jobStatuses[jobID].Message = "Fazendo upload para o bucket..."
		
		// Fazer upload dos arquivos para o GCS
		if songFolder, exists := result["song_folder"]; exists {
			if folderPath, ok := songFolder.(string); ok {
				gcsFolder := ""
				if gcsF, exists := result["gcs_folder"]; exists {
					if gcsFolderStr, ok := gcsF.(string); ok {
						gcsFolder = gcsFolderStr
					}
				}
				
				ctx := context.Background()
				uploadedFiles, err := gcsService.UploadSongFiles(ctx, folderPath, gcsFolder)
				if err != nil {
					log.Printf("⚠️ Erro ao fazer upload para GCS: %v", err)
					jobStatuses[jobID].Message = "Processamento concluído (erro no upload para bucket)"
				} else {
					jobStatuses[jobID].GCSFiles = uploadedFiles
					log.Printf("☁️ Upload para GCS concluído: %d arquivos", len(uploadedFiles))
					
					// Limpar pasta local após upload bem-sucedido
					if err := os.RemoveAll(folderPath); err != nil {
						log.Printf("⚠️ Erro ao limpar pasta local: %v", err)
					}
				}
			}
		}
	}

	// Atualizar status com sucesso
	jobStatuses[jobID].Status = "completed"
	jobStatuses[jobID].Progress = 100
	jobStatuses[jobID].Message = "Processamento concluído com sucesso!"

	// Extrair informações importantes do resultado
	if backingTrackURL, exists := result["backing_track_url"]; exists {
		if url, ok := backingTrackURL.(string); ok {
			if jobStatuses[jobID].Files == nil {
				jobStatuses[jobID].Files = make(map[string]string)
			}
			jobStatuses[jobID].Files["backing_track_url"] = url
		}
	}

	// Extrair arquivos gerados
	if files, exists := result["files"]; exists {
		if filesMap, ok := files.(map[string]interface{}); ok {
			if jobStatuses[jobID].Files == nil {
				jobStatuses[jobID].Files = make(map[string]string)
			}
			
			for key, value := range filesMap {
				if strValue, ok := value.(string); ok {
					jobStatuses[jobID].Files[key] = strValue
				}
			}
		}
	}

	log.Printf("✅ Processamento do job %s concluído com sucesso", jobID)
	
	// Limpar arquivo de upload após processamento bem-sucedido
	if err := os.Remove(audioPath); err != nil {
		log.Printf("⚠️ Aviso: Não foi possível remover arquivo temporário %s: %s", audioPath, err.Error())
	}
}

func getJobStatusHandler(c *gin.Context) {
	jobID := c.Param("job_id")
	
	status, exists := jobStatuses[jobID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Job não encontrado",
		})
		return
	}

	response := StatusResponse{
		Status:   status.Status,
		Progress: status.Progress,
		Message:  status.Message,
	}

	if status.ErrorMessage != "" {
		response.ErrorMessage = status.ErrorMessage
	}

	// Se completado, adicionar URLs dos arquivos
	if status.Status == "completed" {
		if status.Files != nil {
			if backingTrackURL, ok := status.Files["backing_track_url"]; ok {
				response.BackingTrackURL = backingTrackURL
			}
		}
		
		// Se temos arquivos no GCS, incluir eles
		if status.GCSFiles != nil {
			response.GCSFiles = status.GCSFiles
		}
	}

	c.JSON(http.StatusOK, response)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
} 