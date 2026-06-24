package di

import (
	"fmt"
	"sync"
)

type Container struct {
	mu       sync.RWMutex
	services map[string]any
}

func NewContainer() *Container {
	return &Container{
		services: make(map[string]any),
	}
}

func (c *Container) Register(name string, svc any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services[name] = svc
}

func (c *Container) Resolve(name string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.services[name]
}

func (c *Container) MustResolve(name string) any {
	svc := c.Resolve(name)
	if svc == nil {
		panic(fmt.Sprintf("service %s not found in container", name))
	}
	return svc
}
