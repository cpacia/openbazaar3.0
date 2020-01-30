package api

import (
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/models"
	peer "github.com/libp2p/go-libp2p-peer"
	"net/http"
	"testing"
)

func TestChatHandlers(t *testing.T) {
	runAPITests(t, apiTests{
		{
			name:   "Post chat message",
			path:   "/v1/ob/chatmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendChatMessageFunc = func(to peer.ID, message string, orderID models.OrderID, done chan<- struct{}) error {
					return nil
				}
			},
			body:       []byte(`{"peerID": "12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN", "message": "", "orderID": ""}`),
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:   "Post chat message invalid JSON",
			path:   "/v1/ob/chatmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendChatMessageFunc = func(to peer.ID, message string, orderID models.OrderID, done chan<- struct{}) error {
					return nil
				}
			},
			body:       []byte(`"peerID": "12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN", "message": "", "orderID": ""}`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "json: cannot unmarshal string into Go value of type api.message"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post chat message invalid peer ID",
			path:   "/v1/ob/chatmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendChatMessageFunc = func(to peer.ID, message string, orderID models.OrderID, done chan<- struct{}) error {
					return nil
				}
			},
			body:       []byte(`{"peerID": "xxx", "message": "", "orderID": ""}`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "length greater than remaining number of bytes in buffer"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post chat message fail",
			path:   "/v1/ob/chatmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendChatMessageFunc = func(to peer.ID, message string, orderID models.OrderID, done chan<- struct{}) error {
					return errors.New("error")
				}
			},
			body:       []byte(`{"peerID": "12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN", "message": "", "orderID": ""}`),
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "error"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post group chat message",
			path:   "/v1/ob/groupchatmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendChatMessageFunc = func(to peer.ID, message string, orderID models.OrderID, done chan<- struct{}) error {
					return nil
				}
			},
			body:       []byte(`{"peerIDs": ["12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN"], "message": "", "orderID": ""}`),
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:   "Post group chat message invalid JSON",
			path:   "/v1/ob/groupchatmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendChatMessageFunc = func(to peer.ID, message string, orderID models.OrderID, done chan<- struct{}) error {
					return nil
				}
			},
			body:       []byte(`"peerID": "12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN", "message": "", "orderID": ""}`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "json: cannot unmarshal string into Go value of type api.message"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post group chat message invalid peer ID",
			path:   "/v1/ob/groupchatmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendChatMessageFunc = func(to peer.ID, message string, orderID models.OrderID, done chan<- struct{}) error {
					return nil
				}
			},
			body:       []byte(`{"peerIDs": ["xxx"], "message": "", "orderID": ""}`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "length greater than remaining number of bytes in buffer"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post group chat message fail",
			path:   "/v1/ob/groupchatmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendChatMessageFunc = func(to peer.ID, message string, orderID models.OrderID, done chan<- struct{}) error {
					return errors.New("error")
				}
			},
			body:       []byte(`{"peerIDs": ["12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN"], "message": "", "orderID": ""}`),
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "error"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post typing message",
			path:   "/v1/ob/typingmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendTypingMessageFunc = func(to peer.ID, orderID models.OrderID) error {
					return nil
				}
			},
			body:       []byte(`{"peerID": "12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN", "orderID": ""}`),
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:   "Post typing message invalid JSON",
			path:   "/v1/ob/typingmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendTypingMessageFunc = func(to peer.ID, orderID models.OrderID) error {
					return nil
				}
			},
			body:       []byte(`"peerID": "12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN", "orderID": ""}`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "json: cannot unmarshal string into Go value of type api.message"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post typing message invalid peerID",
			path:   "/v1/ob/typingmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendTypingMessageFunc = func(to peer.ID, orderID models.OrderID) error {
					return nil
				}
			},
			body:       []byte(`{"peerID": "xxx", "orderID": ""}`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "length greater than remaining number of bytes in buffer"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post typing message fail",
			path:   "/v1/ob/typingmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendTypingMessageFunc = func(to peer.ID, orderID models.OrderID) error {
					return errors.New("error")
				}
			},
			body:       []byte(`{"peerID": "12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN", "orderID": ""}`),
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "error"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post group typing message",
			path:   "/v1/ob/grouptypingmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendTypingMessageFunc = func(to peer.ID, orderID models.OrderID) error {
					return nil
				}
			},
			body:       []byte(`{"peerIDs": ["12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN"], "orderID": ""}`),
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:   "Post group typing message invalid JSON",
			path:   "/v1/ob/grouptypingmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendTypingMessageFunc = func(to peer.ID, orderID models.OrderID) error {
					return nil
				}
			},
			body:       []byte(`"peerIDs": ["12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN"], "orderID": ""}`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "json: cannot unmarshal string into Go value of type api.message"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post group typing message invalid peerID",
			path:   "/v1/ob/grouptypingmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendTypingMessageFunc = func(to peer.ID, orderID models.OrderID) error {
					return nil
				}
			},
			body:       []byte(`{"peerIDs": ["xxx"], "orderID": ""}`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "length greater than remaining number of bytes in buffer"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post group typing message fail",
			path:   "/v1/ob/grouptypingmessage",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.sendTypingMessageFunc = func(to peer.ID, orderID models.OrderID) error {
					return errors.New("error")
				}
			},
			body:       []byte(`{"peerIDs": ["12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN"], "orderID": ""}`),
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "error"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post mark chat as read",
			path:   "/v1/ob/markchatasread",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.markChatMessagesAsReadFunc = func(to peer.ID, orderID models.OrderID) error {
					return nil
				}
			},
			body:       []byte(`{"peerID": "12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN", "orderID": ""}`),
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:   "Post mark chat as read invalid JSON",
			path:   "/v1/ob/markchatasread",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.markChatMessagesAsReadFunc = func(to peer.ID, orderID models.OrderID) error {
					return nil
				}
			},
			body:       []byte(`"peerID": "12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN", "orderID": ""}`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "json: cannot unmarshal string into Go value of type api.message"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post mark chat as read invalid peerID",
			path:   "/v1/ob/markchatasread",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.markChatMessagesAsReadFunc = func(to peer.ID, orderID models.OrderID) error {
					return nil
				}
			},
			body:       []byte(`{"peerID": "xxx", "orderID": ""}`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "length greater than remaining number of bytes in buffer"}%s`, "\n")), nil
			},
		},
		{
			name:   "Post mark chat as read fail",
			path:   "/v1/ob/markchatasread",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.markChatMessagesAsReadFunc = func(to peer.ID, orderID models.OrderID) error {
					return errors.New("error")
				}
			},
			body:       []byte(`{"peerID": "12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN", "orderID": ""}`),
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "error"}%s`, "\n")), nil
			},
		},
		{
			name:   "Get chat conversations",
			path:   "/v1/ob/chatconversations",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChatConversationsFunc = func() ([]models.ChatConversation, error) {
					return []models.ChatConversation{
						{
							PeerID: "abc",
						},
					}, nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				response := []models.ChatConversation{
					{
						PeerID: "abc",
					},
				}
				return marshalAndSanitizeJSON(&response)
			},
		},
		{
			name:   "Get chat conversations nil",
			path:   "/v1/ob/chatconversations",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChatConversationsFunc = func() ([]models.ChatConversation, error) {
					return nil, nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return []byte(`[]`), nil
			},
		},
		{
			name:   "Get chat conversations fail",
			path:   "/v1/ob/chatconversations",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChatConversationsFunc = func() ([]models.ChatConversation, error) {
					return nil, errors.New("error")
				}
			},
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "error"}%s`, "\n")), nil
			},
		},
		{
			name:   "Get chat messages",
			path:   "/v1/ob/chatmessages/12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChatMessagesByPeerFunc = func(peerID peer.ID, limit int, offsetID string) ([]models.ChatMessage, error) {
					return []models.ChatMessage{
						{
							PeerID: "abc",
						},
					}, nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				response := []models.ChatMessage{
					{
						PeerID: "abc",
					},
				}
				return marshalAndSanitizeJSON(&response)
			},
		},
		{
			name:   "Get chat messages nil",
			path:   "/v1/ob/chatmessages/12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChatMessagesByPeerFunc = func(peerID peer.ID, limit int, offsetID string) ([]models.ChatMessage, error) {
					return nil, nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return []byte(`[]`), nil
			},
		},
		{
			name:   "Get chat messages with limit",
			path:   "/v1/ob/chatmessages/12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN?limit=2",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChatMessagesByPeerFunc = func(peerID peer.ID, limit int, offsetID string) ([]models.ChatMessage, error) {
					if limit != 2 {
						return nil, errors.New("invalid limit")
					}
					return []models.ChatMessage{
						{
							PeerID: "abc",
						},
					}, nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				response := []models.ChatMessage{
					{
						PeerID: "abc",
					},
				}
				return marshalAndSanitizeJSON(&response)
			},
		},
		{
			name:   "Get chat messages invalid limit",
			path:   "/v1/ob/chatmessages/12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN?limit=a",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChatMessagesByPeerFunc = func(peerID peer.ID, limit int, offsetID string) ([]models.ChatMessage, error) {
					if limit != 2 {
						return nil, errors.New("invalid limit")
					}
					return []models.ChatMessage{
						{
							PeerID: "abc",
						},
					}, nil
				}
			},
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "strconv.Atoi: parsing "a": invalid syntax"}%s`, "\n")), nil
			},
		},
		{
			name:   "Get chat messages invalid peerID",
			path:   "/v1/ob/chatmessages/adsf",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChatMessagesByPeerFunc = func(peerID peer.ID, limit int, offsetID string) ([]models.ChatMessage, error) {
					return nil, nil
				}
			},
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "length greater than remaining number of bytes in buffer"}%s`, "\n")), nil
			},
		},
		{
			name:   "Get chat messages fail",
			path:   "/v1/ob/chatmessages/12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChatMessagesByPeerFunc = func(peerID peer.ID, limit int, offsetID string) ([]models.ChatMessage, error) {
					return nil, errors.New("error")
				}
			},
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "error"}%s`, "\n")), nil
			},
		},
		{
			name:   "Get group chat messages",
			path:   "/v1/ob/groupchatmessages/abc",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChatMessagesByOrderIDFunc = func(orderID models.OrderID, limit int, offsetID string) ([]models.ChatMessage, error) {
					return []models.ChatMessage{
						{
							OrderID: "abc",
						},
					}, nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				response := []models.ChatMessage{
					{
						OrderID: "abc",
					},
				}
				return marshalAndSanitizeJSON(&response)
			},
		},
		{
			name:   "Get group chat messages nil",
			path:   "/v1/ob/groupchatmessages/abc",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChatMessagesByOrderIDFunc = func(orderID models.OrderID, limit int, offsetID string) ([]models.ChatMessage, error) {
					return nil, nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return []byte(`[]`), nil
			},
		},
		{
			name:   "Get group chat messages with limit",
			path:   "/v1/ob/groupchatmessages/abc?limit=2",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChatMessagesByOrderIDFunc = func(orderID models.OrderID, limit int, offsetID string) ([]models.ChatMessage, error) {
					if limit != 2 {
						return nil, errors.New("invalid limit")
					}
					return nil, nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return []byte(`[]`), nil
			},
		},
		{
			name:   "Get group chat messages invalid limit",
			path:   "/v1/ob/groupchatmessages/abc?limit=a",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChatMessagesByOrderIDFunc = func(orderID models.OrderID, limit int, offsetID string) ([]models.ChatMessage, error) {
					return nil, nil
				}
			},
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "strconv.Atoi: parsing "a": invalid syntax"}%s`, "\n")), nil
			},
		},
		{
			name:   "Get group chat messages fail",
			path:   "/v1/ob/groupchatmessages/abc",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getChatMessagesByOrderIDFunc = func(orderID models.OrderID, limit int, offsetID string) ([]models.ChatMessage, error) {
					return nil, errors.New("error")
				}
			},
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "error"}%s`, "\n")), nil
			},
		},
		{
			name:   "Delete message",
			path:   "/v1/ob/chatmessage/abc",
			method: http.MethodDelete,
			setNodeMethods: func(n *mockNode) {
				n.deleteChatMessageFunc = func(messageID string) error {
					if messageID != "abc" {
						return errors.New("error")
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
			name:   "Delete chat message fail",
			path:   "/v1/ob/chatmessage/abc",
			method: http.MethodDelete,
			setNodeMethods: func(n *mockNode) {
				n.deleteChatMessageFunc = func(messageID string) error {
					return errors.New("error")
				}
			},
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "error"}%s`, "\n")), nil
			},
		},
		{
			name:   "Delete group chat messages",
			path:   "/v1/ob/groupchatmessages/abc",
			method: http.MethodDelete,
			setNodeMethods: func(n *mockNode) {
				n.deleteGroupChatMessagesFunc = func(orderID models.OrderID) error {
					if orderID.String() != "abc" {
						return errors.New("error")
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
			name:   "Delete group chat messages fail",
			path:   "/v1/ob/groupchatmessages/abc",
			method: http.MethodDelete,
			setNodeMethods: func(n *mockNode) {
				n.deleteGroupChatMessagesFunc = func(orderID models.OrderID) error {
					return errors.New("error")
				}
			},
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "error"}%s`, "\n")), nil
			},
		},
		{
			name:   "Delete chat conversation",
			path:   "/v1/ob/chatconversation/12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN",
			method: http.MethodDelete,
			setNodeMethods: func(n *mockNode) {
				n.deleteChatConversationFunc = func(peerID peer.ID) error {
					if peerID.String() != "12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN" {
						return errors.New("error")
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
			name:   "Delete chat conversation invalid peerID",
			path:   "/v1/ob/chatconversation/xxx",
			method: http.MethodDelete,
			setNodeMethods: func(n *mockNode) {
				n.deleteChatConversationFunc = func(peerID peer.ID) error {
					if peerID.String() != "12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN" {
						return errors.New("error")
					}
					return nil
				}
			},
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "length greater than remaining number of bytes in buffer"}%s`, "\n")), nil
			},
		},
		{
			name:   "Delete chat conversation fail",
			path:   "/v1/ob/chatconversation/12D3KooWLbTBv97L6jvaLkdSRpqhCX3w7PyPDWU7kwJsKJyztAUN",
			method: http.MethodDelete,
			setNodeMethods: func(n *mockNode) {
				n.deleteChatConversationFunc = func(peerID peer.ID) error {
					return errors.New("error")
				}
			},
			statusCode: http.StatusInternalServerError,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf(`{"error": "error"}%s`, "\n")), nil
			},
		},
	})
}
