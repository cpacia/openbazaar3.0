package core

import (
	"context"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core/coreapi"
	fpath "github.com/ipfs/go-path"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-peer"
	"math"
	"math/rand"
	"os"
	"time"
)

// republishInterval is the amount of time to go between republishes.
const republishInterval = time.Hour * 36

// Publish will publish the current public data directory to IPNS.
// It will interrupt the publish if a shutdown happens during.
//
// This cannot be called with the database lock held.
func (n *OpenBazaarNode) Publish(done chan<- struct{}) {
	go func() {
		log.Info("Publishing to IPNS...")

		publishID := rand.Intn(math.MaxInt32)
		n.eventBus.Emit(&events.PublishStarted{
			ID: publishID,
		})

		defer func() {
			n.eventBus.Emit(&events.PublishFinished{
				ID: publishID,
			})
			if n.republishChan != nil {
				n.republishChan <- struct{}{}
			}
			log.Info("Publishing complete")
		}()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			select {
			case <-ctx.Done():
			case <-n.shutdown:
				cancel()
			}
			if done != nil {
				close(done)
			}
		}()

		api, err := coreapi.NewCoreAPI(n.ipfsNode)
		if err != nil {
			log.Errorf("Error building core API: %s", err.Error())
			return
		}

		currentRoot, err := n.ipnsRecordValue()

		// First uppin old root hash
		if err == nil {
			rp, err := api.ResolvePath(context.Background(), path.IpfsPath(currentRoot))
			if err != nil {
				log.Errorf("Error resolving path: %s", err.Error())
				return
			}

			if err := api.Pin().Rm(context.Background(), rp, options.Pin.RmRecursive(true)); err != nil {
				log.Errorf("Error unpinning root: %s", err.Error())
				return
			}
		}

		// Add the directory to IPFS
		stat, err := os.Lstat(n.repo.DB().PublicDataPath())
		if err != nil {
			log.Errorf("Error calling Lstat: %s", err.Error())
			return
		}

		f, err := files.NewSerialFile(n.repo.DB().PublicDataPath(), false, stat)
		if err != nil {
			log.Errorf("Error serializing file: %s", err.Error())
			return
		}

		opts := []options.UnixfsAddOption{
			options.Unixfs.Pin(true),
		}
		pth, err := api.Unixfs().Add(context.Background(), files.ToDir(f), opts...)
		if err != nil {
			log.Errorf("Error adding root: %s", err.Error())
			return
		}

		// Publish
		if err := n.ipfsNode.Namesys.Publish(ctx, n.ipfsNode.PrivateKey, fpath.FromString(pth.Root().String())); err != nil {
			log.Errorf("Error namesys publish: %s", err.Error())
			return
		}

		err = n.repo.DB().Update(func(tx database.Tx) error {
			return tx.Save(&models.Event{Name: "last_publish", Time: time.Now()})
		})
		if err != nil {
			log.Errorf("Error saving last publish time to the db: %s", err.Error())
		}

		// Send the new graph to our connected followers.
		graph, err := n.fetchGraph()
		if err != nil {
			log.Errorf("Error fetching graph: %s", err.Error())
			return
		}

		storeMsg := &pb.StoreMessage{}
		for _, cid := range graph {
			storeMsg.Cids = append(storeMsg.Cids, cid.Bytes())
		}

		any, err := ptypes.MarshalAny(storeMsg)
		if err != nil {
			log.Errorf("Error marshalling store message: %s", err.Error())
			return
		}

		msg := newMessageWithID()
		msg.MessageType = pb.Message_STORE
		msg.Payload = any
		for _, peer := range n.followerTracker.ConnectedFollowers() {
			go n.networkService.SendMessage(context.Background(), peer, msg)
		}
	}()
}

// sendAckMessage saves the incoming message ID in the database so we can
// check for duplicate messages later. Then it sends the ACK message to
// the remote peer.
func (n *OpenBazaarNode) sendAckMessage(messageID string, to peer.ID) {
	err := n.repo.DB().Update(func(tx database.Tx) error {
		return tx.Save(&models.IncomingMessage{ID: messageID})
	})
	if err != nil {
		log.Errorf("Error saving incoming message ID to database: %s", err)
	}
	n.messenger.SendACK(messageID, to)
}

// handleAckMessage is the handler for the ACK message. It sends it off to the messenger
// for processing. If this is an order message it also sends it to the order processor
// to be recorded there as well.
func (n *OpenBazaarNode) handleAckMessage(from peer.ID, message *pb.Message) error {
	if message.MessageType != pb.Message_ACK {
		return errors.New("message is not type ACK")
	}
	ack := new(pb.AckMessage)
	if err := ptypes.UnmarshalAny(message.Payload, ack); err != nil {
		return err
	}

	err := n.repo.DB().Update(func(tx database.Tx) error {
		var outgoingMessage models.OutgoingMessage
		if err := tx.Read().Where("id = ?", ack.AckedMessageID).First(&outgoingMessage).Error; err != nil {
			return err
		}
		if outgoingMessage.MessageType == pb.Message_ORDER.String() {
			if err := n.orderProcessor.ProcessACK(tx, &outgoingMessage); err != nil {
				return err
			}
		}
		return n.messenger.ProcessACK(tx, ack)
	})
	if err != nil {
		return err
	}

	n.eventBus.Emit(&events.MessageACK{MessageID: ack.AckedMessageID})
	return nil
}

// handleOrderMessage is the handler for the ORDER message. It sends it off to the order
// order processor for processing.
func (n *OpenBazaarNode) handleOrderMessage(from peer.ID, message *pb.Message) error {
	defer n.sendAckMessage(message.MessageID, from)

	if n.isDuplicate(message) {
		return nil
	}

	if message.MessageType != pb.Message_ORDER {
		return errors.New("message is not type ORDER")
	}
	order := new(pb.OrderMessage)
	if err := ptypes.UnmarshalAny(message.Payload, order); err != nil {
		return err
	}

	var event interface{}
	err := n.repo.DB().Update(func(tx database.Tx) error {
		var err error
		event, err = n.orderProcessor.ProcessMessage(tx, from, order)
		return err
	})
	if err != nil {
		return err
	}

	if event != nil {
		n.eventBus.Emit(event)
	}
	return nil
}

// handleStoreMessage is the handler for the STORE message. It will download and
// pin any objects sent to it from its followers.
func (n *OpenBazaarNode) handleStoreMessage(from peer.ID, message *pb.Message) error {
	if message.MessageType != pb.Message_STORE {
		return errors.New("message is not type STORE")
	}
	var (
		following models.Following
		err       error
	)
	err = n.repo.DB().View(func(tx database.Tx) error {
		following, err = tx.GetFollowing()
		return err
	})
	if err != nil {
		return err
	}
	if !following.IsFollowing(from) {
		return errors.New("STORE message from peer that is not followed")
	}

	store := new(pb.StoreMessage)
	if err := ptypes.UnmarshalAny(message.Payload, store); err != nil {
		return err
	}

	var cids []cid.Cid
	for _, b := range store.Cids {
		cid, err := cid.Cast(b)
		if err != nil {
			return fmt.Errorf("store handler cid cast error: %s", err)
		}
		cids = append(cids, cid)
		if err := n.pin(path.Join(path.New("/ipfs"), cid.String())); err != nil {
			return fmt.Errorf("store handler error pinning file: %s", err)
		}
	}
	n.eventBus.Emit(&events.MessageStore{
		Peer: from,
		Cids: cids,
	})
	log.Infof("Received STORE message from %s", from)
	return nil
}

// isDuplicate checks if the message ID exists in the incoming messages database.
func (n *OpenBazaarNode) isDuplicate(message *pb.Message) bool {
	err := n.repo.DB().View(func(tx database.Tx) error {
		return tx.Read().Where("id = ?", message.MessageID).First(&models.IncomingMessage{}).Error
	})
	return err == nil
}

// syncMessages listens for new connections to peers and checks to see if we have
// any outgoing messages for them. If so we send the messages over the direct
// connection.
func (n *OpenBazaarNode) syncMessages() {
	connectedSub, err := n.eventBus.Subscribe(&events.PeerConnected{})
	if err != nil {
		log.Error("Error subscribing to PeerConnected event: %s", err)
	}
	for {
		select {
		case event := <-connectedSub.Out():
			notif, ok := event.(*events.PeerConnected)
			if !ok {
				log.Error("syncMessages type assertion failed on PeerConnected")
				continue
			}
			var messages []models.OutgoingMessage
			err = n.repo.DB().View(func(tx database.Tx) error {
				return tx.Read().Where("recipient = ?", notif.Peer.Pretty()).Find(&messages).Error
			})
			if err != nil && !gorm.IsRecordNotFoundError(err) {
				log.Error("syncMessages outgoing messages lookup error: %s", err)
				continue
			}
			for _, om := range messages {
				var message pb.Message
				if err := proto.Unmarshal(om.SerializedMessage, &message); err != nil {
					log.Error("syncMessages unmarshal error: %s", err)
					continue
				}
				recipient, err := peer.IDB58Decode(om.Recipient)
				if err != nil {
					log.Error("syncMessages peer decode error: %s", err)
					continue
				}
				go n.networkService.SendMessage(context.Background(), recipient, &message)
			}
		case <-n.shutdown:
			return
		}
	}
}

// republish is a loop that runs and republishes the IPNS record. It shoots for
// 36 hours from the last republish so as to not slam the network on startup
// every time.
func (n *OpenBazaarNode) republish() {
	var lastPublish time.Time
	err := n.repo.DB().View(func(tx database.Tx) error {
		var event models.Event
		if err := tx.Read().Where("name = ?", "last_publish").First(&event).Error; err != nil {
			return err
		}
		lastPublish = event.Time
		return nil
	})
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		log.Error("Error loading last republish time: %s", err.Error())
	}

	n.republishChan = make(chan struct{})

	tick := time.After(republishInterval - time.Since(lastPublish))
	for {
		select {
		case <-tick:
			lastPublish = time.Now()
			tick = time.After(republishInterval - time.Since(lastPublish))
			err = n.repo.DB().Update(func(tx database.Tx) error {
				return tx.Save(&models.Event{Name: "last_publish", Time: lastPublish})
			})
			if err != nil {
				log.Errorf("Error saving last publish time to the db: %s", err.Error())
			}
			n.Publish(nil)
		case <-n.republishChan:
			lastPublish = time.Now()
			tick = time.After(republishInterval - time.Since(lastPublish))
		case <-n.shutdown:
			return
		}
	}
}
