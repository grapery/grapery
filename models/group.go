package models

// Group ...
type Group struct {
	Base
	Name      string `json:"name,omitempty"`
	ShortDesc string `json:"short_desc,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Gtype     string `json:"gtype,omitempty"`
	Members   int    `json:"members,omitempty"`
	CreatorID int64  `json:"creator_id,omitempty"`
	IsPrivate bool   `json:"is_private,omitempty"`
}

func (g Group) TableNamse() string {
	return "group"
}

func (g *Group) Create() error {
	if !database.NewRecord(g) {
		database.Create(g)
	}
	return nil
}

func (g *Group) Update() error {
	database.Model(g).Update("short_desc", g.ShortDesc)
	return nil
}

func (g *Group) Get() error {
	database.First(g)
	return nil
}

func (g *Group) Delete() error {
	database.Delete(g)
	return nil
}
