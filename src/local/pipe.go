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
	ErrRead        = errors.New("Reading pipe error")
	ErrWrite       = errors.New("Writing pipe error")
	ErrNoSignal    = errors.New("Signal timeout error")
	ErrSignalFalse = errors.New("Signal false")
)

type pipeParam struct {
	src  net.Conn
	dst  net.Conn
	rto  time.Duration
	wto  time.Duration
	stat *common.Statistic
	sig  chan bool
}

func handleBuffer(pp pipeParam, buffer **common.Buffer, signaled *bool) (tempBuffer common.Buffer, result error) {
	if buffer != nil && *buffer != nil {
		tempBuffer = *(*buffer)
		common.Debug("try to write old data:")

		for i := 0; i < len(tempBuffer); i++ {
			if _, err := pp.dst.Write(tempBuffer[i].Bytes()); err != nil {
				common.Error("write old data to outbound error: ", err)
				result = err
				return
			}
			pp.stat.IncreaseTotalUpload(uint64(tempBuffer[i].Len()))
		}

		if !*signaled {
			common.Debug("signal the paired goroutine, written old data to outbound")
			pp.sig <- true
			*signaled = true
		}
	}
	return
}

func finalOutput(pp pipeParam, buffer **common.Buffer, signaled *bool, bytesRead int, tempBuffer *common.Buffer, result error) {
	if !*signaled {
		pp.sig <- false
	} else {
		if bytesRead < 10*1024*1024 {
			*buffer = tempBuffer
		} else {
			*buffer = nil
		}

		pp.stat.IncreaseTotalUpload(uint64(bytesRead))
	}
	if result != nil {
		common.Warning("piping inbound to outbound goroutine exits with error:", result)
		pp.sig <- false
	} else {
		pp.sig <- true
	}
}

func PipeInboundToOutbound(pp pipeParam, buffer **common.Buffer) (result error) {
	signaled := false
	bytesRead := 0
	var tempBuffer common.Buffer
	defer func() {
		finalOutput(pp, buffer, &signaled, bytesRead, &tempBuffer, result)
	}()

	if tempBuffer, result = handleBuffer(pp, buffer, &signaled); result != nil {
		return
	}

	buf := ds.GlobalLeakyBuf.Get()
	defer ds.GlobalLeakyBuf.Put(buf)
	for {
		//common.Debugf("try to read something from inbound with timeout %v at %v\n", rto, time.Now().Add(rto))
		pp.src.SetReadDeadline(time.Now().Add(pp.rto))
		nr, err := pp.src.Read(buf)
		bytesRead += nr
		if nr > 0 {
			if bytesRead < 10*1024*1024 {
				tempBuffer = append(tempBuffer, bytes.NewBuffer(buf[:nr]))
			}

			if !signaled {
				common.Debug("signal the paired goroutine")
				pp.sig <- true
				signaled = true
			}

			//common.Debugf("read something from inbound, and try to write to outbound with timeout %v at %v\n", wto, time.Now().Add(wto))
			pp.dst.SetWriteDeadline(time.Now().Add(pp.wto))
			nw, err := pp.dst.Write(buf[:nr])
			if err != nil {
				if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
					common.Error("write to outbound err: ", ErrWrite)
					result = ErrWrite
					return
				}
				common.Error("writing to outbound err: ", err)
				result = err
				return
			}
			common.Debug("written ", nw, "bytes to outbound and ", nr, "bytes are expected, read", bytesRead, " bytes totally")
		}
		if err != nil {
			if !signaled {
				pp.sig <- true
				signaled = true
				if err == io.EOF {
					common.Debug("pipe inbound to outbound eof with ", bytesRead, " bytes")
					result = err
				} else {
					common.Error("reading from inbound error:", err)
					result = ErrRead
				}
			}

			break
		}
	}

	return
}

func PipeOutboundToInbound(pp pipeParam) (err error) {
	bytesRead := 0
	signaled := false
	defer func() {
		if signaled == false {
			<-pp.sig
		}
		pp.src.Close()
		common.Debug("R/W end signaled")
	}()

	s := <-pp.sig // wait for paired goroutine to start
	common.Debug("R/W begin signaled")
	if !s {
		return ErrSignalFalse
	}

	buf := ds.GlobalLeakyBuf.Get()
	defer ds.GlobalLeakyBuf.Put(buf)
	var nr int
	var nw int
	for {
		//common.Debugf("try to read something from outbound with timeout %v at %v\n", rto, time.Now().Add(rto))
		pp.src.SetReadDeadline(time.Now().Add(pp.rto))
		nr, err = pp.src.Read(buf)
		bytesRead += nr
		if nr > 0 {
			pp.stat.BytesDownload(uint64(nr))
			//common.Debugf("read something from outbound, and try to write to inbound with timeout %v at %v\n", wto, time.Now().Add(wto))
			pp.dst.SetWriteDeadline(time.Now().Add(pp.wto))
			nw, err = pp.dst.Write(buf[:nr])
			if err != nil {
				common.Error("write to inbound error: ", err)
				err = ErrWrite
				break
			}
			common.Debug("written ", nw, "bytes to inbound and", nr, "bytes are expected, read", bytesRead, "bytes totally")
		}
		if err != nil {
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() && bytesRead == 0 {
				common.Error("read from outbound timeout, seems the server has been null")
				err = ErrRead
				break
			}
			if err == io.EOF {
				common.Debug("read from outbound eof with", bytesRead, "bytes")
				err = nil
				break
			}
			common.Error("reading from outbound error:", err)
			break
		}
		select {
		case result := <-pp.sig:
			signaled = true
			if !result {
				common.Debug("paired piping inbound to outbound goroutine exited, so this goroutine piping outbound to inbound just exit too")
				return nil
			}
			common.Debug("no matter paired goroutine exiting, go on reading outbound input with", bytesRead, "bytes")
		default:
			common.Debug("go on reading outbound input with", bytesRead, "bytes")
		}
	}

	return
}
