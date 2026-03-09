package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/crateos/crateos/internal/platform"
)

type Client struct {
	httpc *http.Client
	user  string
}

func NewClient(user string) *Client {
	return &Client{
		httpc: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return net.Dial("unix", platform.AgentSocket)
				},
			},
		},
		user: user,
	}
}

func (c *Client) do(method, path string, body interface{}, out interface{}) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
	}
	req, err := http.NewRequest(method, "http://unix"+path, &buf)
	if err != nil {
		return err
	}
	if c.user != "" {
		req.Header.Set("X-CrateOS-User", c.user)
	}
	resp, err := c.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("api %s %s: %s", method, path, resp.Status)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) Status() (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.do("GET", "/status", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ActorDiagnostics() (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.do("GET", "/diagnostics/actors", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) Services() (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.do("GET", "/services", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
func (c *Client) EnableService(name string) error {
	return c.do("POST", "/services/enable", map[string]string{"name": name}, nil)
}

func (c *Client) DisableService(name string) error {
	return c.do("POST", "/services/disable", map[string]string{"name": name}, nil)
}

func (c *Client) StartService(name string) error {
	return c.do("POST", "/services/start", map[string]string{"name": name}, nil)
}

func (c *Client) StopService(name string) error {
	return c.do("POST", "/services/stop", map[string]string{"name": name}, nil)
}

func (c *Client) Users() (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.do("GET", "/users", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
func (c *Client) AddUser(name, role string, perms []string) error {
	body := map[string]interface{}{"name": name, "role": role}
	if perms != nil {
		body["permissions"] = perms
	}
	return c.do("POST", "/users/add", body, nil)
}

func (c *Client) DeleteUser(name string) error {
	return c.do("POST", "/users/delete", map[string]string{"name": name}, nil)
}
func (c *Client) UpdateUser(targetName, name, role string, perms []string) error {
	body := map[string]interface{}{"target_name": targetName}
	if name != "" {
		body["name"] = name
	}
	if role != "" {
		body["role"] = role
	}
	if perms != nil {
		body["permissions"] = perms
	}
	return c.do("POST", "/users/update", body, nil)
}

func (c *Client) Bootstrap(adminName string) error {
	return c.do("POST", "/bootstrap", map[string]string{"admin_name": adminName}, nil)
}

func (c *Client) CompleteFTPUpload(path string) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.do("POST", "/uploads/ftp/complete", map[string]string{"path": path}, &out); err != nil {
		return nil, err
	}
	return out, nil
}
