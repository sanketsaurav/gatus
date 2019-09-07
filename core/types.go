package core

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Result struct {
	HttpStatus      int
	Hostname        string
	Ip              string
	Duration        time.Duration
	Errors          []error
	ConditionResult []*ConditionResult
}

type Service struct {
	Name             string       `yaml:"name"`
	Url              string       `yaml:"url"`
	Interval         int          `yaml:"interval,omitempty"`
	FailureThreshold int          `yaml:"failure-threshold,omitempty"`
	Conditions       []*Condition `yaml:"conditions"`
}

func (service *Service) getIp(result *Result) {
	urlObject, err := url.Parse(service.Url)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return
	}
	result.Hostname = urlObject.Hostname()
	ips, err := net.LookupIP(urlObject.Hostname())
	if err != nil {
		result.Errors = append(result.Errors, err)
		return
	}
	result.Ip = ips[0].String()
}

func (service *Service) getStatus(result *Result) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	startTime := time.Now()
	response, err := client.Get(service.Url)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return
	}
	result.Duration = time.Now().Sub(startTime)
	result.HttpStatus = response.StatusCode
}

func (service *Service) EvaluateConditions() *Result {
	result := &Result{}
	service.getStatus(result)
	service.getIp(result)
	for _, condition := range service.Conditions {
		condition.Evaluate(result)
	}
	return result
}

type ConditionResult struct {
	Condition   *Condition
	Success     bool
	Explanation string
}

type Condition string

func (c *Condition) Evaluate(result *Result) {
	condition := string(*c)
	if strings.Contains(condition, "==") {
		parts := sanitizeAndResolve(strings.Split(condition, "=="), result)
		if parts[0] == parts[1] {
			result.ConditionResult = append(result.ConditionResult, &ConditionResult{
				Condition:   c,
				Success:     true,
				Explanation: fmt.Sprintf("%s is equal to %s", parts[0], parts[1]),
			})
		} else {
			result.ConditionResult = append(result.ConditionResult, &ConditionResult{
				Condition:   c,
				Success:     false,
				Explanation: fmt.Sprintf("%s is not equal to %s", parts[0], parts[1]),
			})
		}
	} else if strings.Contains(condition, "!=") {
		parts := sanitizeAndResolve(strings.Split(condition, "!="), result)
		if parts[0] != parts[1] {
			result.ConditionResult = append(result.ConditionResult, &ConditionResult{
				Condition:   c,
				Success:     true,
				Explanation: fmt.Sprintf("%s is not equal to %s", parts[0], parts[1]),
			})
		} else {
			result.ConditionResult = append(result.ConditionResult, &ConditionResult{
				Condition:   c,
				Success:     false,
				Explanation: fmt.Sprintf("%s is equal to %s", parts[0], parts[1]),
			})
		}
	}
}

func sanitizeAndResolve(list []string, result *Result) []string {
	var sanitizedList []string
	for _, element := range list {
		element = strings.TrimSpace(element)
		switch strings.ToUpper(element) {
		case "$STATUS":
			element = strconv.Itoa(result.HttpStatus)
		case "$IP":
			element = result.Ip
		default:
		}
		sanitizedList = append(sanitizedList, element)
	}
	return sanitizedList
}