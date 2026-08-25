package service

import (
	"errors"
	"sync"
	"time"

	"awesomeProject/internal/repository"
)

// Интерфейс, который должен реализовать наш репозиторий
type CatalogRepository interface {
	GetProduct(id string) (repository.Product, error)
	ListProducts() ([]repository.Product, error)
}

// catalogListTTL — короткий кэш списка: каталог, корзина и заказы
// дергают ListProducts почти одновременно, а сток за 10с почти не меняется.
const catalogListTTL = 10 * time.Second

type CatalogService struct {
	repo CatalogRepository

	mu       sync.RWMutex
	cached   []repository.Product
	cachedAt time.Time
}

func NewCatalogService(repo CatalogRepository) *CatalogService {
	return &CatalogService{repo: repo}
}

// GetProduct возвращает товар по ID
func (s *CatalogService) GetProduct(id string) (repository.Product, error) {
	// Базовая бизнес-проверка (для ручного тестирования)
	if id == "" {
		return repository.Product{}, errors.New("product_id cannot be empty")
	}

	return s.repo.GetProduct(id)
}

// ListProducts возвращает список товаров
// В будущем сюда можно добавить логику пагинации и искусственные задержки
func (s *CatalogService) ListProducts() ([]repository.Product, error) {
	s.mu.RLock()
	if s.cached != nil && time.Since(s.cachedAt) < catalogListTTL {
		out := s.cached
		s.mu.RUnlock()
		return out, nil
	}
	s.mu.RUnlock()

	products, err := s.repo.ListProducts()
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cached = products
	s.cachedAt = time.Now()
	s.mu.Unlock()
	return products, nil
}
