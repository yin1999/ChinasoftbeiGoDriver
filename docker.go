package main

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/moby/moby/client"
)

const (
	execDocker = "/usr/bin/docker"
)

var (
	imageMapping = map[fileType]string{
		python2: "registry-vpc.cn-shanghai.aliyuncs.com/yin199909/centos_7:origin",
		python3: "registry-vpc.cn-shanghai.aliyuncs.com/yin199909/centos_7:python3.6",
		golang:  "registry-vpc.cn-shanghai.aliyuncs.com/yin199909/centos_7:origin",
	}
	errNotSupport = errors.New("Path type not support")
)

type containerID string

// PathType 路径类型
type PathType byte

func newProcess(ctx context.Context, p programInfo, argv string, dbList []dbInfo) (string, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return "", err
	}
	cmd := []string{"sh", "-c"}
	switch p.file {
	case python2:
		cmd = append(cmd, "pip2 install -r requirements.txt && python2 main.py "+argv)
	case python3:
		cmd = append(cmd, "pip3 install -r requirements.txt && python3 main.py "+argv)
	case golang:
		cmd = append(cmd, "./main "+argv)
	}
	pb := nat.PortBinding{HostPort: "2105"}
	portNum, err := GetFreePort()
	if err != nil {
		return "", err
	}
	port := strconv.Itoa(portNum)
	exportPort := nat.Port(port + "/tcp")
	body, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        imageMapping[p.file],
		Cmd:          cmd,
		WorkingDir:   "/app",
		ExposedPorts: nat.PortSet{exportPort: struct{}{}},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{exportPort: []nat.PortBinding{pb}},
	}, nil, "")
	if err != nil {
		cli.Close()
		return "", err
	}
	if err = copyToContainer(ctx, cli, body.ID, "/app/", p.dir); err != nil {
		cli.Close()
		return "", err
	}
	if err = cli.ContainerStart(ctx, body.ID, types.ContainerStartOptions{}); err != nil {
		cli.Close()
		return "", err
	}
	addIDToMapping(body.ID, port)
	go containerListenAndServe(ctx, cli, body.ID)
	return body.ID, nil
}

func containerListenAndServe(ctx context.Context, cli *client.Client, containerID string) {
	returnCode, err := cli.ContainerWait(ctx, containerID)
	logger.Printf("Container: %s return %d.\n", containerID, returnCode)
	if err != nil {
		logger.Printf("Exit with error: %s.\n", err.Error())
	}
	cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{Force: true})
	dataRead(containerID)
	dbInfoRemove(containerID)
	cli.Close()
}

func copyToContainer(ctx context.Context, cli *client.Client, containerID, dst, src string) error {
	switch pathStat(src) {
	case notExist:
		return os.ErrNotExist
	case directory:
		if dst[len(dst)-1] != '/' {
			dst += "/"
		}
		if src[len(src)-1] != '/' {
			src += "/"
		}
		files, err := ioutil.ReadDir(src)
		if err != nil {
			return err
		}
		for i := range files {
			fileName := files[i].Name()
			err = copyToContainer(ctx, cli, containerID, dst+fileName, src+fileName)
			if err != nil {
				return err
			}
		}
		return err
	case file:
		f, err := os.Open(src)
		if err != nil {
			return err
		}
		err = cli.CopyToContainer(ctx, containerID, dst, f, types.CopyToContainerOptions{AllowOverwriteDirWithFile: true})
		f.Close()
		return err
	default:
		return errNotSupport
	}
}

const (
	notExist PathType = iota
	directory
	file
)

// pathStat 判断所给路径类型
// return: directory: 文件夹, file: 文件, notExist: 不存在
func pathStat(path string) PathType {
	s, err := os.Stat(path)
	if err != nil {
		return notExist
	}
	if s.IsDir() {
		return directory
	}
	return file
}