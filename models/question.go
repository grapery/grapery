package models

// base on project
type Question struct {
	IDBase
	UserID    uint64 `json:"user_id,omitempty"`
	GroupID   uint64 `json:"group_id,omitempty"`
	ProjectID uint64 `json:"project_id,omitempty"`
	Tital     string `json:"tital,omitempty"`
	Content   string `json:"description,omitempty"`
	Tags      uint64 `json:"tags,omitempty"`
	RefId     uint64 `json:"ref_id,omitempty"`
}

func (q Question) TableName() string {
	return "question"
}

func GetProjectQuestions(projectID int) (list []*Question, err error) {
	return nil, nil
}

func GetQuestion() (err error) {
	return nil
}

func GetQuestionByTags() (err error) {
	return nil
}

func GetQuestionByTitle() (err error) {
	return nil
}

func CreateQuestion() (err error) {
	return nil
}

func EditQuestion() (err error) {
	return nil
}

func CloseQuestion() (err error) {
	return nil
}

func DeleteQuestion() (err error) {
	return nil
}
