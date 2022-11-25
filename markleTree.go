/*
* Mieux : https://github.com/cbergoon/merkletree/blob/master/merkle_tree.go
* tout est super bien coder dessus, il n'y a plus qu'à comprendre 
*/





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
	} else if existingNode.right == nil {
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
