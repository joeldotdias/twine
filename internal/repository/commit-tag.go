package repository

import (
	"bytes"
	"fmt"
	"strings"
	"time"
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

func (t *Tag) Kind() string {
	return "tag"
}

func (t *Tag) Serialize() []byte {
	return serializeKvlm(t.toStringMap(), t.message)
}

func (t *Tag) Deserialize(data []byte) {
	kv, message := parseKvlm(data)
	t.fromStringMap(kv)
	t.message = message
}

func (c *Commit) parseCommitLog(sha string, checkRef func() (string, bool)) (string, string, error) {
	var cmtStr strings.Builder

	cmtStr.WriteString("\033[33m" + "commit " + sha + "\033[0m")
	if ref, isHead := checkRef(); ref != "" {
		cmtStr.WriteString("\033[33m (")
		if isHead {
			cmtStr.WriteString("\033[36;1mHEAD \033[0m\033[33m-> \033[0m")
		}
		cmtStr.WriteString("\033[32;1m")
		cmtStr.WriteString(ref)
		cmtStr.WriteString("\033[0m\033[33m)\033[0m")
	}

	stats, _ := c.getField("author")
	parts := strings.Split(stats, "> ")
	author, timestampStr := parts[0], parts[1]
	var timestamp int64
	var tzOffset string
	_, err := fmt.Sscanf(timestampStr, "%d %s", &timestamp, &tzOffset)
	if err != nil {
		return "", "", fmt.Errorf("Malformed unix timestamp: %s", err)
	}
	tz, err := time.Parse("-0700", tzOffset)
	if err != nil {
		return "", "", err
	}

	t := time.Unix(timestamp, 0).In(tz.Location())

	// author, date and message
	cmtStr.WriteString("\nAuthor: " + author)
	cmtStr.WriteString(">\nDate: " + t.Format("Mon Jan 2 15:04:05 2006"))
	cmtStr.WriteString(" " + tzOffset)

	cmtStr.WriteString("\n\n\t" + c.message + "\n")

	parent, _ := c.getField("parent")

	return cmtStr.String(), parent, nil
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

func (c *Commit) getField(key string) (string, error) {
	field := CommitField(key)
	values, ok := c.metaKV[field]
	if !ok || len(values) == 0 {
		return "", fmt.Errorf("Field %s does not exist", key)
	}
	return values[0], nil
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
