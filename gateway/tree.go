package gateway

import "strings"

// TreeNode represents a node in the routing tree structure.
// 网管路由树节点
type TreeNode struct {
	Name       string
	Children   []*TreeNode // 子节点
	RouterName string      // 路由名称
	IsEnd      bool        // 是否是尾节点标识
	GwName     string      // 网关名称
}

func (t *TreeNode) Put(path string, gwName string) {
	root := t
	strs := strings.Split(path, "/")
	for index, name := range strs {
		if index == 0 {
			continue
		}
		children := t.Children
		isMatch := false
		for _, node := range children {
			if node.Name == name {
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
			node := &TreeNode{
				Name:     name,
				Children: make([]*TreeNode, 0),
				IsEnd:    isEnd,
				GwName:   gwName,
			}
			children = append(children, node)
			t.Children = children
			t = node
		}
	}
	t = root
}

func (t *TreeNode) Get(path string) *TreeNode {
	strs := strings.Split(path, "/")
	routerName := ""
	for index, name := range strs {
		if index == 0 {
			continue
		}
		children := t.Children
		isMatch := false

		for _, node := range children {
			if node.Name == name || node.Name == "*" || strings.Contains(node.Name, ":") {
				isMatch = true
				routerName += "/" + node.Name
				node.RouterName = routerName
				t = node
				if index == len(strs)-1 {
					return node
				}
				break
			}
		}

		if !isMatch {
			for _, node := range children {
				if node.Name == "**" {
					routerName += "/" + node.Name
					node.RouterName = routerName
					return node
				}
			}
		}
	}
	return nil
}
