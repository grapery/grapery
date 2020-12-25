package graph

import (
	"github.com/dgraph-io/dgo/v200"
)

var globalGraphDB *dgo.Dgraph

func GetGrapthDB() *dgo.Dgraph {
	return globalGraphDB
}

func newGraphDBClient() {

}
