package reporting

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unsafe"
)

func bindata_read(data, name string) ([]byte, error) {
	var empty [0]byte
	sx := (*reflect.StringHeader)(unsafe.Pointer(&data))
	b := empty[:]
	bx := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bx.Data = sx.Data
	bx.Len = len(data)
	bx.Cap = bx.Len

	gz, err := gzip.NewReader(bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	return buf.Bytes(), nil
}

var _templates_index_html = "\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xac\x54\xc1\x6e\xdb\x30\x0c\xbd\xf7\x2b\x88\x5c\xea\x00\xab\x85\x5d\x7a\x68\x65\xa3\x5d\x37\x0c\x1b\xba\x15\x48\xd1\x7b\x1d\x89\x8b\xd5\xd9\xa2\x20\xd1\x41\x83\xc0\xff\x3e\xd8\x72\x52\x2b\xd8\xa1\x87\x9e\x2c\xf2\x3d\x3d\x3e\x51\x94\x65\xcd\x6d\x53\x9e\x01\xc8\x1a\x2b\x3d\x2c\x00\x24\x1b\x6e\xb0\x5c\xa1\x23\xcf\x52\xc4\x28\x22\x41\x79\xe3\x18\x82\x57\xc5\xa2\x66\x76\xe1\x4a\x88\xce\xba\xbf\x9b\x5c\x51\x2b\x3c\x56\x8a\x6f\x3e\x5f\x8a\xae\xd5\x31\xc8\x35\x6e\xb1\x21\xd7\xa2\xe5\xfc\x25\x2c\x40\x79\x0a\x81\xbc\xd9\x18\x5b\x4a\x11\xe5\xde\xaf\x7d\xa1\xa9\x4d\xf4\x87\xc4\xc7\xd5\x58\x57\x6b\x6c\x2e\x02\x57\x56\x57\x0d\x59\xbc\xb9\x8c\xa9\xbc\x35\x76\x50\x9e\xa9\x25\x72\xbc\x73\x58\x2c\x18\x5f\x39\xf2\x17\xb1\x1a\x80\x6a\xaa\x10\xe0\xd6\x39\xc0\x57\x46\xab\x03\xac\xc6\xae\xdc\x51\xeb\xc8\xa2\x65\xd8\x4f\x4c\x00\x45\x36\xb0\xef\x14\x93\xcf\x9c\x27\x17\x96\x33\x10\x20\x74\x0e\x0f\xc0\xf5\xd9\x0c\xe0\xda\x84\xbc\xa1\xcd\x93\x6f\xa0\x80\xe7\xfd\x1e\xf2\x7b\xda\x3c\xad\xee\xa1\xef\x9f\x4f\x79\x7e\xbc\xd3\x19\x35\x5e\xf2\xcf\xc7\x87\xdf\x23\xfd\x94\x1f\xb8\x62\x84\x02\xf6\x7d\x52\xf3\x0f\xb2\xaa\xb3\x54\x71\x39\xc3\x73\xae\xd1\x66\x90\x79\x0c\x8e\x6c\xc0\x25\x14\x65\x72\x1a\x00\x8f\xdc\x79\x0b\x07\x46\xfe\x12\xc8\x66\x73\x8d\x7e\x79\x50\x19\xa0\xff\x28\x44\x83\xc8\x8f\x83\xc7\x48\x4a\xb6\x5f\x1f\xa3\xfe\xcd\xbb\x47\xab\xd1\x67\x69\x6f\x27\x2f\x59\x22\x2f\xb5\xd9\x96\x49\x06\x40\xba\xf2\x8e\xda\xd6\xf0\x15\xec\xdf\xfa\x93\x7f\x37\x1c\xd3\xbd\x14\x2e\xdd\x23\xc5\x89\x4c\x62\x6b\xfe\x9d\x4f\xaa\x14\xf1\x35\x0e\xcb\x35\xe9\xdd\x34\xbd\xda\x6c\xc1\xe8\xe2\xdc\x13\xf1\x79\x39\x49\xbf\x6f\x12\xc7\xb9\xfb\xfa\xf0\x2b\x9f\x1a\x70\x34\x21\x6f\x9d\x13\xe5\xa7\x63\xac\x49\x75\xe3\x4b\xda\x20\x7f\x6b\x70\x58\x7e\xd9\xfd\xd0\x59\x2c\x7a\xe8\xf0\x74\x8a\xd4\x73\x74\x2a\x45\xfc\xa5\xfc\x0b\x00\x00\xff\xff\xcb\x85\xb8\x21\x5a\x04\x00\x00"

func templates_index_html() ([]byte, error) {
	return bindata_read(
		_templates_index_html,
		"templates/index.html",
	)
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		return f()
	}
	return nil, fmt.Errorf("Asset %s not found", name)
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
var _bindata = map[string]func() ([]byte, error){
	"templates/index.html": templates_index_html,
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
	for name := range node.Children {
		rv = append(rv, name)
	}
	return rv, nil
}

type _bintree_t struct {
	Func     func() ([]byte, error)
	Children map[string]*_bintree_t
}

var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"templates": {nil, map[string]*_bintree_t{
		"index.html": {templates_index_html, map[string]*_bintree_t{}},
	}},
}}
