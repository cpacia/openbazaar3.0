package channels

import (
	"context"
	"github.com/OpenBazaar/jsonpb"
	"github.com/cpacia/openbazaar3.0/channels/pb"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-routing"
	"github.com/op/go-logging"
	"io"
	"strings"
	"sync"
	"time"
)

var (
	log       = logging.MustGetLogger("CHNL")
	marshaler = jsonpb.Marshaler{Indent: "    "}
)

type Channel struct {
	name     string
	w        io.Writer
	ps       iface.PubSubAPI
	db       database.Database
	privKey  crypto.PrivKey
	identity peer.ID
	mtx      sync.Mutex

	sub iface.PubSubSubscription
}

func Subscribe(name string, ipfsNode *core.IpfsNode, db database.Database, writer io.Writer) (*Channel, error) {
	err := db.Update(func(tx database.Tx) error {
		return tx.Save(&models.ChannelInfo{Name: name})
	})
	if err != nil {
		return nil, err
	}

	api, err := coreapi.NewCoreAPI(ipfsNode)
	if err != nil {
		return nil, err
	}

	blk, err := api.Block().Put(context.Background(), strings.NewReader("floodsub:"+name))
	if err != nil {
		return nil, err
	}

	c := &Channel{
		name:     name,
		ps:       api.PubSub(),
		privKey:  ipfsNode.PrivateKey,
		identity: ipfsNode.Identity,
		db:       db,
		w:        writer,
		mtx:      sync.Mutex{},
	}
	if err := c.subscribe(blk.Path().Cid(), ipfsNode); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Channel) PublishMessage(message string, link string) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	var channelInfo models.ChannelInfo
	err := c.db.View(func(tx database.Tx) error {
		return tx.Read().Where("name=?", c.name).Find(&channelInfo).Error
	})
	if err != nil {
		return err
	}

	cids, err := channelInfo.GetLastCIDs()
	if err != nil {
		return err
	}

	chatMessage := &pb.Message{
		FromPeerID: c.identity.Pretty(),
		Message: &pb.Message_ChatMessage{
			ChatMessage: &pb.Message_Chat{
				Message:      message,
				Timestamp:    ptypes.TimestampNow(),
				Link:         link,
				PreviousCIDs: cids,
			},
		},
	}

	ser, err := proto.Marshal(chatMessage)
	if err != nil {
		return err
	}

	sig, err := c.privKey.Sign(ser)
	if err != nil {
		return err
	}

	chatMessage.Signature = sig

	ser, err = proto.Marshal(chatMessage)
	if err != nil {
		return err
	}

	return c.ps.Publish(context.Background(), c.name, ser)
}

func (c *Channel) SendTypingMessage() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	typing := &pb.Message{
		FromPeerID: c.identity.Pretty(),
		Message: &pb.Message_TypingMessage{
			TypingMessage: &pb.Message_Typing{},
		},
	}

	ser, err := proto.Marshal(typing)
	if err != nil {
		return err
	}

	sig, err := c.privKey.Sign(ser)
	if err != nil {
		return err
	}

	typing.Signature = sig

	ser, err = proto.Marshal(typing)
	if err != nil {
		return err
	}

	return c.ps.Publish(context.Background(), c.name, ser)
}

func (c *Channel) Close() error {
	return c.sub.Close()
}

func (c *Channel) Unsubscribe() error {
	if err := c.sub.Close(); err != nil {
		return err
	}
	return c.db.Update(func(tx database.Tx) error {
		return tx.Delete("name", c.name, nil, &models.ChannelInfo{})
	})
}

func (c *Channel) subscribe(cid cid.Cid, ipfsNode *core.IpfsNode) error {
	sub, err := c.ps.Subscribe(context.Background(), c.name, options.PubSub.Discover(false))
	if err != nil {
		return err
	}
	c.sub = sub

	connectToPubSubPeers(context.Background(), ipfsNode.Routing, ipfsNode.PeerHost, cid)

	peers, err := c.ps.Peers(context.Background(), options.PubSub.Topic(c.name))
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(peers))
	lastIDMap := make(map[string]bool)
	for _, pid := range peers {
		go func(p peer.ID) {
			// TODO: query and ask for last CIDs.
		}(pid)
	}
	wg.Wait()
	var lastCIDs []string
	for id := range lastIDMap {
		lastCIDs = append(lastCIDs, id)
	}

	err = c.db.Update(func(tx database.Tx) error {
		ci := &models.ChannelInfo{
			Name: c.name,
		}
		if err := ci.SetLastCIDs(lastCIDs); err != nil {
			return err
		}
		return tx.Save(ci)
	})
	if err != nil {
		return err
	}

	go func() {
		for {
			msg, err := sub.Next(context.Background())
			if err != nil {
				log.Errorf("Pubsub error: %s", err)
				continue
			}

			message := new(pb.Message)
			if err := proto.Unmarshal(msg.Data(), message); err != nil {
				log.Errorf("Error unmarshalling pubsub message: %s", err)
				continue
			}

			cpy := proto.Clone(message)
			cpy.(*pb.Message).Signature = nil

			ser, err := proto.Marshal(cpy)
			if err != nil {
				log.Errorf("Error validating pubsub message: %s", err)
				continue
			}

			from, err := peer.IDB58Decode(message.FromPeerID)
			if err != nil {
				log.Errorf("Error validating pubsub message: %s", err)
				continue
			}

			pub, err := from.ExtractPublicKey()
			if err != nil {
				log.Errorf("Error validating pubsub message: %s", err)
				continue
			}

			valid, err := pub.Verify(ser, message.Signature)
			if err != nil {
				log.Errorf("Error validating pubsub message: %s", err)
				continue
			}

			if !valid {
				return
			}

			out, err := marshaler.MarshalToString(message)
			if err != nil {
				log.Errorf("Error unmarshalling pubsub message: %s", err)
				continue
			}

			if _, err := c.w.Write([]byte(out)); err != nil {
				log.Errorf("Error writing pubsub message to websockets: %s", err)
				continue
			}
		}
	}()
	return nil
}

func connectToPubSubPeers(ctx context.Context, r routing.ContentRouting, ph host.Host, cid cid.Cid) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	provs := r.FindProvidersAsync(ctx, cid, 10)
	var wg sync.WaitGroup
	for p := range provs {
		wg.Add(1)
		go func(pi peerstore.PeerInfo) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()
			err := ph.Connect(ctx, pi)
			if err != nil {
				log.Info("pubsub discover: ", err)
				return
			}
			log.Info("connected to pubsub peer:", pi.ID)
		}(p)
	}

	wg.Wait()
}
