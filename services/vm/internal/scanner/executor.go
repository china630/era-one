package scanner

import "era/services/vm/internal/models"

// Executor выполняет шаблон сканирования против одной цели и возвращает находки.
type Executor interface {
	Execute(target string, tpl *models.Template) ([]models.Finding, error)
}
