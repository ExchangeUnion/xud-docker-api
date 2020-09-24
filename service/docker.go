package service

import (
	docker "github.com/docker/docker/client"
)

type DockerClientFactory interface {
	GetSharedInstance() *docker.Client
	NewInstance() (*docker.Client, error)
}

type ClientFactory struct {
	shared *docker.Client
}

func NewClientFactory() (*ClientFactory, error) {
	client, err := createClient()
	if err != nil {
		return nil, err
	}
	return &ClientFactory{
		shared: client,
	}, nil
}

func createClient() (*docker.Client, error) {
	client, err := docker.NewEnvClient()
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (t *ClientFactory) GetSharedInstance() *docker.Client {
	return t.shared
}

func (t *ClientFactory) NewInstance() (*docker.Client, error) {
	client, err := createClient()
	if err != nil {
		return nil, err
	}
	return client, nil
}
