package main

import (
	"crypto/sha256"
	"fmt"
	"os"
)

type MerkleNode struct {
	ParentNode *MerkleNode
	Children   []*MerkleNode
	IsLeaf     bool
	Hash       []byte
}

type MerkleTree struct {
	Root     *MerkleNode
	MaxArity int
}

func CreateTree(UdpDatagrams [][]byte, maxArity int) *MerkleTree {
	var merkleTree MerkleTree
	merkleTree.MaxArity = maxArity

	leafs := merkleTree.createLeafNodes(UdpDatagrams)

	root := merkleTree.createNodes(leafs)
	merkleTree.Root = root

	return &merkleTree
}

func (merkleTree *MerkleTree) createLeafNodes(UdpDatagrams [][]byte) []*MerkleNode {
	var leafs []*MerkleNode

	for i := 0; i < len(UdpDatagrams); i++ {
		hash := sha256.New()

		_, errorMessage := hash.Write(UdpDatagrams[i])
		if errorMessage != nil {
			os.Exit(1)
		}

		leafs = append(leafs, &MerkleNode{Hash: hash.Sum(nil), IsLeaf: true, Children: nil, ParentNode: nil})
	}

	return leafs
}

func (merkleTree *MerkleTree) createNodes(leafNodes []*MerkleNode) *MerkleNode {
	var merkleNodes []*MerkleNode
	fmt.Println("len(leafNodes) = ", len(leafNodes))

	for i := 0; i < len(leafNodes); i += merkleTree.MaxArity {
		var hashesConcatenation []byte
		var children []*MerkleNode
		fmt.Println("i = ", i)
		fmt.Printf("hash : %x \n", leafNodes[i].Hash)

		merkleNode := &MerkleNode{}
		for j := i; j < len(leafNodes); j++ {
			fmt.Println("\t j = ", j)

			fmt.Printf("hash : %x \n", leafNodes[j].Hash)

			// Each internal node is a concatenation of byte strings of the hashes of its children
			hashesConcatenation = append(hashesConcatenation, leafNodes[j].Hash...)
			children = append(children, leafNodes[j])
			leafNodes[j].ParentNode = merkleNode
		}

		merkleNode.Children = children
		merkleNode.Hash = hashesConcatenation
		merkleNodes = append(merkleNodes, merkleNode)

		fmt.Println("len(merkleNodes) = ", len(merkleNodes))

		if len(leafNodes) <= merkleTree.MaxArity {
			return merkleNode
		}

	}

	return merkleTree.createNodes(merkleNodes)
}

func main() {
	var merkleTree *MerkleTree

	datagram1 := []byte("test")
	datagram2 := []byte("test2")
	datagram3 := []byte("XV")
	datagram4 := []byte("tt")

	udpDatagrams := []([]byte){datagram1, datagram2, datagram3, datagram4}

	merkleTree = CreateTree(udpDatagrams, 32)

	fmt.Printf("Root node hash : %x \n", merkleTree.Root.Hash)
	fmt.Printf("Root node children : %d \n", len(merkleTree.Root.Children))

	fmt.Printf("Child1 hash : %x \n", merkleTree.Root.Children[0].Hash)
	fmt.Printf("Child2 hash : %x \n", merkleTree.Root.Children[1].Hash)
	fmt.Printf("Child3 hash : %x \n", merkleTree.Root.Children[2].Hash)
	fmt.Printf("Child4 hash : %x \n", merkleTree.Root.Children[3].Hash)

}
