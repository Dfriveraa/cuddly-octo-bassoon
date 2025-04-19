package service

import (
	"context"
	"testing"
	"time"

	domainErrors "tiny-url/internal/domain/errors"
	"tiny-url/internal/domain/model"
	"tiny-url/internal/domain/ports/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestRegister_Success(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockUserRepository(t)
	service := NewAuthService(mockRepo)

	username := "testuser"
	email := "test@example.com"
	password := "password123"
	ctx := context.Background()

	// Configurar el comportamiento del mock
	mockRepo.EXPECT().GetByUsername(ctx, username).Return(nil, domainErrors.ErrUserNotFound)
	mockRepo.EXPECT().GetByEmail(ctx, email).Return(nil, domainErrors.ErrUserNotFound)
	mockRepo.EXPECT().CreateUser(mock.AnythingOfType("*model.User")).Return(nil)

	// Act
	user, token, err := service.Register(ctx, username, email, password)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotEmpty(t, token)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, email, user.Email)
	assert.NotEqual(t, password, user.Password) // La contraseña debe estar hasheada
}

func TestRegister_UsernameExists(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockUserRepository(t)
	service := NewAuthService(mockRepo)

	username := "existinguser"
	email := "new@example.com"
	password := "password123"
	ctx := context.Background()

	existingUser := &model.User{
		ID:        1,
		Username:  username,
		Email:     "existing@example.com",
		Password:  "hashedpassword",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Configurar el comportamiento del mock
	mockRepo.EXPECT().GetByUsername(ctx, username).Return(existingUser, nil)

	// Act
	user, token, err := service.Register(ctx, username, email, password)

	// Assert
	assert.Error(t, err)
	assert.True(t, domainErrors.Is(err, domainErrors.ErrUserAlreadyExists))
	assert.Nil(t, user)
	assert.Empty(t, token)
}

func TestRegister_EmailExists(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockUserRepository(t)
	service := NewAuthService(mockRepo)

	username := "newuser"
	email := "existing@example.com"
	password := "password123"
	ctx := context.Background()

	existingUser := &model.User{
		ID:        1,
		Username:  "otheruser",
		Email:     email,
		Password:  "hashedpassword",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Configurar el comportamiento del mock
	mockRepo.EXPECT().GetByUsername(ctx, username).Return(nil, domainErrors.ErrUserNotFound)
	mockRepo.EXPECT().GetByEmail(ctx, email).Return(existingUser, nil)

	// Act
	user, token, err := service.Register(ctx, username, email, password)

	// Assert
	assert.Error(t, err)
	assert.True(t, domainErrors.Is(err, domainErrors.ErrUserAlreadyExists))
	assert.Nil(t, user)
	assert.Empty(t, token)
}

func TestLogin_Success(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockUserRepository(t)
	service := NewAuthService(mockRepo)

	username := "testuser"
	password := "password123"
	ctx := context.Background()

	// Crear un usuario de prueba para verificar el login
	// Encriptar la contraseña como lo haría el servicio real
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := &model.User{
		ID:       1,
		Username: username,
		Password: string(hashedPassword),
	}

	// Configurar el comportamiento del mock
	mockRepo.EXPECT().GetByUsername(ctx, username).Return(user, nil)

	// Act
	token, err := service.Login(username, password)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockUserRepository(t)
	service := NewAuthService(mockRepo)

	username := "testuser"
	correctPassword := "correctpassword"
	wrongPassword := "wrongpassword"
	ctx := context.Background()

	// Crear un usuario de prueba con la contraseña correcta
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(correctPassword), bcrypt.DefaultCost)
	user := &model.User{
		ID:       1,
		Username: username,
		Password: string(hashedPassword),
	}

	// Configurar el comportamiento del mock
	mockRepo.EXPECT().GetByUsername(ctx, username).Return(user, nil)

	// Act - Intentar login con contraseña incorrecta
	token, err := service.Login(username, wrongPassword)

	// Assert
	assert.Error(t, err)
	assert.True(t, domainErrors.Is(err, domainErrors.ErrInvalidCredentials))
	assert.Empty(t, token)
}

func TestLogin_UserNotFound(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockUserRepository(t)
	service := NewAuthService(mockRepo)

	username := "nonexistentuser"
	password := "password123"
	ctx := context.Background()

	// Configurar el comportamiento del mock
	mockRepo.EXPECT().GetByUsername(ctx, username).Return(nil, domainErrors.ErrUserNotFound)

	// Act
	token, err := service.Login(username, password)

	// Assert
	assert.Error(t, err)
	assert.True(t, domainErrors.Is(err, domainErrors.ErrInvalidCredentials))
	assert.Empty(t, token)
}

func TestGetUser_Success(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockUserRepository(t)
	service := NewAuthService(mockRepo)

	userID := uint(1)
	ctx := context.Background()
	expectedUser := &model.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	// Configurar el comportamiento del mock
	mockRepo.EXPECT().GetByID(ctx, userID).Return(expectedUser, nil)

	// Act
	user, err := service.GetUser(ctx, userID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
}

func TestGetUser_NotFound(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockUserRepository(t)
	service := NewAuthService(mockRepo)

	userID := uint(999)
	ctx := context.Background()

	// Configurar el comportamiento del mock
	mockRepo.EXPECT().GetByID(ctx, userID).Return(nil, domainErrors.ErrUserNotFound)

	// Act
	user, err := service.GetUser(ctx, userID)

	// Assert
	assert.Error(t, err)
	assert.True(t, domainErrors.Is(err, domainErrors.ErrUserNotFound))
	assert.Nil(t, user)
}

func TestValidateToken_Success(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockUserRepository(t)
	service := NewAuthService(mockRepo)

	userID := uint(1)

	// Generar un token real
	token, err := service.GenerateToken(userID)
	assert.NoError(t, err)

	// Act
	resultUserID, err := service.ValidateToken(token)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, userID, resultUserID)
}

func TestValidateToken_Invalid(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockUserRepository(t)
	service := NewAuthService(mockRepo)

	// Act
	userID, err := service.ValidateToken("invalid.token.string")

	// Assert
	assert.Error(t, err)
	assert.True(t, domainErrors.Is(err, domainErrors.ErrInvalidToken))
	assert.Equal(t, uint(0), userID)
}
