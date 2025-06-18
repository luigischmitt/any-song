package api

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type UserInfo struct {
	UID   string `json:"uid"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (api *API) AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extrair token do header Authorization
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token de autorização necessário",
			})
		}

		// Remover "Bearer " do início do token
		token := strings.Replace(authHeader, "Bearer ", "", 1)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token inválido",
			})
		}

		// Verificar token com Firebase
		decodedToken, err := api.Auth.VerifyIDToken(context.Background(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token inválido ou expirado",
			})
		}

		userInfo := UserInfo{
			UID: decodedToken.UID,
		}
		if email, ok := decodedToken.Claims["email"].(string); ok {
			userInfo.Email = email
		}
		if name, ok := decodedToken.Claims["name"].(string); ok {
			userInfo.Name = name
		}

		c.Locals("user", userInfo)

		return c.Next()
	}
}

func (api *API) AdminRequiredMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := GetUserFromContext(c)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Acesso negado. Falha na autenticação."})
		}

		adminListKey := "groups:admins:members"
		var adminUIDs []string

		// 1. Tenta buscar no CACHE
		found, err := api.CacheService.Get(adminListKey, &adminUIDs)
		if err != nil {
			log.Printf("AVISO: Erro no cache de admins: %v. Buscando no DB.", err)
		}

		// 2. CACHE HIT
		if found {
			for _, adminUID := range adminUIDs {
				if adminUID == user.UID {
					return c.Next()
				}
			}
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Acesso negado. Requer permissão de administrador."})
		}

		// 3. CACHE MISS, busca no FIRESTORE
		doc, err := api.Firestore.Collection("groups").Doc("admins").Get(c.Context())
		if err != nil {
			log.Printf("ERRO: Falha ao buscar grupo de admins no Firestore: %v", err)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Acesso negado."})
		}

		// --- BLOCO CORRIGIDO ---
		data := doc.Data()
		if data == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Acesso negado. Grupo de admins não encontrado."})
		}

		membersData, keyExists := data["members"]
		if !keyExists {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Acesso negado. Grupo de admins mal configurado."})
		}
		// --- FIM DO BLOCO CORRIGIDO ---

		isUserAdmin := false
		// Converte o `interface{}` para um slice de `interface{}`
		if membersSlice, ok := membersData.([]interface{}); ok {
			// Itera sobre o slice para extrair os UIDs como strings
			for _, member := range membersSlice {
				if uid, ok := member.(string); ok {
					adminUIDs = append(adminUIDs, uid)
					if uid == user.UID {
						isUserAdmin = true
					}
				}
			}
		}

		// 4. Salva a lista recém-buscada no CACHE
		api.CacheService.Set(adminListKey, adminUIDs, 15*time.Minute)

		if isUserAdmin {
			return c.Next()
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Acesso negado. Requer permissão de administrador."})
	}
}
func GetUserFromContext(c *fiber.Ctx) (UserInfo, bool) {
	user := c.Locals("user")
	if user == nil {
		return UserInfo{}, false
	}

	userInfo, ok := user.(UserInfo)
	return userInfo, ok
}
