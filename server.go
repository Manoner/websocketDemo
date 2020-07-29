package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
	"net/http"
	"runtime"
)

type Message struct {
	Sender    string `json:"sender,omitempty"`
	Recipient string `json:"recipient,omitempty"`
	Content   string `json:"content,omitempty"`
}

// 抽象的ws连接器（在后端不停的处理）
// 处理ws中的各种逻辑
type ClientManager struct {
	clients    map[*Client]bool // clients 注册了连接器
	broadcast  chan []byte      // 从连接器发送的信息
	register   chan *Client     // 从连接器注册请求
	unregister chan *Client     // 从连接器销毁请求
}

// 连接器初始化
var manager = ClientManager{
	broadcast:  make(chan []byte),
	register:   make(chan *Client),
	unregister: make(chan *Client),
	clients:    make(map[*Client]bool),
}

func (manager *ClientManager) start() {
	for {
		select {
		case conn := <-manager.register:
			manager.clients[conn] = true
			jsonMessage, _ := json.Marshal(&Message{Content: "/A new socket has connected."})
			manager.send(jsonMessage, conn)

		case conn := <-manager.unregister:
			if _, ok := manager.clients[conn]; ok {
				close(conn.send)
				delete(manager.clients, conn)
				jsonMessage, _ := json.Marshal(&Message{Content: "/A socket has disconnected."})
				manager.send(jsonMessage, conn)
			}
		case message := <-manager.broadcast:
			for conn := range manager.clients {
				select {
				case conn.send <- message:
				default:
					close(conn.send)
					delete(manager.clients, conn)
				}
			}
		}
	}
}

func (manager *ClientManager) send(message []byte, ignore *Client) {
	//fmt.Println("send message=", string(message), "ignore=", *ignore)
	//fmt.Println("send manager.clients=", manager.clients)
	for conn := range manager.clients {
		//fmt.Println("send conn = ", conn)
		if conn != ignore {
			conn.send <- message
		}
	}
}

type Client struct {
	id string
	// ws 连接器
	socket *websocket.Conn
	// 管道 发送数据，数据为切片类型
	send chan []byte
}

// ws 连接中读数据
func (c *Client) read() {
	defer func() {
		manager.unregister <- c
		c.socket.Close()
	}()

	for {
		_, message, err := c.socket.ReadMessage()
		// 如果 该连接 读取失败，说明可能该连接已经断开
		if err != nil {
			// 把该链接放到释放的chan中等待处理
			manager.unregister <- c
			c.socket.Close()
			break
		}
		//fmt.Printf("read message = %#v, id = %s \n", message, c.id)
		//fmt.Println("read message = ", string(message), "c.id=", c.id)
		jsonMessage, _ := json.Marshal(&Message{Sender: c.id, Content: string(message)})
		manager.broadcast <- jsonMessage

	}
}

// ws 连接中写数据
func (c *Client) write() {
	defer func() {
		c.socket.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			//fmt.Println("write data = ", message)
			if !ok {
				c.socket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.socket.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func main() {
	fmt.Println("Starting application...")
	go func() {
		for {
			fmt.Println("------协程数量=", runtime.NumGoroutine())
		}
	}()
	go manager.start()
	http.HandleFunc("/ws", wsPage)
	fmt.Println("Starting application...1111111")
	http.ListenAndServe(":12346", nil)
	fmt.Println("Starting application...2222222")
}

func wsPage(res http.ResponseWriter, req *http.Request) {
	fmt.Println("*********** wsPage **************")
	conn, error := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
		return true
	}}).Upgrade(res, req, nil)
	if error != nil {
		http.NotFound(res, req)
		return
	}
	client := &Client{id: uuid.NewV4().String(), socket: conn, send: make(chan []byte)}
	fmt.Println("*********** client =", client)

	manager.register <- client

	go client.read()
	go client.write()

}
