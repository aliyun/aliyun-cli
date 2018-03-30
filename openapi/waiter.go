package openapi
//
//import (
//	"github.com/aliyun/aliyun-cli/cli"
//	"time"
//	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
//	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
//	"fmt"
//	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
//	"github.com/jmespath/go-jmespath"
//	"encoding/json"
//	"errors"
//	"strconv"
//	"strings"
//)
//
//type Waiter struct {
//	client   *sdk.Client
//	request  *requests.CommonRequest
//	waitExpr string
//	targets  []string
//	timeout  time.Duration
//	interval time.Duration
//
//}
//
//func NewWaiterWithCTX(ctx *cli.Context, client *sdk.Client, request *requests.CommonRequest) *Waiter {
//	waitForExprFlag := ctx.Flags().Get(WaitForExprFlag.Name, WaitForExprFlag.Shorthand)
//	waitForTargetFlag := ctx.Flags().Get(WaitForTargetFlag.Name, WaitForTargetFlag.Shorthand)
//	waitTimeoutFlag := ctx.Flags().Get(WaitTimeoutFlag.Name, WaitTimeoutFlag.Shorthand)
//	waitIntervalFlag := ctx.Flags().Get(WaitIntervalFlag.Name, WaitIntervalFlag.Shorthand)
//
//	timeout, err := strconv.Atoi(waitTimeoutFlag.GetValueOrDefault(ctx, "0"))
//	if err != nil {
//		fmt.Println(err)
//		timeout = 0
//	}
//
//	if timeout < 0 {
//		timeout = 0
//	}
//
//	interval, err := strconv.Atoi(waitIntervalFlag.GetValueOrDefault(ctx, "1"))
//	if err != nil {
//		fmt.Println(err)
//		interval = 1
//	}
//
//	if interval < 0 {
//		interval = 1
//	}
//
//	return &Waiter{
//		client: client,
//		request: request,
//		waitExpr: waitForExprFlag.GetValueOrDefault(ctx, ""),
//		targets: strings.Split(waitForTargetFlag.GetValueOrDefault(ctx, ""), ","),
//		timeout:time.Duration(timeout) * time.Second,
//		interval:time.Duration(interval) * time.Second,
//	}
//}
//
//
//
//func (w *Waiter)Wait() (response *responses.CommonResponse, err error){
//	if w.timeout == 0 {
//		return w.client.ProcessCommonRequest(w.request)
//	}
//
//
//	doRequestAndCheck := func() (bool, *responses.CommonResponse, error) {
//		var v interface{}
//
//		resp, err := w.client.ProcessCommonRequest(w.request)
//
//		if err != nil {
//			fmt.Println(err)
//			return false, nil, err
//		}
//
//		if len(w.targets) == 0 {
//			return true, resp, fmt.Errorf("no target targets: %v", w.targets)
//		}
//
//		err = json.Unmarshal(resp.GetHttpContentBytes(), &v)
//
//		if err != nil {
//			return false, resp, err
//		}
//
//		intf, err := jmespath.Search(w.waitExpr, v)
//
//		if err != nil {
//			return false, resp, fmt.Errorf("jmespath: '%s' failed %s", w.waitExpr, err)
//		}
//
//		res := fmt.Sprintf("%v", intf)
//		if res == w.targets[0] {
//			return true, resp, nil
//		} else {
//			//fmt.Println("response value: " + res)
//		}
//
//		return false, resp, err
//	}
//
//	t := time.NewTimer(w.timeout)
//
//	for {
//
//		find, req, err := doRequestAndCheck()
//
//		if find {
//			//fmt.Printf("Find targets %v\n", w.targets)
//			return req, err
//		}
//
//		if err != nil {
//			fmt.Println(err)
//		}
//
//		select {
//		case <- time.After(w.interval):
//			//fmt.Println("interval done")
//		case <- t.C:
//			return req, errors.New("Timeout")
//		}
//
//	}
//}
