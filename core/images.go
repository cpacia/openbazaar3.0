package core

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/disintegration/imaging"
	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core/coreapi"
	ipath "github.com/ipfs/interface-go-ipfs-core/path"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"image"
	"image/jpeg"
	"io"
	"strings"
)

// GetImage loads the image from the network or cache and returns a
// reader to the underlying data.
func (n *OpenBazaarNode) GetImage(ctx context.Context, cid cid.Cid) (io.ReadSeeker, error) {
	api, err := coreapi.NewCoreAPI(n.ipfsNode)
	if err != nil {
		return nil, err
	}
	pth := ipath.IpfsPath(cid)

	nd, err := api.Unixfs().Get(ctx, pth)
	if err != nil {
		return nil, fmt.Errorf("%w: image not found", coreiface.ErrNotFound)
	}
	f, ok := nd.(files.File)
	if !ok {
		return nil, errors.New("error asserting ipfs file type")
	}
	return f, nil
}

// GetAvatar loads the image from the network or cache and returns a
// reader to the underlying data.
func (n *OpenBazaarNode) GetAvatar(ctx context.Context, peerID peer.ID, size models.ImageSize, useCache bool) (io.ReadSeeker, error) {
	pth, err := n.resolve(ctx, peerID, useCache)
	if err != nil {
		return nil, err
	}

	api, err := coreapi.NewCoreAPI(n.ipfsNode)
	if err != nil {
		return nil, err
	}

	nd, err := api.Unixfs().Get(ctx, ipath.Join(pth, "images", string(size), "avatar"))
	if err != nil {
		return nil, fmt.Errorf("%w: avatar not found", coreiface.ErrNotFound)
	}
	f, ok := nd.(files.File)
	if !ok {
		return nil, errors.New("error asserting ipfs file type")
	}
	return f, nil
}

// GetHeader loads the image from the network or cache and returns a
// reader to the underlying data.
func (n *OpenBazaarNode) GetHeader(ctx context.Context, peerID peer.ID, size models.ImageSize, useCache bool) (io.ReadSeeker, error) {
	pth, err := n.resolve(ctx, peerID, useCache)
	if err != nil {
		return nil, err
	}

	api, err := coreapi.NewCoreAPI(n.ipfsNode)
	if err != nil {
		return nil, err
	}

	nd, err := api.Unixfs().Get(ctx, ipath.Join(pth, "images", string(size), "header"))
	if err != nil {
		return nil, fmt.Errorf("%w: header not found", coreiface.ErrNotFound)
	}
	f, ok := nd.(files.File)
	if !ok {
		return nil, errors.New("error asserting ipfs file type")
	}
	return f, nil
}

// SetAvatarImage saves the avatar image, updates the profile, and republishes
func (n *OpenBazaarNode) SetAvatarImage(base64ImageData string, done chan struct{}) (models.ImageHashes, error) {
	var (
		hashes         models.ImageHashes
		profileUpdated bool
		err            error
	)
	err = n.repo.DB().Update(func(tx database.Tx) error {
		hashes, err = n.resizeAndAddImage(tx, base64ImageData, "avatar", 60, 60)
		if err != nil {
			return err
		}

		profile, err := tx.GetProfile()
		if err == nil {
			profile.AvatarHashes = hashes
			profileUpdated = true
			return tx.SetProfile(profile)
		}
		return nil
	})
	if err != nil {
		maybeCloseDone(done)
		return models.ImageHashes{}, err
	}

	if profileUpdated {
		n.Publish(done)
	}
	return hashes, nil
}

// SetHeaderImage saves the header image, updates the profile, and republishes
func (n *OpenBazaarNode) SetHeaderImage(base64ImageData string, done chan struct{}) (models.ImageHashes, error) {
	var (
		hashes         models.ImageHashes
		err            error
		profileUpdated bool
	)
	err = n.repo.DB().Update(func(tx database.Tx) error {
		hashes, err = n.resizeAndAddImage(tx, base64ImageData, "header", 315, 90)
		if err != nil {
			return err
		}

		profile, err := tx.GetProfile()
		if err == nil {
			profile.HeaderHashes = hashes
			profileUpdated = true
			return tx.SetProfile(profile)
		}
		return nil
	})
	if err != nil {
		maybeCloseDone(done)
		return models.ImageHashes{}, err
	}

	if profileUpdated {
		n.Publish(done)
	}
	return hashes, nil
}

// SetProductImage saves the image with the provided filename. This method does NOT
// publish the changes as it's generally expected that the product images will be
// added prior to saving a listing and the listing save will trigger the publish.
func (n *OpenBazaarNode) SetProductImage(base64ImageData string, filename string) (models.ImageHashes, error) {
	var (
		hashes models.ImageHashes
		err    error
	)
	err = n.repo.DB().Update(func(tx database.Tx) error {
		hashes, err = n.resizeAndAddImage(tx, base64ImageData, filename, 120, 120)
		if err != nil {
			return err
		}

		return nil
	})
	return hashes, err
}

func (n *OpenBazaarNode) resizeAndAddImage(dbtx database.Tx, base64ImageData, filename string, baseWidth, baseHeight int) (models.ImageHashes, error) {
	img, err := decodeImageData(base64ImageData)
	if err != nil {
		return models.ImageHashes{}, fmt.Errorf("%w: invalid image: %s", coreiface.ErrBadRequest, err.Error())
	}

	t, err := n.addResizedImage(dbtx, img, 1*baseWidth, 1*baseHeight, filename, models.ImageSizeTiny)
	if err != nil {
		return models.ImageHashes{}, err
	}
	s, err := n.addResizedImage(dbtx, img, 2*baseWidth, 2*baseHeight, filename, models.ImageSizeSmall)
	if err != nil {
		return models.ImageHashes{}, err
	}
	m, err := n.addResizedImage(dbtx, img, 4*baseWidth, 4*baseHeight, filename, models.ImageSizeMedium)
	if err != nil {
		return models.ImageHashes{}, err
	}
	l, err := n.addResizedImage(dbtx, img, 8*baseWidth, 8*baseHeight, filename, models.ImageSizeLarge)
	if err != nil {
		return models.ImageHashes{}, err
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		return models.ImageHashes{}, err
	}

	o, err := n.addImage(dbtx, models.Image{
		Name:       filename,
		Size:       models.ImageSizeOriginal,
		ImageBytes: buf.Bytes(),
	})
	if err != nil {
		return models.ImageHashes{}, err
	}

	return models.ImageHashes{Tiny: t.String(), Small: s.String(), Medium: m.String(), Large: l.String(), Original: o.String(), Filename: filename}, nil
}

func (n *OpenBazaarNode) addImage(dbtx database.Tx, img models.Image) (cid.Cid, error) {
	if err := dbtx.SetImage(img); err != nil {
		return cid.Cid{}, err
	}

	return n.cid(img.ImageBytes)
}

func (n *OpenBazaarNode) addResizedImage(dbtx database.Tx, img image.Image, w, h int, name string, size models.ImageSize) (cid.Cid, error) {
	width, height := getImageAttributes(w, h, img.Bounds().Max.X, img.Bounds().Max.Y)
	newImg := imaging.Resize(img, width, height, imaging.Lanczos)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, newImg, nil); err != nil {
		return cid.Cid{}, err
	}

	m := models.Image{
		ImageBytes: buf.Bytes(),
		Size:       size,
		Name:       name,
	}

	return n.addImage(dbtx, m)
}

func decodeImageData(base64ImageData string) (image.Image, error) {
	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(base64ImageData))
	img, err := imaging.Decode(reader, imaging.AutoOrientation(true))
	if err != nil {
		return nil, err
	}
	return img, err
}

func getImageAttributes(targetWidth, targetHeight, imgWidth, imgHeight int) (width, height int) {
	targetRatio := float32(targetWidth) / float32(targetHeight)
	imageRatio := float32(imgWidth) / float32(imgHeight)
	var h, w float32
	if imageRatio > targetRatio {
		h = float32(targetHeight)
		w = float32(targetHeight) * imageRatio
	} else {
		w = float32(targetWidth)
		h = float32(targetWidth) * (float32(imgHeight) / float32(imgWidth))
	}
	return int(w), int(h)
}
