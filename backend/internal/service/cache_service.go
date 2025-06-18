package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type CacheService struct {
	redisClient *redis.Client
}

func NewCacheService(client *redis.Client) *CacheService {
	return &CacheService{redisClient: client}
}

// Set serializa um valor para JSON e o armazena no cache com um TTL.
// Aceita qualquer tipo de valor (`interface{}`) e o transforma em JSON.
func (s *CacheService) Set(key string, value interface{}, ttl time.Duration) error {
	// 1. Serializar o valor para o formato JSON, que é um texto.
	// O Redis armazena dados como texto ou bytes.
	dataToCache, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("erro ao serializar valor para o cache (chave: %s): %w", key, err)
	}

	// 2. Chamar o comando SET do Redis.
	// O `context.Background()` é usado porque esta é uma operação rápida.
	err = s.redisClient.Set(context.Background(), key, dataToCache, ttl).Err()
	if err != nil {
		return fmt.Errorf("erro ao salvar no redis (chave: %s): %w", key, err)
	}

	return nil
}

// Get recupera um item do cache e o desserializa para a estrutura de destino.
// `dest` deve ser um ponteiro para a variável onde você quer guardar o resultado (ex: &models.User{}).
// Retorna `true` se encontrou (Cache Hit), ou `false` se não encontrou (Cache Miss).
func (s *CacheService) Get(key string, dest interface{}) (bool, error) {
	// 1. Chamar o comando GET do Redis.
	val, err := s.redisClient.Get(context.Background(), key).Bytes()

	// 2. Checar o tipo de erro. Se for `redis.Nil`, significa que a chave não existe.
	// Isso é um "Cache Miss", e não um erro do sistema.
	if err == redis.Nil {
		return false, nil // Cache Miss
	} else if err != nil {
		return false, fmt.Errorf("erro ao obter do redis (chave: %s): %w", key, err)
	}

	// 3. Se encontrou o valor (que está em formato JSON), desserializa ele
	// para dentro da variável de destino (`dest`) que foi passada como ponteiro.
	err = json.Unmarshal(val, dest)
	if err != nil {
		return false, fmt.Errorf("erro ao desserializar valor do cache (chave: %s): %w", key, err)
	}

	return true, nil 
}

// Delete remove uma ou mais chaves do cache. É usado para invalidação.
func (s *CacheService) Delete(keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	// O comando DEL do Redis pode remover múltiplas chaves de uma vez.
	err := s.redisClient.Del(context.Background(), keys...).Err()
	if err != nil {
		return fmt.Errorf("erro ao deletar chaves do redis: %w", err)
	}

	return nil
}
