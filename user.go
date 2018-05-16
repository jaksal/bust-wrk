package main

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// User ...
type User struct {
	idx      uint32
	param    map[string]string
	respSize int
	client   *http.Client
	cycle    int
}

func (u *User) replaceParam(param string) string {
	startIdx := strings.Index(param, "[")
	endIdx := strings.Index(param, "]")
	if startIdx == -1 || endIdx == -1 {
		return param
	}
	newKey := param[startIdx+1 : endIdx]

	if newVal, ok := u.param[newKey]; ok {
		return strings.Replace(param, param[startIdx:endIdx+1], newVal, -1)
	}
	log.Fatalln("not found user param", newKey)
	return ""
}

func (u *User) parseParam(param map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range param {
		/*
			if v[0] == '[' && v[len(v)-1] == ']' {
				valKey := v[1 : len(v)-1]
				if newVal, ok := u.param[valKey]; ok {
					result[k] = newVal
				} else {
					log.Fatalln("not found user param", valKey)
				}
			} else {
				result[k] = v
			}
		*/
		result[k] = u.replaceParam(v)
	}
	return result
}

func (u *User) setParam(srcParam map[string]string, destParam map[string]interface{}) {
	for k, v := range srcParam {
		newKey := k[1 : len(k)-1]

		var vTemp interface{}
		vList := strings.Split(v, ".")

		for i := 0; i < len(vList); i++ {
			newValKey := vList[i]
			if i == len(vList)-1 {
				vTemp = destParam[newValKey]
				break
			} else {
				destParam = destParam[newValKey].(map[string]interface{})
			}
		}

		switch vTemp.(type) {
		case string:
			u.param[newKey] = vTemp.(string)
		case float64:
			u.param[newKey] = strconv.FormatFloat(vTemp.(float64), 'f', 0, 64)
		case int, int32, int64:
			u.param[newKey] = strconv.Itoa(vTemp.(int))
		}

		//log.Printf("[%04d] set param key=%s val=%s\n", u.idx, newKey, u.param[newKey])
	}
}

// Pre ...
func (u *User) Pre(preSenario []*Task) (time.Duration, error) {

	start := time.Now()
	for _, task := range preSenario {

		newURL := u.replaceParam(task.URL)
		// log.Printf("[%4d] start task=%v\n", u.idx, task)
		var headerList map[string]string
		if task.UseToken {
			headerList = make(map[string]string)
			headerList["Authorization"] = "Bearer " + u.param["ACCESS_TOKEN"]
		}

		res, _, _, err := doRequest(u.client, srvaddr+newURL, task.Method, headerList, u.parseParam(task.URLParam), u.parseParam(task.Body))
		if err != nil {
			log.Printf("[%04d] task=%+v err=%s\n", u.idx, task, err)
			break
		}
		u.setParam(task.SetParam, res)
		//log.Printf("[%4d] nick:%s due=%f\n", u.idx, userID, due.Seconds())
	}

	due := time.Now().Sub(start)
	// log.Printf("[%4d] finish! cycle=%d due=%f\n", u.idx, cycle, due.Seconds())

	return due, nil
}

func parseWaitSec(str string) int {
	if str == "" {
		return 0
	}
	ps := strings.Split(str, ":")

	sec := 0
	if len(ps) == 1 {
		sec, _ = strconv.Atoi(ps[0])
	} else if len(ps) == 2 {
		min, _ := strconv.Atoi(ps[0])
		max, _ := strconv.Atoi(ps[1])
		sec = rand.Intn(max-min) + min
	}
	return sec
}

// Run ...
func (u *User) Run(runSenario, preSteps []*Task) (time.Duration, error) {
	u.respSize = 0
	start := time.Now()

	for _, task := range runSenario {
		if task.IsOnce && u.cycle > 0 {
			continue
		}
		u.param["STEP_NAME"] = task.Step

		for _, step := range preSteps {
			if step.URL != "" {
				newURL := u.replaceParam(step.URL)

				var headerList map[string]string
				if step.UseToken {
					if u.param["ACCESS_TOKEN"] == "" {
						continue
					}
					headerList = make(map[string]string)
					headerList["Authorization"] = "Bearer " + u.param["ACCESS_TOKEN"]
				}

				res, _, _, err := doRequest(u.client, srvaddr+newURL, step.Method, headerList, u.parseParam(step.URLParam), u.parseParam(step.Body))
				if err != nil {
					log.Printf("[%04d] url=%s err=%s\n", u.idx, srvaddr+newURL, err)
					break
				}
				u.setParam(step.SetParam, res)
			}

			if sec := parseWaitSec(step.WaitSec); sec > 0 {
				time.Sleep(time.Second * time.Duration(sec))
			}
		}

		if task.URL != "" {
			// log.Printf("[%4d] start task=%v\n", u.idx, task)
			stats := &RequesterStats{Title: task.Step, MinRequestTime: time.Minute}

			newURL := u.replaceParam(task.URL)

			var headerList map[string]string
			if task.UseToken {
				headerList = make(map[string]string)
				headerList["Authorization"] = "Bearer " + u.param["ACCESS_TOKEN"]
			}

			res, due, respSize, err := doRequest(u.client, srvaddr+newURL, task.Method, headerList, u.parseParam(task.URLParam), u.parseParam(task.Body))
			if err != nil {
				stats.Err()
				statsAggregator <- stats
				log.Printf("[%04d] url=%s err=%s\n", u.idx, srvaddr+newURL, err)
				break
			}
			u.respSize += respSize

			u.setParam(task.SetParam, res)

			//log.Printf("[%4d] nick:%s due=%f\n", u.idx, userID, due.Seconds())
			stats.Calc(due, respSize)
			statsAggregator <- stats
		}

		if sec := parseWaitSec(task.WaitSec); sec > 0 {
			time.Sleep(time.Second * time.Duration(sec))
		}
	}
	u.cycle++
	return time.Now().Sub(start), nil
}
