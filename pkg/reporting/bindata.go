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

var _templates_index_html = "\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xac\x93\xcd\x6e\xdb\x30\x0c\xc7\xef\x7d\x0a\x22\x97\x3a\xc0\x2a\x61\x97\x1e\x5a\xd9\x68\xd7\x0d\xc3\x06\x6c\x05\xda\x17\xa8\x23\x71\xb1\x3a\x5b\x14\x24\x3a\x68\x60\xf8\xdd\x07\x47\x4e\x6a\x0d\x3b\xe4\xb0\x93\xf9\xa5\x1f\xff\xa2\x29\xd5\x70\xd7\x56\x17\x00\xaa\xc1\xda\x4c\x06\x80\x62\xcb\x2d\x56\x4f\xe8\x29\xb0\x92\xc9\x4b\x99\xa8\x83\xf5\x0c\x31\xe8\x72\xd5\x30\xfb\x78\x23\x65\xef\xfc\xef\xad\xd0\xd4\xc9\x80\xb5\xe6\xbb\x8f\xd7\xb2\xef\x4c\x72\x84\xc1\x1d\xb6\xe4\x3b\x74\x2c\x5e\xe3\x0a\x74\xa0\x18\x29\xd8\xad\x75\x95\x92\x09\x77\x3e\xfb\xca\x50\x97\xf1\xa7\xc0\xff\xeb\xb1\xa9\x37\xd8\x5e\x45\xae\x9d\xa9\x5b\x72\x78\x77\x9d\x42\xa2\xb3\x6e\x22\x2f\x68\x19\x8e\xf7\x1e\xcb\x15\xe3\x1b\xa7\xfa\x55\xea\x06\xa0\xdb\x3a\x46\xb8\xf7\x1e\xf0\x8d\xd1\x99\x08\x4f\x87\xa9\x3c\x50\xe7\xc9\xa1\x63\x18\xe6\x4a\x00\x4d\x2e\x72\xe8\x35\x53\x28\x7c\x20\x1f\xd7\x8b\x24\x40\xec\x3d\x1e\x13\xb7\x17\x8b\x04\x37\x36\x8a\xc8\x35\x23\x94\x30\x8c\x59\xee\x17\xb2\x6e\x8a\x97\x61\x00\x91\xfe\xe6\xf7\xe7\xc7\x9f\x30\x8e\x2f\xeb\x45\x91\xe0\x06\x5d\x01\x45\xc0\xe8\xc9\x45\x5c\x43\x59\x65\xad\x01\x02\x72\x1f\x1c\x1c\x2b\xc4\x6b\x24\x57\x2c\x19\xe3\xfa\x48\x99\x52\xff\x20\x24\x95\xc8\xcf\x93\xd0\x54\x94\x1d\xbf\x3d\x79\xe3\xfb\x05\x02\x3a\x83\xa1\xc8\x07\x31\x6b\x29\x32\xbc\x32\x76\x57\x65\x11\x00\xe5\xab\x07\xea\x3a\xcb\x37\x30\xbc\x0f\x49\x7c\xb5\x9c\xc2\xa3\x92\x3e\x3f\xa3\xe4\x5f\x98\x4c\xd6\xf2\xbb\x5c\x2b\x25\xd3\xd3\x99\xcc\x0d\x99\xfd\xbc\x6a\xc6\xee\xc0\x9a\xf2\x32\x10\xf1\x65\x35\xa3\xcf\x5b\x9b\xc3\x92\x7c\x7e\xfc\x21\xe6\x01\x9c\x44\xa8\x7b\xef\x65\xf5\xe1\xe4\x1b\xd2\xfd\x61\xed\xb7\xc8\x5f\x5a\x9c\xcc\x4f\xfb\x6f\xa6\x48\x4d\x8f\x13\x9e\x6f\x91\x6b\x4e\x4a\x95\x4c\xef\xff\x4f\x00\x00\x00\xff\xff\xf0\x4a\x26\x45\x07\x04\x00\x00"

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
	Func func() ([]byte, error)
	Children map[string]*_bintree_t
}
var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"templates": &_bintree_t{nil, map[string]*_bintree_t{
		"index.html": &_bintree_t{templates_index_html, map[string]*_bintree_t{
		}},
	}},
}}
