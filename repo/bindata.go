// Code generated by go-bindata.
// sources:
// sample-openbazaar.conf
// DO NOT EDIT!

package repo

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

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _sampleOpenbazaarConf = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xc4\x5a\x5b\x73\x1b\xb7\x92\x7e\xe7\xaf\xe8\x87\x9c\x3a\xbb\x55\xf2\x50\xa2\x2d\xf9\xc2\xd5\x56\xd1\x96\x6c\xeb\x44\x91\x58\x12\x7d\x89\xde\xc0\x41\x0f\x07\x2b\x0c\x30\x01\x30\xa4\x98\xad\xcd\x6f\xdf\xea\x06\x30\x43\x2a\x76\x1e\x4e\x59\x15\xfb\xc1\x12\x2e\xdd\x8d\xbe\x7e\xdd\xe3\x29\x3c\xfb\xa1\x7f\x46\x53\x38\x13\x41\x80\xc7\x10\x94\x59\xf9\xd1\x0f\x67\x30\x9a\xc2\xa2\x46\x90\xca\x61\x19\xac\xdb\x42\xb0\xe0\x83\x75\x08\x92\x19\x77\x65\x0d\xc2\x43\xa8\x11\x6c\x8b\x66\x29\x7e\x17\xc2\xf1\xde\x52\x78\x3c\x00\xd5\x56\x1e\x1a\x0c\x82\x96\x0e\x40\x18\x39\x9a\x42\xdb\x2d\xb5\x2a\xf9\x54\x91\x19\x60\x25\x3a\x1d\x40\x79\xf8\x63\x5c\xec\x90\xb2\x06\xe6\xd7\xb7\x17\x5f\xe1\xfa\x16\xfd\x01\xfc\x74\x79\xfd\x6e\x76\x39\x9b\xcf\xcf\x66\x8b\xd9\xf8\xba\x45\xf3\xb6\x3f\xf7\x45\x19\x69\x37\xfe\x60\x34\x85\x3f\xc6\x97\x6a\xe9\x84\xdb\x8e\x67\x6d\xab\x55\x29\x82\xb2\x06\x6e\xbb\xb6\xb5\x2e\x3c\xba\xf6\x8b\x28\xe1\xfa\x96\x65\x83\x9f\x6a\xdb\xe0\x78\x8f\xfd\x68\x0a\x73\x2d\xcc\xeb\x02\xe0\xdc\xac\x95\xb3\xa6\x41\x13\x60\x2d\x9c\x12\x4b\x8d\x1e\x84\x43\xc0\x87\x56\x18\x89\x12\xbc\x25\x5d\x6c\xa1\x11\x5b\x58\x22\x74\x1e\x65\x01\x70\x75\xbd\x38\x7f\x93\xe5\x1b\x4d\x01\xbf\x4b\x28\x6c\x5b\x55\x0a\xad\xb7\xf0\x8f\xcf\xb3\x9b\x8b\xd9\xdb\xcb\xf3\x7f\x1c\xc0\xb2\x0b\x89\x6c\xe7\x03\xd1\x15\x65\x89\xde\xa3\x84\x8d\x0a\xf5\x68\x0a\x3f\xe5\xc3\x50\xa3\xc3\x02\x60\xa6\xbd\x3d\x80\x3f\x48\x9f\xbd\x6c\xc1\xee\xab\x6f\x47\x67\x64\x06\x32\x87\x54\xee\x74\x4f\xff\xa3\xd1\x8f\xf7\xa9\x29\x5c\x61\xd8\x58\x77\xff\xb4\x7e\xfb\xc9\x23\x04\xf4\xc1\x60\xa0\xe7\xa5\x1f\x4f\x8f\x78\xcf\xa8\x35\x3a\x2f\x34\xcc\x75\xb7\x62\xd3\xcf\xb5\xd8\xc2\x7f\x7c\x9a\x9b\xf9\x7f\x82\xe8\x82\x6d\x44\x48\x96\x20\x6d\x44\x17\xd7\xca\x07\x34\x40\x4e\x04\x76\x19\x84\x32\x24\x3a\xed\xe0\x43\x40\x67\x84\x86\x8b\x39\x08\x29\x1d\x7a\x0f\x95\xb3\x0d\xf8\xe8\x73\x28\x41\xe2\x5a\x95\xe8\x0b\x58\xd4\xca\x83\x6d\xd9\x25\xa5\xf2\xd1\xf8\x8a\x85\x34\xb6\x6b\x4d\x1b\x65\xfc\xd5\x76\xec\x46\xbe\xc5\x52\x55\x5b\xb0\x06\xc1\x3a\x68\x28\xf8\xfc\x46\xb8\x26\x33\x42\x4f\xa6\x4d\xb2\x59\x03\x95\x75\xa0\x4c\x69\x1b\x65\x56\x60\xa2\xaa\x47\x53\x28\xad\x31\x58\x12\x57\x96\x01\x3d\xee\x10\x20\x47\x25\xc7\x52\x06\x04\xac\x85\x56\x12\x9a\x4e\x07\x45\x27\x88\x60\x23\x58\x3e\xe6\x4b\x6b\xa7\x63\xd5\xbe\x18\x1f\x16\xfc\x77\x1c\xca\x76\xfc\xe2\xf0\xf0\xe8\xf1\x89\x93\xf1\x9b\x37\xdf\xdd\xdc\xbf\xfe\xfa\xf0\xf0\x78\xcc\xc1\xf1\x6d\x0a\x79\x3f\xe5\x8b\x95\x08\xb8\x11\xdb\x5e\xd7\x2c\x6c\xab\xf1\x01\x3d\x2c\x6d\xa8\xd9\x28\x17\xf3\xf7\xb7\xfd\xc9\xd9\xfc\x82\xed\xbc\x9f\xaa\x46\x53\xde\xb0\x6b\x74\xbc\xe3\x45\xd3\xab\x85\xb5\xb4\xc3\xc1\xd7\x49\x43\xdf\xd7\x4f\x62\x36\x3c\xf1\x68\xf2\x92\x1f\x79\x94\xd5\x30\xa1\x17\xbc\xb5\x36\xf8\x20\xda\x1d\x03\x50\xf0\xb3\x11\x82\x85\xff\xb1\xca\xb0\x34\xc9\x78\x05\x5c\x1b\xf0\x41\xb8\x10\x57\xad\x44\xd8\x28\xad\xa1\x11\xf7\x38\x9a\x82\xed\xc2\xca\x92\xb1\x77\x4c\x4c\x74\xe8\xf0\x92\x59\x39\xd1\x42\x8b\xe8\x3c\xab\xa0\xa3\xc8\xa8\xb1\xa1\x33\x52\xf9\x92\x5f\x6f\x43\x8d\xa4\x8e\x78\xec\x91\x00\xa3\xe9\x40\x68\x78\xdc\x43\xc1\x7f\x7b\x0b\x8f\xdb\x49\x3b\x3e\x9a\x9c\x3d\xff\xd9\xda\x2f\xf3\xbb\xe7\x0f\x6f\xaf\x6e\x3e\x3c\xbc\xa8\xea\x9b\x65\xf5\xeb\xac\xfc\xfa\xa9\x2e\xef\xea\xc5\xdd\xe4\xf2\xdd\xfd\xbf\x5e\xbe\xb8\xff\xd7\xd7\x0f\xd5\xef\xaf\x17\x9f\x2f\x17\xa4\x93\x5b\xae\x2a\x24\x5e\x65\xdd\x46\x38\x09\x1e\xdd\x9a\x45\xde\x51\x8d\xc3\x12\xd5\x1a\xa1\x41\xef\xc5\x0a\x3d\x6c\x6a\x72\xfa\xaa\xd2\xca\x60\x01\x73\x44\x77\x71\xc6\x5e\xc4\x51\xa3\x50\x72\x46\x8c\xea\x5a\x22\x65\x9c\xfc\xb6\xd6\xd9\x4a\xe9\xc8\x92\x1f\xcf\x8a\xf5\xf1\x68\xac\x71\x99\xcb\x68\xca\x89\x36\x2a\x4d\x55\x31\x17\x97\xc2\x18\x1b\xb2\xce\xa3\xbe\x95\x67\x22\x39\xbe\x76\x5f\x10\x48\xd0\xdf\x3a\x74\x5b\x0a\xf8\xd1\xb4\x77\xc6\xc1\x9c\xd2\x6e\x8c\xb6\x42\x0e\xaf\xe3\x14\x42\x5c\x8b\xd1\xd4\x9b\x2a\xd2\x3b\xfd\x77\x55\xfc\x83\x73\xec\x14\x16\xd6\x3d\x6d\x0e\x3f\xfd\xa1\x7f\x46\x53\xf8\xde\x9f\x2f\xb3\x9b\xab\x8b\xab\x0f\xf0\xec\x19\x9c\xcd\xae\x3e\x9c\xdf\xc0\xdd\xf5\xd5\x39\xfd\x9a\x76\x46\x53\xd8\x81\x0d\x1d\x27\xdd\x9c\x2f\x28\x64\xe0\xe2\x8c\x13\xaf\x20\xe7\x41\xef\x63\x9a\xbd\xa8\x60\x6b\xbb\x7d\x1f\xc1\x1d\x42\x94\xf2\x53\x2d\xc4\x35\x67\xef\x12\xb3\x7f\x96\x1a\x85\x3b\xa0\xfb\x0e\x1c\xee\x97\x96\x04\x2f\x5a\x74\x8d\x30\x68\x82\x26\xc4\xd1\xb6\x31\x46\xe8\x46\x0a\x64\x92\x8a\xfc\x6c\xad\xbc\x5a\x6a\xa4\xdd\x18\xdf\xf6\x51\x82\x49\x82\x92\xa3\x2a\x13\xd0\x48\x4a\x27\xc1\x72\xaa\x20\x2b\x07\x0b\x8d\xf0\x54\x46\x58\x9e\x41\x14\x16\x30\xe2\x92\xab\xf3\xcf\xe7\x37\x29\x4f\xed\xe8\x8a\x22\xc7\x76\x01\x3a\x4f\x34\x17\xd6\x15\x70\x65\x43\x7e\x2f\x89\x31\x9a\x42\xa5\x9c\x0f\xf1\x6e\xc1\x0c\x33\xd2\x29\xad\xa9\xd4\xaa\x73\x28\x73\xea\x92\x74\x0b\xd7\xe8\xb6\x40\x14\x35\xc6\x6b\x5d\x9b\x5f\x41\xb1\x55\x96\x4a\xa2\x09\x5c\xbf\x79\x1b\xe5\x5f\xca\x14\x9f\xf1\xcb\xa7\xdb\x05\x48\xd4\x18\x30\xbe\x93\x41\x6e\x0f\x7e\x53\xd0\xc6\x17\x52\xd2\x2c\xe0\x8c\x0e\xb3\xae\x6a\x7c\x74\x3a\xc6\x74\x65\x5d\xb9\x6b\xf1\xac\x54\x3a\x58\x55\xe8\xd0\x84\xc1\x56\x05\x17\x7d\xbe\xa7\x2d\x1d\x32\x5b\xae\xeb\x14\x5f\x07\x60\x9d\x44\xc7\xff\x42\x69\x95\xf1\x2c\x72\x2d\xd6\xe4\x85\x6b\x94\x94\x98\x68\x45\x5a\xf0\xb6\xf8\xf1\xc1\x93\xe2\xbd\xe9\xd3\x55\xd4\x83\x30\x80\xcd\x12\x25\x21\x4c\xda\x97\x02\x1b\x6b\x28\xbb\x3e\x6c\x63\x29\xee\xb1\x08\x67\xda\x6f\xd4\x2a\x2a\x61\xb9\x00\x13\x89\xde\x2b\x19\x29\x31\x43\x0e\x38\xda\xc3\x87\x52\x77\x5e\xad\x51\x6f\x99\x1e\xa5\xe0\x3e\x5a\xd8\x77\x5d\x06\x7c\xd6\x45\x20\x75\xd6\x09\x16\xb6\xbc\xdf\x11\x9e\x10\x74\x1b\x06\xd9\xf6\x4a\x67\x6d\x5d\xb7\xaa\xa3\xf4\xc4\x74\x76\x75\x36\x30\x19\x4d\x07\x36\x94\xe7\x1d\x56\xdc\x0f\x75\x42\xef\x30\x51\x9e\x50\x3f\xb4\x4e\xad\x45\xc0\x02\xae\xbf\x55\xa3\x53\x55\x1a\x4d\xa1\x11\x12\x07\x25\xec\x3f\x06\x3a\xa3\x29\xe8\x83\xd0\xf7\x29\x2c\x45\xac\x1a\xae\x33\x86\x56\x76\x95\xb2\xc4\x5a\x71\x97\x45\x91\x46\xb0\x3e\xcb\x15\x95\xf1\x83\x73\xf4\x14\xae\x48\x90\xdb\x54\x04\xe0\x19\x63\xa6\xca\x6a\x6d\x37\x24\x59\x84\xb9\x4f\xd7\x98\x9a\xae\x59\x12\x78\xa9\xc0\xa1\x6f\xad\x49\x60\x78\x23\x54\xe0\x74\xcc\xf0\xa0\x11\xac\xb7\x8b\xf9\xd5\x2d\x57\x60\xd5\xa3\x70\xe5\x41\x40\x70\x42\xa2\xad\x2a\x02\x39\x18\x36\x88\x31\x37\x8a\xb2\xec\x9c\x28\xb7\x44\x9c\x7e\xe7\xda\xdd\x57\x6d\xdf\x22\x4a\xd2\xaf\x6a\x8d\xff\xad\xb3\xae\x6b\x4e\x19\xdb\x9d\x45\x44\xcf\x87\x28\xd0\x6d\x15\x19\xcf\xbb\xa5\xef\x96\x31\xc2\x5b\x67\x97\x62\xa9\xb7\xb0\x11\x86\xab\x82\x4c\xe0\x21\x86\x70\x44\x22\x24\x1c\xbb\x0c\x31\x49\x3f\xd2\xd9\x25\xe6\x07\x09\xd0\xc2\xad\x76\x95\xb0\xfb\x44\xea\xcb\x55\x88\x3e\x46\x82\xb0\x0f\x35\xd8\xd8\xf8\x0a\x7a\xad\x30\x72\xa3\x64\xa8\x63\xeb\x41\x2f\x69\x7d\x74\x13\x02\xc5\x9f\x6e\x2e\x73\xb6\xaa\x62\xe4\xd5\xc2\xac\x10\x9c\x08\xa4\xc0\x5f\x28\x43\x53\x7a\xb6\xae\xc9\x95\xed\xad\x0a\x94\x9a\x66\x6b\x74\x62\x85\x3b\xc0\x38\x5f\xa6\xbb\xad\xb3\x6b\x25\xd1\x9d\xd6\x21\xb4\xfe\xcd\x78\x1c\x54\x79\x8f\x6e\xa7\xeb\x2c\xac\x5b\x8d\x45\xab\x76\xf5\x49\x85\x75\x27\x8d\x3a\xd4\x82\x92\x7a\xd5\x19\x0e\x26\xa1\x55\xd8\x12\x1b\x8a\xea\x1e\xfc\xb3\x1e\xc9\x64\xf1\xb7\x98\x57\x94\x59\x45\xc3\x55\xde\x1a\xbd\x4d\x0f\x6e\x5b\x34\x12\x04\x94\xb6\xe1\xbe\x3c\xbd\xa8\xf3\xe8\x40\xac\x68\x25\xe3\xc6\x61\x7a\x31\x24\xfb\x62\x34\xed\x44\xba\x7a\x9a\xfe\x7d\x92\x70\x23\xc3\xfc\x2d\xd1\x96\xdb\xd1\x8d\xf2\x35\x29\x07\x0d\x9b\xe5\xf6\xf6\x32\x83\x09\x12\x6d\xc8\x6e\x43\x84\xd5\x6a\x55\x13\x42\x71\x18\x15\x23\x91\x9c\x4f\x0d\x88\x23\xa7\x31\x8e\x2b\x86\xb8\x44\x52\x80\xc3\xc6\x06\xf2\xf6\xb2\x56\x06\xc9\x9f\x2b\xa1\x74\xe7\x30\xbb\x25\x31\x27\xff\xa6\xc2\x4c\x3a\xa0\x82\x49\x6d\x72\xb0\xbb\x90\x8b\xec\x5f\x5a\x13\x9c\xd5\x43\x74\x1d\x50\xea\xd7\x1d\xe3\x1c\xe9\x84\xea\x05\xd8\x08\xad\x63\x01\xf1\x5e\x47\xdf\x58\x0c\xdc\xb6\xb9\x3e\x1b\x8c\x60\x4b\x68\x6f\xfb\x16\x9d\xdd\x43\x84\x9a\x73\x50\xdf\x86\x96\xc8\x65\x52\xc2\x3d\x6e\x81\x5a\x0e\x32\x10\x45\x14\x0b\x43\xbb\xaa\x52\x25\x55\x89\xc8\x94\x56\xe8\xd8\xe9\x98\x68\x8d\x83\x1d\x7b\xaf\x0b\x5a\x8d\xfb\xf7\xb8\xfd\xf3\xf6\x3d\x6e\x73\x4e\x1c\xfc\x21\xf5\x1d\xb0\x14\x5e\x95\x20\xba\x50\x43\xe9\x90\x80\x91\x12\xda\xb3\x0c\xd9\x70\xc9\x1c\xd9\xba\x9d\xe7\x16\xa5\xa3\xae\x25\xa4\xf9\x19\xe3\x36\x22\x28\xc2\x00\xfa\x48\x31\xfc\x52\xd2\x0e\xd5\xc9\xfd\x3b\x8c\x38\x9d\x0d\x58\x92\xf0\xbd\x49\xa3\x95\x0b\xb8\x08\xff\xf4\x51\x85\xe4\x24\xbb\x3e\x32\xb0\x61\xb4\xb4\x4f\x94\xb0\x23\x81\x06\x03\xda\x96\x42\xd7\xd6\x87\xc8\x88\x36\x42\xea\xe6\x5a\x67\x57\x4e\x34\xa9\x89\x8a\x13\xb3\x6c\xe4\xd9\xfc\x82\x27\x8f\xe2\x9e\xfa\xaf\xfc\xa8\xac\x8b\x56\x78\xbf\xb1\x4e\xc2\x12\xc9\xa9\x32\x14\xa5\xed\x1a\x1f\x00\x4d\x69\x09\xed\xdc\x7e\x9c\x4d\x8e\x4f\xa0\x16\xbe\x06\x5b\xa5\x41\x90\x28\x03\xc1\x8d\x4c\x62\x88\x02\x99\x1c\x33\x69\x23\xf9\x4a\x62\xb4\xa9\xa9\x13\x55\x01\xbc\x0a\x9e\x3b\x56\x46\x19\xd1\x7d\x18\x01\xb3\xe3\x14\xf0\x85\xea\x19\x2b\x9f\x44\x17\x86\xe5\x75\xf8\x5b\x87\x3e\x0c\xce\x49\x74\xf3\xf5\xce\x3c\x23\x09\x39\xe6\x7a\x7e\xb9\x8a\xb1\xec\xb9\x37\x2e\x6d\xd3\x0a\x17\xdd\xba\xdf\x8c\xd0\x92\xa7\x8a\xa3\xa9\x68\x15\xe5\x43\x23\x1a\x3c\x15\x5a\x95\xc8\x4b\x99\xea\xe9\x31\xbe\x7a\xf5\xe2\xd5\xeb\x57\x52\x4c\x5e\x1d\xbe\x78\x79\x74\x7c\x24\x0f\xf1\xf8\xa4\x7a\x25\xcb\x93\xc9\xeb\xc9\xcb\x97\xcf\x4f\x0e\x9f\xcb\x43\x79\x22\xc4\x72\x29\xe5\xc9\x44\x1c\x1d\x61\xf5\x72\x72\x24\x8f\x8e\x5f\x4c\xe4\x2b\xce\xc3\x9e\x5e\x25\x34\x8f\xd3\x02\xb5\xfa\x14\x4a\x83\xff\x72\x3b\x25\x0c\x7b\x45\x69\xed\xbd\x62\xef\xa6\xee\xe0\x91\xaf\x2e\xb8\xaf\x68\x9d\x6a\x84\xdb\xc6\xe3\x22\x55\xb2\x90\x4c\x42\x3f\xf7\x5e\xc2\x1e\x90\x7e\xeb\x47\x7f\xc3\xd0\x25\x7a\x2c\xc3\xca\x3d\x13\x92\x27\xc1\x17\xa4\x0a\x4e\x50\x74\xf0\xdf\xe8\x08\x44\x23\x66\xeb\xc8\x75\x2d\x74\x97\x3a\x3c\xe5\x93\x69\xa9\x12\x77\x81\xca\x2a\xbb\xad\x88\x6e\xaa\x52\xc1\x71\x96\xa0\x68\x74\x84\xa6\x21\xc3\x69\x4a\x86\x29\xd5\xc7\xe1\x7b\x7c\x0e\xf1\xef\x4d\x1d\x53\xd9\xf6\x71\xf8\xf7\x1e\xa0\x7c\xb4\x67\xd4\xe1\xe9\xaf\x5f\xaf\xee\xef\x9a\xf7\xbf\xdf\x7d\x78\xdf\xdc\x7d\xbc\xaa\xef\x3e\x5e\x35\xc3\xda\x5d\x5d\x4e\x6e\x9a\xbb\xe6\xfd\xfd\xdd\x2a\x77\x02\xe4\xb3\x01\xa9\x3b\xc9\xb3\x96\x72\xa7\x2d\x44\x7f\x00\x6d\x9c\x5a\x37\xbd\xf7\x50\x5a\x42\xa9\xda\xd3\xc9\xab\xe2\xc5\x71\x71\xf2\xb2\x38\x7a\x79\xbc\xbb\xfe\x7c\x52\x4c\x9e\xbf\x2e\x8e\x0e\x5f\x17\x47\xc7\x9c\x7a\xdf\x5d\xdf\xdc\xf2\x10\x9b\xab\x8d\x84\xe5\x36\x7f\x2a\xa0\x36\x31\x8f\x4f\x79\xac\x13\xf6\x52\x5f\xb0\x50\x09\xed\x89\xaf\xb1\xa5\x75\x09\xd7\x5c\xec\xa7\xb9\x58\x35\xfa\xb9\x4d\x82\x57\xdc\x64\x0a\x82\x86\xa9\xd6\x13\x50\xc9\xb3\xbd\x83\x34\x3e\x53\xdc\xb6\xc4\x29\x2e\x59\x25\x43\xad\x2c\x52\x4c\x38\x89\x09\x87\x29\x1a\xd9\x5a\x65\x82\x27\xd5\x95\x75\x3e\x11\x7b\x2a\x55\x6d\x47\xd3\x3c\x0e\xfa\xa7\x4f\xe0\x3f\x36\x2e\x21\x62\x18\x26\x4f\x88\x25\x89\x5d\x61\xa0\xc2\xb8\x8a\x50\x84\x9c\x39\x8d\xb1\xd2\x44\x81\xc7\x59\xc5\x68\x1a\x1f\x91\xc4\x7f\xa2\x2e\xe0\x0b\x57\xcd\xbf\x07\x99\x9c\x9b\x1e\x70\x0f\x0c\x63\x19\x8f\x31\x4e\x06\x53\x66\x07\x3d\xf2\x40\x95\x9a\xeb\x64\x2c\x99\x8f\xf3\xa0\x2f\x66\x43\xaa\x54\x94\xfa\xf2\x08\x2e\x75\x8c\x28\xa1\xec\x9c\x43\x53\x12\xc4\xa6\x5a\x24\xca\x3a\x37\xe9\xc5\x28\xf9\x69\x24\x77\xfa\xf6\xdd\xc7\xc7\x2b\x8b\x77\x8f\x56\x2e\xff\xb4\x72\x77\xfe\x6e\x34\xdd\x5f\x3a\x5f\x7c\x7c\x12\xb3\xc5\xa1\xeb\xcc\x48\x78\x9f\x86\xae\xb7\x11\x7f\xfd\x7d\x86\xec\x81\x20\x89\xf6\x4c\x18\xf9\x6c\x7f\x1e\x9c\xda\xf7\x3f\x07\xae\xad\x2a\x74\x69\x70\x1b\xfb\x9b\xdd\x8b\xaa\xc4\x7e\x26\x3e\x8c\xd5\x1f\x8f\x7d\x97\x08\x22\xcf\xc9\x3a\x5f\x0f\x83\xd8\x38\x14\xc3\x44\x35\x0f\x9c\x77\x66\xea\xa1\xb6\x1e\xbf\x43\xca\x61\x70\x0a\xd7\xd1\x45\xf7\x27\xd7\xa1\xc6\x2d\x7f\x9b\x69\x28\x4d\x97\xf7\x14\xdf\x3c\xc9\x4e\x50\x2b\xf7\xa8\xad\xdd\xa0\x8b\xcd\x48\xaa\x27\x05\xdc\xf4\xb0\x59\xf9\xac\x1c\x5f\xdb\x4e\x73\xfe\xef\x3f\x24\x2e\x31\x62\x0f\x86\xd4\x4b\xfb\x10\x47\xd9\x02\xb4\x0d\xd4\x32\x46\xca\x71\xae\x65\xb9\x6b\x13\x3e\x21\x0c\xbe\x4b\xab\xb1\x09\x15\xb0\xb2\x56\x82\x44\xa1\xe9\x62\xfa\x7c\x1b\x1d\x75\x67\x38\xdd\x0f\xf3\xbf\x61\xbc\x38\x0a\x17\x3b\xa3\xc8\x28\x0d\x07\x51\xcc\x5e\x19\x94\xb6\x9d\x6b\x6d\xec\x9f\x1d\xa6\x6f\xb8\x2c\x06\xf3\x7d\x9c\xc8\x07\x52\x14\xd6\x91\x52\x66\x99\x21\x03\x52\x4a\x25\xda\xca\xe5\x81\x9b\xcf\xb5\xc9\x9b\x8a\x96\xfa\xd1\xfa\xdb\xf3\xdb\x72\x12\x6e\xcd\xfa\xf3\x0d\x36\x3f\x7b\x7f\xf6\x8b\xfa\xf9\xf2\x0e\x7f\xae\x3e\xdd\xd4\x9b\xaf\x62\x73\xf7\x45\x28\xfb\x9b\x9f\x3f\x5f\x1f\x6d\x9e\x24\x32\xcf\x70\xd9\xad\x9e\x24\xca\x98\x32\x68\xbb\x5a\x91\xf3\x68\x5c\xa3\x26\x2c\xfc\x99\x3f\x6a\xf1\xaf\xd1\x4a\xff\x2b\xe9\x20\xf5\x49\x95\x3d\x20\x70\xa1\x4a\x3c\x80\x8d\x70\xe4\x74\x07\x80\xce\x59\x77\x00\xa5\x53\x0c\x96\xfe\x6f\x34\x25\x9a\x7c\xff\x94\xae\xfc\xc5\x7f\x1d\xd0\x76\xd5\xf7\x41\xda\xae\xfe\xf4\xd1\x79\xac\xed\xaa\xff\xd2\xc7\x1f\x5b\xf3\xe7\x9f\xf4\x91\x93\x7c\xe4\xe3\x62\x31\xef\xbf\xe1\x24\x04\xec\x0b\x88\x77\xd2\xf2\x4e\xc6\xe0\xe1\xce\x90\xef\xf9\x23\xce\xf0\x19\x36\xa1\xa7\xfe\xa3\xd1\x23\x3a\xca\xc4\x49\x06\x1d\x25\x4f\xe2\x91\x5d\xff\x0d\x5e\x04\x06\x08\x6f\xc6\xe3\xbe\x1b\x79\xf3\x5f\xe9\x2a\x49\xff\xdf\x63\xd6\xe4\xb8\xa5\xb5\x38\xe2\x4f\x1d\x6f\xc1\x08\x95\x0f\x9e\x9e\x1c\x9e\x70\xe8\x7c\x71\x2a\x20\xbc\x9b\x7f\xea\xb9\xa7\xac\x35\x7c\xd1\xe2\x56\x80\xb2\x46\xdb\xe5\xdb\xe3\xd0\xb4\x3b\xff\x6f\xa1\xa0\xf5\xd1\xff\x07\x00\x00\xff\xff\x80\xfd\xc5\x73\x6f\x22\x00\x00")

func sampleOpenbazaarConfBytes() ([]byte, error) {
	return bindataRead(
		_sampleOpenbazaarConf,
		"sample-openbazaar.conf",
	)
}

func sampleOpenbazaarConf() (*asset, error) {
	bytes, err := sampleOpenbazaarConfBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "sample-openbazaar.conf", size: 8815, mode: os.FileMode(420), modTime: time.Unix(1583004073, 0)}
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
	"sample-openbazaar.conf": sampleOpenbazaarConf,
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
	"sample-openbazaar.conf": &bintree{sampleOpenbazaarConf, map[string]*bintree{}},
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
