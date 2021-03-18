package item

var itemServer ItemServer

func init() {
	itemServer = NewItemService()
}

func GetItemServer() ItemServer {
	return itemServer
}

func NewItemService() *ItemService {
	return &ItemService{}
}

type ItemServer interface{}

type ItemService struct{}
