package elastic

import (
	"fmt"

	"github.com/grapery/grapery/models"
)

type ElasticUser struct {
	*models.User
}

func (e *ElasticUser) Index() string {
	return "user_index"
}

func (e *ElasticUser) Type() string {
	return ""
}
func (e *ElasticUser) ElasticID() string {
	return fmt.Sprintf("%d", e.ID)
}

func (e *ElasticUser) LastUsedTime() int64 {
	return int64(e.UpdatedAt.Unix())
}

func (e *ElasticUser) SetLastUsedTime(int64) {

}

type ElasticGroup struct {
}

func (e *ElasticGroup) Index() string {
	return "group_index"
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
}

func (e *ElasticProject) Index() string {
	return "project_index"
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

type ElasticTeam struct {
}

func (e *ElasticTeam) Index() string {
	return "team_index"
}

func (e *ElasticTeam) Type() string {
	return ""
}
func (e *ElasticTeam) ElasticID() string {
	return ""
}

func (e *ElasticTeam) LastUsedTime() int64 {
	return 0
}

func (e *ElasticTeam) SetLastUsedTime(int64) {

}

type ElasticItem struct {
}

func (e *ElasticItem) Index() string {
	return "item_index"
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
