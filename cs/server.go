package cs

import(
	codecs "sanguo/codec/cs"
	_ "sanguo/protocol/cs" //触发pb注册
	cs_proto "sanguo/protocol/cs/message"
	"github.com/sniperHW/kendynet"
	"github.com/sniperHW/kendynet/socket/stream_socket/tcp"
	"sync/atomic"
	"fmt"
	"time"
	"sanguo/common"
)

var (
	server     listener
	started    int32
	dispatcher ServerDispatcher
)

type listener interface {
	Close()
	Start() error
}

type tcpListener struct {
	l *tcp.Listener
}

func newTcpListener(nettype,service string) (*tcpListener,error) {
	var err error
	l := &tcpListener{}
	l.l,err = tcp.NewListener(nettype,service)

	if nil == err {
		return l,nil
	} else {
		return nil,err
	}
}

func (this *tcpListener) Close() {
	this.l.Close()
}

func (this *tcpListener) Start() error {
	if nil == this.l {
		return fmt.Errorf("invaild listener")
	} 
	return this.l.Start(func(session kendynet.StreamSession) {
		session.SetRecvTimeout(common.HeartBeat_Timeout * time.Second)
		session.SetReceiver(codecs.NewReceiver("cs"))
		session.SetEncoder(codecs.NewEncoder("sc"))
		session.SetCloseCallBack(func (sess kendynet.StreamSession, reason string) {
			dispatcher.OnClose(sess,reason)
		})

		dispatcher.OnNewClient(session)

		session.Start(func (event *kendynet.Event) {
			if event.EventType == kendynet.EventTypeError {
				event.Session.Close(event.Data.(error).Error(),0)
			} else {
				msg := event.Data.(*codecs.Message)
				switch msg.GetData().(type) {
        			case *cs_proto.HeartbeatToS:
        				fmt.Printf("on HeartbeatToS\n")
        				Heartbeat := &cs_proto.HeartbeatToC{}
						session.Send(codecs.NewMessage(0,Heartbeat))
        				break	
        			default:
						dispatcher.Dispatch(session,msg)	
						break
				}			
			}
		})
	})
}


func StartTcpServer(nettype ,service string,d ServerDispatcher) error {
	l,err := newTcpListener(nettype,service)
	if nil != err {
		return err
	}
	return startServer(l,d)
}

func startServer(l listener,d ServerDispatcher) error {
	if !atomic.CompareAndSwapInt32(&started,0,1) {
		return fmt.Errorf("server already started")
	}

	server = l
	dispatcher = d

	go func() {
		err := server.Start()
		if nil != err {
			kendynet.Errorf("server.Start() error:%s\n",err.Error())
		}
	}()

	return nil
}

func StopServer() {
	if !atomic.CompareAndSwapInt32(&started,1,0) {
		return
	}
	server.Close()
} 