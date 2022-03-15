package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type RpcTarget struct {
	Host     string
	Port     string
	User     string
	Password string
}

func NewRpcTarget(host string, port string, user string, pw string) *RpcTarget {
	return &RpcTarget{
		Host:     host,
		Port:     port,
		User:     user,
		Password: pw,
	}
}

type RpcRequest struct {
	target *RpcTarget
	Method string
	Params interface{}
}

// Wait until an RpcTarget is available
func (target *RpcTarget) WaitUntilAvailable() {
	for {
		pingReq := target.NewRequest("uptime", map[string]interface{}{})
		_, err := pingReq.Send()
		if err == nil {
			log.Println("RPC available")
			return
		}
		log.Println("Waiting for RPC")
		time.Sleep(time.Second)
	}
}

func (target *RpcTarget) NewRequest(method string, params interface{}) *RpcRequest {
	return &RpcRequest{
		target: target,
		Method: method,
		Params: params,
	}
}

func (req *RpcRequest) Send() (interface{}, error) {
	body := map[string]interface{}{
		"method": req.Method,
		"params": req.Params,
		"id":     "foo",
	}

	bodyRaw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	targetRequest, err := http.NewRequest(
		"POST",
		"http://"+req.target.Host+":"+req.target.Port,
		bytes.NewReader(bodyRaw),
	)
	//log.Println("Requesting http://" + req.target.Host + ":" + req.target.Port)

	auth := base64.StdEncoding.EncodeToString([]byte(req.target.User + ":" + req.target.Password))
	targetRequest.Header.Add("Authorization", "Basic "+auth)

	client := http.Client{}
	resp, err := client.Do(targetRequest)
	if err != nil {
		return nil, err
	} else {
		res, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var jsonRes map[string]interface{} = make(map[string]interface{})
		err = json.Unmarshal(res, &jsonRes)
		if err != nil {
			return nil, err
		}

		jsonErr, hasError := jsonRes["error"]
		if hasError && jsonErr != nil {
			msg := jsonErr.(map[string]interface{})
			return nil, errors.New(msg["message"].(string))
		}

		data, hasData := jsonRes["result"]
		if hasData {
			return data, nil
		}
		return nil, nil
	}
}

func TargetFromHost(host string) *RpcTarget {
	target := NewRpcTarget(host, RPC_PORT, "dashrpc", "rpcpassword")
	target.WaitUntilAvailable()
	return target
}
