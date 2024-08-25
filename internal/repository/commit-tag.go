package repository

import (
	"bytes"
	"strings"
)

type (
	CommitField string
	TagField    string
)

const (
	TreeField      CommitField = "tree"
	ParentField    CommitField = "parent"
	AuthorField    CommitField = "author"
	CommitterField CommitField = "committer"

	ObjectField  TagField = "object"
	TypeField    TagField = "type"
	TagNameField TagField = "tag"
	TaggerField  TagField = "tagger"
)

// these are so similar
// i feel stupid having different structs for them

type Commit struct {
	metaKV  map[CommitField][]string
	message string
}

type Tag struct {
	metaKV  map[TagField][]string
	message string
}

func (c *Commit) Kind() string {
	return "commit"
}

func (c *Commit) Serialize() []byte {
	return serializeKvlm(c.toStringMap(), c.message)
}

func (c *Commit) Deserialize(data []byte) {
	kv, message := parseKvlm(data)
	c.fromStringMap(kv)
	c.message = message
}

func (t *Tag) Serialize() []byte {
	return serializeKvlm(t.toStringMap(), t.message)
}

func (t *Tag) Deserialize(data []byte) {
	kv, message := parseKvlm(data)
	t.fromStringMap(kv)
	t.message = message
}

// kvlm -> Key Value List with Message
// this format is taken from Thibault Polge's "Write yourself a Git!" article
// real lifesaver
func parseKvlm(data []byte) (map[string][]string, string) {
	kvlm := make(map[string][]string)
	lines := bytes.Split(data, []byte{'\n'})

	var messageStartIndex int
	var currentKey string

	for i, line := range lines {
		if len(line) == 0 {
			messageStartIndex = i + 1
			break
		}

		if line[0] == ' ' {
			// Continuation of previous key
			kvlm[currentKey] = append(kvlm[currentKey], string(line[1:]))
		} else {
			parts := bytes.SplitN(line, []byte{' '}, 2)
			if len(parts) == 2 {
				key := string(parts[0])
				value := string(parts[1])
				currentKey = key
				if existingValue, ok := kvlm[key]; ok {
					kvlm[key] = append(existingValue, value)
				} else {
					kvlm[key] = []string{value}
				}
			}
		}
	}

	message := string(bytes.Join(lines[messageStartIndex:], []byte{'\n'}))
	return kvlm, message
}

func serializeKvlm(klvm map[string][]string, message string) []byte {
	var buffer bytes.Buffer

	for key, values := range klvm {
		for _, value := range values {
			buffer.WriteString(key)
			buffer.WriteByte(' ')
			buffer.WriteString(strings.Replace(value, "\n", "\n ", -1))
			buffer.WriteByte('\n')
		}
	}

	buffer.WriteByte('\n')
	buffer.WriteString(message)

	return buffer.Bytes()
}

func (c *Commit) toStringMap() map[string][]string {
	kv := make(map[string][]string)
	for k, v := range c.metaKV {
		kv[string(k)] = v
	}
	return kv
}

func (c *Commit) fromStringMap(kv map[string][]string) {
	c.metaKV = make(map[CommitField][]string)
	for k, v := range kv {
		c.metaKV[CommitField(k)] = v
	}
}

func (t *Tag) toStringMap() map[string][]string {
	kv := make(map[string][]string)
	for k, v := range t.metaKV {
		kv[string(k)] = v
	}
	return kv
}

func (t *Tag) fromStringMap(kv map[string][]string) {
	t.metaKV = make(map[TagField][]string)
	for k, v := range kv {
		t.metaKV[TagField(k)] = v
	}
}
