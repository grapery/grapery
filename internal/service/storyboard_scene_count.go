package service

import "fmt"

const (
	MinStoryboardSceneCount     = 2
	MaxStoryboardSceneCount     = 8
	DefaultStoryboardSceneCount = 3
)

func NormalizeStoryboardSceneCount(sceneCount int) (int, error) {
	if sceneCount == 0 {
		return DefaultStoryboardSceneCount, nil
	}
	if sceneCount < MinStoryboardSceneCount || sceneCount > MaxStoryboardSceneCount {
		return 0, fmt.Errorf("sceneCount must be between %d and %d", MinStoryboardSceneCount, MaxStoryboardSceneCount)
	}
	return sceneCount, nil
}
