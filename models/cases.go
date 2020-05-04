package models

import (
	"encoding/json"
	"github.com/OpenBazaar/jsonpb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
)

type Case struct {
	ID OrderID `gorm:"primary_key"`

	BuyerContract  json.RawMessage
	VendorContract json.RawMessage

	SerializedDisputeOpen  json.RawMessage
	SerializedDisputeClose json.RawMessage
}

func (c *Case) DisuteOpenMessage() (*pb.DisputeOpen, error) {
	if c.SerializedDisputeOpen == nil || len(c.SerializedDisputeOpen) == 0 {
		return nil, ErrMessageDoesNotExist
	}
	disputeOpen := new(pb.DisputeOpen)
	if err := jsonpb.UnmarshalString(string(c.SerializedDisputeOpen), disputeOpen); err != nil {
		return nil, err
	}
	return disputeOpen, nil
}

func (c *Case) PutDisputeOpen(disputeOpen *pb.DisputeOpen) error {
	if disputeOpen.OpenedBy == pb.DisputeOpen_BUYER {
		c.BuyerContract = disputeOpen.Contract
	} else {
		c.VendorContract = disputeOpen.Contract
	}

	disputeOpen.Contract = nil
	out, err := marshaler.MarshalToString(disputeOpen)
	if err != nil {
		return err
	}

	c.SerializedDisputeOpen = []byte(out)
	return nil
}

func (c *Case) PutDisputeUpdate(disputeUpdate *pb.DisputeUpdate) error {
	disputeOpen, err := c.DisuteOpenMessage()
	if err != nil {
		return err
	}

	if disputeOpen.OpenedBy == pb.DisputeOpen_BUYER {
		c.VendorContract = disputeUpdate.Contract
	} else {
		c.BuyerContract = disputeUpdate.Contract
	}

	return nil
}
