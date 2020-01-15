package models

import "encoding/json"

// ChannelInfo holds info about a chat room channel. It's basically
// just the name and the last CID(s) seen in the channel.
type ChannelInfo struct {
	Name     string `gorm:"primary_key"`
	LastCIDs json.RawMessage
}

// GetLastCIDs unmarshals the CID slice and returns it.
func (ci *ChannelInfo) GetLastCIDs() ([]string, error) {
	var cids []string
	if err := json.Unmarshal(ci.LastCIDs, cids); err != nil {
		return nil, err
	}
	return cids, nil
}

// SetLastCIDs marshals and sets the CIDs.
func (ci *ChannelInfo) SetLastCIDs(cids []string) error {
	out, err := json.MarshalIndent(cids, "", "    ")
	if err != nil {
		return err
	}
	ci.LastCIDs = out
	return nil
}
