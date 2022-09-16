package plugins

import (
	"strconv"
	"strings"

	"github.com/niudaii/zpscan/pkg/crack/plugins/wmiexec"
)

func WmiHashCrack(serv *Service) (int, error) {
	err := wmiexec.WMIExec(serv.Ip+":"+strconv.Itoa(serv.Port), serv.User, "", serv.Pass, "", "", "", serv.Timeout, nil)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			return CrackError, err
		}
		return CrackFail, nil
	}
	return CrackSuccess, nil
}
