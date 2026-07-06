package scanner

import (
	"sync"

	"era/services/vm/internal/models"
)

// Engine координирует параллельное сканирование списка целей по набору шаблонов.
// Поля executor и templates в ходе Run только читаются; не вызывайте Run одновременно
// из нескольких горутин без внешней синхронизации.
type Engine struct {
	executor    Executor
	templates   []*models.Template
	concurrency int
}

// NewEngine создаёт движок с заданным исполнителем, шаблонами и уровнем параллелизма.
// Если concurrency < 1, используется 1 воркер.
func NewEngine(ex Executor, templates []*models.Template, concurrency int) *Engine {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Engine{
		executor:    ex,
		templates:   templates,
		concurrency: concurrency,
	}
}

// Run запускает worker pool: каждая цель обрабатывается одним воркером, для неё
// последовательно выполняются все шаблоны. Находки со всех целей объединяются в один слайс.
func (e *Engine) Run(targets []string) []models.Finding {
	if e == nil || e.executor == nil {
		return nil
	}
	workers := e.concurrency
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan string)
	results := make(chan []models.Finding)

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for target := range jobs {
				results <- e.scanTarget(target)
			}
		}()
	}

	// После завершения всех воркеров закрываем results — только так, иначе возможна гонка с читателем.
	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		for _, t := range targets {
			jobs <- t
		}
		close(jobs)
	}()

	// Сбор результатов выполняется в одной горутине — без мьютекса на итоговый слайс.
	var all []models.Finding
	for batch := range results {
		all = append(all, batch...)
	}
	return all
}

// scanTarget выполняет все шаблоны для одной цели в текущем воркере (без гонок с другими воркерами).
func (e *Engine) scanTarget(target string) []models.Finding {
	var agg []models.Finding
	for _, tpl := range e.templates {
		if tpl == nil {
			continue
		}
		fs, err := e.executor.Execute(target, tpl)
		if err != nil {
			// MVP: пропускаем ошибку шаблона и продолжаем остальные.
			continue
		}
		agg = append(agg, fs...)
	}
	return agg
}
