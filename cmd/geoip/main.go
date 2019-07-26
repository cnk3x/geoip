package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"shu.run/geoip"
	"shu.run/log"
)

func main() {
	log.SetLevel(log.LevelInfo)

	var lang string
	var ip string
	var ipList []string
	var err error
	var update bool

	flag.StringVar(&ip, "ip", "", "ip地址")
	flag.StringVar(&lang, "ln", "zh-CN", "语言")
	flag.BoolVar(&update, "update", false, "更新数据库")
	flag.Parse()

	if ip == "" {
		ipList, err = getIP()
	} else {
		ipList = strings.Split(ip, ",")
	}

	dir := os.Getenv("HOME")
	var geo = geoip.New(filepath.Join(dir, ".geoip"))
	err = geo.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer geo.Close()

	if update {
		errs := make(chan error)
		geo.Update(errs)
		select {
		case err = <-errs:
			if err != nil {
				log.Info(err)
				os.Exit(0)
			}
			log.Info("更新完成")
			os.Exit(0)
		}
	}

	ipList = removeRepByLoop(ipList)
	for _, ip := range ipList {
		ip = strings.TrimSpace(ip)
		ret, err := geo.Find(ip, lang)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(ret.String())
	}
}

func getIP() ([]string, error) {
	resp, err := http.Get("http://httpbin.org/ip")
	if err != nil {
		return nil, err
	}

	var origin ipOrigin
	err = json.NewDecoder(resp.Body).Decode(&origin)
	if err != nil {
		return nil, err
	}
	ipList := strings.Split(origin.Origin, ",")
	return ipList, nil
}

type ipOrigin struct {
	Origin string `json:"origin"`
}

func removeRepByLoop(slc []string) []string {
	result := []string{} // 存放结果
	for i := range slc {
		flag := true
		s := strings.TrimSpace(slc[i])
		for j := range result {
			if s == result[j] {
				flag = false // 存在重复元素，标识为false
				break
			}
		}
		if flag { // 标识为false，不添加进结果
			result = append(result, s)
		}
	}
	return result
}
