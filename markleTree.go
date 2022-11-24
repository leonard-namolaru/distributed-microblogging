package main

import "fmt"

type MarkleNode struct {
	right *MarkleNode
	left  *MarkleNode
	hash  string
	label string
}

type MarkleTree struct {
	root *MarkleNode
}

func (markleTree *MarkleTree) InitMarkleTree(rootNode *MarkleNode) {
	markleTree.root = rootNode
}

func (markleTree *MarkleTree) addNode(existingNode *MarkleNode, newNode *MarkleNode) bool {

	if existingNode.left == nil {
		existingNode.left = newNode
	}

	if existingNode.right == nil {
		existingNode.left = newNode
	}
	return true
}

func main() {
	var markleTree MarkleTree
	var rootNode = &MarkleNode{right: nil, left: nil, hash: "", label: "1"}
	markleTree.InitMarkleTree(rootNode)

	fmt.Printf("Root node label : %s \n", markleTree.root.label)
}
