package services

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"creative-studio-server/models"
	"creative-studio-server/pkg/database"
	"creative-studio-server/pkg/logger"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService() *UserService {
	return &UserService{
		db: database.GetDB(),
	}
}

func (s *UserService) CreateUser(req *models.UserCreateRequest) (*models.User, error) {
	// Check if user already exists
	var existingUser models.User
	if err := s.db.Where("email = ? OR username = ?", req.Email, req.Username).First(&existingUser).Error; err == nil {
		if existingUser.Email == req.Email {
			return nil, errors.New("user with this email already exists")
		}
		return nil, errors.New("user with this username already exists")
	}

	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Role:     "user",
		IsActive: true,
	}

	if err := user.HashPassword(); err != nil {
		logger.Errorf("Failed to hash password: %v", err)
		return nil, errors.New("failed to process password")
	}

	if err := s.db.Create(user).Error; err != nil {
		logger.Errorf("Failed to create user: %v", err)
		return nil, errors.New("failed to create user")
	}

	logger.Infof("User created successfully: %s", user.Email)
	return user, nil
}

func (s *UserService) AuthenticateUser(req *models.UserLoginRequest) (*models.User, error) {
	var user models.User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid credentials")
		}
		logger.Errorf("Failed to find user: %v", err)
		return nil, errors.New("authentication failed")
	}

	if !user.IsActive {
		return nil, errors.New("account is disabled")
	}

	if err := user.CheckPassword(req.Password); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Update last login
	now := time.Now()
	user.LastLogin = &now
	s.db.Save(&user)

	return &user, nil
}

func (s *UserService) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		logger.Errorf("Failed to get user: %v", err)
		return nil, errors.New("failed to get user")
	}

	return &user, nil
}

func (s *UserService) UpdateUser(userID uint, req *models.UserUpdateRequest) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, errors.New("failed to get user")
	}

	// Check for duplicate username/email if they're being changed
	if req.Username != "" && req.Username != user.Username {
		var existingUser models.User
		if err := s.db.Where("username = ? AND id != ?", req.Username, userID).First(&existingUser).Error; err == nil {
			return nil, errors.New("username already taken")
		}
		user.Username = req.Username
	}

	if req.Email != "" && req.Email != user.Email {
		var existingUser models.User
		if err := s.db.Where("email = ? AND id != ?", req.Email, userID).First(&existingUser).Error; err == nil {
			return nil, errors.New("email already taken")
		}
		user.Email = req.Email
	}

	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}

	if err := s.db.Save(&user).Error; err != nil {
		logger.Errorf("Failed to update user: %v", err)
		return nil, errors.New("failed to update user")
	}

	return &user, nil
}

func (s *UserService) ChangePassword(userID uint, currentPassword, newPassword string) error {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return errors.New("user not found")
	}

	if err := user.CheckPassword(currentPassword); err != nil {
		return errors.New("current password is incorrect")
	}

	user.Password = newPassword
	if err := user.HashPassword(); err != nil {
		return errors.New("failed to process new password")
	}

	if err := s.db.Save(&user).Error; err != nil {
		logger.Errorf("Failed to update password: %v", err)
		return errors.New("failed to update password")
	}

	return nil
}

func (s *UserService) DeleteUser(userID uint) error {
	if err := s.db.Delete(&models.User{}, userID).Error; err != nil {
		logger.Errorf("Failed to delete user: %v", err)
		return errors.New("failed to delete user")
	}

	return nil
}

func (s *UserService) ListUsers(page, limit int, role string) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := s.db.Model(&models.User{})
	
	if role != "" {
		query = query.Where("role = ?", role)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}

	return users, total, nil
}