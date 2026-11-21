// Code generated for package schema by go-bindata DO NOT EDIT. (@generated)
// sources:
// mutation.graphql
// query.graphql
// schema.graphql
// type/job.graphql
package schema

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _mutationGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x2a\xa9\x2c\x48\x55\xf0\x2d\x2d\x49\x2c\xc9\xcc\xcf\x53\xa8\xe6\x52\x50\x50\x56\x70\x2e\x4a\x4d\x2c\x49\x55\xf0\x4b\x2d\x57\xf0\xca\x4f\xe2\x52\x50\x70\xcd\x2b\x2c\x4d\x2d\x4d\xd5\x28\x49\x2c\xce\xb6\x52\x08\x2e\x29\xca\xcc\x4b\x57\xd4\xb4\x02\xc9\x2a\x72\xd5\x02\x02\x00\x00\xff\xff\x9e\x45\x59\x6f\x43\x00\x00\x00")

func mutationGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_mutationGraphql,
		"mutation.graphql",
	)
}

func mutationGraphql() (*asset, error) {
	bytes, err := mutationGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "mutation.graphql", size: 67, mode: os.FileMode(420), modTime: time.Unix(1765158559, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _queryGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x54\x8f\xc1\x6a\xc3\x30\x10\x44\xef\xfe\x8a\x31\xbd\x38\xbf\xa0\x63\x09\x85\x84\x5c\x8a\x8f\x25\x07\xc9\x5a\xac\x2d\x8a\x14\xbc\x2b\x8a\x28\xfd\xf7\x22\x2b\xa5\xe4\xb6\x0c\xf3\x98\xb7\x5a\xef\x84\xf7\x42\x5b\xc5\xf7\x00\xbc\xe0\xc2\xa2\xb0\x31\xe2\x33\x3b\x01\x27\x68\x20\x48\x15\xa5\xdb\x00\x9c\xb3\x13\x83\x8f\x73\x76\xe3\x75\x1c\x76\xe0\x8d\x74\x09\xb0\x10\x4e\x6b\xa4\x86\xc1\x55\x9c\x8e\xbd\x3d\xb1\x37\x98\x75\xe3\xb4\x8e\x07\xd3\x92\x4e\xcd\x21\x7f\xed\x5d\x51\xab\x45\xb0\xe4\x92\x54\x3a\x33\xef\x91\xf9\x3f\x9f\x96\x6e\x25\x2a\xdf\xfb\x92\xf4\x29\xc1\x54\x84\x04\x47\xab\xf6\x92\xad\xa7\x0d\xce\xea\x12\x38\xad\x87\x87\xf4\x6b\x3d\x79\x99\xd8\x37\xfb\x87\xce\xb5\x09\xfd\xbd\xf2\x33\xfc\x06\x00\x00\xff\xff\x28\xd8\x9d\xa2\x0a\x01\x00\x00")

func queryGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_queryGraphql,
		"query.graphql",
	)
}

func queryGraphql() (*asset, error) {
	bytes, err := queryGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "query.graphql", size: 266, mode: os.FileMode(420), modTime: time.Unix(1765188414, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _schemaGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x2a\x4e\xce\x48\xcd\x4d\x54\xa8\xe6\x52\x50\x50\x50\x28\x2c\x4d\x2d\xaa\xb4\x52\x08\x04\x51\x60\x81\xdc\xd2\x92\xc4\x92\xcc\xfc\x3c\x2b\x05\x5f\x28\x8b\xab\x16\x10\x00\x00\xff\xff\x8e\x43\x79\x00\x32\x00\x00\x00")

func schemaGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_schemaGraphql,
		"schema.graphql",
	)
}

func schemaGraphql() (*asset, error) {
	bytes, err := schemaGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "schema.graphql", size: 50, mode: os.FileMode(420), modTime: time.Unix(1765158559, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _typeJobGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x4c\xcc\x41\x0a\x83\x40\x10\x44\xd1\xb5\x7d\x8a\xf2\x1a\xde\x20\xd9\x7a\x82\x49\xa6\x23\x43\xb4\x6d\xec\x72\x11\xc4\xbb\x07\x42\x1c\x66\xf9\x8a\x4f\xf1\xe3\x8a\xfb\xfa\xc0\x21\x5d\xc9\x03\x46\x6e\xc5\xa6\x5e\x3a\xa6\x78\x37\x0c\x26\xee\xd1\x0c\x89\xd4\xc5\x19\x03\x6e\xc6\x5e\x4e\x91\xeb\x6a\xfc\xa5\x38\x04\x70\xb5\x5c\x6c\xfa\x37\xc0\xb6\x9b\xb5\x7e\xa5\x32\x6b\xae\x7c\xae\x8b\xcf\xca\xba\x9c\xdf\x00\x00\x00\xff\xff\x68\xc0\x48\x58\x9e\x00\x00\x00")

func typeJobGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_typeJobGraphql,
		"type/job.graphql",
	)
}

func typeJobGraphql() (*asset, error) {
	bytes, err := typeJobGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "type/job.graphql", size: 158, mode: os.FileMode(420), modTime: time.Unix(1765158559, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"mutation.graphql": mutationGraphql,
	"query.graphql":    queryGraphql,
	"schema.graphql":   schemaGraphql,
	"type/job.graphql": typeJobGraphql,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"mutation.graphql": &bintree{mutationGraphql, map[string]*bintree{}},
	"query.graphql":    &bintree{queryGraphql, map[string]*bintree{}},
	"schema.graphql":   &bintree{schemaGraphql, map[string]*bintree{}},
	"type": &bintree{nil, map[string]*bintree{
		"job.graphql": &bintree{typeJobGraphql, map[string]*bintree{}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
