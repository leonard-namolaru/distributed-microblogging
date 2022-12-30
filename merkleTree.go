package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"time"
)

const NODE_TYPE_INTERNAL = 1
const NODE_TYPE_MESSAGE = 0

/* The format of the Data field of each node */
const NODE_TYPE_BYTE = 0
const MESSAGE_DATE_FIRST_BYTE = 1
const MESSAGE_DATE_LENGTH = 4
const MESSAGE_IN_REPLY_TO_FIRST_BYTE = 5
const MESSAFE_IN_REPLY_TO_LENGTH = 32
const MESSAFE_LENGTH_FIRST_BYTE = 37
const MESSAGE_LENGTH_LENGTH = 2
const MESSAGE_BODY_FIRST_BYTE = 39
const MESSAGE_TOTAL_MIN_LENGTH = 1 + MESSAGE_DATE_LENGTH + MESSAFE_IN_REPLY_TO_LENGTH + MESSAGE_LENGTH_LENGTH // 1 for the type byte

type MerkleNode struct {
	ParentNode *MerkleNode   // Pointer to the parent node
	Children   []*MerkleNode // A table of pointers to the child nodes
	Hash       []byte
	Data       []byte
}

type MerkleTree struct {
	Root     *MerkleNode // Pointer to the root node
	MaxArity int         // The maximum number of children of each node
}

var JANUARY_1_2022 = time.Date(2022, 1, 1, 1, 0, 0, 0, time.Local) // January 1, 2022

/* A function that receives a list of messages as well as the maximum number of children in each node,
 * and returns a pointer to a Merkel tree containing all these messages.
 */
func CreateTree(messages [][]byte, maxArity int) *MerkleTree {
	var merkleTree MerkleTree
	merkleTree.MaxArity = maxArity

	leafs := merkleTree.createLeafNodes(messages)

	root := merkleTree.createNodes(leafs)
	merkleTree.Root = root

	return &merkleTree
}

func CreateEmptyTree(maxArity int) *MerkleTree {
	var merkleTree MerkleTree
	merkleTree.MaxArity = maxArity

	root := &MerkleNode{ParentNode: nil, Hash: []byte{}, Data: nil}
	merkleTree.Root = root

	return &merkleTree
}

func (merkleTree *MerkleTree) AddNode(hash []byte, nodeData []byte) bool {

	if !CheckHash(hash, nodeData) {
		return false
	}

	node := merkleTree.DepthFirstSearch(0, merkleTree.SearchParent, hash)
	if node != nil {
		if len(node.Children) < merkleTree.MaxArity {

			node.Children = append(node.Children, &MerkleNode{
				Hash:       hash,
				Children:   nil,
				ParentNode: node,
				Data:       nodeData,
			})
		} else {
			return false
		}
	} else {
		merkleTree.Root.Hash = hash
		merkleTree.Root.Data = nodeData

		if merkleTree.Root.Data[NODE_TYPE_BYTE] == NODE_TYPE_INTERNAL {
			for i := 0; i < len(merkleTree.Root.Children); i++ {
				hashChild := fmt.Sprintf("%x", merkleTree.Root.Children[i].Hash)
				hashFoundInParentNode := false
				for j := NODE_TYPE_BYTE + 1; j < len(merkleTree.Root.Data) && !hashFoundInParentNode; j += HASH_LENGTH {
					nodeHashString := fmt.Sprintf("%x", merkleTree.Root.Data[j:j+HASH_LENGTH])

					if hashChild == nodeHashString {
						hashFoundInParentNode = true
					}
				}

				if !hashFoundInParentNode {
					if i < len(merkleTree.Root.Children)-1 {
						merkleTree.Root.Children = append(merkleTree.Root.Children[:i], merkleTree.Root.Children[i+1:]...)
					} else {
						fmt.Printf("Test %d \n", len(merkleTree.Root.Children))
						if len(merkleTree.Root.Children)-1 == 0 {
							merkleTree.Root.Children = []*MerkleNode{}
						} else {
							merkleTree.Root.Children = merkleTree.Root.Children[:i]
						}
					}
				}
			}
		}

	}

	return true
}

/* Internal function. This function receives a list messages and returns leaves for those messages
 * (list of leaf pointers).
 */
func (merkleTree *MerkleTree) createLeafNodes(messages [][]byte) []*MerkleNode {
	var leafs []*MerkleNode

	for i := 0; i < len(messages); i++ {
		hash := sha256.New()

		_, errorMessage := hash.Write(messages[i])
		if errorMessage != nil {
			log.Fatalf("Error : unable to generate a hash for the message %s \n", messages[i])
		}

		leafs = append(leafs, &MerkleNode{
			Hash:       hash.Sum(nil),
			Children:   nil,
			ParentNode: nil, // Will be determined later
			Data:       messages[i],
		})
	}

	return leafs
}

/* Building a Merkel tree starts from the bottom.
 * After we created the leaves, we can create the upper nodes in the tree.
 */
func (merkleTree *MerkleTree) createNodes(leafNodes []*MerkleNode) *MerkleNode {
	var merkleNodes []*MerkleNode

	// Particular case: if we have a tree that includes only one element, this element will be the root of the tree
	if len(leafNodes) == 1 {
		return leafNodes[0]
	}

	// We go through the list of leaves
	for i := 0; i < len(leafNodes); i += merkleTree.MaxArity {
		var hashesConcatenation []byte
		var children []*MerkleNode

		// If we see that we have only one element left in the list, we want to create a direct link with the element above it,
		// without creating an element in the bins, so we add this single element directly to the list that the function returns
		if len(leafNodes[i:]) == 1 {
			merkleNodes = append(merkleNodes, leafNodes[i])
		} else {
			hash := sha256.New()
			merkleNode := &MerkleNode{}
			for j := i; j < len(leafNodes) && (j-i) < merkleTree.MaxArity; j++ {
				// Each internal node is a concatenation of byte strings of the hashes of its children
				hashesConcatenation = append(hashesConcatenation, leafNodes[j].Hash...)
				children = append(children, leafNodes[j])
				leafNodes[j].ParentNode = merkleNode
			}

			hashesConcatenation = append([]byte{NODE_TYPE_INTERNAL}, hashesConcatenation...)
			_, errorMessage := hash.Write(hashesConcatenation)
			if errorMessage != nil {
				log.Fatalf("Error : unable to generate a hash for hashes concatenation : %s \n", hashesConcatenation)
			}

			merkleNode.Children = children
			merkleNode.Hash = hash.Sum(nil)
			merkleNode.Data = hashesConcatenation
			merkleNodes = append(merkleNodes, merkleNode)

			if len(leafNodes) <= merkleTree.MaxArity {
				return merkleNode
			}
		}

	}

	return merkleTree.createNodes(merkleNodes)
}

/* Depth First Search (Prefix search)
 */
func (merkleTree *MerkleTree) DepthFirstSearch(nodesHeightCountInitialization int, function func(int, *MerkleNode, []byte) bool, hashSearch []byte, arg ...*MerkleNode) *MerkleNode {
	var merkleNode *MerkleNode // Node from which the search will start

	if len(arg) == 0 {
		merkleNode = merkleTree.Root // The default value
	} else {
		merkleNode = arg[0]
	}

	funcResult := function(nodesHeightCountInitialization, merkleNode, hashSearch)
	if funcResult == true {
		return merkleNode
	}

	if len(merkleNode.Data) == 0 {
		return nil
	}

	if merkleNode.Data[NODE_TYPE_BYTE] != NODE_TYPE_INTERNAL {
		return nil
	}

	for i := 0; i < len(merkleNode.Children); i++ {
		result := merkleTree.DepthFirstSearch(nodesHeightCountInitialization+1, function, hashSearch, merkleNode.Children[i])
		if result != nil {
			return result
		}
	}

	return nil
}

/******************************************************************************************/
func (merkleTree *MerkleTree) SearchParent(nodeHeight int, merkleNode *MerkleNode, hashSearch []byte) bool {
	hashSearchString := fmt.Sprintf("%x", hashSearch)

	if len(merkleNode.Data) != 0 && merkleNode.Data[NODE_TYPE_BYTE] == NODE_TYPE_INTERNAL {
		for i := NODE_TYPE_BYTE + 1; i < len(merkleNode.Data); i += HASH_LENGTH {
			nodeHashString := fmt.Sprintf("%x", merkleNode.Data[i:i+HASH_LENGTH])

			if hashSearchString == nodeHashString {
				return true
			}
		}
	}

	return false
}

func (merkleTree *MerkleTree) GetNodeByHash(nodeHeight int, merkleNode *MerkleNode, hashSearch []byte) bool {
	nodeHashString := fmt.Sprintf("%x", merkleNode.Hash)
	hashSearchString := fmt.Sprintf("%x", hashSearch)

	return (nodeHashString == hashSearchString)
}

func (merkleTree *MerkleTree) PrintNodeHash(nodeHeight int, merkleNode *MerkleNode, hashSearch []byte) bool {
	for i := 0; i < nodeHeight; i++ {
		fmt.Printf("\t")
	}

	fmt.Printf("Hash : %x \n", merkleNode.Hash)
	return false
}

func (merkleTree *MerkleTree) PrintNumberChildren(nodeHeight int, merkleNode *MerkleNode, hashSearch []byte) bool {
	for i := 0; i < nodeHeight; i++ {
		fmt.Printf("\t")
	}

	fmt.Printf("Number of children : %d \n", len(merkleNode.Children))

	return false
}

func (merkleTree *MerkleTree) PrintNodesData(nodeHeight int, merkleNode *MerkleNode, hashSearch []byte) bool {
	for i := 0; i < nodeHeight; i++ {
		fmt.Printf("\t")
	}
	fmt.Printf("Node hash : %x \n", merkleNode.Hash)

	for i := 0; i < nodeHeight; i++ {
		fmt.Printf("\t")
	}
	fmt.Printf("Node data : %s \n", nodeDataToString(merkleNode.Data, nodeHeight))

	return false
}

/******************************************************************************************/
func CreateMessage(body string, inReplyTo []byte) []byte {
	timeSinceJanuary := fmt.Sprintf("%d", int(time.Since(JANUARY_1_2022).Seconds()))

	messageBodyLength := len(body)
	messageLength := MESSAGE_TOTAL_MIN_LENGTH + messageBodyLength
	message := make([]byte, messageLength)

	message[NODE_TYPE_BYTE] = NODE_TYPE_MESSAGE
	copy(message[MESSAGE_DATE_FIRST_BYTE:MESSAGE_DATE_FIRST_BYTE+MESSAGE_DATE_LENGTH], []byte(timeSinceJanuary))
	copy(message[MESSAGE_IN_REPLY_TO_FIRST_BYTE:MESSAGE_IN_REPLY_TO_FIRST_BYTE+MESSAFE_IN_REPLY_TO_LENGTH], inReplyTo)
	message[MESSAFE_LENGTH_FIRST_BYTE] = byte(messageBodyLength >> 8)
	message[MESSAFE_LENGTH_FIRST_BYTE+1] = byte(messageBodyLength & 0xFF)

	copy(message[MESSAGE_BODY_FIRST_BYTE:], []byte(body))
	return message
}

func CreateMessagesForMerkleTree(numMessages int) [][]byte {
	messages := make([][]byte, numMessages)

	for i := 0; i < len(messages); i++ {
		var inReplyTo []byte
		messageBody := fmt.Sprintf("Message %d", i+1)

		if i%2 != 0 {
			hash := sha256.New()
			_, errorMessage := hash.Write(messages[i-1])
			if errorMessage != nil {
				log.Fatalf("Error : unable to generate a hash for the message %s \n", messages[i-1])
			}
			inReplyTo = hash.Sum(nil)
		} else {
			inReplyTo = inReplyToZeroes()
		}

		messages[i] = CreateMessage(messageBody, inReplyTo)
	}
	return messages
}

// In-reply-to indicates the hash of the message to which a message replies.
// It is 0 if a message does not respond to another message. Field size : 32 bytes.
func inReplyToZeroes() []byte {
	inReplyTo := make([]byte, MESSAFE_IN_REPLY_TO_LENGTH)

	for i := 0; i < len(inReplyTo); i++ {
		inReplyTo[i] = 0
	}

	return inReplyTo
}

/******************************************************************************************/
func nodeDataToString(nodeData []byte, tabulationNum int) string {
	str := ""
	nodeType := nodeData[NODE_TYPE_BYTE]

	if nodeType == 0 { // Type 0 indicates that it is a message
		messageDate := nodeData[MESSAGE_DATE_FIRST_BYTE : MESSAGE_DATE_FIRST_BYTE+MESSAGE_DATE_LENGTH]
		messageInReplyTo := nodeData[MESSAGE_IN_REPLY_TO_FIRST_BYTE : MESSAGE_IN_REPLY_TO_FIRST_BYTE+MESSAFE_IN_REPLY_TO_LENGTH]
		messageLength := int(nodeData[MESSAFE_LENGTH_FIRST_BYTE])<<8 | int(nodeData[MESSAFE_LENGTH_FIRST_BYTE+1])
		messageBody := nodeData[MESSAGE_BODY_FIRST_BYTE:]

		messageDateSec := int(messageDate[0]) + int(messageDate[1]) + int(messageDate[2]) + int(messageDate[3])
		messageDateTime := JANUARY_1_2022.Add(time.Duration(messageDateSec) * time.Second).String()

		for i := 0; i < tabulationNum; i++ {
			str += fmt.Sprintf("\t")
		}
		str += fmt.Sprintf("Node type :  %d \n", nodeType)

		for i := 0; i < tabulationNum; i++ {
			str += fmt.Sprintf("\t")
		}
		str += fmt.Sprintf("Date :  %s \n", messageDateTime)

		for i := 0; i < tabulationNum; i++ {
			str += fmt.Sprintf("\t")
		}
		str += fmt.Sprintf("In reply to : %x \n", messageInReplyTo)

		for i := 0; i < tabulationNum; i++ {
			str += fmt.Sprintf("\t")
		}
		str += fmt.Sprintf("Length :  %d \n", messageLength)

		for i := 0; i < tabulationNum; i++ {
			str += fmt.Sprintf("\t")
		}
		str += fmt.Sprintf("Body :  %s \n", messageBody)

	} else if nodeType == 1 {
		str += fmt.Sprintf("Node type :  %d \n", nodeType)
		hashCount := 0
		for i := NODE_TYPE_BYTE + 1; i < len(nodeData); i += HASH_LENGTH {
			hashCount++
			for j := 0; j < tabulationNum; j++ {
				str += fmt.Sprintf("\t")
			}
			str += fmt.Sprintf("Hash %d : %x \n", hashCount, nodeData[i:i+HASH_LENGTH])
		}
	}

	return str
}

func CheckHash(hash []byte, data []byte) bool {

	dataHash := sha256.New()
	_, errorMessage := dataHash.Write(data)
	if errorMessage != nil {
		log.Fatalf("Error : unable to generate a hash for the message %s \n", data)
	}

	hashString := fmt.Sprintf("%x", hash)
	dataHashString := fmt.Sprintf("%x", dataHash.Sum(nil))
	return (hashString == dataHashString)
}
