package local

type Stat struct {
	Id                       string
	Address                  string
	ProtocolType             string
	FailedCount              uint32
	Latency                  int64
	TotalUpload              uint64
	TotalDownload            uint64
	HighestLastSecondBps     uint64
	HighestLastMinuteBps     uint64
	HighestLastTenMinutesBps uint64
	HighestLastHourBps       uint64
	LastSecondBps            uint64
	LastMinuteBps            uint64
	LastTenMinutesBps        uint64
	LastHourBps              uint64
}

type Stats []*Stat

func (slice Stats) Len() int {
	return len(slice)
}

func (slice Stats) Less(i, j int) bool {
	return slice[i].Address < slice[j].Address
}

func (slice Stats) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

type byLastSecondBps struct{ Stats }

func (slice byLastSecondBps) Less(i, j int) bool {
	return slice.Stats[i].LastSecondBps < slice.Stats[j].LastSecondBps
}

type byLastMinuteBps struct{ Stats }

func (slice byLastMinuteBps) Less(i, j int) bool {
	return slice.Stats[i].LastMinuteBps < slice.Stats[j].LastMinuteBps
}

type byLastTenMinutesBps struct{ Stats }

func (slice byLastTenMinutesBps) Less(i, j int) bool {
	return slice.Stats[i].LastTenMinutesBps < slice.Stats[j].LastTenMinutesBps
}

type byLastHourBps struct{ Stats }

func (slice byLastHourBps) Less(i, j int) bool {
	return slice.Stats[i].LastHourBps < slice.Stats[j].LastHourBps
}

type byHighestLastSecondBps struct{ Stats }

func (slice byHighestLastSecondBps) Less(i, j int) bool {
	return slice.Stats[i].HighestLastSecondBps < slice.Stats[j].HighestLastSecondBps
}

type byHighestLastMinuteBps struct{ Stats }

func (slice byHighestLastMinuteBps) Less(i, j int) bool {
	return slice.Stats[i].HighestLastMinuteBps < slice.Stats[j].HighestLastMinuteBps
}

type byHighestLastTenMinutesBps struct{ Stats }

func (slice byHighestLastTenMinutesBps) Less(i, j int) bool {
	return slice.Stats[i].HighestLastTenMinutesBps < slice.Stats[j].HighestLastTenMinutesBps
}

type byHighestLastHourBps struct{ Stats }

func (slice byHighestLastHourBps) Less(i, j int) bool {
	return slice.Stats[i].HighestLastHourBps < slice.Stats[j].HighestLastHourBps
}

type byLatency struct{ Stats }

func (slice byLatency) Less(i, j int) bool {
	return slice.Stats[i].Latency < slice.Stats[j].Latency
}

type byFailedCount struct{ Stats }

func (slice byFailedCount) Less(i, j int) bool {
	return slice.Stats[i].FailedCount < slice.Stats[j].FailedCount
}

type byTotalUpload struct{ Stats }

func (slice byTotalUpload) Less(i, j int) bool {
	return slice.Stats[i].TotalUpload < slice.Stats[j].TotalUpload
}

type byTotalDownload struct{ Stats }

func (slice byTotalDownload) Less(i, j int) bool {
	return slice.Stats[i].TotalDownload < slice.Stats[j].TotalDownload
}

type byProtocolType struct{ Stats }

func (slice byProtocolType) Less(i, j int) bool {
	return slice.Stats[i].ProtocolType < slice.Stats[j].ProtocolType
}
