package go_rookie

import (
	"fmt"
	"testing"
)

func TestTreeNode(t *testing.T) {
	root := &treeNode{
		name:     "/",
		children: make([]*treeNode, 0),
	}
	root.Put("/user/get/:id")
	root.Put("/user/create/hello")
	root.Put("/user/create/aaa")
	root.Put("/order/get/aaa")

	node := root.Get("/user/get/1")
	fmt.Println(node)

	node = root.Get("/user/create/hello")
	fmt.Println(node)

	node = root.Get("/user/create/aaa")
	fmt.Println(node)

	node = root.Get("/order/get/aaa")
	fmt.Println(node)
}
