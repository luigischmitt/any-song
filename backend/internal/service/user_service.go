package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/josevitorrodriguess/any-song/backend/internal/models"
	"gorm.io/gorm"
)

type UserService struct {
	DB    *gorm.DB
	cache *CacheService
}

func NewUserService(db *gorm.DB, cache *CacheService) *UserService {
	return &UserService{
		DB:    db,
		cache: cache,
	}
}

func (s *UserService) CreateUser(user *models.User) error {
	return s.DB.Create(user).Error
}

func (s *UserService) GetUserByEmail(email string) (*models.User, error) {
	cacheKey := fmt.Sprintf("user:email:%s", email)
	var user models.User

	found, err := s.cache.Get(cacheKey, &user)
	if err != nil {
		log.Printf("AVISO: Erro no cache ao buscar por email %s: %v", email, err)
	}
	if found {
		return &user, nil
	}
	if err := s.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Usuário não encontrado - retorna nil sem erro
			return nil, nil
		}
		// Outros erros são retornados
		return nil, err
	}

	s.cache.Set(cacheKey, user, 1*time.Hour)
	return &user, nil
}

func (s *UserService) GetUserByName(name string) (*models.User, error) {
	var user models.User
	if err := s.DB.Where("name = ?", name).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("usuário com nome '%s' não encontrado", name)
		}
		return nil, err
	}
	return &user, nil
}

func (s *UserService) GetUserByFirebaseUID(firebaseUID string) (*models.User, error) {
	cacheKey := fmt.Sprintf("user:uid:%s", firebaseUID)
	var user models.User

	found, err := s.cache.Get(cacheKey, &user)
	if err != nil {
		log.Printf("AVISO: Erro no cache ao buscar por UID %s: %v", firebaseUID, err)
	}
	if found {
		return &user, nil
	}

	if err := s.DB.Where("firebase_uid = ?", firebaseUID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("usuário com Firebase UID '%s' não encontrado", firebaseUID)
		}
		return nil, err
	}
	s.cache.Set(cacheKey, user, 1*time.Hour)
	return &user, nil
}

func (s *UserService) UpdateUser(user *models.User) error {
	if err := s.DB.Save(user).Error; err != nil {
		return err
	}

	uidKey := fmt.Sprintf("user:uid:%s", user.FirebaseUID)
	emailKey := fmt.Sprintf("user:email:%s", user.Email)
	s.cache.Delete(uidKey, emailKey)

	return nil
}

func (s *UserService) DeleteUser(firebaseUID string) error {
	user, err := s.GetUserByFirebaseUID(firebaseUID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("usuário não encontrado para deletar")
	}

	if err := s.DB.Where("firebase_uid = ?", firebaseUID).Delete(&models.User{}).Error; err != nil {
		return err
	}

	uidKey := fmt.Sprintf("user:uid:%s", user.FirebaseUID)
	emailKey := fmt.Sprintf("user:email:%s", user.Email)
	s.cache.Delete(uidKey, emailKey)

	return nil
}
