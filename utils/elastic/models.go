package elastic

import (
	"fmt"

	"github.com/grapery/grapery/models"
)

type ElasticUser struct {
	*models.User
}

func (e *ElasticUser) Index() string {
	return e.TableName()
}

func (e *ElasticUser) Type() string {
	return ""
}
func (e *ElasticUser) ElasticID() string {
	return fmt.Sprintf("%d", e.ID)
}

func (e *ElasticUser) LastUsedTime() int64 {
	return int64(e.UpdateAt.Unix())
}

func (e *ElasticUser) SetLastUsedTime(int64) {

}

type ElasticGroup struct {
	*models.Group
}

func (e *ElasticGroup) Index() string {
	return e.TableName()
}

func (e *ElasticGroup) Type() string {
	return ""
}
func (e *ElasticGroup) ElasticID() string {
	return ""
}

func (e *ElasticGroup) LastUsedTime() int64 {
	return 0
}

func (e *ElasticGroup) SetLastUsedTime(int64) {

}

type ElasticStory struct {
	*models.Story
}

func (e *ElasticStory) Index() string {
	return e.TableName()
}

func (e *ElasticStory) Type() string {
	return ""
}
func (e *ElasticStory) ElasticID() string {
	return ""
}

func (e *ElasticStory) LastUsedTime() int64 {
	return 0
}

func (e *ElasticStory) SetLastUsedTime(int64) {

}

type ElasticStoryBoard struct {
	*models.StoryBoard
}

func (e *ElasticStoryBoard) Index() string {
	return e.TableName()
}

func (e *ElasticStoryBoard) Type() string {
	return ""
}
func (e *ElasticStoryBoard) ElasticID() string {
	return ""
}

func (e *ElasticStoryBoard) LastUsedTime() int64 {
	return 0
}

func (e *ElasticStoryBoard) SetLastUsedTime(int64) {

}

type ElasticStoryBoardRole struct {
	*models.StoryBoardRole
}

func (e *ElasticStoryBoardRole) Index() string {
	return e.TableName()
}

func (e *ElasticStoryBoardRole) Type() string {
	return ""
}

func (e *ElasticStoryBoardRole) ElasticID() string {
	return ""
}

func (e *ElasticStoryBoardRole) LastUsedTime() int64 {
	return 0
}

func (e *ElasticStoryBoardRole) SetLastUsedTime(int64) {

}
