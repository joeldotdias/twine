package repository

import "sort"

type Blob struct {
	contents []byte
}

type Tree struct {
	leaves []*TreeLeaf
}

func (b *Blob) Kind() string {
	return "blob"
}

func (b *Blob) Serialize() []byte {
	return b.contents
}

func (b *Blob) Deserialize(raw []byte) {
	b.contents = raw
}

func (t *Tree) Kind() string {
	return "tree"
}

func (t *Tree) Serialize() []byte {
	sort.Slice(t.leaves, func(i, j int) bool {
		return sortLeafByKey(t.leaves[i]) < sortLeafByKey(t.leaves[j])
	})

	var serialized []byte
	for _, leaf := range t.leaves {
		serialized = append(serialized, leaf.mode...)
		serialized = append(serialized, ' ')
		serialized = append(serialized, []byte(leaf.path)...)
		serialized = append(serialized, 0)
		serialized = append(serialized, leaf.sha...)
	}
	return serialized
}

func (t *Tree) Deserialize(data []byte) {
	t.leaves = treeParseEntirety(data)
}
