package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	madns "github.com/multiformats/go-multiaddr-dns"

	vole "github.com/ipfs-shipyard/vole/lib"
	"github.com/urfave/cli/v2"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multibase"
)

func init() {
	// Lets us discover our own public address with a single observation
	identify.ActivationThresh = 1
}

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
					{
						Name:        "ping",
						ArgsUsage:   "<multiaddr>",
						Usage:       "ping a DHT node",
						Description: "sends a DHT ping to the target",
						Action: func(c *cli.Context) error {
							if c.NArg() != 1 {
								return fmt.Errorf("invalid number of arguments")
							}
							maStr := c.Args().Get(0)
							protoID := c.String("protocolID")

							ma, err := multiaddr.NewMultiaddr(maStr)
							if err != nil {
								return err
							}

							err = vole.DhtPing(c.Context, protocol.ID(protoID), ma)
							if err != nil {
								return err
							}

							fmt.Println("Ok")
							return nil
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
			{
				Name:  "libp2p",
				Usage: "tools for working with libp2p",
				Subcommands: []*cli.Command{
					{
						Name:      "identify",
						ArgsUsage: "<multiaddr>",
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name: "allow-unknown-peer",
								Usage: `if the multiaddr does not end with /p2p/PeerID allow trying to determine the peerID at the destination.
Note: connecting to a peer without knowing its peerID is generally insecure, however it is situationally useful.
Note: may not work with some transports such as p2p-circuit (not applicable) and webtransport (requires certificate hashes).
`,
								DefaultText: "false",
								Value:       false,
							},
						},
						Usage:       "learn about the peer with the given multiaddr",
						Description: "connects to the target address and runs identify against the peer",
						Action: func(c *cli.Context) error {
							if c.NArg() != 1 {
								return fmt.Errorf("invalid number of arguments")
							}
							allowUnknownPeer := c.Bool("allow-unknown-peer")
							resp, err := vole.IdentifyRequest(c.Context, c.Args().First(), allowUnknownPeer)
							if err != nil {
								return err
							}

							fmt.Printf("PeerID: %q\n", resp.PeerId)
							fmt.Printf("Protocol version: %q\n", resp.ProtocolVersion)
							fmt.Printf("Agent version: %q\n", resp.AgentVersion)

							fmt.Println("Listen addresses:")
							for _, a := range resp.Addresses {
								fmt.Printf("\t- %q\n", a)
							}

							fmt.Println("Protocols:")
							for _, p := range resp.Protocols {
								fmt.Printf("\t- %q\n", p)
							}
							return nil
						},
					}, {
						Name:      "ping",
						ArgsUsage: "<multiaddr>",
						Flags: []cli.Flag{

							&cli.BoolFlag{
								Name:        "force-relay",
								Usage:       `Ping the peer over a relay instead of a direct connection`,
								DefaultText: "false",
								Value:       false,
							},
						},
						Usage:       "ping a peer",
						Description: "connects to the target address and pings",
						Action: func(c *cli.Context) error {
							if c.NArg() != 1 {
								return fmt.Errorf("invalid number of arguments")
							}
							maStr := c.Args().First()
							ai, err := peer.AddrInfoFromString(maStr)
							if err != nil {
								return err
							}
							return vole.Ping(c.Context, c.Bool("force-relay"), ai)
						},
					}, {
						Name:      "http",
						ArgsUsage: "<multiaddr>",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:        "socket-path",
								Usage:       `Use the specified path for the unix socket instead of making a new one.`,
								DefaultText: "",
								Value:       "",
							},
						},
						Usage: "Make http requests to the given multiaddr with a unix socket",
						Description: `This command creates a unix socket that can be used with curl to make HTTP requests to the provided multiaddr.
Example:
	vole libp2p http <multiaddr>
	# Output:
	# Proxying on:
	# /tmp/libp2phttp-abc.sock

	# In another terminal
	curl --unix-socket /tmp/libp2phttp-abc.sock http://.well-known/libp2p/protocols`,
						Action: func(c *cli.Context) error {
							if c.NArg() != 1 {
								return fmt.Errorf("invalid number of arguments")
							}

							socketPath := c.String("socket-path")
							if socketPath == "" {
								f, err := os.CreateTemp("", "libp2phttp-*.sock")
								if err != nil {
									return err
								}
								// Remove this file since the listen will create it. We just wanted a random unused file path.
								f.Close()
								os.Remove(f.Name())
								socketPath = f.Name()
							}

							fmt.Println("Proxying on:")
							fmt.Println(socketPath)

							fmt.Println("\nExample curl request:")
							fmt.Println("curl --unix-socket", socketPath, "http://example.com/")

							m, err := multiaddr.NewMultiaddr(c.Args().First())
							if err != nil {
								return err
							}

							err = vole.Libp2pHTTPSocketProxy(c.Context, m, socketPath)
							if err == http.ErrServerClosed {
								return nil
							}
							return err
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
	}()
	err := app.RunContext(ctx, os.Args)
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

		return vole.GetBitswapCID(root, ai)
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

		output, err := vole.CheckBitswapCID(c.Context, nil, bsCid, ma, getBlock)
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
