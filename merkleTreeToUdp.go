package main

const INTERNAL_NODE_DATAGRAM_MIN_LENGTH = 1     // For the Type field
const LEAF_DATAGRAM_MIN_LENGTH = 1 + 4 + 32 + 2 // For the Type, Date, In-reply-toLength and Length fields

func merkleInternalNodeToUdp(merkleNode *MerkleNode) []byte {
	var datagram_type byte = 1 // Type 1 indicates that this is an internal node.

	datagram_length := INTERNAL_NODE_DATAGRAM_MIN_LENGTH + len(merkleNode.Hash)
	datagram := make([]byte, datagram_length)

	datagram[0] = datagram_type
	copy(datagram[1:], merkleNode.Hash)

	return datagram
}

func leafToUdp(merkleNode *MerkleNode) []byte {
	var datagram_type byte = 0 // Type 0 indicates that it is a message

	message_body_length := len(merkleNode.message.Body)
	datagram_length := LEAF_DATAGRAM_MIN_LENGTH + message_body_length
	datagram := make([]byte, datagram_length)

	datagram[0] = datagram_type
	copy(datagram[1:4], []byte(merkleNode.message.Date.String())) // warning : signed or nor signed ?
	copy(datagram[4:36], merkleNode.message.InReplyTo)
	copy(datagram[4:36], merkleNode.message.InReplyTo)

	datagram[36] = byte(message_body_length >> 8)
	datagram[37] = byte(message_body_length & 0xFF)

	copy(datagram[LEAF_DATAGRAM_MIN_LENGTH:], merkleNode.message.Body)

	return datagram
}

func main() { // For testing purposes only

}
