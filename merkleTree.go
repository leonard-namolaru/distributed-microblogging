package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"time"
)

type Message struct {
	Date      time.Duration // Encoded as a number of seconds since January 1, 2022
	InReplyTo []byte        // The hash of the message to which this message replies, or 0
	Body      string        // The message itself, encoded in UTF-8
}

type MerkleNode struct {
	ParentNode *MerkleNode   // Pointer to the parent node
	Children   []*MerkleNode // A table of pointers to the child nodes
	IsLeaf     bool          // Boolean value. Is it a leaf or an internal/root node?
	Hash       []byte        // The label of each node is a hash. In the case of a leaf, it is a hash of a UDP datagram
	message    Message       // Only if the node is a leaf
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

/* Depth First Search (Prefix search)
 */
func (merkleTree *MerkleTree) DepthFirstSearch(count int, function func(int, *MerkleNode), arg ...*MerkleNode) {
	var merkleNode *MerkleNode // Node from which the search will start

	if len(arg) == 0 {
		merkleNode = merkleTree.Root // The default value
	} else {
		merkleNode = arg[0]
	}

	function(count, merkleNode)

	if merkleNode.IsLeaf {
		return
	}

	for i := 0; i < len(merkleNode.Children); i++ {
		merkleTree.DepthFirstSearch(count+1, function, merkleNode.Children[i])
	}
}

func (merkleTree *MerkleTree) PrintNodeHash(counter int, merkleNode *MerkleNode) {
	for i := 0; i < counter; i++ {
		fmt.Printf("\t")
	}

	fmt.Printf("Hash : %x \n", merkleNode.Hash)
}

func (merkleTree *MerkleTree) PrintNodeHashInBytes(counter int, merkleNode *MerkleNode) {
	for i := 0; i < counter; i++ {
		fmt.Printf("\t")
	}

	fmt.Printf("Hash : %v \n", merkleNode.Hash)
}

func (merkleTree *MerkleTree) PrintNumberChildren(counter int, merkleNode *MerkleNode) {
	for i := 0; i < counter; i++ {
		fmt.Printf("\t")
	}

	fmt.Printf("Number of children : %d \n", len(merkleNode.Children))
}

func main() { // For testing purposes only
	var merkleTree *MerkleTree

	datagram1 := []byte("test")
	datagram2 := []byte("test2")
	datagram3 := []byte("XV")
	datagram4 := []byte("tt")

	udpDatagrams := []([]byte){datagram1, datagram2, datagram3, datagram4}

	merkleTree = CreateTree(udpDatagrams, 2)
	merkleTree.DepthFirstSearch(0, merkleTree.PrintNodeHash)
	merkleTree.DepthFirstSearch(0, merkleTree.PrintNumberChildren)

	fmt.Printf("\n\n\n")

	merkleTree = CreateTree(udpDatagrams, 3)
	merkleTree.DepthFirstSearch(0, merkleTree.PrintNodeHash)
	merkleTree.DepthFirstSearch(0, merkleTree.PrintNumberChildren)

	fmt.Printf("\n\n\n")

	merkleTree = CreateTree(udpDatagrams, 4)
	merkleTree.DepthFirstSearch(0, merkleTree.PrintNodeHash)
	merkleTree.DepthFirstSearch(0, merkleTree.PrintNodeHashInBytes)
	merkleTree.DepthFirstSearch(0, merkleTree.PrintNumberChildren)
}
