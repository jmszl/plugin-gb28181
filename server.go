package gb28181

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ghettovoice/gosip"
	"github.com/ghettovoice/gosip/log"
	"github.com/ghettovoice/gosip/sip"
	"go.uber.org/zap"
	. "m7s.live/engine/v4"
	"m7s.live/plugin/gb28181/v4/utils"
)

var srvUdp gosip.Server
var srvTcp gosip.Server

const MaxRegisterCount = 3

func FindChannel(deviceId string, channelId string) (c *Channel) {
	if v, ok := Devices.Load(deviceId); ok {
		d := v.(*Device)
		if v, ok := d.channelMap.Load(channelId); ok {
			return v.(*Channel)
		}
	}
	return
}

var levelMap = map[string]log.Level{
	"trace": log.TraceLevel,
	"debug": log.DebugLevel,
	"info":  log.InfoLevel,
	"warn":  log.WarnLevel,
	"error": log.ErrorLevel,
	"fatal": log.FatalLevel,
	"panic": log.PanicLevel,
}

func GetSipServer(transport string) gosip.Server {
	if strings.ToLower(transport) == "tcp" {
		return srvTcp
	} else {
		return srvUdp
	}
}

var sn = 0

// func CreateRequest(exposedId string, Method sip.RequestMethod, recipient *sip.Address, netAddr string) (req sip.Request) {
//
//		sn++
//
//		callId := sip.CallID(utils.RandNumString(10))
//		userAgent := sip.UserAgentHeader("Monibuca")
//		cseq := sip.CSeq{
//			SeqNo:      uint32(sn),
//			MethodName: Method,
//		}
//		port := sip.Port(conf.SipPort)
//		serverAddr := sip.Address{
//			//DisplayName: sip.String{Str: d.config.Serial},
//			Uri: &sip.SipUri{
//				FUser: sip.String{Str: exposedId},
//				FHost: conf.SipIP,
//				FPort: &port,
//			},
//			Params: sip.NewParams().Add("tag", sip.String{Str: utils.RandNumString(9)}),
//		}
//		req = sip.NewRequest(
//			"",
//			Method,
//			recipient.Uri,
//			"SIP/2.0",
//			[]sip.Header{
//				serverAddr.AsFromHeader(),
//				recipient.AsToHeader(),
//				&callId,
//				&userAgent,
//				&cseq,
//				serverAddr.AsContactHeader(),
//			},
//			"",
//			nil,
//		)
//
//		req.SetTransport(conf.SipNetwork)
//		req.SetDestination(netAddr)
//		//fmt.Printf("构建请求参数:%s", *&req)
//		// requestMsg.DestAdd, err2 = d.ResolveAddress(requestMsg)
//		// if err2 != nil {
//		// 	return nil
//		// }
//		//intranet ip , let's resolve it with public ip
//		// var deviceIp, deviceSourceIP net.IP
//		// switch addr := requestMsg.DestAdd.(type) {
//		// case *net.UDPAddr:
//		// 	deviceIp = addr.IP
//		// case *net.TCPAddr:
//		// 	deviceIp = addr.IP
//		// }
//
//		// switch addr2 := d.SourceAddr.(type) {
//		// case *net.UDPAddr:
//		// 	deviceSourceIP = addr2.IP
//		// case *net.TCPAddr:
//		// 	deviceSourceIP = addr2.IP
//		// }
//		// if deviceIp.IsPrivate() && !deviceSourceIP.IsPrivate() {
//		// 	requestMsg.DestAdd = d.SourceAddr
//		// }
//		return
//	}
func RequestForResponse(transport string, request sip.Request,
	options ...gosip.RequestWithContextOption) (sip.Response, error) {
	return (GetSipServer(transport)).RequestWithContext(context.Background(), request, options...)
}

func (c *GB28181Config) startServer() {
	addr := c.ListenAddr + ":" + strconv.Itoa(int(c.SipPort))

	logger := utils.NewZapLogger(GB28181Plugin.Logger, "GB SIP Server", nil)
	logger.SetLevel(uint32(levelMap[EngineConfig.LogLevel]))
	// logger := log.NewDefaultLogrusLogger().WithPrefix("GB SIP Server")
	srvConf := gosip.ServerConfig{}
	if c.SipIP != "" {
		srvConf.Host = c.SipIP
	}
	srvUdp = gosip.NewServer(srvConf, nil, nil, logger)
	_ = srvUdp.OnRequest(sip.REGISTER, c.OnRegister)
	_ = srvUdp.OnRequest(sip.MESSAGE, c.OnMessage)
	_ = srvUdp.OnRequest(sip.NOTIFY, c.OnNotify)
	_ = srvUdp.OnRequest(sip.BYE, c.OnBye)
	GB28181Plugin.Info(fmt.Sprint("c.SipNetwork = ", "UDP"))
	GB28181Plugin.Info(fmt.Sprint("c.MediaNetwork = ", c.MediaNetwork))
	err := srvUdp.Listen("udp", addr)
	if err != nil {
		GB28181Plugin.Logger.Error("gb28181 server listen", zap.Error(err))
	} else {
		GB28181Plugin.Info(fmt.Sprint("Server gb28181 start at", addr))
	}

	if c.MediaNetwork == "tcp" {
		c.tcpPorts.Init(c.MediaPortMin, c.MediaPortMax)
	} else {
		c.udpPorts.Init(c.MediaPortMin, c.MediaPortMax)
	}
	go c.startJob()
}

func (c *GB28181Config) startServerTCP() {

	addr := c.ListenAddr + ":" + strconv.Itoa(int(c.SipPort))

	logger := utils.NewZapLogger(GB28181Plugin.Logger, "GB SIP Server TCP", nil)
	logger.SetLevel(uint32(levelMap[EngineConfig.LogLevel]))
	// logger := log.NewDefaultLogrusLogger().WithPrefix("GB SIP Server")
	srvConf := gosip.ServerConfig{}
	if c.SipIP != "" {
		srvConf.Host = c.SipIP
	}
	srvTcp = gosip.NewServer(srvConf, nil, nil, logger)
	_ = srvTcp.OnRequest(sip.REGISTER, c.OnRegister)
	_ = srvTcp.OnRequest(sip.MESSAGE, c.OnMessage)
	_ = srvTcp.OnRequest(sip.NOTIFY, c.OnNotify)
	_ = srvTcp.OnRequest(sip.BYE, c.OnBye)
	GB28181Plugin.Info(fmt.Sprint("c.SipNetwork = ", "TCP"))
	GB28181Plugin.Info(fmt.Sprint("c.MediaNetwork = ", c.MediaNetwork))
	err := srvTcp.Listen("tcp", addr)
	if err != nil {
		GB28181Plugin.Logger.Error("gb28181 server listen", zap.Error(err))
	} else {
		GB28181Plugin.Info(fmt.Sprint("Server gb28181 start at", addr))
	}
}

// func queryCatalog(config *transaction.Config) {
// 	t := time.NewTicker(time.Duration(config.CatalogInterval) * time.Second)
// 	for range t.C {
// 		Devices.Range(func(key, value interface{}) bool {
// 			device := value.(*Device)
// 			if time.Since(device.UpdateTime) > time.Duration(config.RegisterValidity)*time.Second {
// 				Devices.Delete(key)
// 			} else if device.Channels != nil {
// 				go device.Catalog()
// 			}
// 			return true
// 		})
// 	}
// }

// 定时任务
func (c *GB28181Config) startJob() {
	statusTick := time.NewTicker(c.HeartbeatInterval / 2)
	banTick := time.NewTicker(c.RemoveBanInterval)
	linkTick := time.NewTicker(time.Millisecond * 100)
	GB28181Plugin.Debug("start job")
	for {
		select {
		case <-banTick.C:
			if c.Username != "" || c.Password != "" {
				c.removeBanDevice()
			}
		case <-statusTick.C:
			c.statusCheck()
		case <-linkTick.C:
			RecordQueryLink.cleanTimeout()
		}
	}
}

func (c *GB28181Config) removeBanDevice() {
	DeviceRegisterCount.Range(func(key, value interface{}) bool {
		if value.(int) > MaxRegisterCount {
			DeviceRegisterCount.Delete(key)
		}
		return true
	})
}

// statusCheck
// -  当设备超过 3 倍心跳时间未发送过心跳（通过 UpdateTime 判断）, 视为离线
// - 	当设备超过注册有效期内为发送过消息，则从设备列表中删除
// UpdateTime 在设备发送心跳之外的消息也会被更新，相对于 LastKeepaliveAt 更能体现出设备最会一次活跃的时间
func (c *GB28181Config) statusCheck() {
	Devices.Range(func(key, value any) bool {
		d := value.(*Device)
		if time.Since(d.UpdateTime) > c.RegisterValidity {
			Devices.Delete(key)
			GB28181Plugin.Info("Device register timeout",
				zap.String("id", d.ID),
				zap.Time("registerTime", d.RegisterTime),
				zap.Time("updateTime", d.UpdateTime),
			)
		} else if time.Since(d.UpdateTime) > c.HeartbeatInterval*3 {
			d.Status = DeviceOfflineStatus
			d.channelMap.Range(func(key, value any) bool {
				ch := value.(*Channel)
				ch.Status = ChannelOffStatus
				return true
			})
			GB28181Plugin.Info("Device offline", zap.String("id", d.ID), zap.Time("updateTime", d.UpdateTime))
		}
		return true
	})
}
