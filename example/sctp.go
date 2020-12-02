package main

import (
	"crypto/rand"
	"encoding/binary"
	"flag"
	"log"
	"net"
	"strings"
	"time"

	"github.com/ishidawataru/sctp"
)

func serveClient(conn net.Conn, bufsize int) error {
	for {
		buf := make([]byte, bufsize+128) // add overhead of SCTPSndRcvInfoWrappedConn
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("read failed: %v", err)
			return err
		}

		log.Printf("header:%v", buf[:32])
		log.Printf("read: %d bytes:\n\tsinfo_stream %v\n\tsinfo_ssn %v\n"+
			"\tsinfo_flags %v\n\tsinfo_ppid %v\n\tsinfo_context %v\n\tttl %v\n"+
			"\tsinfo_tsn %v\n\tsinfo_cumtsn %v\n\tsinfo_assoc_id %v\nData:%s",
			n, binary.LittleEndian.Uint16(buf[0:]), binary.LittleEndian.Uint16(buf[2:]),
			binary.LittleEndian.Uint16(buf[4:]), binary.LittleEndian.Uint32(buf[8:]),
			binary.LittleEndian.Uint32(buf[12:]), binary.LittleEndian.Uint32(buf[16:]),
			binary.LittleEndian.Uint32(buf[20:]), binary.LittleEndian.Uint32(buf[24:]),
			binary.LittleEndian.Uint32(buf[28:]), string(buf[32:]))
		// n, err = conn.Write(buf[:n])
		// if err != nil {
		// 	log.Printf("write failed: %v", err)
		// 	return err
		// }
		// log.Printf("write: %d", n)
	}
}

func main() {
	var server = flag.Bool("server", false, "")
	var ip = flag.String("ip", "0.0.0.0", "")
	var port = flag.Int("port", 0, "")
	var lport = flag.Int("lport", 0, "")
	var bufsize = flag.Int("bufsize", 1024, "")
	var sndbuf = flag.Int("sndbuf", 0, "")
	var rcvbuf = flag.Int("rcvbuf", 0, "")

	flag.Parse()

	ips := []net.IPAddr{}

	for _, i := range strings.Split(*ip, ",") {
		if a, err := net.ResolveIPAddr("ip", i); err == nil {
			log.Printf("Resolved address '%s' to %s", i, a)
			ips = append(ips, *a)
		} else {
			log.Printf("Error resolving address '%s': %v", i, err)
		}
	}

	addr := &sctp.SCTPAddr{
		IPAddrs: ips,
		Port:    *port,
	}
	log.Printf("raw addr: %+v\n", addr.ToRawSockAddrBuf())

	if *server {
		ln, err := sctp.ListenSCTP("sctp", addr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		log.Printf("Listen on %s", ln.Addr())

		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Fatalf("failed to accept: %v", err)
			}
			log.Printf("Accepted Connection from RemoteAddr: %s", conn.RemoteAddr())
			wconn := sctp.NewSCTPSndRcvInfoWrappedConn(conn.(*sctp.SCTPConn))
			if *sndbuf != 0 {
				err = wconn.SetWriteBuffer(*sndbuf)
				if err != nil {
					log.Fatalf("failed to set write buf: %v", err)
				}
			}
			if *rcvbuf != 0 {
				err = wconn.SetReadBuffer(*rcvbuf)
				if err != nil {
					log.Fatalf("failed to set read buf: %v", err)
				}
			}
			*sndbuf, err = wconn.GetWriteBuffer()
			if err != nil {
				log.Fatalf("failed to get write buf: %v", err)
			}
			*rcvbuf, err = wconn.GetWriteBuffer()
			if err != nil {
				log.Fatalf("failed to get read buf: %v", err)
			}
			log.Printf("SndBufSize: %d, RcvBufSize: %d", *sndbuf, *rcvbuf)

			go serveClient(wconn, *bufsize)
		}

	} else {
		var laddr *sctp.SCTPAddr
		if *lport != 0 {
			laddr = &sctp.SCTPAddr{
				Port: *lport,
			}
		}
		conn, err := sctp.DialSCTP("sctp", laddr, addr)
		if err != nil {
			log.Fatalf("failed to dial: %v", err)
		}

		log.Printf("Dail LocalAddr: %s; RemoteAddr: %s", conn.LocalAddr(), conn.RemoteAddr())

		if *sndbuf != 0 {
			err = conn.SetWriteBuffer(*sndbuf)
			if err != nil {
				log.Fatalf("failed to set write buf: %v", err)
			}
		}
		if *rcvbuf != 0 {
			err = conn.SetReadBuffer(*rcvbuf)
			if err != nil {
				log.Fatalf("failed to set read buf: %v", err)
			}
		}

		*sndbuf, err = conn.GetWriteBuffer()
		if err != nil {
			log.Fatalf("failed to get write buf: %v", err)
		}
		*rcvbuf, err = conn.GetReadBuffer()
		if err != nil {
			log.Fatalf("failed to get read buf: %v", err)
		}
		log.Printf("SndBufSize: %d, RcvBufSize: %d", *sndbuf, *rcvbuf)

		ppid := 41
		info := &sctp.SndRcvInfo{
			Stream: uint16(0),
			PPID:   uint32(ppid),
		}
		for {
			// ppid += 1
			conn.SubscribeEvents(sctp.SCTP_EVENT_DATA_IO)
			buf := make([]byte, *bufsize)
			// buf := []byte("hello Baicells\nthis is intel.")
			n, err := rand.Read(buf)
			if n != *bufsize {
				log.Fatalf("failed to generate random string len: %d", *bufsize)
			}
			n, err = conn.SCTPWrite(buf, info)
			if err != nil {
				log.Fatalf("failed to write: %v", err)
			}
			log.Printf("write: len %d", n)
			// n, info, err = conn.SCTPRead(buf)
			// if err != nil {
			// 	log.Fatalf("failed to read: %v", err)
			// }
			// log.Printf("read: len %d, info: %+v", n, info)
			time.Sleep(time.Second * 5)
		}
	}
}
