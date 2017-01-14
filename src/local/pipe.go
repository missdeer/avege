package local

import (
	"errors"
	"io"
	"net"
	"time"
	"common"
)

var (
	ERR_READ     = errors.New("Reading pipe error")
	ERR_WRITE    = errors.New("Writing pipe error")
	ERR_NOSIG    = errors.New("Signal timeout error")
	ERR_SIGFALSE = errors.New("Signal false")
)

func PipeInboundToOutbound(src net.Conn, dst net.Conn, rto time.Duration, wto time.Duration, stat *common.Statistic) error {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		n, e := wt.WriteTo(dst)
		stat.IncreaseTotalUpload(uint64(n))
		return e
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		n, e := rt.ReadFrom(src)
		stat.IncreaseTotalUpload(uint64(n))
		return e
	}
	buf := common.GlobalLeakyBuf.Get()
	defer common.GlobalLeakyBuf.Put(buf)
	var written int64
	var err error
	for {
		src.SetReadDeadline(time.Now().Add(rto))
		nr, er := src.Read(buf)
		if nr > 0 {
			dst.SetWriteDeadline(time.Now().Add(wto))
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			err = er
			break
		}
	}
	stat.IncreaseTotalUpload(uint64(written))
	if err != io.ErrShortWrite {
		dst.Close()
	}
	return err
}

func PipeOutboundToInbound(src net.Conn, dst net.Conn, rto time.Duration, wto time.Duration, stat *common.Statistic) error {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		n, e := wt.WriteTo(dst)
		stat.IncreaseTotalUpload(uint64(n))
		return e
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		n, e := rt.ReadFrom(src)
		stat.IncreaseTotalUpload(uint64(n))
		return e
	}
	buf := common.GlobalLeakyBuf.Get()
	defer common.GlobalLeakyBuf.Put(buf)
	var written int64
	var err error
	for {
		src.SetReadDeadline(time.Now().Add(rto))
		nr, er := src.Read(buf)
		if nr > 0 {
			dst.SetWriteDeadline(time.Now().Add(wto))
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			err = er
			break
		}
	}
	if err == nil {
		stat.BytesDownload(uint64(written))
	}
	return err
}
