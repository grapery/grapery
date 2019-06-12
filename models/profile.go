package models

type Profile struct {
	Base
	UserID    int64 `json:"user_id,omitempty"`
	Followers int64 `json:"followers,omitempty"`
	Following int64 `json:"following,omitempty"`
	//
	Emotion   int    `json:"emotion,omitempty"`
	ShortDesc string `json:"short_desc,omitempty"`
	//

}

func (p Profile) TableNamse() string {
	return "profile"
}

func (p *Profile) Create() error {
	if !database.NewRecord(a) {
		database.Create(a)
	}
	return nil
}

func (p *Profile) Update() error {
	database.Model(a).Update("password", a.Password)
	return nil
}

func (p *Profile) Get() error {
	database.First(a)
	return nil
}

func (p *Profile) Delete() error {
	database.Delete(a)
	return nil
}
