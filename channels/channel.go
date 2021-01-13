package channels

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/cpacia/openbazaar3.0/channels/pb"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/net"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-merkledag"
	iface "github.com/ipfs/interface-go-ipfs-core"
	caopts "github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/op/go-logging"
	"gorm.io/gorm"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	topicPrefix       = "/channel/"
	maxBootstrapPeers = 5
)

var log = logging.MustGetLogger("CHAN")

// Channel represents an open chat channel.
type Channel struct {
	bus      events.Bus
	pubsub   iface.PubSubAPI
	object   iface.ObjectAPI
	topic    string
	ns       *net.NetworkService
	db       database.Database
	privKey  crypto.PrivKey
	identity peer.ID
	cache    map[cid.Cid]bool
	cacheMtx sync.RWMutex

	boostrapped bool
	shutdown    chan struct{}
}

// NewChannel instantiates a new chat channel, subscribes to the pubsub topic, and bootstraps the initial messages.
func NewChannel(topic string, ipfsNode *core.IpfsNode, ns *net.NetworkService, bus events.Bus, db database.Database) (*Channel, error) {
	api, err := coreapi.NewCoreAPI(ipfsNode)
	if err != nil {
		return nil, err
	}

	c := &Channel{
		bus:      bus,
		pubsub:   api.PubSub(),
		object:   api.Object(),
		topic:    strings.ToLower(topic),
		db:       db,
		ns:       ns,
		privKey:  ipfsNode.PrivateKey,
		identity: ipfsNode.Identity,
		cache:    make(map[cid.Cid]bool),
		cacheMtx: sync.RWMutex{},
		shutdown: make(chan struct{}),
	}
	if err := c.run(); err != nil {
		return nil, err
	}
	return c, nil
}

// Topic returns the topic of this channel.
func (c *Channel) Topic() string {
	return c.topic
}

// Publish broadcasts a message to this chat channel. The message will contain
// a pointer to the previous message(s) in the channel so that the channel
// history can be loaded by traversing the DAG backwards.
func (c *Channel) Publish(ctx context.Context, message string) error {
	msg := &pb.ChannelMessage{
		Message:   message,
		Topic:     c.topic,
		PeerID:    c.identity.Pretty(),
		Timestamp: ptypes.TimestampNow(),
	}
	ser, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	sig, err := c.privKey.Sign(ser)
	if err != nil {
		return err
	}
	msg.Signature = sig

	serializedMsg, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	nd, err := c.object.New(ctx)
	if err != nil {
		return err
	}

	pth, err := c.object.SetData(ctx, path.IpldPath(nd.Cid()), bytes.NewReader(serializedMsg))
	if err != nil {
		return err
	}

	var channelRec models.Channel
	err = c.db.View(func(tx database.Tx) error {
		return tx.Read().Where("topic=?", c.topic).First(&channelRec).Error
	})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil {
		links, err := channelRec.GetHead()
		if err != nil {
			return err
		}
		for i, link := range links {
			pth, err = c.object.AddLink(ctx, pth, "previousMessage"+strconv.Itoa(i), path.IpldPath(link))
			if err != nil {
				return err
			}
		}
	}

	ind, err := c.object.Get(ctx, pth)
	if err != nil {
		return err
	}

	pnd, ok := ind.(*merkledag.ProtoNode)
	if !ok {
		return errors.New("protoNode type assertion error")
	}
	serializedObj, err := pnd.Marshal()
	if err != nil {
		return err
	}

	return c.pubsub.Publish(ctx, topicPrefix+c.topic, serializedObj)
}

// Messages returns the next `limit` messages starting from the `from` cid.
// If no cid is provided this will return messages starting at the head of
// the channel. If no more messages could be found it will return a nil slice.
func (c *Channel) Messages(ctx context.Context, from *cid.Cid, limit int) ([]models.ChannelMessage, error) {
	if limit <= 0 {
		limit = 20
	}

	level := make(map[cid.Cid]bool)
	if from != nil {
		c.cacheMtx.RLock()
		if c.cache[*from] {
			level = c.cache
		}
		c.cacheMtx.RUnlock()
	} else {
		var channelRec models.Channel
		err := c.db.View(func(tx database.Tx) error {
			return tx.Read().Where("topic=?", c.topic).First(&channelRec).Error
		})
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.ChannelMessage{}, nil
		} else if err != nil {
			return nil, err
		}

		ids, err := channelRec.GetHead()
		if err != nil {
			return nil, err
		}

		for _, id := range ids {
			level[id] = true
		}
	}

	ret := make([]models.ChannelMessage, 0, limit)
	for {
		nextLevel := make(map[cid.Cid]bool)
		for id := range level {
			nd, err := c.object.Get(ctx, path.IpldPath(id))
			if err != nil {
				continue
			}
			pnd, ok := nd.(*merkledag.ProtoNode)
			if !ok {
				continue
			}

			var cm pb.ChannelMessage
			if err := proto.Unmarshal(pnd.Data(), &cm); err != nil {
				continue
			}

			valid, err := validateMessage(&cm)
			if err != nil || !valid {
				continue
			}
			if from == nil || nd.Cid().String() != from.String() {
				ret = append(ret, models.ChannelMessage{
					PeerID:    cm.PeerID,
					Topic:     c.topic,
					Message:   cm.Message,
					Timestamp: time.Unix(cm.Timestamp.Seconds, int64(cm.Timestamp.Nanos)),
					Cid:       nd.Cid().String(),
				})
			}

			for _, link := range nd.Links() {
				nextLevel[link.Cid] = true
			}
		}
		if len(nextLevel) == 0 || len(ret) >= limit {
			break
		}
		level = nextLevel
	}
	c.cacheMtx.Lock()
	c.cache = level
	c.cacheMtx.Unlock()

	sort.Slice(ret, func(i, j int) bool {
		return ret[j].Timestamp.Before(ret[i].Timestamp)
	})

	return ret, nil
}

// Close will shutdown the channel.
func (c *Channel) Close() {
	close(c.shutdown)
}

// run is subscribing to the pubsub topic and bootstrapping the last
// known messages in the channel. If a new message is received on the channel
// before the bootstrap finishes we will set that message as the head and
// terminate the bootstrapping.
func (c *Channel) run() error {
	ctx, cancel := context.WithCancel(context.Background()) //nolint
	sub, err := c.pubsub.Subscribe(ctx, topicPrefix+c.topic, caopts.PubSub.Discover(true))
	if err != nil {
		log.Errorf("Error subscribing to channel, topic %s: %s", c.topic, err)
		return err //nolint
	}

	go c.bootstrapState()

	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil && err != context.Canceled && err != pubsub.ErrSubscriptionCancelled {
				log.Errorf("Error fetching next channel message, topic %s: %s", c.topic, err)
				continue
			}
			if err == context.Canceled || err == pubsub.ErrSubscriptionCancelled {
				return
			}

			pnd, err := merkledag.DecodeProtobuf(msg.Data())
			if err != nil {
				log.Errorf("Error decoding channel object, topic %s: %s", c.topic, err)
				continue
			}

			channelMsg := new(pb.ChannelMessage)
			if err := proto.Unmarshal(pnd.Data(), channelMsg); err != nil {
				log.Errorf("Error decoding channel protobuf, topic %s: %s", c.topic, err)
				continue
			}

			valid, err := validateMessage(channelMsg)
			if err != nil {
				log.Error("Error validating message, topic %s: %s", c.topic, err)
				continue
			}
			if !valid {
				log.Error("Message invalid, topic %s: %s", c.topic, err)
				continue
			}

			pth, err := c.object.Put(context.Background(), bytes.NewReader(msg.Data()), caopts.Object.InputEnc("protobuf"), caopts.Object.Pin(true))
			if err != nil {
				log.Errorf("Error putting message to IPFS, topic %s: peer %s: %s", c.topic, channelMsg.PeerID, err)
				continue
			}

			nd, err := c.object.Get(context.Background(), pth)
			if err != nil {
				log.Errorf("Error getting IPFS object, topic %s: peer %s: %s", c.topic, channelMsg.PeerID, err)
				continue
			}

			wasBoostrapped := c.boostrapped
			err = c.db.Update(func(tx database.Tx) error {
				var channelRec models.Channel
				err := tx.Read().Where("topic=?", c.topic).First(&channelRec).Error
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
				channelRec.Topic = c.topic
				channelRec.LastMessage = time.Now()
				if !c.boostrapped {
					if err := channelRec.SetHead([]cid.Cid{nd.Cid()}); err != nil {
						return err
					}
				} else {
					if err := channelRec.UpdateHead(nd); err != nil {
						return err
					}
				}
				return tx.Save(&channelRec)
			})
			if err != nil {
				log.Errorf("Error updating database with new cids, topic %s: peer %s: %s", c.topic, channelMsg.PeerID, err)
				continue
			}

			if !wasBoostrapped {
				log.Infof("Bootstrapped channel %s with %d cid(s)", c.topic, 1)
				c.bus.Emit(&events.ChannelBootstrapped{Topic: c.topic})
			}

			c.boostrapped = true

			c.bus.Emit(&events.ChannelMessage{
				PeerID:    channelMsg.PeerID,
				Topic:     c.topic,
				Message:   channelMsg.Message,
				Timestamp: time.Unix(channelMsg.Timestamp.Seconds, int64(channelMsg.Timestamp.Nanos)),
				Cid:       nd.Cid().String(),
			})
		}
	}()

	go func() {
		<-c.shutdown
		cancel()
		sub.Close()
	}()

	return nil
}

// bootstrapState loops until some channel peers connect. Once they connect
// it queries each of them for the cid(s) they believe to be the head of the
// channel. The responses are set as the head in our database.
func (c *Channel) bootstrapState() {
	var (
		peers []peer.ID
		err   error
	)
	ticker := time.NewTicker(time.Second * 4)
	for ; true; <-ticker.C {
		if c.boostrapped {
			return
		}
		peers, err = c.pubsub.Peers(context.Background(), caopts.PubSub.Topic(topicPrefix+c.topic))
		if err != nil {
			log.Debugf("No pubsub peers found for topic: %s", c.topic)
			continue
		}
		if len(peers) > 0 {
			break
		}
	}

	if c.ns == nil {
		c.boostrapped = true
		c.bus.Emit(&events.ChannelBootstrapped{Topic: c.topic})
		return
	}

	max := len(peers)
	if max > maxBootstrapPeers {
		max = maxBootstrapPeers
	}

	respChan := make(chan []cid.Cid)
	var wg sync.WaitGroup
	wg.Add(max)
	log.Debugf("Bootstrapping channel %s with %d peers", c.topic, max)
	go func() {
		for _, p := range peers[:max] {
			go func(pid peer.ID) {
				defer wg.Done()
				channelResp := npb.ChannelRequestMessage{
					Topic: c.topic,
				}

				payload, err := ptypes.MarshalAny(&channelResp)
				if err != nil {
					log.Errorf("Error serializing payload: %s", err)
					return
				}

				msgID := make([]byte, 20)
				rand.Read(msgID)

				req := &npb.Message{
					MessageID:   hex.EncodeToString(msgID),
					MessageType: npb.Message_CHANNEL_REQUEST,
					Payload:     payload,
				}

				if err := c.ns.SendMessage(context.Background(), pid, req); err != nil {
					return
				}

				sub, err := c.bus.Subscribe(&events.ChannelRequestResponse{}, events.MatchFields(map[string]string{
					"Topic":  c.topic,
					"PeerID": pid.Pretty(),
				}))
				if err != nil {
					log.Errorf("Error subscribing to bus: %s", err)
					return
				}

				select {
				case <-time.After(time.Second * 10):
					return
				case event := <-sub.Out():
					respChan <- event.(*events.ChannelRequestResponse).Cids
				}
			}(p)
		}
		wg.Wait()
		close(respChan)
	}()

	cidMap := make(map[cid.Cid]bool)
	for resp := range respChan {
		for _, id := range resp {
			cidMap[id] = true
		}
	}
	ids := make([]cid.Cid, 0, len(cidMap))
	for id := range cidMap {
		ids = append(ids, id)
	}

	if c.boostrapped || len(ids) == 0 {
		return
	}

	err = c.db.Update(func(tx database.Tx) error {
		var channelRec models.Channel
		err := tx.Read().Where("topic=?", c.topic).First(&channelRec).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		channelRec.Topic = c.topic
		channelRec.LastMessage = time.Now()
		if err := channelRec.SetHead(ids); err != nil {
			return err
		}
		return tx.Save(&channelRec)
	})
	if err != nil {
		log.Errorf("Error updating db with cids from peers: %s", err)
	}

	log.Infof("Bootstrapped channel %s with %d cid(s)", c.topic, len(ids))

	c.boostrapped = true
	c.bus.Emit(&events.ChannelBootstrapped{Topic: c.topic})
}

func validateMessage(cm *pb.ChannelMessage) (bool, error) {
	cloneMsg := proto.Clone(cm)
	cloneMsg.(*pb.ChannelMessage).Signature = nil
	ser, err := proto.Marshal(cloneMsg)
	if err != nil {
		return false, err
	}
	peerID, err := peer.Decode(cm.PeerID)
	if err != nil {
		return false, err
	}
	pubkey, err := peerID.ExtractPublicKey()
	if err != nil {
		return false, err
	}
	valid, err := pubkey.Verify(ser, cm.Signature)
	if err != nil {
		return false, err
	}
	if !valid {
		return false, nil
	}
	return true, nil
}
