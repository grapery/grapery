package mysql

import (
	"context"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Repository is the main repository struct that implements domain.Repository
// It contains database access methods for all entities
type Repository struct {
	db      *gorm.DB
	log     *zap.Logger
	recoCfg config.RecommendationConfig
	cache   cache.Cache
}

// NewRepository creates a new Repository instance
func NewRepository(db *gorm.DB, log *zap.Logger, recoCfg config.RecommendationConfig) *Repository {
	return &Repository{
		db:      db,
		log:     log,
		recoCfg: recoCfg,
	}
}

// SetCache wires optional Redis for membership cache invalidation on writes.
func (r *Repository) SetCache(c cache.Cache) {
	r.cache = c
}

// DB returns the underlying GORM database instance
func (r *Repository) DB() *gorm.DB {
	return r.db
}

// Log returns the logger
func (r *Repository) Log() *zap.Logger {
	return r.log
}

// WithTransaction executes a function within a database transaction
// If the function returns an error, the transaction is rolled back
// If the function returns nil, the transaction is committed
func (r *Repository) WithTransaction(ctx context.Context, fn func(tx domain.Repository) error) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Create a transaction-scoped repository
	txRepo := &transactionRepository{
		Repository: &Repository{
			db:      tx,
			log:     r.log,
			recoCfg: r.recoCfg,
			cache:   r.cache,
		},
		tx: tx,
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-throw panic after rollback
		}
	}()

	if err := fn(txRepo); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// transactionRepository wraps a Repository with transaction support
type transactionRepository struct {
	*Repository
	tx *gorm.DB
}

// ========== Helper methods for domain conversion ==========

// storyToDomain converts a Story model to domain.Story
func (r *Repository) storyToDomain(s Story) domain.Story {
	if &s == nil {
		return domain.Story{}
	}
	domainStory := ModelToStory(&s)
	if domainStory == nil {
		return domain.Story{}
	}
	return *domainStory
}

// characterToDomain converts a Character model to domain.Character
func (r *Repository) characterToDomain(c Character) domain.Character {
	if &c == nil {
		return domain.Character{}
	}
	domainCharacter := ModelToCharacter(&c)
	if domainCharacter == nil {
		return domain.Character{}
	}
	return *domainCharacter
}

// panelToDomainWithCharacters converts a Panel model to domain.Panel with associated characters
func (r *Repository) panelToDomainWithCharacters(p Panel, characters []*domain.Character) domain.Panel {
	domainPanel := ModelToPanel(&p)
	if domainPanel == nil {
		return domain.Panel{}
	}

	// Characters are already in domain format
	domainCharacters := make([]domain.Character, len(characters))
	for i, c := range characters {
		if c != nil {
			domainCharacters[i] = *c
		}
	}
	domainPanel.Characters = domainCharacters

	return *domainPanel
}

// userToDomain converts a User model to domain.User (non-pointer version)
func (r *Repository) userToDomain(u User) domain.User {
	result := r.userToDomainPtr(&u)
	if result == nil {
		return domain.User{}
	}
	return *result
}
