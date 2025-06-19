package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type LyricsRequest struct {
	MusicName string `json:"music_name" validate:"required"`
	Timeout   int    `json:"timeout,omitempty"` // em segundos
}

type LyricsResponse struct {
	Lyrics    string `json:"lyrics"`
	TrackID   string `json:"track_id"`
	MusicName string `json:"music_name"`
	FilePath  string `json:"file_path"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Duration  string `json:"duration"`
}

func (api *API) CatchLyricsHandler(c *fiber.Ctx) error {
	var req LyricsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// Validar se o nome da música foi fornecido
	if strings.TrimSpace(req.MusicName) == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "music_name is required",
		})
	}

	// Definir timeout (padrão: 30 segundos)
	timeout := 30 * time.Second
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()

	// Executar o script Python
	result, err := api.executeCatchLyricsScript(ctx, req.MusicName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":    "Failed to execute lyrics script",
			"details":  err.Error(),
			"duration": time.Since(start).String(),
		})
	}

	result.Duration = time.Since(start).String()
	
	if !result.Success {
		return c.Status(500).JSON(result)
	}

	return c.JSON(result)
}

func (api *API) executeCatchLyricsScript(ctx context.Context, musicName string) (*LyricsResponse, error) {
	// Criar um script temporário que chama a função lyrics com o parâmetro
	pythonCode := fmt.Sprintf(`
import sys
import os
sys.path.append('%s')
from catch_lyrics import lyrics
import json

try:
    lyrics('%s')
    
    # Ler o arquivo JSON gerado
    lyrics_file = '/root/any-song/backend/utils/lyrics/%s.json'
    if os.path.exists(lyrics_file):
        with open(lyrics_file, 'r', encoding='utf-8') as f:
            data = json.load(f)
        
        result = {
            'success': True,
            'lyrics': data.get('lyrics', ''),
            'track_id': str(data.get('track_id', '')),
            'music_name': data.get('music_name', ''),
            'file_path': lyrics_file
        }
    else:
        result = {
            'success': False,
            'error': 'Lyrics file not created'
        }
    
    print(json.dumps(result))
    
except Exception as e:
    result = {
        'success': False,
        'error': str(e)
    }
    print(json.dumps(result))
    sys.exit(1)
`, filepath.Join("backend", "utils"), musicName, musicName)

	// Executar o código Python
	cmd := exec.CommandContext(ctx, "python3", "-c", pythonCode)
	
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return &LyricsResponse{
			Success: false,
			Error:   fmt.Sprintf("Script execution failed: %s. Output: %s", err.Error(), string(output)),
		}, nil
	}

	// Parse do resultado JSON
	var result LyricsResponse
	if err := json.Unmarshal(output, &result); err != nil {
		return &LyricsResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse script output: %s. Raw output: %s", err.Error(), string(output)),
		}, nil
	}
	return &result, nil
}