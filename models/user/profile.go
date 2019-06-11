package user

import "time"

type Profile struct {
	UserProfileID int64 `json:"user_profile_id,omitempty"`
	UserID        int64 `json:"user_id,omitempty"`
	Followers     int64 `json:"followers,omitempty"`
	Following     int64 `json:"following,omitempty"`
	//
	Emotion   int    `json:"emotion,omitempty"`
	ShortDesc string `json:"short_desc,omitempty"`
	//

	CreateAt time.Time `json:"create_at,omitempty"`
	UpdateAt time.Time `json:"update_at,omitempty"`
	Deleted  bool      `json:"deleted,omitempty"`
}

func NewProfile() *Profile {
	return &Profile{}
}
