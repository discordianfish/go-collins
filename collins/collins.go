package collins

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type AssetState struct {
	ID     int `json:"ID"`
	Status struct {
		Name        string `json:"NAME"`
		Description string `json:"DESCRIPTION"`
	} `json:"STATUS,omitempty"`
	Name        string `json:"NAME"`
	Label       string `json:"LABEL,omitempty"`
	Description string `json:"DESCRIPTION,omitempty"`
}

type Status struct {
	Status string `json:"status"`
}

// incomplete
type Asset struct {
	Status
	Data AssetDetails `json:"data"`
}

type AssetDetails struct {
	Asset struct {
		ID     int    `json:"ID"`
		Tag    string `json:"TAG"`
		State  AssetState
		Status string `json:"STATUS"`
		Type   string `json:"TYPE"`
	} `json:"ASSET"`
	Attributes map[string]map[string]string `json:"ATTRIBS"`
	IPMI       struct {
		Address  string `json:"IPMI_ADDRESS"`
		Username string `json:"IPMI_USERNAME"`
		Password string `json:"IPMI_PASSWORD"`
	} `json:"IPMI"`
}

type AssetAddress struct {
	ID      int    `json:"ID"`
	Pool    string `json:"POOL"`
	Address string `json:"ADDRESS"`
	Netmask string `json:"NETMASK"`
	Gateway string `json:"GATEWAY"`
}

type AssetAddresses struct {
	Status
	Data struct {
		Addresses []AssetAddress
	}
}

type Assets struct {
	Status
	Data struct {
		Data []AssetDetails `json:"data"`
	} `json:"Data"`
}

type Client struct {
	client   *http.Client
	user     string
	password string
	url      string
}

func New(user, password, url string) *Client {
	return &Client{
		client:   &http.Client{},
		user:     user,
		password: password,
		url:      url,
	}
}

func (c *Client) Request(method string, path string, params *url.Values) ([]byte, error) {
	url := c.url + path
	if params != nil {
		url = url + "?" + params.Encode()
	}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	log.Printf("> %s", req.URL)
	req.SetBasicAuth(c.user, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Error %d: %s", resp.StatusCode, body)
	}
	return body, nil
}

func (c *Client) GetAssetAddresses(tag string) (*AssetAddresses, error) {
	body, err := c.Request("GET", "/asset/"+tag+"/addresses", nil)
	if err != nil {
		return nil, err
	}
	adresses := &AssetAddresses{}
	return adresses, json.Unmarshal(body, &adresses)
}

func (c *Client) GetAsset(tag string) (*Asset, error) {
	if tag == "" {
		return nil, fmt.Errorf("Tag required")
	}
	body, err := c.Request("GET", "/asset/"+tag, nil)
	if err != nil {
		return nil, err
	}
	asset := &Asset{}
	return asset, json.Unmarshal(body, &asset)
}

func (c *Client) FindAllAssets() (*Assets, error) {
	return c.FindAssets(nil)
}

func (c *Client) FindAssets(params *url.Values) (*Assets, error) {
	body, err := c.Request("GET", "/assets", params)
	if err != nil {
		return nil, err
	}
	assets := &Assets{}
	return assets, json.Unmarshal(body, &assets)
}

func (c *Client) AddAssetLog(tag, mtype, message string) error {
	v := url.Values{}
	v.Set("message", message)
	v.Set("type", mtype)

	req, err := http.NewRequest("PUT", c.url+"/asset/"+tag+"/log?"+v.Encode(), nil)
	if err != nil {
		return err
	}
	log.Printf("> %s", req.URL)
	req.SetBasicAuth(c.user, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Status code %d unexpected", resp.StatusCode)
	}
	return nil
}

func (c *Client) SetStatus(tag, status, reason string) error {
	params := &url.Values{}
	params.Set("status", status)
	params.Set("reason", reason)

	body, err := c.Request("POST", "/asset/"+tag+"/status", params)
	if err != nil {
		return err
	}
	s := &Status{}
	if err := json.Unmarshal(body, &s); err != nil {
		return fmt.Errorf("Couldn't unmarshal %s: %s", body, err)
	}
	if s.Status != "success:ok" {
		return fmt.Errorf("Couldn't set status to %s", status)
	}
	return nil
}
