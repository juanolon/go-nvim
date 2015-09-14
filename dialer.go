package nvim

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/ugorji/go/codec"
)

type connType int

const (
	CONN_NET connType = iota
	CONN_STD
)

// WRCloser r/w wrapper
type WRCloser struct {
	r io.ReadCloser
	w io.WriteCloser
}

func NewWRCloser(r io.ReadCloser, w io.WriteCloser) *WRCloser {
	return &WRCloser{r, w}
}

func (b *WRCloser) Read(p []byte) (n int, err error) {
	// log.Printf("reading: %#v", p)
	fmt.Printf("read: %#+v", p)
	spew.Dump("reading", p)
	return b.r.Read(p)
}

func (b *WRCloser) Write(data []byte) (n int, err error) {
	// log.Printf("writing: %#v", data)
	fmt.Printf("read: %#+v", data)
	spew.Dump("writing: ", data)
	return b.w.Write(data)
}

func (b *WRCloser) Close() error {
	err := b.r.Close()
	if err != nil {
		return err
	}
	err = b.w.Close()
	if err != nil {
		return err
	}
	return nil
}

type buffer struct {
	bytes.Buffer
}

// Add a Close method to our buffer so that we satisfy io.ReadWriteCloser.
func (b *buffer) Close() error {
	b.Buffer.Reset()
	return nil
}

func Dial(network connType, addr string) (*Vim, error) {
	var conn io.ReadWriteCloser
	var err error
	switch network {
	case CONN_NET:
		conn, err = net.Dial("unix", addr)
		if err != nil {
			// return nil, errors.New(fmt.Sprintf("Fail connecting to Nvim: %s", err))
			log.Fatalf("Fail connecting to Nvim: %s", err)
		}
	case CONN_STD:
		cmd := exec.Command("nvim", "--embed")

		stdin, err := cmd.StdinPipe()
		if err != nil {
			// return nil, errors.New(fmt.Sprintf("Fail getting stdin: %s", err))
			log.Fatalf("Fail getting stdin: %s", err)
		}
		// defer stdin.Close() // do i need to close it here?

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			// return nil, errors.New(fmt.Sprintf("Fail getting stdout: %s", err))
			log.Fatalf("Fail getting stdout: %s", err)
		}
		// defer stdout.Close()

		cmd.Stderr = os.Stderr
		conn = NewWRCloser(stdout, stdin)
		// conn = &buffer{}

		// TODO check this
		err = cmd.Start()
		if err != nil {
			log.Fatal("cmd Start: ", err)
		}
	}

	var h codec.MsgpackHandle
	h.RawToString = true
	h.WriteExt = true
	h.SetExt(reflect.TypeOf(BufferIdentifier{}), Buffer_TAG, &nvimExt{})
	h.SetExt(reflect.TypeOf(WindowIdentifier{}), Window_TAG, &nvimExt{})
	h.SetExt(reflect.TypeOf(TabpageIdentifier{}), Tabpage_TAG, &nvimExt{})

	rpcCodec := codec.MsgpackSpecRpc.ClientCodec(conn, &h)
	client := rpc.NewClientWithCodec(rpcCodec)

	// TODO handle errors correctly
	return &Vim{
		client,
	}, nil
}
