package wskt

import (
	"log"
	"fmt"
	"time"
	"io"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 1024
)

type ExClient interface {
	ReceiveWebsocket( id int, wsbuf *WSBuf ) error
	CreateSendCliBuf( buf *[]byte ) ( *[]byte, error )

	CheckSend( check *WSBuf ) ( int, error )
	OnClose( id int )
	GetHTMLText() string
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	id int

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	exC ExClient
}

// The application runs receiveWebSocket in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func ( c *Client ) receiveWebSocket() {
	log.Printf("Start wskt.receiveWebSocket.")
	defer func() {
		log.Printf( "wskt.receiveWebSocket close id:%d.", c.id )
//		c.conn.Close()		// この2つはOnCloseの呼び出し先のRemoveClientでリストから外した後にcloseされる
//		close( c.send )
		c.exC.OnClose( c.id )
	}()
	c.conn.SetReadLimit( maxMessageSize )
	c.conn.SetReadDeadline( time.Now().Add( pongWait ) )
	c.conn.SetPongHandler( func( string ) error { c.conn.SetReadDeadline( time.Now().Add( pongWait ) ); return nil })
	for loop := true ; loop; {
		messageType, buf, err := c.conn.ReadMessage()
		log.Printf("wskt.receiveWebSocket c.conn.ReadMessage() id:%d MessageType:%d buf:%v.\n\t%+v", c.id, messageType, buf, err)
		if err != nil {
			if websocket.IsUnexpectedCloseError( err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure ) {
				log.Printf( "error: %v", err )
			}
			log.Printf( "wskt.receiveWebSocket c.Conn.ReadMessage() id:%d.\n\t%+v", c.id, err )
			loop = false	// ループから抜ける
		} else if messageType != websocket.BinaryMessage {
			log.Printf( "wskt.receiveWebSocket c.Conn.ReadMessage() Invalid MessageType:%d id:%d.", messageType, c.id )
			loop = false	// ループから抜ける
		} else {
			wsBuf := WSBuf{ index: 0, inc:0, buf: &buf }
			if err = c.exC.ReceiveWebsocket( c.id, &wsBuf ); err != nil {
				log.Printf( "wskt.receiveWebSocket.\n\t%+v", err )
//				loop = false	// ループから抜ける
			}
		}
	}
	log.Printf("End wskt.receiveWebSocket.")
}

// A goroutine running receiveChannel is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func ( c *Client ) receiveChannel() {
	ticker := time.NewTicker( pingPeriod )
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case buf, ok := <-c.send:
log.Printf( "wskt.client receiveChannel c.send buf:%+v.", buf )
			c.conn.SetWriteDeadline( time.Now().Add( writeWait ) )
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage( websocket.CloseMessage, []byte{} )
				return
			}

			w, err := c.conn.NextWriter( websocket.BinaryMessage )
			if err != nil {
				return
			}

			if err = c.sendClient( &w, &buf ); err != nil { log.Printf( "wskt.Client.receiveChannel failed.\n\t%v", err ); return }

			// Add queued chat messages to the current websocket message.
			n := len( c.send )
			for i := 0; i < n; i++ {
				buf = <-c.send
				if err = c.sendClient( &w, &buf ); err != nil { log.Printf( "wskt.Client.receiveChannel failed.\n\t%v", err ); return }
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
log.Printf("wskt.client receiveChannel ticker.C.")
			c.conn.SetWriteDeadline( time.Now().Add( writeWait ) )
			if err := c.conn.WriteMessage( websocket.PingMessage, nil ); err != nil {
				return
			}
		}
	}
}

func ( c *Client ) sendClient( w *io.WriteCloser, buf *[]byte ) error {
log.Printf( "wskt.Client.sendClient:buf:%+v.", buf )
	if c.exC != nil {	// Server.RemoveClientが呼ばれるとnilがセットされるので
		// 途中ずっとポインタで受け渡しをして最後にバイト列で送信
		if send, err := c.exC.CreateSendCliBuf( buf ); err == nil {
log.Printf( "wskt.Client.sendClient:size:%d send:%+v.", len( *send ), send )
			(*w).Write( *send )
		} else {
			return fmt.Errorf( "wskt.Client.sendClient:ExClient.CreateSendCliBuf failed.\n\t%v", err )
		}
	}
	return nil
}

func ( c *Client ) GetHTMLText() string {
	return fmt.Sprintf("<tr><td scope=\"row\">%d</td>%s</tr>", c.id, c.exC.GetHTMLText() )
}
