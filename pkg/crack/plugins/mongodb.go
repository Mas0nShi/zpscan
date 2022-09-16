package plugins

import (
	"fmt"
	"net"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
)

func MongodbCrack(serv *Service) (int, error) {
	// 未授权
	if res, err := MongodbUnAuth(serv); res != -1 {
		return res, err
	}
	// 口令爆破
	url := fmt.Sprintf("mongodb://%v:%v@%v:%v/%v", serv.User, serv.Pass, serv.Ip, serv.Port, serv.User)
	session, err := mgo.Dial(url)
	if err != nil {
		if strings.Contains(err.Error(), "Authentication failed") {
			return CrackFail, nil
		}
		return CrackError, err
	}
	defer session.Close()
	err = session.Ping()
	if err != nil {
		return CrackFail, nil
	}
	return CrackSuccess, nil
}

var senddata = []byte{72, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 212, 7, 0, 0, 0, 0, 0, 0, 97, 100, 109, 105, 110, 46, 36, 99, 109, 100, 0, 0, 0, 0, 0, 1, 0, 0, 0, 33, 0, 0, 0, 2, 103, 101, 116, 76, 111, 103, 0, 16, 0, 0, 0, 115, 116, 97, 114, 116, 117, 112, 87, 97, 114, 110, 105, 110, 103, 115, 0, 0}

func MongodbUnAuth(serv *Service) (int, error) {
	addr := fmt.Sprintf("%v:%v", serv.Ip, serv.Port)
	conn, err := net.DialTimeout("tcp", addr, time.Duration(serv.Timeout)*time.Second)
	if err != nil {
		return CrackError, err
	}
	err = conn.SetDeadline(time.Now().Add(time.Duration(serv.Timeout) * time.Second))
	if err != nil {
		return CrackError, err
	}
	defer conn.Close()
	_, err = conn.Write(senddata)
	if err != nil {
		return CrackError, err
	}
	buf := make([]byte, 1024)
	count, err := conn.Read(buf)
	if err != nil {
		return CrackError, err
	}
	text := string(buf[0:count])
	if strings.Contains(text, "totalLinesWritten") {
		return CrackSuccess, nil
	}
	return -1, nil
}
