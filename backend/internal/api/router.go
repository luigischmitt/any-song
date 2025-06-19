package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func (api *API) SetupRoutes() {
	// CORS middleware
	api.Router.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000",
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	}))

	api.Router.Post("/signin", api.SignInHandler)
	api.Router.Post("/logout", api.AuthMiddleware(), api.LogoutHandler)

	userRoutes := api.Router.Group("/user", api.AuthMiddleware())
	userRoutes.Get("/:username", api.FindUserByNameHandler)
	userRoutes.Put("/update", api.UpdateUserHandler)
	userRoutes.Delete("/deleteAccount", api.DeleteUserHandler)

	artistRoutes := api.Router.Group("/artist")
	artistRoutes.Post("/create", api.CreateArtistHandler)
	artistRoutes.Get("/search", api.SearchArtistsHandler)
	artistRoutes.Get("/id/:id", api.GetArtistByIDHandler)
	artistRoutes.Get("/", api.GetAllArtistsHandler)
	artistRoutes.Put("/update", api.UpdateArtistHandler)
	artistRoutes.Delete("/delete/:id", api.DeleteArtistHandler)

	// Song download route
	api.Router.Post("/download-song", api.AuthMiddleware(), api.DownloadSongHandler)

	// Song search route
	api.Router.Post("/search-song", api.AuthMiddleware(), api.SearchSongHandler)

	// Lyrics route - Nova rota adicionada
	api.Router.Post("/lyrics/fetch", api.AuthMiddleware(), api.CatchLyricsHandler)

	// Transcription routes 
	api.Router.Post("/transcribe", api.AuthMiddleware(), api.TranscribeAudioHandler)

	// Karaoke processing routes
	api.Router.Post("/karaoke/process", api.AuthMiddleware(), api.ProcessKaraokeHandler)
	api.Router.Get("/karaoke/status/:job_id", api.AuthMiddleware(), api.GetKaraokeJobStatusHandler)

	// Audio files route
	api.Router.Get("/audio-files", api.AuthMiddleware(), api.ListAudioFilesHandler)

	api.Router.Get("/protected", api.AuthMiddleware(), api.ProtectedHandler)
	api.Router.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
}

func (api *API) ProtectedHandler(c *fiber.Ctx) error {
	user, exists := GetUserFromContext(c)
	if !exists {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Usuário não encontrado",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Rota protegida acessada com sucesso!",
		"user":    user,
	})
}
