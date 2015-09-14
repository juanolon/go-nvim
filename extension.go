package nvim

import "log"

// Identifier represents raw vim objects like Buffer..
type Identifier interface {
	GetId() uint8
}

type nvimExt struct{}

func (e nvimExt) WriteExt(val interface{}) []byte {
	var b = make([]byte, 1)
	switch v := val.(type) {
	case BufferIdentifier:
		b[0] = v.Id
	case WindowIdentifier:
		b[0] = v.Id
	case TabpageIdentifier:
		b[0] = v.Id
	default:
		log.Fatalf("Fail writing msgpack-Extension. Got %T", val)
	}
	return b
}

func (e nvimExt) ReadExt(dest interface{}, src []byte) {
	switch v := dest.(type) {
	case *BufferIdentifier:
		v.Id = src[0]
	case *WindowIdentifier:
		v.Id = src[0]
	case *TabpageIdentifier:
		v.Id = src[0]
	default:
		log.Fatalf("Fail reading msgpack-Extension. Got %T", dest)
	}
}

func (nvimExt) ConvertExt(v interface{}) interface{} { panic("no used") }

func (nvimExt) UpdateExt(dst interface{}, src interface{}) { panic("no used") }
