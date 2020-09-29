package models

import (
	"encoding/json"
	"errors"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/jsonpb"
)

type Case struct {
	ID OrderID `gorm:"primary_key"`

	BuyerContract  json.RawMessage
	VendorContract json.RawMessage

	ValidationErrors json.RawMessage

	SerializedDisputeOpen  json.RawMessage
	SerializedDisputeClose json.RawMessage

	ParkedUpdate json.RawMessage
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

func (c *Case) OpenedBy() (pb.DisputeOpen_Party, error) {
	do, err := c.DisuteOpenMessage()
	if err != nil {
		return pb.DisputeOpen_BUYER, err
	}
	return do.OpenedBy, nil
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

	if c.ParkedUpdate != nil {
		if disputeOpen.OpenedBy == pb.DisputeOpen_BUYER {
			c.VendorContract = c.ParkedUpdate
			c.ParkedUpdate = nil
		} else {
			c.BuyerContract = c.ParkedUpdate
			c.ParkedUpdate = nil
		}
	}
	return nil
}

func (c *Case) PutDisputeUpdate(disputeUpdate *pb.DisputeUpdate) error {
	disputeOpen, err := c.DisuteOpenMessage()
	if err != nil && !errors.Is(err, ErrMessageDoesNotExist) {
		return err
	}

	if errors.Is(err, ErrMessageDoesNotExist) {
		c.ParkedUpdate = disputeUpdate.Contract
		return nil
	}

	if disputeOpen.OpenedBy == pb.DisputeOpen_BUYER {
		if c.VendorContract != nil {
			return errors.New("DISPUTE_UPDATE already exists")
		}
		c.VendorContract = disputeUpdate.Contract
	} else {
		if c.BuyerContract != nil {
			return errors.New("DISPUTE_UPDATE already exists")
		}
		c.BuyerContract = disputeUpdate.Contract
	}

	return nil
}

func (c *Case) PutValidationErrors(validationErrors []error) error {
	errStrs := make([]string, 0, len(validationErrors))
	for _, err := range validationErrors {
		errStrs = append(errStrs, err.Error())
	}

	out, err := json.MarshalIndent(errStrs, "", "    ")
	if err != nil {
		return err
	}

	c.ValidationErrors = out
	return nil
}
