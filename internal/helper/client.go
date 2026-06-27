package helper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type Client interface {
	Identify() ([]Device, error)
	List(deviceUUID string) ([]File, error)
	Download(deviceUUID, handle, targetPath string) error
}

type SubprocessClient struct {
	helperPath string
}

func NewSubprocessClient(helperPath string) *SubprocessClient {
	return &SubprocessClient{helperPath: helperPath}
}

func (c *SubprocessClient) Identify() ([]Device, error) {
	var resp IdentifyResponse
	if err := c.runJSON("identify", &resp); err != nil {
		return nil, err
	}
	return resp.Devices, nil
}

func (c *SubprocessClient) List(deviceUUID string) ([]File, error) {
	var resp ListResponse
	args := []string{"list", "--device", deviceUUID}
	if err := c.runJSONArgs(args, &resp); err != nil {
		return nil, err
	}
	return resp.Files, nil
}

func (c *SubprocessClient) Download(deviceUUID, handle, targetPath string) error {
	cmd := exec.Command(c.helperPath, "download", "--device", deviceUUID, "--handle", handle, "--to", targetPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("download %s: %w (stderr: %s)", handle, err, stderr.String())
	}
	return nil
}

func (c *SubprocessClient) runJSON(subcmd string, target interface{}) error {
	return c.runJSONArgs([]string{subcmd}, target)
}

func (c *SubprocessClient) runJSONArgs(args []string, target interface{}) error {
	cmd := exec.Command(c.helperPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helper %v: %w (stderr: %s)", args, err, stderr.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), target); err != nil {
		return fmt.Errorf("parse helper output: %w (raw: %s)", err, stdout.String())
	}
	return nil
}
