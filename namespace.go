package socketioclient

import (
	"sync"
)

type Namespace struct {
	Namespace string
	Sid       string
	isInit    bool
	mu        sync.Mutex
}

func (n *Namespace) IsInit() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.isInit
}

func (n *Namespace) SetInit(init bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.isInit = init
}

func (n *Namespace) GetNamespace() string {
	return n.Namespace
}
