package orders

import (
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/ptypes"
)

func (op *OrderProcessor) handleOrderOpenMessage(order *models.Order, message *npb.OrderMessage) error {
	orderOpen := new(pb.OrderOpen)
	if err := ptypes.UnmarshalAny(message.Message, orderOpen); err != nil {
		return err
	}

	return nil
}
