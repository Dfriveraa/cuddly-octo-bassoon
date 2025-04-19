package repository

import (
	"fmt"
	"testing"
	"time"

	"tiny-url/internal/domain/errors"
	"tiny-url/internal/domain/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateUniqueUserData genera datos únicos para cada test de usuario
func generateUniqueUserData(testName string, index int) (string, string) {
	timestamp := time.Now().UnixNano()
	username := fmt.Sprintf("user-%s-%d-%d", testName, index, timestamp)
	email := fmt.Sprintf("%s-%d-%d@example.com", testName, index, timestamp)
	return username, email
}

func TestUserRepository_CreateUser_GetByID(t *testing.T) {
	// Arrange
	tx, ctx, cleanup := setupTest(t)
	defer cleanup()

	repo := NewUserRepository(tx)
	username, email := generateUniqueUserData("create-get", 1)

	user := &model.User{
		Username:  username,
		Email:     email,
		Password:  "password123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Act
	err := repo.CreateUser(user)
	assert.NoError(t, err)
	assert.NotZero(t, user.ID, "El ID de usuario debería haber sido generado")

	// Retrieve the user by ID
	retrievedUser, err := repo.GetByID(ctx, user.ID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, retrievedUser)
	assert.Equal(t, user.ID, retrievedUser.ID)
	assert.Equal(t, user.Username, retrievedUser.Username)
	assert.Equal(t, user.Email, retrievedUser.Email)
	assert.Equal(t, user.Password, retrievedUser.Password)
}

func TestUserRepository_GetByUsername(t *testing.T) {
	// Arrange
	tx, ctx, cleanup := setupTest(t)
	defer cleanup()

	repo := NewUserRepository(tx)
	username, email := generateUniqueUserData("get-username", 1)

	user := &model.User{
		Username:  username,
		Email:     email,
		Password:  "password123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.CreateUser(user)
	require.NoError(t, err)

	// Act
	retrievedUser, err := repo.GetByUsername(ctx, username)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, retrievedUser)
	assert.Equal(t, user.ID, retrievedUser.ID)
	assert.Equal(t, username, retrievedUser.Username)
}

func TestUserRepository_GetByEmail(t *testing.T) {
	// Arrange
	tx, ctx, cleanup := setupTest(t)
	defer cleanup()

	repo := NewUserRepository(tx)
	username, email := generateUniqueUserData("get-email", 1)

	user := &model.User{
		Username:  username,
		Email:     email,
		Password:  "password123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.CreateUser(user)
	require.NoError(t, err)

	// Act
	retrievedUser, err := repo.GetByEmail(ctx, email)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, retrievedUser)
	assert.Equal(t, user.ID, retrievedUser.ID)
	assert.Equal(t, email, retrievedUser.Email)
}

func TestUserRepository_UpdateUser(t *testing.T) {
	// Arrange
	tx, ctx, cleanup := setupTest(t)
	defer cleanup()

	repo := NewUserRepository(tx)
	username, email := generateUniqueUserData("update", 1)

	user := &model.User{
		Username:  username,
		Email:     email,
		Password:  "password123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.CreateUser(user)
	require.NoError(t, err)

	// Act - Update the user
	updatedEmail := fmt.Sprintf("updated-%s", email)
	user.Email = updatedEmail
	err = repo.UpdateUser(user)

	// Assert
	assert.NoError(t, err)

	// Verify the update
	updatedUser, err := repo.GetByID(ctx, user.ID)
	assert.NoError(t, err)
	assert.Equal(t, updatedEmail, updatedUser.Email)
}

func TestUserRepository_DeleteUser(t *testing.T) {
	// Arrange
	tx, ctx, cleanup := setupTest(t)
	defer cleanup()

	repo := NewUserRepository(tx)
	username, email := generateUniqueUserData("delete", 1)

	user := &model.User{
		Username:  username,
		Email:     email,
		Password:  "password123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.CreateUser(user)
	require.NoError(t, err)

	// Act
	err = repo.DeleteUser(user.ID)

	// Assert
	assert.NoError(t, err)

	// Verify it's deleted
	_, err = repo.GetByID(ctx, user.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errors.ErrUserNotFound))
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	// Arrange
	tx, ctx, cleanup := setupTest(t)
	defer cleanup()

	repo := NewUserRepository(tx)

	// Act
	_, err := repo.GetByID(ctx, 9999) // Un ID que no debería existir

	// Assert
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errors.ErrUserNotFound))
}

func TestUserRepository_GetByUsername_NotFound(t *testing.T) {
	// Arrange
	tx, ctx, cleanup := setupTest(t)
	defer cleanup()

	repo := NewUserRepository(tx)

	// Act
	_, err := repo.GetByUsername(ctx, "nonexistentuser")

	// Assert
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errors.ErrUserNotFound))
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	// Arrange
	tx, ctx, cleanup := setupTest(t)
	defer cleanup()

	repo := NewUserRepository(tx)

	// Act
	_, err := repo.GetByEmail(ctx, "nonexistent@example.com")

	// Assert
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errors.ErrUserNotFound))
}
