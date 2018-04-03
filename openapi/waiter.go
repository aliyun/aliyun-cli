package openapi

import (
	"github.com/aliyun/aliyun-cli/cli"
	"time"
	"fmt"
	"github.com/aliyun/aliyun-cli/i18n"
	"strconv"
)

var WaiterFlag = &cli.Flag {Category:"helper",
	Name: "waiter",
	AssignedMode: cli.AssignedRepeatable,
	Short: i18n.T(
		"use `--waiter expr=<jmesPath> to=<value>` to pull api until result equal to expected value",
		"使用 `--waiter expr=<jmesPath> to=<value>` 来轮询调用OpenAPI，知道返回期望的值"),
	Long: i18n.T(
		"",
		""),
	Fields: []cli.Field {
		{Key:"expr", Required:true, Short: i18n.T("", "")},
		{Key:"to", Required:true, Short: i18n.T("", "")},
		{Key:"timeout",DefaultValue:"180", Short:i18n.T("", "")},
		{Key:"interval",DefaultValue:"5", Short:i18n.T("","")},
	},
	ExcludeWith:[]string{"pager"},
}

type Waiter struct {
	expr string
	to string
//	timeout  time.Duration
//	interval time.Duration
}

func GetWaiter() *Waiter {
	if !WaiterFlag.IsAssigned() {
		return nil
	}

	waiter := &Waiter {
	}
	waiter.expr, _ = WaiterFlag.GetFieldValue("expr")
	waiter.to, _ = WaiterFlag.GetFieldValue("to")
	//waiter.timeout = time.Duration(time.Second * 180)
	//waiter.interval = time.Duration(time.Second * 5)

	//timeout, err := strconv.Atoi(waitTimeoutFlag.GetStringOrDefault(ctx, "0"))
	//if err != nil {
	//	fmt.Println(err)
	//	timeout = 0
	//}
	//
	//if timeout < 0 {
	//	timeout = 0
	//}
	//
	//interval, err := strconv.Atoi(waitIntervalFlag.GetStringOrDefault(ctx, "1"))
	//if err != nil {
	//	fmt.Println(err)
	//	interval = 1
	//}
	//
	//if interval < 0 {
	//	interval = 1
	//}
	//
	return waiter
}

func (a *Waiter) CallWith(invoker Invoker) (string, error) {
	//
	// timeout is 1-180 seconds
	timeout := time.Duration(time.Second * 180)
	if s, ok := WaiterFlag.GetFieldValue("timeout"); ok {
		if n, err := strconv.Atoi(s); err == nil {
			if n <= 0 && n > 180 {
				return "", fmt.Errorf("--waiter timeout=%s must between 1-180 (seconds)", s)
			}
			timeout = time.Duration(time.Second * time.Duration(n))
		} else {
			return "", fmt.Errorf("--waiter timeout=%s must be integer", s)
		}
	}
	//
	// interval is 2-10 seconds
	interval := time.Duration(time.Second * 5)
	if s, ok := WaiterFlag.GetFieldValue("interval"); ok {
		if n, err := strconv.Atoi(s); err == nil {
			if n <= 1 && n > 10 {
				return "", fmt.Errorf("--waiter interval=%s must between 2-10 (seconds)", s)
			}
			interval = time.Duration(time.Second * time.Duration(n))
		} else {
			return "", fmt.Errorf("--waiter interval=%s must be integer", s)
		}
	}
	//waiter.timeout = time.Duration(time.Second * 180)
	//waiter.interval = time.Duration(time.Second * 5)
	begin := time.Now()
	for {
		resp, err := invoker.Call()
		if err != nil {
			return "", err
		}

		v, err := evaluateExpr(resp.GetHttpContentBytes(), a.expr)
		if err != nil {
			return "", err
		}

		if v == a.to {
			return resp.GetHttpContentString(), nil
		}
		duration := time.Now().Sub(begin)
		if duration > timeout {
			return "", fmt.Errorf("wait '%s' to '%s' timeout(%dseconds), last='%s'",
				a.expr, a.to, timeout / time.Second, v)
		}
		time.Sleep(interval)
	}
}
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
