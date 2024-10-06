package socketioclient

type ClientManager struct {
	clients map[string]*Client
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]*Client),
	}
}

func (c *ClientManager) AddClient(client *Client) {
	c.clients[client.Namespace.Namespace] = client
}

func (c *ClientManager) RemoveClient(client *Client) {
	delete(c.clients, client.Namespace.Namespace)
}
