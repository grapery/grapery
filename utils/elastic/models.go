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

type ElasticProject struct {
	*models.Project
}

func (e *ElasticProject) Index() string {
	return e.TableName()
}

func (e *ElasticProject) Type() string {
	return ""
}
func (e *ElasticProject) ElasticID() string {
	return ""
}

func (e *ElasticProject) LastUsedTime() int64 {
	return 0
}

func (e *ElasticProject) SetLastUsedTime(int64) {

}

type ElasticItem struct {
	*models.StoryItem
}

func (e *ElasticItem) Index() string {
	return e.TableName()
}

func (e *ElasticItem) Type() string {
	return ""
}
func (e *ElasticItem) ElasticID() string {
	return ""
}

func (e *ElasticItem) LastUsedTime() int64 {
	return 0
}

func (e *ElasticItem) SetLastUsedTime(int64) {

}
