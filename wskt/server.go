package wskt

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

const (
	BUFFER_SIZE = 1024
)

type ExServer interface {
	ReceiveChannel( buf *[]byte ) error
}

// Server maintains the set of active clients and broadcasts messages to the
// clients.
type Server struct {
	clientID int
	clients map[ int ]*Client

	// Inbound messages from the clients.
	broadcast chan []byte

	exS ExServer
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  BUFFER_SIZE,
	WriteBufferSize: BUFFER_SIZE,
}

func ( s *Server ) Run() {
	for {
		select {
			case buf := <-s.broadcast:
				log.Printf( "wskt.Server.Run buf:%v.", buf )
				if err := s.exS.ReceiveChannel( &buf ); err != nil {
					log.Printf( "wskt.Server.Run:wskt.ExServer.ReceiveChannel failure buf:%v.\n\t%v.", buf, err )
				}
		}
	}
}

func ( s *Server ) RegistClient( exC ExClient, w http.ResponseWriter, r *http.Request ) *Client {
	ws, err := upgrader.Upgrade( w, r, nil )
	if err != nil {
		log.Fatal(err)
		return nil
	}

	s.clientID++
	client := &Client{
		id: s.clientID,
		conn: ws,
		send: make( chan []byte, 256 ),
		exC: exC,
	}

	s.clients[ client.id ] = client

	go client.receiveChannel()
	go client.receiveWebSocket()

	return client
}

func ( s *Server ) SendServerBroadcast( wsBuf *WSBuf ) {
	// 途中ずっとポインタで受け渡しをして最後にバイト列で受信
	buf := wsBuf.GetSendBuf()
	s.broadcast <- *buf
}

func ( s *Server ) ClientLoop( check, send *WSBuf ) ( int, error ) {
	sum := 0
log.Printf( "wskt.Server.ClientLoop:s.clients:%+v.", s.clients )
	for i, c := range s.clients {
log.Printf( "wskt.Server.ClientLoop:c:%+v.", c )
log.Printf( "wskt.Server.ClientLoop:c.exC:%+v.", c.exC )
log.Printf( "wskt.Server.ClientLoop:check:%+v.", check )
		if val, err := c.exC.CheckSend( check ); err != nil {
			return 0, fmt.Errorf( "wskt.Server.ClientLoop:wskt.ExClient.CheckSend failure check:%v send:%v i:%d c:%v.\n\t%v", check, send, i, c, err )
		} else {
			if val > 0 && send != nil {
				buf := send.GetSendBuf()
				c.send <- *buf
			}
			sum += val
		}
	}
	return sum, nil
}

func ( s *Server ) RemoveClient( id int ) error {
	if c, ok := s.clients[ id ]; c != nil && ok {
		c.exC = nil
		delete( s.clients, id )
		c.conn.Close()
		close( c.send )
log.Printf( "wskt.Server.RemoveClient:id:%d c:%+v.", id, c )
	} else {
		return fmt.Errorf( "wskt.Server.RemoveClient:Can't get Client id:%d, s.clients:%+v.", id, s.clients )
	}
	return nil
}

func ( s *Server ) SendWSBCli( clientID int, buf *[]byte ) error {
	if c, ok := s.clients[ clientID ]; c != nil && ok {
		// 途中ずっとポインタで受け渡しをして最後にバイト列で送信
		send := buf
		c.send <- *send
	} else {
		return fmt.Errorf( "wskt.Server.SendWSBCli:Can't get Client clientID:%d, s.clients:%+v.", clientID, s.clients )
	}
	return nil
}

func StartServerProcess( exS ExServer, num int ) *Server {
	s := &Server{
		clientID:   0,
		clients:    make( map[ int ]*Client, num ),
		broadcast:  make( chan []byte ),
		exS:        exS,
	}
	go s.Run()
	return s
}

func ( s *Server ) PrintHTMLClientList() string {
	buf := ""
	for _, v := range s.clients {
		buf += v.GetHTMLText()
	}
	return buf
}
