package orders

import (
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/ptypes"
)

func (op *OrderProcessor) handleOrderOpenMessage(order *models.Order, message *npb.OrderMessage) (interface{}, error) {
	dup, err := isDuplicate(message, order.SerializedOrderOpen)
	if err != nil {
		return nil, err
	}
	if order.SerializedOrderOpen != nil && !dup {
		log.Error("Duplicate ORDER_OPEN message does not match original for order: %s", order.ID)
		return nil, ErrChangedMessage
	} else if dup {
		return nil, nil
	}

	orderOpen := new(pb.OrderOpen)
	if err := ptypes.UnmarshalAny(message.Message, orderOpen); err != nil {
		return nil, err
	}

	event := &events.OrderNotification{
		ID: string(order.ID),
	}

	return event, nil
}
