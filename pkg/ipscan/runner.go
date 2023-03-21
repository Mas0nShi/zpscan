package ipscan

import (
	"github.com/niudaii/zpscan/pkg/ipscan/ipweb"
	"github.com/niudaii/zpscan/pkg/ipscan/naabu"
	"github.com/niudaii/zpscan/pkg/ipscan/portfinger"
	"github.com/niudaii/zpscan/pkg/ipscan/qqwry"
	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/naabu/v2/pkg/runner"
	"time"
)

type Options struct {
	Hosts     []string
	PortRange string
	Rate      int
	Threads   int
	Proxy     string
	MaxPort   int
	Process   bool
	QQwry     *qqwry.QQwry
	NmapProbe *portfinger.NmapProbe
}

type Runner struct {
	options          *Options
	naabuRunner      *runner.Runner
	portFingerEngine *portfinger.Engine
}

func NewRunner(options *Options) (*Runner, error) {
	naabuRunner, err := naabu.NewRunner(options.Hosts, options.PortRange, options.Rate, options.Threads, options.Proxy, options.Process)
	if err != nil {
		return nil, err
	}
	portFingerEngine, err := portfinger.NewEngine(options.Proxy, options.NmapProbe)
	if err != nil {
		return nil, err
	}
	return &Runner{
		options:          options,
		naabuRunner:      naabuRunner,
		portFingerEngine: portFingerEngine,
	}, nil
}

type Ip struct {
	Ip      string
	Ports   string
	Country string
	Area    string
	OS      string
}

type Service struct {
	Address  string
	Protocol string
	Version  string
}

func (r *Runner) Run() (results []*portfinger.Result) {
	start := time.Now()
	gologger.Info().Msgf("开始端口扫描")
	defer r.naabuRunner.Close()
	err := r.naabuRunner.RunEnumeration()
	if err != nil {
		gologger.Error().Msgf("naabuRunner.RunEnumeration() err, %v", err)
		return
	}
	gologger.Info().Msgf("开放端口的host: %v", len(naabu.Results))
	if len(naabu.Results) == 0 {
		return
	}
	// 去除开放端口数大于maxPort
	naabuResults := naabu.Results
	naabu.Results = make(naabu.Result)
	for k := range naabuResults {
		if len(naabuResults[k]) > r.options.MaxPort {
			gologger.Info().Msgf("%v 开放端口大于 %v", k, r.options.MaxPort)
			// 如果端口开放大于 max-port，进行 web 探活
			naabuResults[k] = ipweb.Run(k, 1, 50, "")
		}
	}
	gologger.Info().Msgf("端口开放扫描完成: %v", time.Since(start))
	// 开放的端口使用nmap指纹识别
	gologger.Info().Msgf("端口协议识别")
	results = r.portFingerEngine.Run(naabuResults)
	gologger.Info().Msgf("端口协议扫描完成: %v", time.Since(start))
	return
}
