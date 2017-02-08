package local

import (
	"bytes"
	"errors"
	"io"
	"net"
	"time"

	"common"
	"common/ds"
)

var (
	ERR_READ     = errors.New("Reading pipe error")
	ERR_WRITE    = errors.New("Writing pipe error")
	ERR_NOSIG    = errors.New("Signal timeout error")
	ERR_SIGFALSE = errors.New("Signal false")
)

func PipeInboundToOutbound(src net.Conn, dst net.Conn, rto time.Duration, wto time.Duration, stat *common.Statistic, sig chan bool, buffer **common.Buffer) (result error) {
	bytesRead := 0
	var tempBuffer common.Buffer
	signaled := false
	defer func() {
		if !signaled {
			sig <- false
		} else {
			if bytesRead < 10*1024*1024 {
				*buffer = &tempBuffer
			} else {
				*buffer = nil
			}

			stat.IncreaseTotalUpload(uint64(bytesRead))
		}
		if result != nil {
			common.Warning("piping inbound to outbound goroutine exits with error:", result)
			sig <- false
		} else {
			sig <- true
		}
	}()

	if buffer != nil && *buffer != nil {
		tempBuffer = *(*buffer)
		common.Debug("try to write old data:")

		for i := 0; i < len(tempBuffer); i++ {
			if _, err := dst.Write(tempBuffer[i].Bytes()); err != nil {
				common.Error("write old data to outbound error: ", err)
				result = err
				return
			}
			stat.IncreaseTotalUpload(uint64(tempBuffer[i].Len()))
		}

		if !signaled {
			common.Debug("signal the paired goroutine, written old data to outbound")
			sig <- true
			signaled = true
		}
	}

	buf := ds.GlobalLeakyBuf.Get()
	defer ds.GlobalLeakyBuf.Put(buf)
	for {
		//common.Debugf("try to read something from inbound with timeout %v at %v\n", rto, time.Now().Add(rto))
		src.SetReadDeadline(time.Now().Add(rto))
		n, err := src.Read(buf)
		bytesRead += n
		if n > 0 {
			if bytesRead < 10*1024*1024 {
				tempBuffer = append(tempBuffer, bytes.NewBuffer(buf[0:n]))
			}

			if !signaled {
				common.Debug("signal the paired goroutine")
				sig <- true
				signaled = true
			}

			//common.Debugf("read something from inbound, and try to write to outbound with timeout %v at %v\n", wto, time.Now().Add(wto))
			dst.SetWriteDeadline(time.Now().Add(wto))
			nw, err := dst.Write(buf[0:n])
			if err != nil {
				if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
					common.Error("write to outbound err: ", ERR_WRITE)
					result = ERR_WRITE
					return
				}
				common.Error("writing to outbound err: ", err)
				result = err
				return
			}
			common.Debug("written ", n, "bytes to outbound and ", nw, "bytes are expected, read", bytesRead, " bytes totally")
		}
		if err != nil {
			if !signaled {
				sig <- true
				signaled = true
				if err == io.EOF {
					common.Debug("pipe inbound to outbound eof with ", bytesRead, " bytes")
					result = err
				} else {
					common.Error("reading from inbound error:", err)
					result = ERR_READ
				}
			}

			break
		}
	}

	return
}

func PipeOutboundToInbound(src net.Conn, dst net.Conn, rto time.Duration, wto time.Duration, stat *common.Statistic, sig chan bool) error {
	bytesRead := 0
	signaled := false
	defer func() {
		if signaled == false {
			<-sig
		}
		src.Close()
		common.Debug("R/W end signaled")
	}()

	s := <-sig // wait for paired goroutine to start
	common.Debug("R/W begin signaled")
	if !s {
		return ERR_SIGFALSE
	}

	buf := ds.GlobalLeakyBuf.Get()
	defer ds.GlobalLeakyBuf.Put(buf)
	for {
		//common.Debugf("try to read something from outbound with timeout %v at %v\n", rto, time.Now().Add(rto))
		src.SetReadDeadline(time.Now().Add(rto))
		n, err := src.Read(buf)
		bytesRead += n
		if n > 0 {
			stat.BytesDownload(uint64(n))
			//common.Debugf("read something from outbound, and try to write to inbound with timeout %v at %v\n", wto, time.Now().Add(wto))
			dst.SetWriteDeadline(time.Now().Add(wto))
			nw, err := dst.Write(buf[0:n])
			if err != nil {
				common.Error("write to inbound error: ", err)
				return ERR_WRITE
			}
			common.Debug("written ", nw, "bytes to inbound and", n, "bytes are expected, read", bytesRead, "bytes totally")
		}
		if err != nil {
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() && bytesRead == 0 {
				common.Error("read from outbound timeout, seems the server has been null")
				return ERR_READ
			}
			if err == io.EOF {
				common.Debug("read from outbound eof with", bytesRead, "bytes")
				return nil
			}
			common.Error("reading from outbound error:", err)
			return nil
		}
		select {
		case result := <-sig:
			signaled = true
			if !result {
				common.Debug("paired piping inbound to outbound goroutine exited, so this goroutine piping outbound to inbound just exit too")
				return nil
			} else {
				common.Debug("no matter paired goroutine exiting, go on reading outbound input with", bytesRead, "bytes")
			}
		default:
			common.Debug("go on reading outbound input with", bytesRead, "bytes")
		}
	}

	common.Debug("outbound to inbound seems ok")
	return nil
}
