package local

import (
	"bytes"
	"encoding/gob"
	"math/rand"
	"net"
	"sync"

	"common"
	"common/cache"
)

type remoteAddr struct {
	rawAddr []byte
}

var (
	// statistics collections contains all remote servers statistics
	statistics                = NewStatisticWrapper()
	remoteAddresses           []remoteAddr
	oneResolveRemoteAddresses sync.Once
)

func resolvServer(bi *BackendInfo) {
	host, _, _ := net.SplitHostPort(bi.address)
	if ips, err := net.LookupIP(host); err == nil {
		backends.Lock()
		bi.ips = ips
		bi.ipv6 = false
		for _, ip := range ips {
			if ip.To4() == nil {
				bi.ipv6 = true
				break
			}
		}
		backends.Unlock()
	}
}

type StatisticMap map[*BackendInfo]*common.Statistic

type StatisticWrapper struct {
	sync.RWMutex
	StatisticMap
}

func NewStatisticWrapper() *StatisticWrapper {
	sw := &StatisticWrapper{}
	sw.StatisticMap = make(StatisticMap)
	return sw
}

func (m *StatisticWrapper) Get(bi *BackendInfo) (s *common.Statistic, ok bool) {
	m.RLock()
	defer m.RUnlock()
	s, ok = m.StatisticMap[bi]
	return s, ok
}

func (m *StatisticWrapper) Delete(si *BackendInfo) {
	m.Lock()
	defer m.Unlock()
	delete(m.StatisticMap, si)
}

func (m *StatisticWrapper) Insert(si *BackendInfo, s *common.Statistic) {
	m.Lock()
	defer m.Unlock()
	m.StatisticMap[si] = s
}

func (m *StatisticWrapper) UpdateServerIP() {
	m.RLock()
	defer m.RUnlock()
	for si := range m.StatisticMap {
		resolvServer(si)
	}
}

func (m *StatisticWrapper) UpdateBps() {
	m.RLock()
	defer m.RUnlock()
	for _, stat := range m.StatisticMap {
		stat.Tick()
	}
}

func (m *StatisticWrapper) UpdateLatency() {
	var rawAddr []byte
	if len(remoteAddresses) == 0 {
		// addr = "104.28.31.28:443"
		rawAddr = []byte{1, 104, 28, 31, 28, 1, 187}
		oneResolveRemoteAddresses.Do(func() {
			go func() {
				remotes := []string{
					"api.twitter.com",
					"dev.twitter.com",
					"www.twitter.com",
					"twitter.com",
				}
				for _, r := range remotes {
					ips, err := net.LookupIP(r)
					if err != nil {
						common.Warning("looking up IP for ", r, " failed.", err)
						continue
					}
					for _, ip := range ips {
						if ip.To4() == nil {
							common.Info("skip nil v4 ip:", ip)
							continue
						}
						ipv4 := []byte(ip.To4())
						ra := remoteAddr{
							[]byte{1, ipv4[0], ipv4[1], ipv4[2], ipv4[3], 1, 187},
						}
						remoteAddresses = append(remoteAddresses, ra)
					}
				}
			}()
		})
	} else {
		index := rand.Intn(len(remoteAddresses))
		rawAddr = remoteAddresses[index].rawAddr
	}

	m.RLock()
	for bi := range m.StatisticMap {
		go bi.testLatency(rawAddr)
	}
	m.RUnlock()

	m.SaveToRedis()
}

func (m *StatisticWrapper) LoadFromCache() {
	m.Lock()
	defer m.Unlock()
	for server, stat := range m.StatisticMap {
		statistic, _ := cache.Instance.Get(server.address)
		if statistic == nil {
			common.Info("no cache item for", server.address)
			continue
		}
		b, ok := statistic.([]byte)
		if !ok {
			common.Error("to []byte failed")
			continue
		}
		var buf bytes.Buffer
		buf.Write(b)
		decoder := gob.NewDecoder(&buf)
		var s Stat
		if err := decoder.Decode(&s); err != nil {
			common.Error("to Stat failed")
			continue
		}
		if len(s.Id) == 0 {
			s.Id = common.GenerateRandomString(4)
		}
		server.id = s.Id
		stat.SetLatency(s.Latency)
		stat.SetFailedCount(s.FailedCount)
		stat.SetTotalDownload(s.TotalDownload)
		stat.SetTotalUploaded(s.TotalUpload)
		stat.SetHighestLastHourBps(s.HighestLastHourBps)
		stat.SetHighestLastMinuteBps(s.HighestLastMinuteBps)
		stat.SetHighestLastSecondBps(s.HighestLastSecondBps)
		stat.SetHighestLastTenMinutesBps(s.HighestLastTenMinutesBps)
		stat.SetLastHourBps(s.LastHourBps)
		stat.SetLastMinuteBps(s.LastMinuteBps)
		stat.SetLastSecondBps(s.LastSecondBps)
		stat.SetLastTenMinutesBps(s.LastTenMinutesBps)
	}
	totalDownload, _ := cache.Instance.Get("totaldownload")
	if totalDownload == nil {
		common.Info("no cache item for total download")
		return
	}
	rawb, ok := totalDownload.([]byte)
	if !ok {
		common.Error("total download to []byte failed")
		return
	}
	var bufDownload bytes.Buffer
	bufDownload.Write(rawb)
	decoderDownload := gob.NewDecoder(&bufDownload)
	var b uint64
	if err := decoderDownload.Decode(&b); err != nil {
		common.Error("to TotalStat failed", err)
		return
	}
	common.TotalStat.SetDownload(b)

	totalUpload, _ := cache.Instance.Get("totalupload")
	if totalUpload == nil {
		common.Info("no cache item for total upload")
		return
	}
	rawu, ok := totalUpload.([]byte)
	if !ok {
		common.Error("total upload to []byte failed")
		return
	}
	var bufUpload bytes.Buffer
	bufUpload.Write(rawu)
	decoderUpload := gob.NewDecoder(&bufUpload)
	var u uint64
	if err := decoderUpload.Decode(&u); err != nil {
		common.Error("to TotalStat failed", err)
		return
	}
	common.TotalStat.SetUpload(u)
}

func (m *StatisticWrapper) SaveToRedis() {
	m.RLock()
	defer m.RUnlock()
	for server, stat := range m.StatisticMap {
		s := Stat{
			Id:          server.id,
			Address:     server.address,
			FailedCount: stat.GetFailedCount(),
			Latency:     stat.GetLatency(),
			totalStat: totalStat{
				TotalDownload: stat.GetTotalDownload(),
				TotalUpload:   stat.GetTotalUploaded(),
			},
			highestStat: highestStat{
				HighestLastHourBps:       stat.GetHighestLastHourBps(),
				HighestLastTenMinutesBps: stat.GetHighestLastTenMinutesBps(),
				HighestLastMinuteBps:     stat.GetHighestLastMinuteBps(),
				HighestLastSecondBps:     stat.GetHighestLastSecondBps(),
			},
			lastStat: lastStat{
				LastHourBps:       stat.GetLastHourBps(),
				LastTenMinutesBps: stat.GetLastTenMinutesBps(),
				LastMinuteBps:     stat.GetLastMinuteBps(),
				LastSecondBps:     stat.GetLastSecondBps(),
			},
		}
		var buf bytes.Buffer
		encoder := gob.NewEncoder(&buf)
		if err := encoder.Encode(s); err != nil {
			continue
		}
		cache.Instance.Put(server.address, interface{}(buf.Bytes()))
	}
	var bufDownload bytes.Buffer
	encoderDownload := gob.NewEncoder(&bufDownload)
	if err := encoderDownload.Encode(common.TotalStat.GetDownload()); err != nil {
		common.Error("encoding total download failed", err)
		return
	}
	cache.Instance.Put("totaldownload", interface{}(bufDownload.Bytes()))

	var bufUpload bytes.Buffer
	encoderUpload := gob.NewEncoder(&bufUpload)
	if err := encoderUpload.Encode(common.TotalStat.GetUpload()); err != nil {
		common.Error("encoding total upload failed", err)
		return
	}
	cache.Instance.Put("totalupload", interface{}(bufUpload.Bytes()))
}
