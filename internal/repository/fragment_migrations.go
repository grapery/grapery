package repository

import (
	"fmt"

	"github.com/grapestree/voyager/grapery/internal/domain"
	"gorm.io/gorm"
)

// MigrateFragments runs fragment-related migrations
func MigrateFragments(db *gorm.DB) error {
	// Check if fragments table already exists
	if db.Migrator().HasTable(&domain.Fragment{}) {
		return nil // Skip if already migrated
	}

	// Create fragments table
	if err := db.AutoMigrate(&domain.Fragment{}); err != nil {
		return fmt.Errorf("failed to create fragments table: %w", err)
	}

	// Create fragment_likes table
	if err := db.AutoMigrate(&domain.FragmentLike{}); err != nil {
		return fmt.Errorf("failed to create fragment_likes table: %w", err)
	}

	// Create indexes
	if err := createFragmentIndexes(db); err != nil {
		return fmt.Errorf("failed to create fragment indexes: %w", err)
	}

	return nil
}

// createFragmentIndexes creates indexes for fragments table
func createFragmentIndexes(db *gorm.DB) error {
	// Index on creator_id for quick user fragment lookup
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_fragments_creator_id ON fragments(creator_id)").Error; err != nil {
		// Ignore error if index already exists
		fmt.Printf("Warning: failed to create idx_fragments_creator_id: %v\n", err)
	}

	// Index on visibility for filtering by visibility
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_fragments_visibility ON fragments(visibility)").Error; err != nil {
		fmt.Printf("Warning: failed to create idx_fragments_visibility: %v\n", err)
	}

	// Index on created_at for sorting
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_fragments_created_at ON fragments(created_at DESC)").Error; err != nil {
		fmt.Printf("Warning: failed to create idx_fragments_created_at: %v\n", err)
	}

	// Composite index for visibility + created_at
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_fragments_visibility_created ON fragments(visibility, created_at DESC)").Error; err != nil {
		fmt.Printf("Warning: failed to create idx_fragments_visibility_created: %v\n", err)
	}

	// Indexes for fragment_likes table
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_fragment_likes_fragment_id ON fragment_likes(fragment_id)").Error; err != nil {
		fmt.Printf("Warning: failed to create idx_fragment_likes_fragment_id: %v\n", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_fragment_likes_user_id ON fragment_likes(user_id)").Error; err != nil {
		fmt.Printf("Warning: failed to create idx_fragment_likes_user_id: %v\n", err)
	}

	// Composite index for fragment_likes (unique check)
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_fragment_likes_fragment_user ON fragment_likes(fragment_id, user_id)").Error; err != nil {
		fmt.Printf("Warning: failed to create idx_fragment_likes_fragment_user: %v\n", err)
	}

	return nil
}
