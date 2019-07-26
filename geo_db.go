package geoip

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"shu.run/log"

	"gopkg.in/cheggaaa/pb.v1"
)

const (
	// updateURL = "https://geolite.maxmind.com/download/geoip/database/GeoLite2-Country.tar.gz"
	updateURL = "https://geolite.maxmind.com/download/geoip/database/GeoLite2-City.tar.gz"
	lastest   = "lastest"
)

//New New
func New(dir string) *GeoDB {
	return &GeoDB{dir: dir}
}

//GeoDB mmdb Updater
type GeoDB struct {
	dir string
	geo *Reader
}

//Open Open
func (u *GeoDB) Open() error {
	errs := make(chan error, 1)
	log.Debug("链接数据库")
	ver, err := u.GetVersion()
	if err != nil {
		log.Debug("数据库不存在，更新:", err)
		u.Update(errs)
		return <-errs
	}

	target := filepath.Join(u.dir, ver+".mmdb")
	_, err = os.Stat(target)
	if err != nil {
		log.Debug("数据库不存在，更新:", err)
		u.Update(errs)
		return <-errs
	}

	return u.open(target)
}

//Update 开始更新
func (u *GeoDB) Update(errs chan error) {
	go u.update(errs)
}

//Find 查找
func (u *GeoDB) Find(ipString string, lang string) (*GeoResult, error) {
	res := &GeoResult{IP: ipString, Lang: lang}

	ip := net.ParseIP(res.IP)

	if ip.IsLoopback() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
		res.Code = "local"
		return res, nil
	}

	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 10 || (ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) || (ip4[0] == 192 && ip4[1] == 168) {
			res.Code = "internal"
			return res, nil
		}

		if u.geo == nil {
			return nil, fmt.Errorf("数据库正在更新，清稍后再试")
		}

		city, err := u.geo.City(ip)
		if err != nil {
			return nil, err
		}

		res.Code = "internet"
		if city.Continent.GeoNameID != 0 {
			res.Continent = &Name{Code: city.Continent.Code}
			res.Continent.SetName(res.Lang, city.Continent.Names)
		}

		if city.Country.GeoNameID != 0 {
			res.Country = &Name{Code: city.Country.IsoCode}
			res.Country.SetName(res.Lang, city.Country.Names)
		}

		if city.City.GeoNameID != 0 {
			res.City = &Name{Code: strconv.Itoa(int(city.City.GeoNameID))}
			res.City.SetName(res.Lang, city.City.Names)
		}

		return res, nil
	}

	return nil, errors.New("ip is not a v4 addr")
}

//SetVersion 保存版本
func (u *GeoDB) SetVersion(ver string) error {
	verFn := filepath.Join(u.dir, lastest)
	return ioutil.WriteFile(verFn, []byte(ver), 0666)
}

//GetVersion 获取版本
func (u *GeoDB) GetVersion() (string, error) {
	verFn := filepath.Join(u.dir, lastest)
	v, err := ioutil.ReadFile(verFn)
	if err != nil {
		return "", err
	}
	return string(v), nil
}

// Close unmaps the database file from virtual memory and returns the
// resources to the system.
func (u *GeoDB) Close() error {
	if u != nil && u.geo != nil {
		return u.geo.Close()
	}
	return nil
}

//Languages 语言
func (u *GeoDB) Languages() []string {
	return u.geo.Metadata().Languages
}

//DatabaseVersion 语言
func (u *GeoDB) DatabaseVersion() string {
	meta := u.geo.Metadata()
	return fmt.Sprintf("v%d.%d.%d", meta.BinaryFormatMajorVersion, meta.BinaryFormatMinorVersion, meta.BuildEpoch)
}

func (u *GeoDB) open(target string) (err error) {
	log.Debug("打开:", target)
	u.geo, err = openRead(target)
	if err != nil {
		log.Debug(err)
		return err
	}

	meta := u.geo.Metadata()
	log.Debug("支持语言:", meta.Languages)
	for ln, desc := range meta.Description {
		log.Debugf("描述[%s]:%s", ln, desc)
	}
	log.Debug("数据库类型:", meta.DatabaseType)
	log.Debugf("数据库版本:%d.%d.%d", meta.BinaryFormatMajorVersion, meta.BinaryFormatMinorVersion, meta.BuildEpoch)
	log.Debug("收录IP节点数量:", meta.NodeCount)
	log.Debug("IP版本:", meta.IPVersion)
	return
}

func (u *GeoDB) update(errs chan error) {
	log.Debug("开始更新数据库:", u.getDownloadURL())

	req, err := http.NewRequest("GET", u.getDownloadURL(), nil)
	if err != nil {
		errs <- err
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		errs <- err
		return
	}

	if resp.StatusCode != 200 {
		errs <- fmt.Errorf("http error.%s", resp.Status)
		return
	}

	if resp.Body == nil {
		errs <- fmt.Errorf("no body")
		return
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		errs <- err
		return
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				err = fmt.Errorf("更新源找不到mmdb文件，失败")
			}
			errs <- err
			return
		}

		if strings.HasSuffix(hdr.Name, ".mmdb") {
			ver := filepath.Dir(hdr.Name)
			target := filepath.Join(u.dir, ver+".mmdb")

			err = os.MkdirAll(u.dir, 0755)
			if err != nil {
				errs <- err
				return
			}

			fi, err := os.Stat(target)
			if err == nil || !os.IsNotExist(err) {
				if err != nil {
					errs <- err
					return
				}

				if fi.Size() == hdr.Size {
					err = fmt.Errorf("已存在最新数据库: %s", ver)
					errs <- err
					return
				}
				err = os.RemoveAll(target)
				errs <- err
				return
			}

			f, err := os.OpenFile(target, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0666)
			if err != nil {
				errs <- err
				return
			}
			bar := pb.New64(hdr.Size)
			bar.SetRefreshRate(time.Second)
			bar.SetUnits(pb.U_BYTES_DEC)
			bar.ShowFinalTime = true
			bar.ShowSpeed = true
			bar.ShowElapsedTime = true
			bar.Start()

			err = copy(f, tr, func(cur int64) {
				bar.Set64(cur)
			})
			bar.Set64(hdr.Size)
			if err != nil {
				bar.Finish()
				errs <- err
				return
			}
			bar.Finish()
			log.Debug("下载完成，更新版本")
			if err = u.SetVersion(ver); err == nil {
				u.Close()
				err = u.open(target)
			}
			if err != nil {
				errs <- err
				return
			}
			errs <- nil
			return
		}
	}
}

func (u *GeoDB) getDownloadURL() string {
	dlu := os.Getenv("GEO_DB")
	if dlu == "" {
		dlu = updateURL
	}
	return dlu
}
