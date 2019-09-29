/**
 * @version: 1.0.0
 * @author: zhangguodong:general_zgd
 * @license: LGPL v3
 * @contact: general_zgd@163.com
 * @site: github.com/generalzgd
 * @software: GoLand
 * @file: main.go
 * @time: 2019/9/29 10:19
 */
package main

import (
	`context`
	`flag`
	`fmt`
	`net`
	`os`
	`os/signal`
	`runtime`
	`strings`
	`syscall`
	`time`

	`github.com/astaxie/beego/logs`
	`github.com/generalzgd/grpc-tcp-gateway-proto/goproto`
	consulapi `github.com/hashicorp/consul/api`
	`google.golang.org/grpc`
	`google.golang.org/grpc/reflection`
)

func init() {
	logger := logs.GetBeeLogger()
	logger.SetLevel(logs.LevelInfo)
	logger.SetLogger(logs.AdapterConsole)
	logger.SetLogger(logs.AdapterFile, `{"filename":"logs/file.log","level":7,"maxlines":1024000000,"maxsize":1024000000,"daily":true,"maxdays":7}`)
	logger.EnableFuncCallDepth(true)
	logger.SetLogFuncCallDepth(3)
	logger.Async(100000)
}

func exit(err error) {
	code := 0
	if err != nil {
		logs.Error("got error:%v", err)
	}
	logs.GetBeeLogger().Flush()
	os.Exit(code)
}

var (
	port = flag.Int("port", 8882, "backend listen port")
)

type BackendSvr struct {
}

func (p *BackendSvr) Method2(context.Context, *gwproto.Method2Request) (*gwproto.Method2Reply, error) {
	return &gwproto.Method2Reply{}, nil
}

func registConsul() (*consulapi.Client, *consulapi.AgentServiceRegistration, error) {
	ip := "127.0.0.1"
	svrName := "BackendSvr1"
	useType := "grpc"
	reg := &consulapi.AgentServiceRegistration{
		ID:      strings.ToLower(fmt.Sprintf("%s_%s_%d", svrName, useType, os.Getpid())),
		Name:    strings.ToLower(svrName),
		Tags:    []string{strings.ToLower(useType)},
		Port:    *port,
		Address: ip,
		Check: &consulapi.AgentServiceCheck{
			TCP:                            fmt.Sprintf(":%d", *port),
			Timeout:                        "1s",
			Interval:                       "15s",
			DeregisterCriticalServiceAfter: "30s",
			Status:                         "passing",
		},
	}

	cfg := consulapi.DefaultConfig()
	cfg.Address = "http://127.0.0.1:8500"
	client, err := consulapi.NewClient(cfg)
	if err != nil {
		return nil, nil, err
	}
	err = client.Agent().ServiceRegister(reg)
	if err != nil {
		return nil, nil, err
	}
	return client, reg, nil
}

func main() {
	var err error
	defer func() {
		exit(err)
	}()

	runtime.GOMAXPROCS(runtime.NumCPU())
	// do something

	s := grpc.NewServer(grpc.ConnectionTimeout(5 * time.Second))
	gwproto.RegisterBackendsvr2Server(s, &BackendSvr{})
	reflection.Register(s)

	var lis net.Listener
	lis, err = net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		return
	}

	go func() {
		if err := s.Serve(lis); err != nil {
			return
		}
	}()

	var client *consulapi.Client
	var reg *consulapi.AgentServiceRegistration
	client, reg, err = registConsul()
	if err != nil {
		return
	}

	logs.Info(reg.Name, "服务启动")
	// catchs system signal
	chSig := make(chan os.Signal)
	signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTERM)
	sig := <-chSig
	logs.Info("siginal:", sig)

	client.Agent().ServiceDeregister(reg.ID)
}
