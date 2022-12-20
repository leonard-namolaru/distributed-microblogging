package main

import (
	"crypto/sha256"
	"fmt"
	"log"
)

type MerkleNode struct {
	ParentNode *MerkleNode   // Pointer to the parent node
	Children   []*MerkleNode // A table of pointers to the child nodes
	IsLeaf     bool          // Boolean value. Is it a leaf or an internal/root node?
	Hash       []byte        // The label of each node is a hash. In the case of a leaf, it is a hash of a UDP datagram
}

type MerkleTree struct {
	Root     *MerkleNode // Pointer to the root node
	MaxArity int         // The maximum number of children of each node
}

/* A function that receives a list of UDP datagrams as well as the maximum number of children in each node,
 * and returns a pointer to a Merkel tree containing all these datagrams.
 */
func CreateTree(UdpDatagrams [][]byte, maxArity int) *MerkleTree {
	var merkleTree MerkleTree
	merkleTree.MaxArity = maxArity

	leafs := merkleTree.createLeafNodes(UdpDatagrams)

	root := merkleTree.createNodes(leafs)
	merkleTree.Root = root

	return &merkleTree
}

/* Internal function. This function receives a list of UDP datagrams and returns leaves for those datagrams
 * (list of leaf pointers).
 */
func (merkleTree *MerkleTree) createLeafNodes(UdpDatagrams [][]byte) []*MerkleNode {
	var leafs []*MerkleNode

	for i := 0; i < len(UdpDatagrams); i++ {
		hash := sha256.New()

		_, errorMessage := hash.Write(UdpDatagrams[i])
		if errorMessage != nil {
			log.Fatal("Error : unable to generate a hash for a UDP datagram \n")
		}

		leafs = append(leafs, &MerkleNode{
			Hash:       hash.Sum(nil),
			IsLeaf:     true,
			Children:   nil,
			ParentNode: nil}) // Will be determined later
	}

	return leafs
}

/* Building a Merkel tree starts from the bottom.
 * After we created the leaves, we can create the upper nodes in the tree.
 */
func (merkleTree *MerkleTree) createNodes(leafNodes []*MerkleNode) *MerkleNode {
	var merkleNodes []*MerkleNode

	// We go through the list of leaves
	for i := 0; i < len(leafNodes); i += merkleTree.MaxArity {
		var hashesConcatenation []byte
		var children []*MerkleNode

		merkleNode := &MerkleNode{}
		for j := i; j < len(leafNodes) && (j-i) < merkleTree.MaxArity; j++ {
			// Each internal node is a concatenation of byte strings of the hashes of its children
			hashesConcatenation = append(hashesConcatenation, leafNodes[j].Hash...)
			children = append(children, leafNodes[j])
			leafNodes[j].ParentNode = merkleNode
		}

		merkleNode.Children = children
		merkleNode.Hash = hashesConcatenation
		merkleNodes = append(merkleNodes, merkleNode)

		if len(leafNodes) <= merkleTree.MaxArity {
			return merkleNode
		}

	}

	return merkleTree.createNodes(merkleNodes)
}

func main() { // For testing purposes only
	var merkleTree *MerkleTree

	datagram1 := []byte("test")
	datagram2 := []byte("test2")
	datagram3 := []byte("XV")
	datagram4 := []byte("tt")

	udpDatagrams := []([]byte){datagram1, datagram2, datagram3, datagram4}

	merkleTree = CreateTree(udpDatagrams, 2)

	fmt.Printf("Root node hash : %x \n", merkleTree.Root.Hash)
	fmt.Printf("Root node children : %d \n", len(merkleTree.Root.Children))

	fmt.Printf("Child1 hash : %x \n", merkleTree.Root.Children[0].Hash)
	fmt.Printf("Child2 hash : %x \n", merkleTree.Root.Children[1].Hash)

	fmt.Printf("Root node children : %d \n", len(merkleTree.Root.Children[0].Children))
	fmt.Printf("Child1 hash : %x \n", merkleTree.Root.Children[0].Children[0].Hash)
	fmt.Printf("Child1 hash : %x \n", merkleTree.Root.Children[0].Children[1].Hash)

	fmt.Printf("Root node children : %d \n", len(merkleTree.Root.Children[1].Children))
	fmt.Printf("Child1 hash : %x \n", merkleTree.Root.Children[1].Children[0].Hash)
	fmt.Printf("Child1 hash : %x \n", merkleTree.Root.Children[1].Children[1].Hash)

	fmt.Printf("\n\n\n")

	merkleTree = CreateTree(udpDatagrams, 3)

	fmt.Printf("Root node hash : %x \n", merkleTree.Root.Hash)
	fmt.Printf("Root node children : %d \n", len(merkleTree.Root.Children))

	fmt.Printf("Child1 hash : %x \n", merkleTree.Root.Children[0].Hash)
	fmt.Printf("Child2 hash : %x \n", merkleTree.Root.Children[1].Hash)

	fmt.Printf("Root node children : %d \n", len(merkleTree.Root.Children[0].Children))
	fmt.Printf("Child1 hash : %x \n", merkleTree.Root.Children[0].Children[0].Hash)
	fmt.Printf("Child1 hash : %x \n", merkleTree.Root.Children[0].Children[1].Hash)
	fmt.Printf("Child1 hash : %x \n", merkleTree.Root.Children[0].Children[2].Hash)

	fmt.Printf("Root node children : %d \n", len(merkleTree.Root.Children[1].Children))
	fmt.Printf("Child1 hash : %x \n", merkleTree.Root.Children[1].Children[0].Hash)

}
