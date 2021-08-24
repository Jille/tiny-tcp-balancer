package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

var (
	port    = flag.Int("port", 6442, "port to listen on")
	targets = flag.String("targets", "", "Comma separated host:port targets")

	d = net.Dialer{
		KeepAlive: 15 * time.Second,
	}

	splitTargets []string
)

func main() {
	flag.Parse()
	if *targets == "" {
		log.Fatal("Flag --targets is mandatory")
	}
	splitTargets = strings.Split(*targets, ",")
	sock, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("net.Listen(): %v", err)
	}
	ctx := context.Background()
	for {
		conn, err := sock.Accept()
		if err != nil {
			log.Fatalf("Accept(): %v", err)
		}
		go handleConn(ctx, conn.(*net.TCPConn))
	}
}

func dialTargets(ctx context.Context) *net.TCPConn {
	ch := make(chan *net.TCPConn, len(splitTargets))
	g, ctx := errgroup.WithContext(ctx)
	for _, t := range splitTargets {
		t := t
		g.Go(func() error {
			r, err := d.DialContext(ctx, "tcp", t)
			if err != nil {
				return err
			}
			ch <- r.(*net.TCPConn)
			return nil
		})
	}
	go func() {
		g.Wait()
		close(ch)
	}()
	ret := <-ch
	go func() {
		for c := range ch {
			c.Close()
		}
	}()
	return ret
}

func handleConn(ctx context.Context, c *net.TCPConn) {
	defer c.Close()
	t := dialTargets(ctx)
	if t == nil {
		return
	}
	defer t.Close()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		io.Copy(c, t)
		c.CloseWrite()
		t.CloseRead()
		wg.Done()
	}()
	go func() {
		io.Copy(t, c)
		t.CloseWrite()
		c.CloseRead()
		wg.Done()
	}()
	wg.Wait()
}
