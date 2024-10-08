package go_rookie

import "strings"

type treeNode struct {
	name       string
	children   []*treeNode
	routerName string
	isEnd      bool // 是否是尾节点标识
}

func (t *treeNode) Put(path string) {
	root := t
	strs := strings.Split(path, "/")
	for index, name := range strs {
		if index == 0 {
			continue
		}
		children := t.children
		isMatch := false
		for _, node := range children {
			if node.name == name {
				isMatch = true
				t = node
				break
			}
		}
		if !isMatch {
			isEnd := false
			if index == len(strs)-1 {
				isEnd = true
			}
			node := &treeNode{
				name:     name,
				children: make([]*treeNode, 0),
				isEnd:    isEnd,
			}
			children = append(children, node)
			t.children = children
			t = node
		}
	}
	t = root
}

func (t *treeNode) Get(path string) *treeNode {
	strs := strings.Split(path, "/")
	routerName := ""
	for index, name := range strs {
		if index == 0 {
			continue
		}
		children := t.children
		isMatch := false

		for _, node := range children {
			if node.name == name || node.name == "*" || strings.Contains(node.name, ":") {
				isMatch = true
				routerName += "/" + node.name
				node.routerName = routerName
				t = node
				if index == len(strs)-1 {
					return node
				}
				break
			}
		}

		if !isMatch {
			for _, node := range children {
				if node.name == "**" {
					routerName += "/" + node.name
					node.routerName = routerName
					return node
				}
			}
		}
	}
	return nil
}
