package thridpart

var thirdParter ThirdParter

func init() {
	thirdParter = NewThirdPart()
}

func GetThirdParter() ThirdParter {
	return thirdParter
}

func NewThirdPart() *ThirdPart {
	return &ThirdPart{}
}

type ThirdParter interface {
}

type ThirdPart struct {
}
