package repository

/*
 *						 Object structure
 * +-------------+------------+--------+-----------+---------+
 * | object_type | whitespace | length | null_byte | content |
 * +-------------+------------+--------+-----------+---------+
 *
 * object_type => blob, tree, commit, tag
 */

type Object interface {
	Kind() string
	Serialize() []byte
	Deserialize(raw []byte)
}
