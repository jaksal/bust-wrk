package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	pb "gopkg.in/cheggaaa/pb.v1"
)

var (
	goroutines  int
	duration    int
	srvaddr     string
	timeout     int
	userCnt     int
	senarioFile string
	redisURL    string
	printCSV    string
	ready       bool
	ramp        int
	check       bool

	usrIdx          uint32
	interrupted     int32
	statsAggregator chan *RequesterStats
	userPool        chan *User
	senario         *Scenario
	isZombi         bool
	throttle        *Throttle
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func parseArgument(name string, args []string) error {
	flagSet := flag.NewFlagSet(name, flag.ContinueOnError)

	flagSet.IntVar(&goroutines, "c", runtime.NumCPU()*2, "Number of goroutines to use (concurrent connections)")
	flagSet.IntVar(&duration, "d", 10, "Duration of test in seconds")
	flagSet.IntVar(&timeout, "timeout", 30, "timeout of test in seconds")
	flagSet.StringVar(&srvaddr, "s", "http://127.0.0.1:2142", "server addr default localhost:2142")
	flagSet.IntVar(&userCnt, "u", runtime.NumCPU()*2, "pre create user count")
	flagSet.StringVar(&senarioFile, "f", "", "senario file path")
	flagSet.StringVar(&redisURL, "r", "", "client mode. redis url")
	flagSet.StringVar(&printCSV, "csv", "", "print result csv mode")
	flagSet.BoolVar(&ready, "ready", false, "ready before run")
	flagSet.IntVar(&ramp, "ramp", 0, "ramp up count")
	flagSet.BoolVar(&check, "check", false, "check senario")

	return flagSet.Parse(args) // Scan the arguments list
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() + goroutines)

	if err := parseArgument(os.Args[0], os.Args[1:]); err != nil {
		panic(err)
	}

	if check {
		Check()
		return
	}

	if redisURL != "" {
		isZombi = true
	}

	// create interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		Stop()
	}()

	if isZombi {
		if err := InitRedis(redisURL); err != nil {
			fmt.Println("redis init error", redisURL, err)
			panic(err)
		}
		defer FinalRedis()

		subscribe("NBA-WRK")
	} else {
		StartTest(ready)
		RunTest()
	}
}

// ParseTest ...
func ParseTest(cmd string) bool {
	cmdList := strings.Split(cmd, " ")
	if cmdList[0] == "PRE" {
		if err := parseArgument("REDIS", cmdList[1:]); err != nil {
			log.Println("cmd parse err", err)
		}
		StartTest(false)
	} else if cmdList[0] == "RUN" {
		RunTest()
	} else if cmdList[0] == "EXIT" {
		return true
	}
	return false
}

func Check() {
	var err error
	senario, err = LoadConfig(senarioFile)
	if err != nil {
		fmt.Println("load senario file error", senarioFile, err)
		panic(err)
	}

	statsAggregator = make(chan *RequesterStats)
	aggStats := make(map[string]*RequesterStats)

	user := senario.newUser()
	if user == nil {
		panic("user create fail")
	}
	user.client = newHTTPClient()

	if senario.IsPre() {
		if _, err := user.Pre(senario.Pre); err != nil {
			panic(err)
		}
	}

	go func() {
		user.Run(senario.Run, senario.PreStep)
		statsAggregator <- &RequesterStats{Title: "total task", MinRequestTime: time.Minute}
	}()

	for {
		stats := <-statsAggregator

		if aggStats[stats.Title] == nil {
			aggStats[stats.Title] = stats
		} else {
			aggStats[stats.Title].Add(stats)
		}

		if stats.Title == "total task" {
			break
		}
	}

	fmt.Println("CHECK Finish!")

	for _, stats := range aggStats {
		result := stats.PrintResult(1)
		fmt.Printf("%s", result)
	}
}

// StartTest ...
func StartTest(wait bool) {
	if ramp > 0 {
		throttle = NewThrottle(ramp)
	}

	var err error
	senario, err = LoadConfig(senarioFile)
	if err != nil {
		fmt.Println("load senario file error", senarioFile, err)
		panic(err)
	}
	if goroutines > userCnt {
		userCnt = goroutines * 2
	}

	fmt.Printf("Running %vs test @ %v,  %v goroutine(s) running concurrently %d user\n", duration, srvaddr, goroutines, userCnt)

	statsAggregator = make(chan *RequesterStats, goroutines)

	if senario.IsPre() {
		userPool = Pre(userCnt)

		if wait {
			// pause
			buf := bufio.NewReader(os.Stdin)
			fmt.Print("start > ")
			if _, err := buf.ReadBytes('\n'); err != nil {
				fmt.Println(err)
			}
		}
	}
}

// RunTest ...
func RunTest() {
	// run senario
	for i := 0; i < goroutines; i++ {
		go Run(userPool)
	}

	responders := 0
	aggStats := make(map[string]*RequesterStats)

	// pregress bar
	pbar := pb.New(duration).Prefix("RUN")
	tickChan := time.NewTicker(time.Second).C

	pbar.Start()
	for responders < goroutines {
		select {
		case <-tickChan:
			pbar.Increment()
			if ramp > 0 {
				throttle.Reset()
			}
		case stats := <-statsAggregator:
			if aggStats[stats.Title] == nil {
				aggStats[stats.Title] = stats
			} else {
				aggStats[stats.Title].Add(stats)
			}
			if stats.Title == "total task" {
				responders++
			}
		}
	}
	pbar.FinishPrint("RUN Finish!")

	for _, stats := range aggStats {
		result := stats.PrintResult(responders)
		fmt.Printf("%s", result)
		if isZombi {
			result = "IP : " + GetMyIP() + "\n" + result
			publish("NBA-WRK", result)
		}
	}

	if printCSV != "" {
		f, err := os.OpenFile(printCSV, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		f.WriteString(PrintCsvHeader())

		for _, stats := range aggStats {
			f.WriteString(stats.PrintCSV(responders))
		}
	}

}

// Pre ...
func Pre(cnt int) chan *User {
	userPool := make(chan *User, cnt)

	httpClient := newHTTPClient()

	pbar := pb.New(cnt).Prefix("PRE")
	pbar.Start()
	for i := 0; i < cnt; i++ {
		if atomic.LoadInt32(&interrupted) != 0 {
			break
		}

		user := senario.newUser()
		if user == nil {
			break
		}

		user.client = httpClient

		if _, err := user.Pre(senario.Pre); err != nil {
			break
		} else {
			userPool <- user
		}
		pbar.Increment()
		runtime.Gosched()
	}
	pbar.FinishPrint("PRE Finish!")
	return userPool
}

// Run ...
func Run(userPool chan *User) {
	httpClient := newHTTPClient()

	stats := &RequesterStats{Title: "total task", MinRequestTime: time.Minute}
	start := time.Now()
	for time.Since(start).Seconds() <= float64(duration) && atomic.LoadInt32(&interrupted) == 0 {

		if ramp > 0 {
			if throttle.CheckLimit() == false {
				continue
			}
		}

		var user *User
		if senario.IsPre() {
			user = <-userPool
		} else {
			user = senario.newUser()
		}
		user.client = httpClient

		reqDur, err := user.Run(senario.Run, senario.PreStep)
		if err == nil {
			stats.TotDuration += reqDur
			stats.TotRespSize += int64(user.respSize)
			stats.MaxRequestTime = MaxDuration(reqDur, stats.MaxRequestTime)
			stats.MinRequestTime = MinDuration(reqDur, stats.MinRequestTime)
			stats.NumRequests++
		} else {
			stats.NumErrs++
		}
		if senario.IsPre() {
			userPool <- user
		}
	}
	statsAggregator <- stats
}

func Stop() {
	atomic.StoreInt32(&interrupted, 1)
	fmt.Printf("stopping...\n")
}
