package repository

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/joeldotdias/twine/internal/helpers"
)

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

func (repo *Repository) makeObject(sha string) (Object, error) {
	path := repo.makePath("objects", sha[:2], sha[2:])
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Didn't find file with sha %s: %s", sha, err)
	}
	defer file.Close()

	zr, err := zlib.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	data, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("Couldn't read compressed data: %s", err)
	}

	nullByte := bytes.IndexByte(data, 0)
	if nullByte == -1 {
		return nil, fmt.Errorf("Malformed object: Didn't find null byte")
	}

	header := string(data[:nullByte])
	contents := data[nullByte+1:]

	var objKind string
	var size int
	_, err = fmt.Sscanf(header, "%s %d", &objKind, &size)
	if err != nil {
		return nil, err
	}

	var obj Object
	switch objKind {
	case "commit":
		obj = &Commit{metaKV: make(map[CommitField][]string)}
	case "tree":
		obj = &Tree{leaves: []*TreeLeaf{}}
	case "blob":
		obj = &Blob{}
	case "tag":
		obj = &Tag{metaKV: make(map[TagField][]string)}
	default:
		return nil, fmt.Errorf("Unknown object type: %s", objKind)
	}

	obj.Deserialize(contents)
	return obj, nil
}

func (repo *Repository) writeObject(obj Object, write bool) (string, error) {
	raw := obj.Serialize()
	// res := []byte(fmt.Sprintf("%s %d\x00", obj.Kind(), len(data)))
	// res = append(res, data...)
	// hash := sha1.Sum(res)
	// sha := hex.EncodeToString(hash[:])

	header := fmt.Sprintf("%s %d\x00", obj.Kind(), len(raw))
	data := append([]byte(header), raw...)
	hash := sha1.Sum(data)
	sha := hex.EncodeToString(hash[:])

	if write {
		path := repo.makePath("objects", sha[:2], sha[2:])
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("Couldn't create directories: %w", err)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			file, err := os.Create(path)
			if err != nil {
				return "", fmt.Errorf("Couldn't create file: %w", err)
			}
			defer file.Close()

			zw := zlib.NewWriter(file)
			if _, err = zw.Write(data); err != nil {
				return "", fmt.Errorf("Couldn't write compressed data: %w", err)
			}

			if err = zw.Close(); err != nil {
				return "", fmt.Errorf("Couldn't close zlib writer: %w", err)
			}
		}
	}

	return sha, nil
}

func (repo *Repository) makeObjectHash(file io.Reader, objKind string, write bool) (string, error) {
	contents, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("Couldn't read file: %v", err)
	}

	var obj Object
	switch objKind {
	case "blob":
		obj = &Blob{
			contents,
		}
	case "tree":
		obj = &Tree{
			leaves: treeParseEntirety(contents),
		}
	case "commit":
		commit := &Commit{
			metaKV: make(map[CommitField][]string),
		}
		commit.Deserialize(contents)
		obj = commit
	default:
		return "", fmt.Errorf("Unexpected object type: %s", err)
	}

	return repo.writeObject(obj, write)
}

func (repo *Repository) findObject(name string) (string, error) {
	// full SHA-1 hash
	if len(name) == 40 && helpers.IsHex(name) {
		return name, nil
	}

	// ref to HEAD
	if name == "HEAD" {
		headPath := repo.makePath("refs", "heads", repo.conf.defaultBranch)
		contents, err := os.ReadFile(headPath)
		if err != nil {
			return "", fmt.Errorf("Didn't find reference to head: %s", err)
		}
		return string(contents[:len(contents)-1]), nil
	}

	// any other ref
	path := repo.makePath("refs", name)
	if _, err := os.Stat(path); err == nil {
		contents, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("Couldn't read ref file: %s", err)
		}
		return string(contents[:len(contents)-1]), nil
	}

	// partial object hashes
	prefix := name[:2]
	path = repo.makePath("objects", prefix)
	if matches, err := filepath.Glob(filepath.Join(path, name[2:]+"*")); err == nil && len(matches) > 0 {
		if len(matches) > 1 {
			return "", fmt.Errorf("Found multiple objects with prefix: %s. Be more specific", name)
		}

		return filepath.Base(prefix + matches[0]), nil
	}

	return "", fmt.Errorf("Didn't find object: %s", name)
}
