package api

import (
	"context"
	"errors"
	"github.com/cpacia/openbazaar3.0/models"
	peer "github.com/libp2p/go-libp2p-peer"
	"net/http"
	"os"
	"testing"
)

func TestFollowHandlers(t *testing.T) {
	runAPITests(t, apiTests{
		{
			name:   "Get my followers",
			path:   "/v1/ob/followers",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getMyFollowersFunc = func() (models.Followers, error) {
					return models.Followers{
						"12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN",
						"12D3KooWBfmETW1ZbkdZbKKPpE3jpjyQ5WBXoDF8y9oE8vMQPKLi",
					}, nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				followers := models.Followers{
					"12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN",
					"12D3KooWBfmETW1ZbkdZbKKPpE3jpjyQ5WBXoDF8y9oE8vMQPKLi",
				}
				return marshalAndSanitizeJSON(&followers)
			},
		},
		{
			name:   "Get my followers nil",
			path:   "/v1/ob/followers",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getMyFollowersFunc = func() (models.Followers, error) {
					return nil, nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return []byte(`[]`), nil
			},
		},
		{
			name:   "Get my followers fail",
			path:   "/v1/ob/followers",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getMyFollowersFunc = func() (models.Followers, error) {
					return nil, errors.New("error")
				}
			},
			statusCode: http.StatusNotFound,
			expectedResponse: func() ([]byte, error) {
				return []byte("error\n"), nil
			},
		},
		{
			name:   "Get my following",
			path:   "/v1/ob/following",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getMyFollowingFunc = func() (models.Following, error) {
					return models.Following{
						"12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN",
						"12D3KooWBfmETW1ZbkdZbKKPpE3jpjyQ5WBXoDF8y9oE8vMQPKLi",
					}, nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				following := models.Following{
					"12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN",
					"12D3KooWBfmETW1ZbkdZbKKPpE3jpjyQ5WBXoDF8y9oE8vMQPKLi",
				}
				return marshalAndSanitizeJSON(&following)
			},
		},
		{
			name:   "Get my following nil",
			path:   "/v1/ob/following",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getMyFollowingFunc = func() (models.Following, error) {
					return nil, nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return []byte(`[]`), nil
			},
		},
		{
			name:   "Get my following fail",
			path:   "/v1/ob/following",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getMyFollowingFunc = func() (models.Following, error) {
					return nil, errors.New("error")
				}
			},
			statusCode: http.StatusNotFound,
			expectedResponse: func() ([]byte, error) {
				return []byte("error\n"), nil
			},
		},
		{
			name:   "Get followers",
			path:   "/v1/ob/followers/12D3KooWKLmVDz6sdzMyX1yQpEdCHB7dtxyEr91wPFNoCXEs2hkv",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getFollowersFunc = func(ctx context.Context, peerID peer.ID, useCache bool) (models.Followers, error) {
					if peerID.Pretty() == "12D3KooWKLmVDz6sdzMyX1yQpEdCHB7dtxyEr91wPFNoCXEs2hkv" {
						return models.Followers{
							"12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN",
							"12D3KooWBfmETW1ZbkdZbKKPpE3jpjyQ5WBXoDF8y9oE8vMQPKLi",
						}, nil
					}
					return nil, os.ErrNotExist
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				followers := models.Followers{
					"12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN",
					"12D3KooWBfmETW1ZbkdZbKKPpE3jpjyQ5WBXoDF8y9oE8vMQPKLi",
				}
				return marshalAndSanitizeJSON(&followers)
			},
		},
		{
			name:   "Get followers invalid peerID",
			path:   "/v1/ob/followers/xxx",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getFollowersFunc = func(ctx context.Context, peerID peer.ID, useCache bool) (models.Followers, error) {
					return nil, nil
				}
			},
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte("multihash length inconsistent: expected 13535, got 0\n"), nil
			},
		},
		{
			name:   "Get followers not found",
			path:   "/v1/ob/followers/12D3KooWKLmVDz6sdzMyX1yQpEdCHB7dtxyEr91wPFNoCXEs2hkv",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getFollowersFunc = func(ctx context.Context, peerID peer.ID, useCache bool) (models.Followers, error) {
					return nil, errors.New("not found")
				}
			},
			statusCode: http.StatusNotFound,
			expectedResponse: func() ([]byte, error) {
				return []byte("not found\n"), nil
			},
		},
		{
			name:   "Get following",
			path:   "/v1/ob/following/12D3KooWKLmVDz6sdzMyX1yQpEdCHB7dtxyEr91wPFNoCXEs2hkv",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getFollowingFunc = func(ctx context.Context, peerID peer.ID, useCache bool) (models.Following, error) {
					if peerID.Pretty() == "12D3KooWKLmVDz6sdzMyX1yQpEdCHB7dtxyEr91wPFNoCXEs2hkv" {
						return models.Following{
							"12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN",
							"12D3KooWBfmETW1ZbkdZbKKPpE3jpjyQ5WBXoDF8y9oE8vMQPKLi",
						}, nil
					}
					return nil, os.ErrNotExist
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				followers := models.Following{
					"12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN",
					"12D3KooWBfmETW1ZbkdZbKKPpE3jpjyQ5WBXoDF8y9oE8vMQPKLi",
				}
				return marshalAndSanitizeJSON(&followers)
			},
		},
		{
			name:   "Get following invalid peerID",
			path:   "/v1/ob/following/xxx",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getFollowingFunc = func(ctx context.Context, peerID peer.ID, useCache bool) (models.Following, error) {
					return nil, nil
				}
			},
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte("multihash length inconsistent: expected 13535, got 0\n"), nil
			},
		},
		{
			name:   "Get following not found",
			path:   "/v1/ob/following/12D3KooWKLmVDz6sdzMyX1yQpEdCHB7dtxyEr91wPFNoCXEs2hkv",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getFollowingFunc = func(ctx context.Context, peerID peer.ID, useCache bool) (models.Following, error) {
					return nil, errors.New("not found")
				}
			},
			statusCode: http.StatusNotFound,
			expectedResponse: func() ([]byte, error) {
				return []byte("not found\n"), nil
			},
		},
		{
			name:   "Post follow",
			path:   "/v1/ob/follow/12D3KooWKLmVDz6sdzMyX1yQpEdCHB7dtxyEr91wPFNoCXEs2hkv",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.followNodeFunc = func(peerID peer.ID, done chan<- struct{}) error {
					if peerID.Pretty() != "12D3KooWKLmVDz6sdzMyX1yQpEdCHB7dtxyEr91wPFNoCXEs2hkv" {
						return errors.New("invalid peerID")
					}
					return nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:   "Post follow fail",
			path:   "/v1/ob/follow/12D3KooWKLmVDz6sdzMyX1yQpEdCHB7dtxyEr91wPFNoCXEs2hkv",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.followNodeFunc = func(peerID peer.ID, done chan<- struct{}) error {
					return errors.New("error")
				}
			},
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte("error\n"), nil
			},
		},
		{
			name:   "Post follow invalid peerID",
			path:   "/v1/ob/follow/xxx",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.followNodeFunc = func(peerID peer.ID, done chan<- struct{}) error {
					return nil
				}
			},
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte("multihash length inconsistent: expected 13535, got 0\n"), nil
			},
		},
		{
			name:   "Post unfollow",
			path:   "/v1/ob/unfollow/12D3KooWKLmVDz6sdzMyX1yQpEdCHB7dtxyEr91wPFNoCXEs2hkv",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.unfollowNodeFunc = func(peerID peer.ID, done chan<- struct{}) error {
					if peerID.Pretty() != "12D3KooWKLmVDz6sdzMyX1yQpEdCHB7dtxyEr91wPFNoCXEs2hkv" {
						return errors.New("invalid peerID")
					}
					return nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:   "Post unfollow fail",
			path:   "/v1/ob/unfollow/12D3KooWKLmVDz6sdzMyX1yQpEdCHB7dtxyEr91wPFNoCXEs2hkv",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.unfollowNodeFunc = func(peerID peer.ID, done chan<- struct{}) error {
					return errors.New("error")
				}
			},
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte("error\n"), nil
			},
		},
		{
			name:   "Post unfollow invalid peerID",
			path:   "/v1/ob/unfollow/xxx",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.unfollowNodeFunc = func(peerID peer.ID, done chan<- struct{}) error {
					return nil
				}
			},
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte("multihash length inconsistent: expected 13535, got 0\n"), nil
			},
		},
	})
}