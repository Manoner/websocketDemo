package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"net/url"
	"time"
)

var addr = flag.String("addr","127.0.0.1:12346","http service address")

func main() {
	fmt.Println("*****************1****")
	u := url.URL{Scheme: "ws",Host: *addr,Path: "/ws"}
	var dialer *websocket.Dialer

	conn,_,err:= dialer.Dial(u.String(),nil)
	if err != nil{
		fmt.Println(err)
		return
	}
	fmt.Println("*****************2****")
	go timeWriter(conn)
	fmt.Println("*****************3****")
	for{
		fmt.Println("*****************4****")
		_,message,err := conn.ReadMessage()
		fmt.Println("*****************5****")
		if err!=nil{
			fmt.Println("read",err)
			return
		}
		fmt.Printf("received:%s\n",message)
	}
}

func timeWriter(conn *websocket.Conn)  {
	for{
		time.Sleep(time.Second*2)
		conn.WriteMessage(websocket.TextMessage,[]byte(time.Now().Format("2006-01-02 15:04:05")))
	}

}