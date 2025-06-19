package redis

import (
	"context"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
)

func ConnectRedis() *redis.Client {
	redisAddr := os.Getenv("REDIS_ADDR")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	// Fallback se REDIS_ADDR não estiver definido
	if redisAddr == "" {
		redisAddr = "redis:6379"
		log.Println("REDIS_ADDR não definido, usando fallback: redis:6379")
	}
	
	log.Printf("Tentando conectar ao Redis em: %s", redisAddr)

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
	})

	// Testa a conexão
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("Erro ao conectar com Redis: %v", err)
	}

	log.Println("Cliente Redis conectado com sucesso!")
	return rdb
}
