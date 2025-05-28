package blockchain

import (
	"crypto/sha256"
)

type MerkleTree struct {
	Root *MerkleNode
}

type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	node := MerkleNode{}

	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		node.Data = hash[:]
	} else {
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		node.Data = hash[:]
	}

	node.Left = left
	node.Right = right
	return &node
}

func NewMerkleTree(data [][]byte) *MerkleTree {

	var nodes []MerkleNode
	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}
	for _, d := range data {
		nodes = append(nodes, *NewMerkleNode(nil, nil, d))
	}

	for i := 0; i < len(data)/2; i ++ {
		var newLevel []MerkleNode
		for j := 0; j < len(nodes); j += 2 {
			if i+1 < len(nodes) {
				newLevel = append(newLevel, *NewMerkleNode(&nodes[i], &nodes[i+1], nil))
			} else {
				newLevel = append(newLevel, nodes[i]) // Odd node, carry it over
			}
		}
		nodes = newLevel
	}

	return &MerkleTree{Root: &nodes[0]}
}
