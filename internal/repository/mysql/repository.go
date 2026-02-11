package mysql

import (
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Repository is the main repository struct that implements domain.Repository
// It contains database access methods for all entities
type Repository struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewRepository creates a new Repository instance
func NewRepository(db *gorm.DB, log *zap.Logger) *Repository {
	return &Repository{
		db:  db,
		log: log,
	}
}

// DB returns the underlying GORM database instance
func (r *Repository) DB() *gorm.DB {
	return r.db
}

// Log returns the logger
func (r *Repository) Log() *zap.Logger {
	return r.log
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
