package repository

import (
	"context"
	"testing"
	"time"

	"tiny-url/internal/domain/errors"
	"tiny-url/internal/domain/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Integration(t *testing.T) {
	// Asegurarse de que la tabla de usuarios esté limpia
	testDB.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")

	// Configurar el repositorio de usuarios
	repo := NewUserRepository(testDB)
	ctx := context.Background()

	t.Run("CreateUser and GetByID", func(t *testing.T) {
		// Arrange
		user := &model.User{
			Username:  "testuser",
			Email:     "test@example.com",
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
	})

	t.Run("GetByUsername", func(t *testing.T) {
		// Arrange
		testDB.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
		username := "usernametest"
		user := &model.User{
			Username:  username,
			Email:     "username@example.com",
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
	})

	t.Run("GetByEmail", func(t *testing.T) {
		// Arrange
		testDB.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
		email := "email@example.com"
		user := &model.User{
			Username:  "emailtest",
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
	})

	t.Run("UpdateUser", func(t *testing.T) {
		// Arrange
		testDB.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
		user := &model.User{
			Username:  "updatetest",
			Email:     "update@example.com",
			Password:  "password123",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.CreateUser(user)
		require.NoError(t, err)

		// Act - Update the user
		updatedEmail := "updated@example.com"
		user.Email = updatedEmail
		err = repo.UpdateUser(user)

		// Assert
		assert.NoError(t, err)

		// Verify the update
		updatedUser, err := repo.GetByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, updatedEmail, updatedUser.Email)
	})

	t.Run("DeleteUser", func(t *testing.T) {
		// Arrange
		testDB.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
		user := &model.User{
			Username:  "deletetest",
			Email:     "delete@example.com",
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
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		// Act
		_, err := repo.GetByID(ctx, 9999) // Un ID que no debería existir

		// Assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, errors.ErrUserNotFound))
	})

	t.Run("GetByUsername_NotFound", func(t *testing.T) {
		// Act
		_, err := repo.GetByUsername(ctx, "nonexistentuser")

		// Assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, errors.ErrUserNotFound))
	})

	t.Run("GetByEmail_NotFound", func(t *testing.T) {
		// Act
		_, err := repo.GetByEmail(ctx, "nonexistent@example.com")

		// Assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, errors.ErrUserNotFound))
	})
}
