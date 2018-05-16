package main

import (
	"fmt"
	"time"
)

// RequesterStats used for colelcting aggregate statistics
type RequesterStats struct {
	Title          string
	TotRespSize    int64
	TotDuration    time.Duration
	MinRequestTime time.Duration
	MaxRequestTime time.Duration
	NumRequests    int
	NumErrs        int
}

// MaxDuration ...
func MaxDuration(d1 time.Duration, d2 time.Duration) time.Duration {
	if d1 > d2 {
		return d1
	}
	return d2
}

// MinDuration ...
func MinDuration(d1 time.Duration, d2 time.Duration) time.Duration {
	if d1 < d2 {
		return d1
	}
	return d2
}

func (rs *RequesterStats) Err() {
	rs.NumErrs++
}

// Calc
func (rs *RequesterStats) Calc(due time.Duration, respSize int) {
	rs.NumRequests++
	rs.TotRespSize += int64(respSize)
	rs.TotDuration += due
	rs.MaxRequestTime = MaxDuration(rs.MaxRequestTime, due)
	rs.MinRequestTime = MinDuration(rs.MinRequestTime, due)
}

// Add ...
func (rs *RequesterStats) Add(new *RequesterStats) {
	rs.NumErrs += new.NumErrs
	rs.NumRequests += new.NumRequests
	rs.TotRespSize += new.TotRespSize
	rs.TotDuration += new.TotDuration
	rs.MaxRequestTime = MaxDuration(rs.MaxRequestTime, new.MaxRequestTime)
	rs.MinRequestTime = MinDuration(rs.MinRequestTime, new.MinRequestTime)
}

// PrintResult ...
func (rs *RequesterStats) PrintResult(responders int) string {
	result := fmt.Sprintf("\nTitle:\t\t\t%v\n", rs.Title)

	if rs.NumRequests == 0 {
		return result + fmt.Sprintf("Number of Errors:\t%v\n", rs.NumErrs)
	}
	avgThreadDur := rs.TotDuration / time.Duration(responders) //need to average the aggregated duration

	reqRate := float64(rs.NumRequests) / avgThreadDur.Seconds()
	avgReqTime := rs.TotDuration / time.Duration(rs.NumRequests)
	bytesRate := float64(rs.TotRespSize) / avgThreadDur.Seconds()

	result += fmt.Sprintf("%v requests in %v, %v read (%v)\n", rs.NumRequests, avgThreadDur, ByteSize{float64(rs.TotRespSize)}, ByteSize{float64(int(rs.TotRespSize) / rs.NumRequests)})
	result += fmt.Sprintf("Requests/sec:\t\t%.2f\nTransfer/sec:\t\t%v\nAvg Req Time:\t\t%v\n", reqRate, ByteSize{bytesRate}, avgReqTime)
	result += fmt.Sprintf("Fastest Request:\t%v\n", rs.MinRequestTime)
	result += fmt.Sprintf("Slowest Request:\t%v\n", rs.MaxRequestTime)
	result += fmt.Sprintf("Number of Errors:\t%v\n", rs.NumErrs)

	return result
}

// PrintCsvHeader ...
func PrintCsvHeader() string {
	return "Title,Requests,Errors,avg Thread Duetime,Total RespSize,Avg RespSize,Requests/sec,Transfer/sec,Avg Req Time,Fastest Request,Slowest Request\n"
}

// PrintCSV ...
func (rs *RequesterStats) PrintCSV(responders int) string {
	result := rs.Title + ","
	result += fmt.Sprintf("%v,%v,", rs.NumRequests, rs.NumErrs)

	if rs.NumRequests == 0 {
		result += "0,0,0,0,0,0,0,0"
	} else {
		avgThreadDur := rs.TotDuration / time.Duration(responders) //need to average the aggregated duration

		reqRate := float64(rs.NumRequests) / avgThreadDur.Seconds()
		avgReqTime := rs.TotDuration / time.Duration(rs.NumRequests)
		bytesRate := float64(rs.TotRespSize) / avgThreadDur.Seconds()

		result += fmt.Sprintf("%v,%v,%.2f,", avgThreadDur, rs.TotRespSize, float64(int(rs.TotRespSize)/rs.NumRequests))
		result += fmt.Sprintf("%.2f,%.2f,%v,", reqRate, bytesRate, avgReqTime)
		result += fmt.Sprintf("%v,%v\n", rs.MinRequestTime, rs.MaxRequestTime)
	}
	return result
}
