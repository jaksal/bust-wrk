package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync/atomic"
)

// Task ...
type Task struct {
	Step     string            `json:"step"`
	URL      string            `json:"url"`
	Method   string            `json:"method"`
	URLParam map[string]string `json:"url_param"`
	Body     map[string]string `json:"body"`
	SetParam map[string]string `json:"set_param"`
	UseToken bool              `json:"use_token"`
	IsOnce   bool              `json:"is_once"`
	WaitSec  string            `json:"wait_sec"`
}

// Scenario ...
type Scenario struct {
	Param   map[string]string `json:"param"`
	Pre     []*Task           `json:"pre"`
	Run     []*Task           `json:"run"`
	PreStep []*Task           `json:"pre_step"`
}

// IsPre ...
func (s *Scenario) IsPre() bool {
	return len(s.Pre) > 0
}

var curStep, maxStep int

// newUser
func (s *Scenario) newUser() *User {
	user := &User{
		idx:   atomic.AddUint32(&usrIdx, 1),
		param: make(map[string]string),
	}
	for key, val := range s.Param {
		startIdx := strings.Index(val, "[")
		endIdx := strings.Index(val, "]")
		if startIdx == -1 || endIdx == -1 {
			user.param[key] = val
		} else {
			valList := strings.Split(val[startIdx+1:endIdx], ":")
			if valList[0] == "RAND" {
				endRand, _ := strconv.Atoi(valList[2])
				startRand, _ := strconv.Atoi(valList[1])
				user.param[key] = strconv.Itoa(rand.Intn(endRand-startRand) + startRand)
			} else if valList[0] == "STEP" {
				if curStep == 0 {
					curStep, _ = strconv.Atoi(valList[1])
					maxStep, _ = strconv.Atoi(valList[2])
					log.Println("step min max : ", curStep, maxStep)
				}
				if curStep >= maxStep {
					return nil
				}
				// log.Println("cur step : ", curStep)
				user.param[key] = strconv.Itoa(curStep)
				curStep++
			} else {
				log.Fatalln("not support param", val)
			}
		}
	}
	return user
}

// LoadConfig ...
func LoadConfig(filePath string) (*Scenario, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	scenario := &Scenario{}
	if err := json.Unmarshal(data, scenario); err != nil {
		return nil, err
	}

	return scenario, nil
}
