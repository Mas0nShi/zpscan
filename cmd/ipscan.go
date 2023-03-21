package cmd

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/niudaii/zpscan/pkg/pocscan"
	"github.com/niudaii/zpscan/pkg/webscan"
	"strings"

	"github.com/niudaii/zpscan/config"
	"github.com/niudaii/zpscan/internal/utils"
	"github.com/niudaii/zpscan/pkg/crack"
	"github.com/niudaii/zpscan/pkg/ipscan"
	"github.com/niudaii/zpscan/pkg/ipscan/portfinger"
	"github.com/niudaii/zpscan/pkg/ipscan/qqwry"

	"github.com/projectdiscovery/gologger"
	"github.com/spf13/cobra"
	"github.com/zu1k/nali/pkg/common"
)

type IpscanOptions struct {
	Proxy     string
	PortRange string
	Rate      int
	Threads   int
	Process   bool
	MaxPort   int

	Os      bool
	Addr    bool
	Crack   bool
	Pocscan bool
}

var ipscanOptions IpscanOptions

func init() {
	ipscanCmd.Flags().StringVarP(&ipscanOptions.PortRange, "port-range", "p", "1-65535", "port range(example: -p '22,80-90,1433,3306')")
	ipscanCmd.Flags().StringVar(&ipscanOptions.Proxy, "proxy", "", "socks5 proxy, (example: --proxy 192.168.31.227:47871')")
	ipscanCmd.Flags().IntVar(&ipscanOptions.Rate, "rate", 1500, "packets to send per second")
	ipscanCmd.Flags().IntVar(&ipscanOptions.Threads, "threads", 25, "number of threads")
	ipscanCmd.Flags().IntVar(&ipscanOptions.MaxPort, "max-port", 200, "discard result if it more than max port")
	ipscanCmd.Flags().BoolVar(&ipscanOptions.Process, "process", false, "show process")
	ipscanCmd.Flags().BoolVar(&ipscanOptions.Os, "os", false, "check os")
	ipscanCmd.Flags().BoolVar(&ipscanOptions.Addr, "addr", false, "get addr")

	ipscanCmd.Flags().BoolVar(&ipscanOptions.Crack, "crack", false, "open crack")
	ipscanCmd.Flags().BoolVar(&ipscanOptions.Pocscan, "pocscan", false, "open pocscan")

	rootCmd.AddCommand(ipscanCmd)
}

var ipscanCmd = &cobra.Command{
	Use:   "ipscan",
	Short: "端口扫描",
	Long:  "端口扫描,对结果进行webscan扫描, 可选crack,pocscan",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ipscanOptions.validateOptions(); err != nil {
			gologger.Fatal().Msgf("Program exiting: %v", err)
		}

		if err := initFinger(); err != nil {
			gologger.Error().Msgf("initFinger() err, %v", err)
		}

		if err := initQqwry(); err != nil {
			gologger.Fatal().Msgf("initQqwry() err, %v", err)
		}

		if err := initNmapProbe(); err != nil {
			gologger.Fatal().Msgf("initNmapProbe() err, %v", err)
		}

		if err := ipscanOptions.configureOptions(); err != nil {
			gologger.Fatal().Msgf("Program exiting: %v", err)
		}

		ipscanOptions.run()
	},
}

func (o *IpscanOptions) validateOptions() error {
	return nil
}

func (o *IpscanOptions) configureOptions() error {
	opt, _ := json.Marshal(o)
	gologger.Debug().Msgf("当前配置: %v", string(opt))

	return nil
}

func initQqwry() error {
	fileData, err := utils.ReadFile(config.Worker.Ipscan.QqwryFile)
	if err != nil {
		return err
	}
	var fileInfo common.FileData
	fileInfo.Data = fileData
	buf := fileInfo.Data[0:8]
	start := binary.LittleEndian.Uint32(buf[:4])
	end := binary.LittleEndian.Uint32(buf[4:])
	config.Worker.Ipscan.Qqwry = &qqwry.QQwry{
		IPDB: common.IPDB{
			Data:  &fileInfo,
			IPNum: (end-start)/7 + 1,
		},
	}
	return nil
}

func initNmapProbe() error {
	config.Worker.Ipscan.NmapProbe = &portfinger.NmapProbe{}
	nmapData, err := utils.ReadFile(config.Worker.Ipscan.NmapFile)
	if err != nil {
		return err
	}
	if err = config.Worker.Ipscan.NmapProbe.Init(nmapData); err != nil {
		return err
	}
	gologger.Info().Msgf("nmap指纹: %v个探针,%v条正则", len(config.Worker.Ipscan.NmapProbe.Probes), config.Worker.Ipscan.NmapProbe.Count())
	return nil
}

func (o *IpscanOptions) run() {
	var hosts []string
	for _, target := range targets {
		tmpHosts, err := ipscan.ParseIP(target)
		if err != nil {
			return
		}
		hosts = append(hosts, tmpHosts...)
	}
	options := &ipscan.Options{
		Hosts:     hosts,
		PortRange: o.PortRange,
		MaxPort:   o.MaxPort,
		Process:   o.Process,
		Rate:      o.Rate,
		Threads:   o.Threads,
		QQwry:     config.Worker.Ipscan.Qqwry,
		NmapProbe: config.Worker.Ipscan.NmapProbe,
		Proxy:     o.Proxy,
	}
	ipscanRunner, err := ipscan.NewRunner(options)
	if err != nil {
		gologger.Error().Msgf("ipscan.NewRunner() err, %v", err)
		return
	}
	var ipResults []*ipscan.Ip
	var servResults []*ipscan.Service
	for _, ip := range hosts {
		ipResult := &ipscan.Ip{
			Ip: ip,
		}
		// 获取地理位置
		if o.Addr {
			if ipResult.Country, ipResult.Area, err = ipscanRunner.GetAddr(ip); err != nil {
				gologger.Error().Msgf("ipscanRunner.GetAddr() err, %v", err)
				return
			}
		}
		// 操作系统识别
		if o.Os {
			if ipResult.OS, err = ipscan.CheckOS(ip); err != nil {
				gologger.Error().Msgf("ipscan.CheckOS() err, %v", err)
				return
			}
		}
		gologger.Info().Msgf("%v [%v %v] [%v]", ipResult.Ip, ipResult.Country, ipResult.Area, ipResult.OS)
		ipResults = append(ipResults, ipResult)
	}
	// 端口扫描
	portscanResults := ipscanRunner.Run()
	if len(portscanResults) == 0 {
		gologger.Info().Msgf("端口扫描结果为空")
		return
	}
	ipPortMap := make(map[string][]string)
	for _, result := range portscanResults {
		t := strings.Split(result.Addr, ":")
		ip := t[0]
		port := t[1]
		ipPortMap[ip] = append(ipPortMap[ip], port)
	}
	for _, ipResult := range ipResults {
		ipResult.Ports = strings.Join(ipPortMap[ipResult.Ip], ",")
	}
	// 结果处理
	var webTargets []string
	var crackTargets []string
	for _, portscanResult := range portscanResults {
		t := strings.Split(portscanResult.Addr, ":")
		ip := t[0]
		port := t[1]
		// unknown 服务也使用 webscan
		if portscanResult.ServiceName == "ssl" {
			if port == "443" {
				webTargets = append(webTargets, "https://"+ip)
			} else {
				webTargets = append(webTargets, "https://"+ip+":"+port)
			}
		} else if portscanResult.ServiceName == "http" {
			if port == "80" {
				webTargets = append(webTargets, "http://"+ip)
			} else {
				webTargets = append(webTargets, "http://"+ip+":"+port)
			}
		} else if portscanResult.ServiceName == "unknown" {
			webTargets = append(webTargets, ip+":"+port)
		} else {
			servResults = append(servResults, &ipscan.Service{
				Address:  portscanResult.Addr,
				Protocol: portscanResult.ServiceName,
				Version:  fmt.Sprintf("%v %v", portscanResult.VendorProduct, portscanResult.Version),
			})
			if crack.SupportProtocols[portscanResult.ServiceName] {
				crackTargets = append(crackTargets, portscanResult.Addr+"|"+portscanResult.ServiceName)
			}
		}
	}
	// 保存 ipscan 结果
	if commonOptions.ResultFile != "" {
		err = utils.SaveMarshal(commonOptions.ResultFile, ipResults)
		if err != nil {
			gologger.Error().Msgf("utils.SaveMarshal() err, %v", err)
		}
	}
	gologger.Info().Msgf("web: %v", len(webTargets))
	gologger.Info().Msgf("service: %v", len(servResults))
	gologger.Info().Msgf("crack: %v", len(crackTargets))
	// webscan
	options2 := &webscan.Options{
		Threads:     webscanOptions.Threads,
		Timeout:     webscanOptions.Timeout,
		Headers:     webscanOptions.Headers,
		NoColor:     commonOptions.NoColor,
		FingerRules: config.Worker.Webscan.FingerRules,
	}
	if ipscanOptions.Proxy != "" {
		options.Proxy = "socks5://" + ipscanOptions.Proxy
	}
	webscanRunner, err := webscan.NewRunner(options2)
	if err != nil {
		gologger.Error().Msgf("webscan.NewRunner() err, %v", err)
		return
	}
	webscanResults := webscanRunner.Run(webTargets)
	var pocscanTargets []string
	for _, webResult := range webscanResults {
		if len(webResult.Fingers) > 0 {
			var pocTags []string
			for _, finger := range webResult.Fingers {
				pocTags = append(pocTags, finger.PocTags...)
			}
			if len(pocTags) > 0 {
				pocscanTargets = append(pocscanTargets, webResult.Url+"|"+strings.Join(pocTags, ","))
			}
		}
	}
	// crack
	if o.Crack {
		options3 := &crack.Options{
			Threads:  crackOptions.Threads,
			Timeout:  crackOptions.Timeout,
			Delay:    crackOptions.Delay,
			CrackAll: crackOptions.CrackAll,
		}
		crackRunner, err := crack.NewRunner(options3)
		if err != nil {
			gologger.Error().Msgf("crack.NewRunner() err, %v", err)
			return
		}
		addrs := crack.ParseTargets(crackTargets)
		addrs = crack.FilterModule(addrs, crackOptions.Module)
		crackRunner.Run(addrs, []string{}, []string{})
	}
	if o.Pocscan {
		err = initPoc()
		if err != nil {
			gologger.Fatal().Msgf("initPoc() err, %v", err)
			return
		}
		options4 := &pocscan.Options{
			Timeout: pocscanOptions.Timeout,
			Headers: pocscanOptions.Headers,
		}
		if ipscanOptions.Proxy != "" {
			options4.Proxy = "socks5://" + ipscanOptions.Proxy
		}
		pocscanRunner, err := pocscan.NewRunner(options4, config.Worker.Pocscan.GobyPocs, config.Worker.Pocscan.XrayPocs, config.Worker.Pocscan.NucleiPocs)
		if err != nil {
			gologger.Error().Msgf("pocscan.NewRunner() err, %v", err)
			return
		}
		scanInputs, err := pocscan.ParsePocInput(pocscanTargets)
		if err != nil {
			gologger.Error().Msgf("pocscan.ParsePocInput() err, %v", err)
			return
		}
		// poc扫描
		pocscanRunner.RunPoc(scanInputs)
	}
}
