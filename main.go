package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/cheggaaa/pb/v3"
	ipld "github.com/ipfs/go-ipld-format"
	madns "github.com/multiformats/go-multiaddr-dns"

	// importing so we can traverse dag-cbor and dag-json nodes in the `bitswap get` command
	_ "github.com/ipld/go-ipld-prime/codec/dagcbor"
	_ "github.com/ipld/go-ipld-prime/codec/dagjson"

	vole "github.com/ipfs-shipyard/vole/lib"
	"github.com/urfave/cli/v2"

	"github.com/ipfs/go-bitswap"
	bsnet "github.com/ipfs/go-bitswap/network"
	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/ipfs/go-merkledag"
	"github.com/libp2p/go-libp2p"
	rhelp "github.com/libp2p/go-libp2p-routing-helpers"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/muxer/mplex"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multibase"
)

func main() {
	app := &cli.App{
		Name:  "vole",
		Usage: "a collection of tools for digging around IPFS nodes",
		Authors: []*cli.Author{
			{
				Name:  "Adin Schmahmann",
				Email: "adin.schmahmann@gmail.com",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "bitswap",
				Usage: "tools for working with bitswap",
				Subcommands: []*cli.Command{
					bitswapGetCmd,
					bitswapCheckCmd,
				},
			},
			{
				Name:  "dht",
				Usage: "tools for working with libp2p kademlia DHTs",
				Subcommands: []*cli.Command{
					{
						Name:        "put",
						ArgsUsage:   "<multibase-bytes-key> <multibase-bytes-value> <multiaddr>",
						Usage:       "put a record to a DHT node",
						Description: "creates a libp2p peer and sends a DHT put request to the target",
						Action: func(c *cli.Context) error {
							if c.NArg() != 3 {
								return fmt.Errorf("invalid number of arguments")
							}
							keyStr := c.Args().Get(0)
							valStr := c.Args().Get(1)
							maStr := c.Args().Get(2)
							protoID := c.String("protocolID")

							_, keyBytes, err := multibase.Decode(keyStr)
							if err != nil {
								return err
							}

							_, valBytes, err := multibase.Decode(valStr)
							if err != nil {
								return err
							}

							ma, err := multiaddr.NewMultiaddr(maStr)
							if err != nil {
								return err
							}

							return vole.DhtPut(c.Context, keyBytes, valBytes, protocol.ID(protoID), ma)
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:        "protocolID",
								Usage:       "the protocol ID",
								DefaultText: "/ipfs/kad/1.0.0",
								Value:       "/ipfs/kad/1.0.0",
							},
						},
					},
					{
						Name:        "get",
						ArgsUsage:   "<multibase-bytes-key> <multiaddr>",
						Usage:       "get a record from a DHT node",
						Description: "creates a libp2p peer and sends a DHT get request to the target",
						Action: func(c *cli.Context) error {
							if c.NArg() != 2 {
								return fmt.Errorf("invalid number of arguments")
							}
							keyStr := c.Args().Get(0)
							maStr := c.Args().Get(1)
							protoID := c.String("protocolID")
							base := c.String("base")

							_, keyBytes, err := multibase.Decode(keyStr)
							if err != nil {
								return err
							}

							ma, err := multiaddr.NewMultiaddr(maStr)
							if err != nil {
								return err
							}

							enc, err := multibase.EncoderByName(base)
							if err != nil {
								return err
							}

							rec, err := vole.DhtGet(c.Context, keyBytes, protocol.ID(protoID), ma)
							if err != nil {
								return err
							}

							fmt.Println(enc.Encode(rec.GetValue()))
							return nil
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:        "protocolID",
								Usage:       "the protocol ID",
								DefaultText: "/ipfs/kad/1.0.0",
								Value:       "/ipfs/kad/1.0.0",
							},
							&cli.StringFlag{
								Name:        "base",
								Aliases:     []string{"b"},
								Usage:       "multibase to encode the result in (e.g. b or base32 for base32 encoding)",
								DefaultText: "base32",
								Value:       "base32",
							},
						},
					},
					{
						Name:        "getprovs",
						ArgsUsage:   "<cid> <multiaddr>",
						Usage:       "gets provider records from a DHT node",
						Description: "creates a libp2p peer and sends a DHT get providers request to the target",
						Action: func(c *cli.Context) error {
							if c.NArg() != 2 {
								return fmt.Errorf("invalid number of arguments")
							}
							cidStr := c.Args().Get(0)
							maStr := c.Args().Get(1)
							protoID := c.String("protocolID")
							showAddrs := c.Bool("show-addrs")

							dataCID, err := cid.Decode(cidStr)
							if err != nil {
								return err
							}

							ma, err := multiaddr.NewMultiaddr(maStr)
							if err != nil {
								return err
							}

							provs, err := vole.DhtGetProvs(c.Context, dataCID.Hash(), protocol.ID(protoID), ma)
							if err != nil {
								return err
							}

							return printPeerIDs(provs, showAddrs)
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:        "protocolID",
								Usage:       "the protocol ID",
								DefaultText: "/ipfs/kad/1.0.0",
								Value:       "/ipfs/kad/1.0.0",
							},
							&cli.BoolFlag{
								Name:        "show-addrs",
								Aliases:     []string{"a"},
								Usage:       "show the peer address or just the IDs",
								DefaultText: "false",
								Value:       false,
							},
						},
					},
					{
						Name:        "gcp",
						ArgsUsage:   "<multibase-bytes-key> <multiaddr>",
						Usage:       "gets the closest peers to the target from a DHT node",
						Description: "creates a libp2p peer and sends a DHT get closest peers request to the target - prints the peers and their addresses",
						Action: func(c *cli.Context) error {
							if c.NArg() != 2 {
								return fmt.Errorf("invalid number of arguments")
							}
							keyStr := c.Args().Get(0)
							maStr := c.Args().Get(1)
							protoID := c.String("protocolID")
							showAddrs := c.Bool("show-addrs")

							_, keyBytes, err := multibase.Decode(keyStr)
							if err != nil {
								return err
							}

							ma, err := multiaddr.NewMultiaddr(maStr)
							if err != nil {
								return err
							}

							ais, err := vole.DhtGetClosestPeers(c.Context, keyBytes, protocol.ID(protoID), ma)
							if err != nil {
								return err
							}

							return printPeerIDs(ais, showAddrs)
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:        "protocolID",
								Usage:       "the protocol ID",
								DefaultText: "/ipfs/kad/1.0.0",
								Value:       "/ipfs/kad/1.0.0",
							},
							&cli.BoolFlag{
								Name:        "show-addrs",
								Aliases:     []string{"a"},
								Usage:       "show the peer address or just the IDs",
								DefaultText: "false",
								Value:       false,
							},
						},
					},
				},
			},
			{
				Name:        "dnsaddr",
				ArgsUsage:   "<domain>",
				Usage:       "get the multiaddrs from a domain name with a dnsaddr",
				Description: "creates a DNSAddr lookup and returns all the multiaddrs",
				Action: func(c *cli.Context) error {
					if c.NArg() != 1 {
						return fmt.Errorf("invalid number of arguments")
					}
					maStr := fmt.Sprintf("/dnsaddr/%s", c.Args().Get(0))

					addr, err := multiaddr.NewMultiaddr(maStr)
					if err != nil {
						return err
					}
					addrs, err := madns.DefaultResolver.Resolve(c.Context, addr)
					if err != nil {
						return err
					}

					for _, a := range addrs {
						fmt.Println(a)
					}
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}

func printPeerIDs(ais []*peer.AddrInfo, showAddrs bool) error {
	for _, a := range ais {
		if showAddrs {
			b, err := a.MarshalJSON()
			if err != nil {
				return err
			}
			var pretty bytes.Buffer
			err = json.Indent(&pretty, b, "", "\t")
			if err != nil {
				return err
			}
			fmt.Println(pretty.String())
		} else {
			fmt.Println(a.ID)
		}
	}

	return nil
}

var bitswapGetCmd = &cli.Command{
	Name: "get",
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() < 2 {
			return fmt.Errorf("must pass cid and multiaddr of peer to fetch from")
		}

		root, err := cid.Decode(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		maddr, err := multiaddr.NewMultiaddr(cctx.Args().Get(1))
		if err != nil {
			return err
		}
		ai, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			return err
		}

		// set up libp2p node...
		ctx := context.Background()
		h, err := libp2p.New(libp2p.DefaultMuxers, libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport))
		if err != nil {
			return err
		}

		ds := sync.MutexWrap(datastore.NewMapDatastore())
		bstore := blockstore.NewBlockstore(ds)

		bsnet := bsnet.NewFromIpfsHost(h, &rhelp.Null{})
		bswap := bitswap.New(ctx, bsnet, bstore)

		bserv := blockservice.New(bstore, bswap)
		dag := merkledag.NewDAGService(bserv)

		// connect to our peer
		if err := h.Connect(ctx, *ai); err != nil {
			return fmt.Errorf("failed to connect to target peer: %w", err)
		}

		bar := pb.StartNew(-1)
		bar.Set(pb.Bytes, true)

		cset := cid.NewSet()

		getLinks := func(ctx context.Context, c cid.Cid) ([]*ipld.Link, error) {
			node, err := dag.Get(ctx, c)
			if err != nil {
				return nil, err
			}
			bar.Add(len(node.RawData()))

			return node.Links(), nil

		}
		if err := merkledag.Walk(ctx, getLinks, root, cset.Visit, merkledag.Concurrency(500)); err != nil {
			return err
		}

		bar.Finish()

		return nil
	},
}
var bitswapCheckCmd = &cli.Command{
	Name:        "check",
	ArgsUsage:   "<cid> <multiaddr>",
	Usage:       "check if a peer has a CID",
	Description: "creates a libp2p peer and sends a bitswap request to the target - prints true if the peer reports it has the CID and false otherwise",
	Action: func(c *cli.Context) error {
		if c.NArg() != 2 {
			return fmt.Errorf("invalid number of arguments")
		}
		cidStr := c.Args().Get(0)
		maStr := c.Args().Get(1)
		getBlock := c.Bool("get-block")

		bsCid, err := cid.Decode(cidStr)
		if err != nil {
			return err
		}

		ma, err := multiaddr.NewMultiaddr(maStr)
		if err != nil {
			return err
		}

		output, err := vole.CheckBitswapCID(c.Context, bsCid, ma, getBlock)
		if err != nil {
			return err
		}

		jsOut, err := json.Marshal(output)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", jsOut)

		return nil
	},
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:        "get-block",
			Usage:       "get the block",
			Value:       true,
			DefaultText: "true",
		},
	},
}
