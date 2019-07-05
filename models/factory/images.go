package factory

import "github.com/cpacia/openbazaar3.0/orders/pb"

func NewImage() *pb.Listing_Item_Image {
	return &pb.Listing_Item_Image{
		Filename: "image.jpg",
		Tiny:     "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
		Small:    "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
		Medium:   "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
		Large:    "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
		Original: "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
	}
}
