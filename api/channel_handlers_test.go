package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/ipfs/go-cid"
	"net/http"
	"testing"
)

func TestChannelHandlers(t *testing.T) {
	runAPITests(t, apiTests{
		{
			name:   "Post channel message",
			path:   "/v1/ob/channelmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.publishChannelMessage = func(ctx context.Context, topic, message string) error {
					return nil
				}
			},
			body:       []byte(`{"message": "hello", "topic": "general"}`),
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:   "Post invalid channel message",
			path:   "/v1/ob/channelmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.publishChannelMessage = func(ctx context.Context, topic, message string) error {
					return nil
				}
			},
			body:       []byte(`{"message": "hello", "topic": "general"`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "unexpected EOF"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post channel message error response",
			path:   "/v1/ob/channelmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.publishChannelMessage = func(ctx context.Context, topic, message string) error {
					return errors.New("error")
				}
			},
			body:       []byte(`{"message": "hello", "topic": "general"}`),
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "error"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post open channel",
			path:   "/v1/ob/openchannel/general",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.openChannel = func(topic string) error {
					return nil
				}
			},
			body:       nil,
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:   "Post open channel error",
			path:   "/v1/ob/openchannel/general",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.openChannel = func(topic string) error {
					return errors.New("error")
				}
			},
			body:       nil,
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "error"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post close channel",
			path:   "/v1/ob/closechannel/general",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.closeChannel = func(topic string) error {
					return nil
				}
			},
			body:       nil,
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:   "Post close channel error",
			path:   "/v1/ob/closechannel/general",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.closeChannel = func(topic string) error {
					return errors.New("error")
				}
			},
			body:       nil,
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "error"}%s`, "\n")), nil
			},
		},
		{
			name:   "Get list channels",
			path:   "/v1/ob/channels",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.listChannels = func() []string {
					return []string{"abc"}
				}
			},
			body:       nil,
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return marshalAndSanitizeJSON([]string{"abc"})
			},
		},
		{
			name:   "Get channel messages",
			path:   "/v1/ob/channelmessages/general",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChannelMessages = func(ctx context.Context, topic string, from *cid.Cid, limit int) ([]models.ChannelMessage, error) {
					ret := []models.ChannelMessage{
						{
							Topic: "general",
						},
					}
					return ret, nil
				}
			},
			body:       nil,
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				ret := []models.ChannelMessage{
					{
						Topic: "general",
					},
				}
				return marshalAndSanitizeJSON(ret)
			},
		},
		{
			name:   "Get channel messages with limit",
			path:   "/v1/ob/channelmessages/general?limit=5",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChannelMessages = func(ctx context.Context, topic string, from *cid.Cid, limit int) ([]models.ChannelMessage, error) {
					ret := []models.ChannelMessage{
						{
							Topic: "general",
						},
					}
					if limit != 5 {
						return nil, errors.New("limit error")
					}
					return ret, nil
				}
			},
			body:       nil,
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				ret := []models.ChannelMessage{
					{
						Topic: "general",
					},
				}
				return marshalAndSanitizeJSON(ret)
			},
		},
		{
			name:   "Get channel messages invalid limit",
			path:   "/v1/ob/channelmessages/general?limit=k",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChannelMessages = func(ctx context.Context, topic string, from *cid.Cid, limit int) ([]models.ChannelMessage, error) {
					ret := []models.ChannelMessage{
						{
							Topic: "general",
						},
					}
					if limit != 5 {
						return nil, errors.New("limit error")
					}
					return ret, nil
				}
			},
			body:       nil,
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "strconv.Atoi: parsing "k": invalid syntax"}%s`, "\n")), nil
			},
		},
		{
			name:   "Get channel messages with offset",
			path:   "/v1/ob/channelmessages/general?offsetID=QmTvGbPiS1PaE7AAn4gEszNiYMgdrbMXwLkGnLKYSADs8K",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChannelMessages = func(ctx context.Context, topic string, from *cid.Cid, limit int) ([]models.ChannelMessage, error) {
					ret := []models.ChannelMessage{
						{
							Topic: "general",
						},
					}
					if from.String() != "QmTvGbPiS1PaE7AAn4gEszNiYMgdrbMXwLkGnLKYSADs8K" {
						return nil, errors.New("offset error")
					}
					return ret, nil
				}
			},
			body:       nil,
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				ret := []models.ChannelMessage{
					{
						Topic: "general",
					},
				}
				return marshalAndSanitizeJSON(ret)
			},
		},
		{
			name:   "Get channel messages invalid offset",
			path:   "/v1/ob/channelmessages/general?offsetID=k",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChannelMessages = func(ctx context.Context, topic string, from *cid.Cid, limit int) ([]models.ChannelMessage, error) {
					ret := []models.ChannelMessage{
						{
							Topic: "general",
						},
					}
					return ret, nil
				}
			},
			body:       nil,
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "cid too short"}%s`, "\n")), nil
			},

		},
		{
			name:   "Get channel messages error",
			path:   "/v1/ob/channelmessages/general",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChannelMessages = func(ctx context.Context, topic string, from *cid.Cid, limit int) ([]models.ChannelMessage, error) {
					return nil, errors.New("error")
				}
			},
			body:       nil,
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "error"}%s`, "\n")), nil
			},
		},
	})
}
