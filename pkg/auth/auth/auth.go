package auth

type AuthService struct {
}

func (au *AuthService) Login() (userId interface{}, err error) {
	return
}

func (au *AuthService) Register(user interface{}) (userID interface{}, err error) {

	return
}
func (au *AuthService) ChangePassword(userID interface{}, npwd, opwd string) (userID interface{}, err error) {
	return
}

func (au *AuthService) LockAuthAccout(userID interface{}) (userID interface{}, err error) {
	return
}
