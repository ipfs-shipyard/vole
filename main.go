package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
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
					{
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

							return checkBitswapCmd(c.Context, cidStr, maStr)
						},
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
