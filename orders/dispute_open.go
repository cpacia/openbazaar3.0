package orders

import (
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/ptypes"
	"github.com/libp2p/go-libp2p-core/peer"
)

func (op *OrderProcessor) processDisputeOpenMessage(dbtx database.Tx, order *models.Order, peer peer.ID, message *npb.OrderMessage) (interface{}, error) {
	disputeOpen := new(pb.DisputeOpen)
	if err := ptypes.UnmarshalAny(message.Message, disputeOpen); err != nil {
		return nil, err
	}
	dup, err := isDuplicate(disputeOpen, order.SerializedDisputeOpen)
	if err != nil {
		return nil, err
	}
	if order.SerializedDisputeOpen != nil && !dup {
		log.Errorf("Duplicate DISPUTE_OPEN message does not match original for order: %s", order.ID)
		return nil, ErrChangedMessage
	} else if dup {
		return nil, nil
	}

	return nil, nil
}
