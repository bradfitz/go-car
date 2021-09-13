package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ipfs/go-cid"
	carv2 "github.com/ipld/go-car/v2"
	"github.com/ipld/go-car/v2/blockstore"
	"github.com/urfave/cli/v2"
)

// FilterCar is a command to select a subset of a car by CID.
func FilterCar(c *cli.Context) error {
	if c.Args().Len() < 2 {
		return fmt.Errorf("an output filename must be provided")
	}

	fd, err := os.Open(c.Args().First())
	if err != nil {
		return err
	}
	defer fd.Close()
	rd, err := carv2.NewBlockReader(fd)
	if err != nil {
		return err
	}

	// Get the set of CIDs from stdin.
	inStream := os.Stdin
	if c.IsSet("cidFile") {
		inStream, err = os.Open(c.String("cidFile"))
		if err != nil {
			return err
		}
		defer inStream.Close()
	}
	cidMap, err := parseCIDS(inStream)
	if err != nil {
		return err
	}
	fmt.Printf("filtering to %d cids\n", len(cidMap))

	outRoots := make([]cid.Cid, 0)
	for _, r := range rd.Roots {
		if _, ok := cidMap[r]; ok {
			outRoots = append(outRoots, r)
		}
	}
	if len(outRoots) == 0 {
		fmt.Fprintf(os.Stderr, "warning: no roots defined after filtering\n")
	}
	bs, err := blockstore.OpenReadWrite(c.Args().Get(1), outRoots)
	if err != nil {
		return err
	}

	for {
		blk, err := rd.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if _, ok := cidMap[blk.Cid()]; ok {
			if err := bs.Put(blk); err != nil {
				return err
			}
		}
	}
	return bs.Finalize()
}

func parseCIDS(r io.Reader) (map[cid.Cid]struct{}, error) {
	cids := make(map[cid.Cid]struct{})
	br := bufio.NewReader(r)
	for {
		line, _, err := br.ReadLine()
		if err != nil {
			if err == io.EOF {
				return cids, nil
			}
			return nil, err
		}
		trimLine := strings.TrimSpace(string(line))
		if len(trimLine) == 0 {
			continue
		}
		c, err := cid.Parse(trimLine)
		if err != nil {
			return nil, err
		}
		if _, ok := cids[c]; ok {
			fmt.Fprintf(os.Stderr, "duplicate cid: %s\n", c)
		}
		cids[c] = struct{}{}
	}
}
