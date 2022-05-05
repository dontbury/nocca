package wskt

import (
	"log"
	"fmt"
)

const (
	BYTE1_SIZE = 1
	BYTE2_SIZE = 2
	BYTE3_SIZE = 3
	BYTE4_SIZE = 4
	BYTE8_SIZE = 8

	STR_SIZE_LENGTH = BYTE2_SIZE	// 文字列の長さを収めるためのバイト数
	CHAR_SIZE = BYTE2_SIZE			// 1文字あたりのバイト数

	WSKTBUF_INCREMENT = 1024
)

type WSBuf struct {
	index	int
	inc		int	// バッファが足らなくなった時の増加分
	buf		*[]byte
}

func ( s *WSBuf ) Create( index, size int ) {
	s.index = index
	s.inc = WSKTBUF_INCREMENT
	buf := make( []byte, size )
	s.buf = &buf
}

func ( s *WSBuf ) CreateBuf( index int, buf *[]byte ) {
	s.index = index
	s.inc = WSKTBUF_INCREMENT
	s.buf = buf
}

func ( s *WSBuf ) CreateWSB( index, headersize int, wsb *WSBuf ) {
	s.index = index
	s.inc = WSKTBUF_INCREMENT
	size := headersize + len( *wsb.buf ) - wsb.index
	buf := make( []byte, size )
	for i := headersize; i < size; i++ {
		buf [ i ] = (*wsb.buf)[ wsb.index + i - headersize ]
	}
	s.buf = &buf
}

func ( s *WSBuf ) GetIndex() int {
	return s.index
}

func ( s *WSBuf ) SetIndexHead() {
	s.index = 0
}

func ( s *WSBuf ) SetIndexTail() {
	s.index = len( *(s.buf) )
}

// バッファをそのまま取得
func ( s *WSBuf ) GetRawBuf() *[]byte {
	return s.buf
}

// 送信用に後ろの未使用部分を切り詰めたバッファを取得
func ( s *WSBuf ) GetSendBuf() *[]byte {
	if s.index < len( *s.buf ) {
		buf := *s.buf
		buf = buf[ : s.index ]
		log.Printf( "WSBuf.GetSendBuf: index:%d size:%d %v -> %v.", s.index, len( *s.buf ), s.buf, buf )
		return &buf
	} else {
		return s.buf
	}
}

func ( s *WSBuf ) Append1Byte( val int ) error {
	if val >= 0x100 {
		return fmt.Errorf( "WSBuf.Append1Byte:val too large:%d.", val )
	}
	if s.index >= len( *s.buf ) {
		if s.inc > 0 {
			stk := *s.buf
			buf := make( []byte, len( *s.buf ) + s.inc )
			s.buf = &buf
			log.Printf( "[INFO] WSData.Append1Byte:Extend buf %d -> %d.", len( buf ), len( *s.buf ) )
			for i, v := range stk {
				(*s.buf)[ i ] = v
			}
		} else {
			return fmt.Errorf( "WSData.Append1Byte:Overflow size:%d index:%d inc:%d value:%v.", len( *s.buf ), s.index, s.inc, s.buf )
		}
	}
	(*s.buf)[ s.index ] = byte(val)
	s.index++
	return nil
}

func ( s *WSBuf ) Append2Bytes( val int ) error {
	if err := s.Append1Byte( val / 0x100 ); err != nil { return fmt.Errorf( "WSData.Append2Bytes:Append1Byte 1 failure.\t\n%+v", err ) }
	if err := s.Append1Byte( val % 0x100 ); err != nil { return fmt.Errorf( "WSData.Append2Bytes:Append1Byte 2 failure.\t\n%+v", err ) }
	return nil
};
func ( s *WSBuf ) Append3Bytes( val int ) error {
	if err := s.Append1Byte( val / 0x10000 ); err != nil { return fmt.Errorf( "WSData.Append3Bytes:Append1Byte failure.\t\n%+v", err ) }
	if err := s.Append2Bytes( val % 0x10000 ); err != nil { return fmt.Errorf( "WSData.Append3Bytes:Append2Bytes failure.\t\n%+v", err ) }
	return nil
}

func ( s *WSBuf ) Append4Bytes( val int ) error {
	if err := s.Append2Bytes( val / 0x10000 ); err != nil { return fmt.Errorf( "WSData.Append4Bytes:Append2Bytes 1 failure.\t\n%+v", err ) }
	if err := s.Append2Bytes( val % 0x10000 ); err != nil { return fmt.Errorf( "WSData.Append4Bytes:Append2Bytes 2 failure.\t\n%+v", err ) }
	return nil
}

func ( s *WSBuf ) Append8Bytes( val int64 ) error {
	if err := s.Append4Bytes( (int)( val / 0x100000000 ) ); err != nil { return fmt.Errorf( "WSData.Append8Bytes:Append4Bytes 1 failure.\t\n%+v", err ) }
	if err := s.Append4Bytes( (int)( val % 0x100000000 ) ); err != nil { return fmt.Errorf( "WSData.Append8Bytes:Append4Bytes 2 failure.\t\n%+v", err ) }
	return nil
}

func ( s *WSBuf ) AppendString( val string ) error {
    sz := len( []rune( val ) )
	if sz >= 0x8000 {		// 2倍して2バイトに収まる必要がある
		return fmt.Errorf( "WSData.AppendString:val too long:%d.", sz )
	} else {
		size := len( *s.buf )
		if s.index + STR_SIZE_LENGTH + sz * CHAR_SIZE > size {
			if s.inc > 0 {
				inc := s.inc
				// 増加分より文字列追加分の方が大きい場合は、格納できる容量を確保する
				if s.index + STR_SIZE_LENGTH + sz * CHAR_SIZE - size > inc {
					inc = s.index + STR_SIZE_LENGTH + sz * CHAR_SIZE - size
				}
				stk := *s.buf
				buf := make( []byte, len( *s.buf ) + inc )
				s.buf = &buf
				log.Printf( "[INFO] WSData.AppendString:Extend buf %d -> %d.", size, len( *s.buf ) )
				for i, v := range stk {
					(*s.buf)[ i ] = v
				}
			} else {
				return fmt.Errorf( "WSData.AppendString:Buffer overflow index:%d size:%d index:%d inc:%d value:%v.", s.index, size, s.index, s.inc, s.buf )
			}
		}
		s.Append2Bytes( sz )
		r := []rune( val )
		for _, v := range r {
			s.Append2Bytes( int(v) )
		}
	}
	return nil
}

func ( s *WSBuf ) Get1Byte() ( int, error ) {
	val := 0
	if s.index < len( *s.buf ) {
		val = (int)( (*s.buf)[ s.index ] )
		s.index++
	} else {
		return 0, fmt.Errorf( "WSBuf.Get1Byte:Invalid index:%d len():%d buf:%v", s.index, len( *s.buf ), s.buf )
	}
	return val, nil
}

func ( s *WSBuf ) Get2Bytes() ( int, error ) {
	var err error
	var v1 int
	if v1, err = s.Get1Byte(); err != nil { return 0, fmt.Errorf( "WSBuf.Get2Bytes:Can't get upper byte.\n\t%+v", err ) }
	var v2 int
	if v2, err = s.Get1Byte(); err != nil { return 0, fmt.Errorf( "WSBuf.Get2Bytes:Can't get lower byte.\n\t%+v", err ) }
	return v1 * 0x100 + v2, nil
}

func ( s *WSBuf ) Geta3Bytes() ( int, error ) {
	var err error
	var v1 int
	if v1, err = s.Get1Byte(); err != nil { return 0, fmt.Errorf( "WSBuf.Geta3Bytes:Can't get upper byte.\n\t%+v", err ) }
	var v2 int
	if v2, err = s.Get2Bytes(); err != nil { return 0, fmt.Errorf( "WSBuf.Geta3Bytes:Can't get lower bytes.\n\t%+v", err ) }
	return v1 * 0x10000 + v2, nil
}

func ( s *WSBuf ) Get4Bytes() ( int, error ) {
	var err error
	var v1 int
	if v1, err = s.Get2Bytes(); err != nil { return 0, fmt.Errorf( "WSBuf.Get4Bytes:Can't get upper bytes.\n\t%+v", err ) }
	var v2 int
	if v2, err = s.Get2Bytes(); err != nil { return 0, fmt.Errorf( "WSBuf.Get4Bytes:Can't get lower bytes.\n\t%+v", err ) }
	return v1 * 0x10000 + v2, nil
}

func ( s *WSBuf ) Get8Bytes() ( int64, error ) {
	var err error
	var v1 int
	if v1, err = s.Get4Bytes(); err != nil { return 0, fmt.Errorf( "WSBuf.Get8Bytes:Can't get upper bytes.\n\t%+v", err ) }
	var v2 int
	if v2, err = s.Get4Bytes(); err != nil { return 0, fmt.Errorf( "WSBuf.Get8Bytes:Can't get lower bytes.\n\t%+v", err ) }
	return (int64)( v1 * 0x100000000 + v2 ), nil
}

func ( s *WSBuf ) GetString() ( string, error ) {
	var err error
	var sz int
	if sz, err = s.Get2Bytes(); err != nil { return "", fmt.Errorf( "WSBuf.GetString:Can't get size.\n\t%+v", err ) }
	var v int
	r := make( []rune, sz )
	for i := 0; sz > i; i++ {
		if v, err = s.Get2Bytes(); err != nil { return "", fmt.Errorf( "WSBuf.GetString:Can't get rune index:%d.\n\t%+v", i, err ) }
		r[ i ] = rune( v )
	}
	return string( r ), nil
}

func CalcStrSize( str string ) int {
	return len( []rune( str ) ) * CHAR_SIZE + STR_SIZE_LENGTH
}

func ( s *WSBuf ) CheckContinue() bool {
	return s.index < len( *s.buf )
}
