package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func postJSON(url string, obj interface{}, token string) (*http.Response, error) {
	tr := &http.Transport{
		MaxIdleConnsPerHost: 10,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   5 * time.Second,
	}

	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(obj); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	return client.Do(req)
}

func clientLogin(baseURL, username, password string) (*loginResp, error) {
	url := baseURL + "/login"
	r, err := postJSON(url, map[string]string{"username": username, "password": password}, "")
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var lresp loginResp
	if err = json.Unmarshal(body, &lresp); err != nil {
		return nil, err
	}

	if r.StatusCode >= 400 {
		err = fmt.Errorf("Login Error: %s", lresp.Msg)
	}

	return &lresp, err
}
