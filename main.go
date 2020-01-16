package main

import (
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/golang/snappy"
	"github.com/urfave/cli"
)

const ChunkSize = 32 * 1024

const EmptyMarker = 0xb8724018f1c8c07d

type SFChunk struct {
	Offset uint64
	Data   []byte //is ChunkSize
}

var totalempty int64
var total int64

func makeChunk(offset uint64, buf []byte) *SFChunk {
	nonempty := false
	rv := &SFChunk{
		Offset: offset,
		Data:   make([]byte, ChunkSize),
	}
	for i := 0; i < ChunkSize/8; i++ {
		le8 := binary.LittleEndian.Uint64(buf[i*8:])
		if le8 != EmptyMarker {
			nonempty = true
		}
		copy(rv.Data[i*8:i*8+8], buf[i*8:i*8+8])
	}
	total += ChunkSize
	if nonempty {
		return rv
	}
	totalempty += ChunkSize
	return nil
}
func GenerateSFMap(inp, out string) {
	off, err := os.Create(out)
	defer off.Close()
	if err != nil {
		fmt.Println("could not create output:", err)
		os.Exit(1)
	}
	w := snappy.NewBufferedWriter(off)
	defer w.Close()
	enc := gob.NewEncoder(w)
	inpf, err := os.Open(inp)
	if err != nil {
		fmt.Println("could not create stream input:", err)
		os.Exit(1)
	}
	buf := make([]byte, ChunkSize)
	var offset uint64
	for {
		nr, err := io.ReadFull(inpf, buf)
		if err != nil {
			fmt.Println("end")
			break
		}
		c := makeChunk(offset, buf)
		offset += uint64(nr)
		if c != nil {
			enc.Encode(c)
		}
	}
	fmt.Printf("done. Image was total %d bytes of which %d were trimmed\n", total, totalempty)
}
func doWrite(ofc chan *SFChunk, oo *os.File, done chan bool) {
	var lastoff uint64
	for c := range ofc {
		if lastoff != c.Offset {
			oo.Seek(int64(c.Offset), os.SEEK_SET)
		}
		nc, err := oo.Write(c.Data)
		if err != nil || nc != ChunkSize {
			fmt.Printf("short write %d : %v\n", nc, err)
			os.Exit(1)
		}
		lastoff = c.Offset + ChunkSize
	}
	err := oo.Sync()
	if err != nil {
		fmt.Printf("could not sync: %v\n", err)
	}
	oo.Close()
	close(done)
}
func ExecuteSFMap(inpf, outpf string) {
	ofc := make(chan *SFChunk, 1000)
	done := make(chan bool, 1)
	oo, err := os.OpenFile(outpf, os.O_WRONLY|os.O_SYNC, 0666)
	if err != nil {
		fmt.Println("could not open output device:", err)
		os.Exit(1)
	}
	go doWrite(ofc, oo, done)
	iff, err := os.Open(inpf)
	if err != nil {
		fmt.Println("could not create stream input:", err)
		os.Exit(1)
	}
	r := snappy.NewReader(iff)
	dec := gob.NewDecoder(r)
	cnt := 0
	for {
		chunk := &SFChunk{}
		err := dec.Decode(chunk)
		if err == io.EOF {
			fmt.Println("finished reading SFMap")
			close(ofc)
			<-done
			break
		}
		if err != nil {
			fmt.Println("could not decode chunk:", err)
			os.Exit(1)
		}
		if cnt%256 == 0 {
			fmt.Printf("processed %.2f MB\n", float64(chunk.Offset)/1024./1024.)
		}
		cnt += 1
		ofc <- chunk
	}
}
func main() {
	app := cli.NewApp()
	app.Name = "superflash"
	app.Usage = "Accelerated image flashing utility"
	app.Version = "2.1"
	app.Commands = []cli.Command{
		{
			Name:      "blank",
			Usage:     "Create a blank image",
			ArgsUsage: "size-in-Mb outputfile",
			Action:    cli.ActionFunc(actionBlank),
		},
		{
			Name:      "encode",
			Usage:     "Create an SFMap from an image",
			ArgsUsage: "imgfile [outputfile]",
			Action:    cli.ActionFunc(actionEncode),
		},
		{
			Name:      "flash",
			Usage:     "Flash an SFMap",
			ArgsUsage: "sfmap outputdevice",
			Action:    cli.ActionFunc(actionFlash),
		},
	}
	app.Run(os.Args)
}

func actionBlank(c *cli.Context) error {
	if len(c.Args()) != 2 {
		fmt.Println("usage: superflash blank <size in MB> <filename>")
		os.Exit(1)
	}
	size, err := strconv.ParseInt(c.Args()[0], 10, 64)
	if err != nil {
		fmt.Println("could not parse size")
		os.Exit(1)
	}
	off, err := os.Create(c.Args()[1])
	if err != nil {
		fmt.Println("could not create output file:", err)
		os.Exit(1)
	}
	buf := make([]byte, 1024*1024)
	mbuf := []byte{0x7d, 0xc0, 0xc8, 0xf1, 0x18, 0x40, 0x72, 0xb8}
	for i := 0; i < (1024*1024)/8; i++ {
		copy(buf[i*8:], mbuf)
	}
	var i int64
	for ; i < size; i++ {
		off.Write(buf)
	}
	off.Close()
	return nil
}
func actionEncode(c *cli.Context) error {
	if len(c.Args()) < 1 || len(c.Args()) > 2 {
		fmt.Println("usage superflash encode imgfile [outfile]")
		os.Exit(1)
	}
	outfile := c.Args()[0] + ".sfmap"
	if len(c.Args()) == 2 {
		outfile = c.Args()[1]
	}
	GenerateSFMap(c.Args()[0], outfile)
	return nil
}
func actionFlash(c *cli.Context) error {
	if len(c.Args()) != 2 {
		fmt.Println("usage superflash flash <sfmap> <device>")
		os.Exit(1)
	}
	ExecuteSFMap(c.Args()[0], c.Args()[1])
	return nil
}
