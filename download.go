package geoip

import (
	"io"
)

func copy(w io.Writer, r io.Reader, fb func(cur int64)) error {
	var buf = make([]byte, 512*1024)
	var cur int64

	for {
		lr, err := r.Read(buf)
		if lr > 0 {
			lw, err := w.Write(buf[0:lr])
			if err != nil {
				return err
			}

			//读取是数据长度不等于写入的数据长度
			if lr != lw {
				return io.ErrShortWrite
			}

			//数据长度大于0
			if lw > 0 && fb != nil {
				cur += int64(lw)
				go fb(cur)
			}
		}

		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}
