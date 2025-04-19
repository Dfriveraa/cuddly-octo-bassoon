package repository

import (
	"gorm.io/gorm"

	"tiny-url/internal/domain/errors"
)

// BaseRepository proporciona operaciones comunes para los repositorios
type BaseRepository struct {
	db *gorm.DB
}

// newBaseRepository crea una nueva instancia del repositorio base
func newBaseRepository(db *gorm.DB) BaseRepository {
	return BaseRepository{db: db}
}

// handleGormError maneja los errores comunes de GORM
func (r *BaseRepository) handleGormError(err error, notFoundErr error, message string) error {
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return notFoundErr
		}
		return errors.Wrap(err, message)
	}
	return nil
}

// findOne busca un único registro usando una condición
func (r *BaseRepository) findOne(dest interface{}, condition string, args ...interface{}) error {
	result := r.db.Where(condition, args...).First(dest)
	return result.Error
}

// findById busca un registro por su ID
func (r *BaseRepository) findById(dest interface{}, id uint) error {
	result := r.db.First(dest, id)
	return result.Error
}

// create crea un nuevo registro
func (r *BaseRepository) create(value interface{}) error {
	result := r.db.Create(value)
	return result.Error
}

// update actualiza un registro existente
func (r *BaseRepository) update(value interface{}) (int64, error) {
	result := r.db.Save(value)
	return result.RowsAffected, result.Error
}

// delete elimina un registro
func (r *BaseRepository) delete(value interface{}, condition string, args ...interface{}) (int64, error) {
	result := r.db.Where(condition, args...).Delete(value)
	return result.RowsAffected, result.Error
}

// deleteById elimina un registro por su ID
func (r *BaseRepository) deleteById(value interface{}, id uint) (int64, error) {
	result := r.db.Delete(value, id)
	return result.RowsAffected, result.Error
}

// findAll busca todos los registros con paginación
func (r *BaseRepository) findAll(dest interface{}, limit, offset int) error {
	result := r.db.Limit(limit).Offset(offset).Find(dest)
	return result.Error
}

// updateColumn actualiza una columna específica
func (r *BaseRepository) updateColumn(model interface{}, condition string, columnName string, value interface{}, args ...interface{}) (int64, error) {
	result := r.db.Model(model).Where(condition, args...).UpdateColumn(columnName, value)
	return result.RowsAffected, result.Error
}
