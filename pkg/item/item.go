package item

var itemServicer ItemServicer

func init() {
	itemServicer = NewItemService()
}

func GetItemServicer() ItemServicer {
	return itemServicer
}

func NewItemService() *ItemService {
	return &ItemService{}
}

type ItemServicer interface{}

type ItemService struct{}
