package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"tiny-url/internal/domain/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormService representa el servicio de base de datos utilizando GORM
type GormService struct {
	db *gorm.DB
}

// NewGormService crea una nueva instancia del servicio de base de datos con GORM
func NewGormService() *GormService {
	// Obtener variables de entorno para la conexión
	database := os.Getenv("BLUEPRINT_DB_DATABASE")
	password := os.Getenv("BLUEPRINT_DB_PASSWORD")
	username := os.Getenv("BLUEPRINT_DB_USERNAME")
	port := os.Getenv("BLUEPRINT_DB_PORT")
	host := os.Getenv("BLUEPRINT_DB_HOST")
	schema := os.Getenv("BLUEPRINT_DB_SCHEMA")

	// Construir la cadena de conexión
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable search_path=%s",
		host, username, password, database, port, schema)

	// Configurar el logger de GORM
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
			Colorful:      true,
		},
	)

	// Conectar a la base de datos
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrar el esquema
	err = db.AutoMigrate(&model.URL{}, &model.User{})
	if err != nil {
		log.Fatalf("Failed to migrate database schema: %v", err)
	}

	return &GormService{
		db: db,
	}
}

// GetDB devuelve la instancia de GORM DB
func (s *GormService) GetDB() *gorm.DB {
	return s.db
}

// Health comprueba el estado de la conexión a la base de datos
func (s *GormService) Health() map[string]string {
	stats := make(map[string]string)

	sqlDB, err := s.db.DB()
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db error: %v", err)
		return stats
	}

	// Comprobar la conexión
	err = sqlDB.Ping()
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		return stats
	}

	// Obtener estadísticas
	stats["status"] = "up"
	stats["message"] = "Database is healthy"
	stats["max_open_connections"] = fmt.Sprintf("%d", sqlDB.Stats().MaxOpenConnections)
	stats["open_connections"] = fmt.Sprintf("%d", sqlDB.Stats().OpenConnections)
	stats["in_use"] = fmt.Sprintf("%d", sqlDB.Stats().InUse)
	stats["idle"] = fmt.Sprintf("%d", sqlDB.Stats().Idle)

	return stats
}

// Close cierra la conexión a la base de datos
func (s *GormService) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
