package common

import (
	"encoding/binary"
	"net"
	"strconv"
	"time"

	"gopkg.in/errgo.v2/fmt/errors"
)

func (f *TimestampModelField) Convert(value string) (interface{}, error) {

	t, err := time.Parse(f.Layout, value)
	if err != nil {
		return nil, err
	}

	return t.UTC().Unix(), nil
}

func (f *ListModelField) Convert(value string) (interface{}, error) {
	v, ok := f.Values[value]
	if ok {
		return v, nil
	}

	if f.Default != nil {
		return *f.Default, nil
	}

	return nil, errors.New("cannot find value for key, and no default value given")
}

func (s *StringModelField) Convert(value string) (interface{}, error) {
	return value, nil
}

func (s *IpToIntModelField) Convert(value string) (interface{}, error) {
	ip := net.ParseIP(value)
	if len(ip) == net.IPv6len {
		return binary.BigEndian.Uint32(ip[12:16]), nil
	}

	return binary.BigEndian.Uint32(ip), nil
}

func (s *Int16ModelField) Convert(value string) (interface{}, error) {
	if v, err := strconv.ParseInt(value, 10, 16); err != nil {
		return nil, err
	} else {
		return int16(v), nil
	}
}

func (s *UInt16ModelField) Convert(value string) (interface{}, error) {
	if v, err := strconv.ParseUint(value, 10, 16); err != nil {
		return nil, err
	} else {
		return uint16(v), nil
	}
}
