package service

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"tiny-url/internal/domain/errors"
	"tiny-url/internal/domain/model"
	"tiny-url/internal/domain/ports"
)

type authService struct {
	userRepo ports.UserRepository
	jwtKey   []byte
}

// NewAuthService crea una nueva instancia del servicio de autenticación
func NewAuthService(userRepo ports.UserRepository) ports.AuthService {
	// En un entorno real, esta clave sería obtenida de variables de entorno o un servicio de secretos
	jwtKey := []byte("mi_clave_secreta_muy_segura")
	return &authService{
		userRepo: userRepo,
		jwtKey:   jwtKey,
	}
}

// Register registra un nuevo usuario en el sistema
func (s *authService) Register(ctx context.Context, username, email, password string) (*model.User, string, error) {
	// Comprobar si el usuario ya existe
	existingUser, _ := s.userRepo.GetByUsername(ctx, username)
	if existingUser != nil {
		return nil, "", errors.ErrUserAlreadyExists
	}

	// Comprobar si el email ya existe
	existingEmail, _ := s.userRepo.GetByEmail(ctx, email)
	if existingEmail != nil {
		return nil, "", errors.ErrUserAlreadyExists
	}

	// Hash de la contraseña
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", errors.Wrap(err, "error al hashear la contraseña")
	}

	// Crear el usuario
	user := &model.User{
		Username: username,
		Email:    email,
		Password: string(hashedPassword),
	}

	// Guardar el usuario en la base de datos
	if err := s.userRepo.CreateUser(user); err != nil {
		return nil, "", err
	}

	// Generar token JWT
	token, err := s.GenerateToken(user.ID)
	if err != nil {
		return nil, "", errors.Wrap(err, "error al generar el token")
	}

	return user, token, nil
}

// Login autentica a un usuario y devuelve un token JWT
func (s *authService) Login(username, password string) (string, error) {
	// Buscar al usuario por nombre de usuario
	user, err := s.userRepo.GetByUsername(context.Background(), username)
	if err != nil {
		return "", errors.ErrInvalidCredentials
	}
	if user == nil {
		return "", errors.ErrUserNotFound
	}

	// Verificar la contraseña
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.ErrInvalidCredentials
	}

	// Generar token JWT
	token, err := s.GenerateToken(user.ID)
	if err != nil {
		return "", errors.Wrap(err, "error al generar el token")
	}

	return token, nil
}

// GetUser obtiene un usuario por su ID
func (s *authService) GetUser(ctx context.Context, userID uint) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.ErrUserNotFound
	}
	return user, nil
}

// ValidateToken valida un token JWT y devuelve el ID del usuario
func (s *authService) ValidateToken(tokenString string) (uint, error) {
	claims := &struct {
		UserID uint `json:"user_id"`
		jwt.RegisteredClaims
	}{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return s.jwtKey, nil
	})

	if err != nil {
		return 0, errors.ErrInvalidToken
	}

	if !token.Valid {
		return 0, errors.ErrInvalidToken
	}

	return claims.UserID, nil
}

// generateToken genera un token JWT para un usuario
func (s *authService) GenerateToken(userID uint) (string, error) {
	// Crear claims con la información del usuario
	claims := &struct {
		UserID uint `json:"user_id"`
		jwt.RegisteredClaims
	}{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Token válido por 24 horas
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// Crear token con claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Firmar token con la clave secreta
	tokenString, err := token.SignedString(s.jwtKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
