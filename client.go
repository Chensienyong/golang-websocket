package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	idleTimeout    = 1 * time.Minute
	maxMessageSize = 1024
	maxPending     = 1024
	host           = "localhost"
	port           = "8099"
)

var dialer = &websocket.Dialer{
	ReadBufferSize:  maxMessageSize,
	WriteBufferSize: 16,
}

var (
	connected int64
	pending   int64
	failed    int64
)

func main() {
	go func() {
		start := time.Now()
		for {
			fmt.Printf("client elapsed=%0.0fs pending=%d connected=%d failed=%d\n", time.Now().Sub(start).Seconds(), atomic.LoadInt64(&pending), atomic.LoadInt64(&connected), atomic.LoadInt64(&failed))
			time.Sleep(1 * time.Second)
		}
	}()

	desired, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	i := 1
	for {
		if atomic.LoadInt64(&connected)+atomic.LoadInt64(&pending) < desired && atomic.LoadInt64(&pending) < maxPending {
			atomic.AddInt64(&pending, 1)
			url := fmt.Sprintf("ws://%s:%s/socket/websocket", host, port)

			user_id := strconv.Itoa(i)
			token := getToken(user_id)
			go createConnection(url, user_id, token)
			i++
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func getToken(user_id string) string {
	hours_since_unix_epoch := int(time.Now().Unix() / 3600)
	days_since_unix_epoch := int(hours_since_unix_epoch / 24)
	return GetMD5Hash(user_id + "undefined" + strconv.Itoa(days_since_unix_epoch))
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func createConnection(url string, user_id string, token string) {
	//fmt.Println("user: ", user_id)
	ws, _, err := dialer.Dial(url, nil)
	if err != nil {
		atomic.AddInt64(&pending, -1)
		atomic.AddInt64(&failed, 1)
		return
	}

	atomic.AddInt64(&pending, -1)
	atomic.AddInt64(&connected, 1)

	ws.SetReadLimit(maxMessageSize)

	message := []byte("{\"topic\":\"room:25\", \"event\":\"phx_join\", \"payload\": {\"user_id\":\"" + user_id + "\", \"token\": \"" + token + "\", \"username\": \"bambang\", \"name\": \"Bambang\", \"avatar\": \"abc\"}, \"ref\": \"live_shopping\"}")
	ws.SetWriteDeadline(time.Now().Add(30 * time.Second))
	err = ws.WriteMessage(websocket.TextMessage, message)

	i, _ := strconv.Atoi(user_id)
	if(i%2==0) {
		message = []byte("{\"topic\":\"room:25\", \"event\":\"post\", \"payload\": {\"message\":\"tes gagal dari user " + user_id + "\"}, \"ref\": \"1\"}")
		ws.SetWriteDeadline(time.Now().Add(30 * time.Second))
		err = ws.WriteMessage(websocket.TextMessage, message)
	} else {
		message = []byte("{\"topic\":\"room:25\", \"event\":\"post\", \"payload\": {\"message\":\"tes dari user " + user_id + "\"}, \"ref\": \"1\"}")
		ws.SetWriteDeadline(time.Now().Add(30 * time.Second))
		err = ws.WriteMessage(websocket.TextMessage, message)
	}

	for {
		ws.SetReadDeadline(time.Now().Add(idleTimeout))
		_, message, err := ws.ReadMessage()
		if err != nil {
			fmt.Println("received err", err)
			break
		}
		if len(message) > 0 {
			fmt.Println("received message", string(message))
		}
	}

	atomic.AddInt64(&connected, -1)
	atomic.AddInt64(&failed, 1)

	ws.Close()
}
