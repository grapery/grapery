package user

import user_model "../../models/user"

type UserService struct {
}
func (u *UserService) GetUser(Uid int64) {
	var  u user_model.User
	u.UID = Uid
	u.GetUser(u.Uid)


}

func (u *UserService) UpdateUser(){

}