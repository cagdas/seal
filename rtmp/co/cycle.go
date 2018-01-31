package co

import (
	"UtilsTools/identify_panic"
	"log"
	"seal/kernel"
	"seal/rtmp/pt"
)

type AckWindowSizeS struct {
	AckWindowSize uint32
	HasAckedSize  uint64
}
type ConnectInfoS struct {
	TcUrl          string
	PageUrl        string
	SwfUrl         string
	ObjectEncoding float64
}

type RtmpConn struct {
	TcpConn         *kernel.TcpSock
	ChunkStreams    map[uint32]*pt.ChunkStream //key:cs id
	InChunkSize     uint32                     //default 128, set by peer
	OutChunkSize    uint32                     //default 128, set by config file.
	Pool            *kernel.MemPool
	AckWindow       AckWindowSizeS
	Requests        map[float64]string //key: transactin id, value:command name
	Role            uint8              //publisher or player.
	StreamName      string
	TokenStr        string        //token str for authentication. it's optional.
	Duration        float64       //for player.used to specified the stop when exceed the duration.
	DefaultStreamId float64       //default stream id for request.
	ConnectInfo     *ConnectInfoS //connect info.
}

func (rc *RtmpConn) Cycle() {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err, ",panic at ", identify_panic.IdentifyPanic())
		}
	}()

	var err error

	err = rc.HandShake()
	if err != nil {
		log.Println("rtmp handshake failed.err=", err)
		return
	}
	log.Println("rtmp handshake success.")

	for {
		var csid uint32
		err, csid = rc.RecvMsg()
		if err != nil {
			break
		}

		err = rc.OnRecvMsg(csid)
		if err != nil {
			break
		}

	}

	log.Println("rtmp cycle finished, begin clean.err=", err)

	rc.clean()

	log.Println("rtmp clean finished, remote=", rc.TcpConn.Conn.RemoteAddr())
}

func (rc *RtmpConn) clean() {

	log.Println("one publisher begin to quit, stream=", rc.StreamName)

	err := rc.TcpConn.Close()
	log.Println("close socket err=", err)

	if RtmpRoleFlashPublisher == rc.Role || RtmpRoleFMLEPublisher == rc.Role {
		rc.DeletePublishStream(rc.StreamName)
		log.Println("delete publisher stream=", rc.StreamName)
	}

	log.Println("publisher quit finish, stream=", rc.StreamName)
}